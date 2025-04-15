// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package echoserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/apiremap"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/auth"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/auth/authn"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/common"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/openapi/api"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/openapi/declarative"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/reconciler"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/utils"
	nexusClient "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/vmware-tanzu/graph-framework-for-microservices/nexus/nexus"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/auditing"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	stopServerTimerTimeout = 30 * time.Second
	stopServerCtxTimeout   = 10 * time.Second
	stopServerWaitTime     = 100 * time.Millisecond
	tmReconcileTime        = 600 * time.Second
)

var (
	appName = "nexus-api-gw-echoserver"
	log     = logging.GetLogger(appName)
)

type TenantData struct {
	TenantName string `json:"tenantName" form:"tenantName"`
	Sku        string `json:"sku,omitempty" form:"sku,omitempty"`
}

type UserLogin struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

var (
	TotalHTTPServerRestartCounter                 = 0
	HTTPServerRestartFromOpenAPISpecUpdateCounter = 0
)

type EchoServer struct {
	Echo               *echo.Echo
	Config             *config.Config
	Client             KubernetesClient
	NexusClient        *nexusClient.Clientset
	TenancyNexusClient *nexusClient.Clientset
	k8sProxy           *httputil.ReverseProxy
	Authenticator      auth.Authenticator
	mu                 sync.Mutex
	restartMu          sync.Mutex // Mutex to protect restart counter
}

type KubernetesClient interface {
	CoreV1() corev1client.CoreV1Interface
}

func InitEcho(stopCh chan struct{}, conf *config.Config, client KubernetesClient,
	nc *nexusClient.Clientset, tenancyNC *nexusClient.Clientset,
) *EchoServer {
	log.Info().Msg("Init Echo")
	e := NewEchoServer(conf, client, nc, tenancyNC)

	if conf.EnableNexusRuntime {
		e.RegisterNexusRoutes()
	}

	if conf.BackendService != "" {
		if err := declarative.Setup(declarative.OpenAPISpecFile); err != nil {
			log.Fatal().Msgf("unable to complete setup, %s", err.Error())
		}

		e.RegisterDeclarativeRoutes()
		e.RegisterDeclarativeRouter()
	}

	if conf.TenancyService {
		log.Info().Msg("Build tenancy API mapping Cache....")
		apiremap.SubscribeToAPIMappingsEvents(tenancyNC)
		e.RegisterTenancyRoutes()
	}
	common.TENANCY = conf.TenancyService
	common.AUTHZDISABLED = conf.DisableAuthz
	tDM := reconciler.NewTenancyManager(tenancyNC, tmReconcileTime)
	tDM.TenancyDmInit()
	go func() {
		if err := tDM.Start(context.Background()); err != nil {
			log.InfraErr(err).Msg("Error starting tDM")
		}
	}()

	e.RegisterDebug()
	e.Start(stopCh)

	if config.Cfg.BackendService != "" {
		WatchForOpenAPISpecChanges(stopCh, declarative.OpenAPISpecDir, declarative.OpenAPISpecFile, e)
	}
	return e
}

func (s *EchoServer) StartHTTPServer() {
	port := "80"
	if s.Config.Server.HTTPPort != "" {
		port = s.Config.Server.HTTPPort
	}

	if err := s.Echo.Start(fmt.Sprintf(":%s", port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Msgf("Server error %v", err)
	}
}

func (s *EchoServer) Start(stopCh chan struct{}) {
	go func() {
		s.mu.Lock()         // Lock the mutex before starting the server
		defer s.mu.Unlock() // Ensure the mutex is unlocked when the function returns

		if s.Config.EnableNexusRuntime {
			// Start watching URI notification
			go func() {
				log.Debug().Msg("NodeUpdateNotifications.. restarting server")
				if err := s.NodeUpdateNotifications(stopCh); err != nil {
					s.mu.Lock()
					s.StopServer()
					InitEcho(stopCh, s.Config, s.Client, s.NexusClient, s.TenancyNexusClient)
					s.restartMu.Lock()
					TotalHTTPServerRestartCounter++
					log.Info().Msgf("TotalHTTPServerRestartCounter: %d", TotalHTTPServerRestartCounter)
					s.restartMu.Unlock()
					s.mu.Unlock()
				}
			}()
		}

		// Start Server
		go func() {
			log.Info().Msg("Start Echo Server")
			if utils.IsServerConfigValid(s.Config) &&
				utils.IsFileExists(s.Config.Server.CertPath) &&
				utils.IsFileExists(s.Config.Server.KeyPath) {
				log.Info().Msgf("Server Config %v", s.Config.Server)
				log.Info().Msg("Start TLS Server")
				err := s.Echo.StartTLS(s.Config.Server.Address, s.Config.Server.CertPath, s.Config.Server.KeyPath)
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatal().Msgf("TLS Server error %v", err)
				}
			} else {
				log.Info().Msg("Certificates or TLS port not configured correctly, hence starting the HTTP Server")
				s.StartHTTPServer()
			}
		}()
	}()
}

type NexusContext struct {
	echo.Context
	NexusURI string
	Codes    nexus.HTTPCodesResponse

	// Kube
	CrdType   string
	GroupName string
	Resource  string
}

func (s *EchoServer) RegisterDebug() {
	s.Echo.GET("/debug/all", DebugAllHandler)
}

func (s *EchoServer) RegisterTenancyRoutes() {
	log.Info().Msg("Registering the routes")
	urlPattern := "/v*/projects/*"

	// Registering route with any method
	s.Echo.Any(urlPattern, s.tenancyapiHandler)
}

func (s *EchoServer) RegisterNexusRoutes() {
	// OpenAPI route
	s.Echo.GET("/:datamodel/openapi.json", func(c echo.Context) error {
		return c.JSON(http.StatusOK, api.Schemas[c.Param("datamodel")])
	})

	// Swagger-UI, datamodel is edge-orchestrator.intel.com
	s.Echo.GET("/:datamodel/docs", SwaggerUI)
}

func (s *EchoServer) RegisterDeclarativeRoutes() {
	s.Echo.GET("/declarative/apis", declarative.ApisHandler)
}

func (s *EchoServer) RegisterRouter(restURI nexus.RestURIs) {
	urlPattern := model.ConstructEchoPathParamURL(restURI.Uri)
	for method, codes := range restURI.Methods {
		log.Info().Msgf("Registered Router Path %s Method %s\n", urlPattern, method)
		nexusContext := s.GetNexusContext(restURI, codes)
		s.registerRoute(string(method), urlPattern, nexusContext)
	}
}

func (s *EchoServer) registerRoute(method, urlPattern string, nexusContext func(next echo.HandlerFunc) echo.HandlerFunc) {
	switch method {
	case "LIST":
		s.registerListRoute(urlPattern, nexusContext)
	case http.MethodGet:
		s.registerGetRoute(urlPattern, nexusContext)
	case http.MethodPut:
		s.registerPutRoute(urlPattern, nexusContext)
	case http.MethodPatch:
		s.registerPatchRoute(urlPattern, nexusContext)
	case http.MethodDelete:
		s.registerDeleteRoute(urlPattern, nexusContext)
	}
}

func (s *EchoServer) registerListRoute(urlPattern string, nexusContext func(next echo.HandlerFunc) echo.HandlerFunc) {
	if common.IsModeAdmin() || common.IsTenancyMode() {
		s.Echo.GET(urlPattern, s.ListHandler, nexusContext)
	} else {
		s.Echo.GET(urlPattern, s.ListHandler, authn.VerifyAuthenticationMiddleware, nexusContext)
	}
}

func (s *EchoServer) registerGetRoute(urlPattern string, nexusContext func(next echo.HandlerFunc) echo.HandlerFunc) {
	if common.IsModeAdmin() || common.IsTenancyMode() {
		s.Echo.GET(urlPattern, s.GetHandler, nexusContext)
	} else {
		s.Echo.GET(urlPattern, s.GetHandler, authn.VerifyAuthenticationMiddleware, nexusContext)
	}
}

func (s *EchoServer) registerPutRoute(urlPattern string, nexusContext func(next echo.HandlerFunc) echo.HandlerFunc) {
	if common.IsModeAdmin() || common.IsTenancyMode() {
		s.Echo.PUT(urlPattern, s.PutHandler, nexusContext)
	} else {
		s.Echo.PUT(urlPattern, s.PutHandler, authn.VerifyAuthenticationMiddleware, nexusContext)
	}
}

func (s *EchoServer) registerPatchRoute(urlPattern string, nexusContext func(next echo.HandlerFunc) echo.HandlerFunc) {
	if common.IsModeAdmin() || common.IsTenancyMode() {
		s.Echo.PATCH(urlPattern, s.PatchHandler, nexusContext)
	} else {
		s.Echo.PATCH(urlPattern, s.PatchHandler, authn.VerifyAuthenticationMiddleware, nexusContext)
	}
}

func (s *EchoServer) registerDeleteRoute(urlPattern string, nexusContext func(next echo.HandlerFunc) echo.HandlerFunc) {
	if common.IsModeAdmin() || common.IsTenancyMode() {
		s.Echo.DELETE(urlPattern, s.deleteHandler, nexusContext)
	} else {
		s.Echo.DELETE(urlPattern, s.deleteHandler, authn.VerifyAuthenticationMiddleware, nexusContext)
	}
}

func (s *EchoServer) GetNexusContext(restURI nexus.RestURIs,
	codes nexus.HTTPCodesResponse,
) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			nc := &NexusContext{
				Context:  c,
				NexusURI: restURI.Uri,
				Codes:    codes,
			}
			return next(nc)
		}
	}
}

func (s *EchoServer) GetNexusCrdContext(crdType, groupName, resource string) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			nc := &NexusContext{
				Context:   c,
				CrdType:   crdType,
				GroupName: groupName,
				Resource:  resource,
			}
			return next(nc)
		}
	}
}

func (s *EchoServer) RegisterCrdRouter(crdType string) {
	crdParts := strings.Split(crdType, ".")
	groupName := strings.Join(crdParts[1:], ".")
	resourcePattern := fmt.Sprintf("/apis/%s/v1/%s", groupName, crdParts[0])
	resourceNamePattern := resourcePattern + "/:name"
	crdContext := s.GetNexusCrdContext(crdType, groupName, crdParts[0])

	// TODO NPT-313 support authentication for kubectl proxy requests
	s.Echo.GET(resourceNamePattern, KubeGetByNameHandler, crdContext)
	s.Echo.GET(resourcePattern, KubeGetHandler, crdContext)
	s.Echo.POST(resourcePattern, KubePostHandler, crdContext)
	s.Echo.DELETE(resourceNamePattern, KubeDeleteHandler, crdContext)
}

func (s *EchoServer) RegisterDeclarativeRouter() {
	for uri, path := range declarative.Paths {
		if path.Get != nil {
			endpointContext := declarative.SetupContext(uri, http.MethodGet, path.Get)

			if endpointContext.Single {
				s.Echo.GET(endpointContext.URI, declarative.GetHandler, declarative.Middleware(endpointContext, true))
				if endpointContext.ShortURI != "" {
					s.Echo.GET(endpointContext.ShortURI, declarative.GetHandler, declarative.Middleware(endpointContext, true))
					log.Debug().Msgf("Registered declarative short get endpoint: %s for uri: %s", endpointContext.ShortURI, uri)
				}

				declarative.AddApisEndpoint(endpointContext)
				log.Debug().Msgf("Registered declarative get endpoint: %s for uri: %s", endpointContext.URI, uri)
			} else {
				s.Echo.GET(endpointContext.URI, declarative.ListHandler, declarative.Middleware(endpointContext, false))
				if endpointContext.ShortURI != "" {
					s.Echo.GET(endpointContext.ShortURI, declarative.ListHandler, declarative.Middleware(endpointContext, false))
					log.Debug().Msgf("Registered declarative short list endpoint: %s for uri: %s", endpointContext.ShortURI, uri)
				}

				declarative.AddApisEndpoint(endpointContext)
				log.Debug().Msgf("Registered declarative list endpoint: %s for uri: %s", endpointContext.URI, uri)
			}
		}

		if path.Put != nil {
			endpointContext := declarative.SetupContext(uri, http.MethodPut, path.Put)
			s.Echo.PUT(endpointContext.URI, declarative.PutHandler, declarative.Middleware(endpointContext, false))
			if endpointContext.ShortURI != "" {
				s.Echo.PUT(endpointContext.ShortURI, declarative.PutHandler, declarative.Middleware(endpointContext, false))
				log.Debug().Msgf("Registered declarative short put endpoint: %s for uri: %s", endpointContext.ShortURI, uri)
			}

			declarative.AddApisEndpoint(endpointContext)
			log.Debug().Msgf("Registered declarative put endpoint: %s for uri: %s", endpointContext.URI, uri)
		}

		if path.Delete != nil {
			endpointContext := declarative.SetupContext(uri, http.MethodDelete, path.Delete)
			s.Echo.DELETE(endpointContext.URI, declarative.DeleteHandler, declarative.Middleware(endpointContext, true))
			if endpointContext.ShortURI != "" {
				s.Echo.DELETE(endpointContext.ShortURI, declarative.DeleteHandler, declarative.Middleware(endpointContext, true))
				log.Debug().Msgf("Registered declarative short delete endpoint: %s for uri: %s", endpointContext.ShortURI, uri)
			}

			declarative.AddApisEndpoint(endpointContext)
			log.Debug().Msgf("Registered declarative delete endpoint: %s for uri: %s", endpointContext.URI, uri)
		}
	}
}

func (s *EchoServer) NodeUpdateNotifications(stopCh chan struct{}) error {
	for {
		select {
		case <-stopCh:
			return fmt.Errorf("stop signal received")
		case restURIs := <-model.RestURIChan:
			log.Debug().Msg("Rest route notification received")
			for _, v := range restURIs {
				if httpCodesResponse, ok := v.Methods[http.MethodPut]; ok {
					v.Methods[http.MethodPatch] = httpCodesResponse
				}
				s.RegisterRouter(v)
			}
		case crdType := <-model.CrdTypeChan:
			log.Debug().Msg("CRD route notification received")
			s.RegisterCrdRouter(crdType)
		}
	}
}

func (s *EchoServer) StopServer() {
	log.Debug().Msg("StopServer invoked")
	ctx, cancel := context.WithTimeout(context.Background(), stopServerCtxTimeout)
	defer cancel()
	if err := s.Echo.Shutdown(ctx); err != nil {
		log.InfraErr(err).Msg("Shutdown signal received")
		return
	}

	log.Debug().Msg("Server exiting")
	address := ":80"
	if s.Config.Server.HTTPPort != "" {
		address = ":" + s.Config.Server.HTTPPort
	}

	if utils.IsServerConfigValid(s.Config) &&
		utils.IsFileExists(s.Config.Server.CertPath) &&
		utils.IsFileExists(s.Config.Server.KeyPath) {
		address = s.Config.Server.Address
	}

	ok := false
	timeout := time.Now().Add(stopServerTimerTimeout)
	for time.Now().Before(timeout) {
		conn, err := net.DialTimeout("tcp", address, stopServerWaitTime)
		if err != nil {
			// informative log. When port is free then error will occur
			log.Debug().Msgf("StopServer: DialTimeout err: %v\n", err)
		}

		if conn == nil {
			ok = true
			break
		}
		conn.Close()
		time.Sleep(stopServerWaitTime)
	}
	if !ok {
		log.InfraError("Error occurred while stopping echo server. TCP port is busy").Msg("")
		return
	}
	log.Debug().Msg("StopServer: success")
}

func NewEchoServer(conf *config.Config, client KubernetesClient,
	nc *nexusClient.Clientset, tenancyNexusClient *nexusClient.Clientset,
) *EchoServer {
	e := echo.New()

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.CORS())
	var k8sProxy *httputil.ReverseProxy
	if conf.EnableNexusRuntime {
		// Setup proxy to api server
		k8sProxy = kubeSetupProxy(e)
	}

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "ACCESS[${time_rfc3339}] method=${method}, uri=${uri}, status=${status}\n",
	}))
	e.Use(auditing.RestEchoMiddleware)

	return &EchoServer{
		// create a new echo_server instance
		Echo:               e,
		Config:             conf,
		Client:             client,
		NexusClient:        nc,
		TenancyNexusClient: tenancyNexusClient,
		k8sProxy:           k8sProxy,
		Authenticator:      &auth.DefaultAuthenticator{},
	}
}

type AssignedInstance struct {
	URL string `json:"url"`
}
type Services struct {
	AssignedInstance []AssignedInstance `json:"allOrgInstances"`
}
type Results struct {
	Services []Services `json:"services"`
}

func WatchForOpenAPISpecChanges(stopCh chan struct{}, openAPISpecDir, openAPISpecFile string, server *EchoServer) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.InfraErr(err).Msg("NewWatcher failed")
		return
	}
	defer watcher.Close()

	go func() {
		for {
			if err := watchOpenAPISpecFile(watcher, stopCh, openAPISpecDir, openAPISpecFile, server); err != nil {
				log.InfraErr(err).Msg("Error watching OpenAPI spec file")
				return
			}
		}
	}()
}

func watchOpenAPISpecFile(watcher *fsnotify.Watcher, stopCh chan struct{},
	openAPISpecDir, openAPISpecFile string, server *EchoServer,
) error {
	_, err := os.Stat(openAPISpecFile)
	if err != nil {
		return watchDirectory(watcher, stopCh, openAPISpecDir, openAPISpecFile, server)
	}
	return watchFile(watcher, stopCh, openAPISpecFile, server)
}

func watchDirectory(watcher *fsnotify.Watcher, stopCh chan struct{},
	openAPISpecDir, openAPISpecFile string, server *EchoServer,
) error {
	if err := watcher.Add(openAPISpecDir); err != nil {
		log.Panic().Msgf("Unable to add watcher for %v: %v", openAPISpecDir, err.Error())
	}
	log.Debug().Msgf("Watching: %v", openAPISpecDir)

	for {
		select {
		case event := <-watcher.Events:
			if handleDirectoryEvent(event, stopCh, openAPISpecFile, server) {
				return nil
			}
		case err := <-watcher.Errors:
			if err != nil {
				log.InfraErr(err).Msg("")
				return err
			}
		}
	}
}

func watchFile(watcher *fsnotify.Watcher, stopCh chan struct{}, openAPISpecFile string, server *EchoServer) error {
	if err := watcher.Add(openAPISpecFile); err != nil {
		log.Panic().Msgf("Unable to add watcher for %v: %v", openAPISpecFile, err.Error())
	}
	log.Debug().Msgf("Watching: %v", openAPISpecFile)

	for {
		select {
		case event := <-watcher.Events:
			if handleFileEvent(event, stopCh, openAPISpecFile, server) {
				return nil
			}
		case err := <-watcher.Errors:
			if err != nil {
				log.InfraErr(err).Msg("")
				return err
			}
		}
	}
}

func handleDirectoryEvent(event fsnotify.Event, stopCh chan struct{}, openAPISpecFile string, server *EchoServer) bool {
	if event.Op == fsnotify.Create && event.Name == openAPISpecFile {
		log.Debug().Msg("Restarting echo server because openApi spec file is created")
		stopCh <- struct{}{}
		server.restartMu.Lock()
		HTTPServerRestartFromOpenAPISpecUpdateCounter++
		log.Info().Msgf("HTTPServerRestartFromOpenAPISpecUpdateCounter: %d", HTTPServerRestartFromOpenAPISpecUpdateCounter)
		server.restartMu.Unlock()
		return true
	}
	log.Trace().Msgf("Received Event on dir watch: %v on file %v", event.Op.String(), event.Name)
	return false
}

func handleFileEvent(event fsnotify.Event, stopCh chan struct{}, openAPISpecFile string, server *EchoServer) bool {
	if event.Op == fsnotify.Write && event.Name == openAPISpecFile {
		log.Debug().Msg("Restarting echo server because openApi spec file is updated")
		stopCh <- struct{}{}
		server.restartMu.Lock()
		HTTPServerRestartFromOpenAPISpecUpdateCounter++
		log.Info().Msgf("HTTPServerRestartFromOpenAPISpecUpdateCounter: %d", HTTPServerRestartFromOpenAPISpecUpdateCounter)
		server.restartMu.Unlock()
		return true
	}
	if event.Op == fsnotify.Remove && event.Name == openAPISpecFile {
		log.Debug().Msg("Restarting echo server because openApi spec file is removed")
		stopCh <- struct{}{}
		server.restartMu.Lock()
		HTTPServerRestartFromOpenAPISpecUpdateCounter++
		log.Info().Msgf("HTTPServerRestartFromOpenAPISpecUpdateCounter: %d", HTTPServerRestartFromOpenAPISpecUpdateCounter)
		server.restartMu.Unlock()
		return true
	}
	log.Trace().Msgf("Received Event on file watch: %v on file %v", event.Op.String(), event.Name)
	return false
}
