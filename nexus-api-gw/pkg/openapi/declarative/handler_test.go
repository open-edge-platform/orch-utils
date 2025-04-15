// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package declarative_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/declarative"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/server/echoserver"
	nexusClient "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

var _ = ginkgo.BeforeSuite(func() {
	log.SetLevel(log.DebugLevel)
	err := declarative.Load(spec)
	gomega.Expect(err).To(gomega.BeNil())
})

var _ = ginkgo.Describe("Handler tests", ginkgo.Ordered, func() {
	ginkgo.It("should test ListHandler for gns list url", func() {
		ec := declarative.SetupContext(URI, http.MethodGet, declarative.Paths[URI].Get)

		// setup test http server for backend service calls
		var requestURI string
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			requestURI = req.URL.String()
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(`[]`))
		}))
		defer server.Close()
		config.Cfg = &config.Config{BackendService: server.URL}

		// setup echo test
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		ec.Context = c

		err := declarative.ListHandler(ec)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(rec.Body.String()).To(gomega.Equal("[]\n"))
		gomega.Expect(requestURI).To(gomega.Equal("/v1alpha1/project/default/global-namespaces"))
	})

	ginkgo.It("should test GetHandler for given gns id", func() {
		ec := declarative.SetupContext(ResourceURI, http.MethodGet, declarative.Paths[ResourceURI].Get)

		// setup test http server for backend service calls
		var requestURI string
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			requestURI = req.URL.String()
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(`{}`))
		}))
		defer server.Close()
		config.Cfg = &config.Config{BackendService: server.URL}

		// setup echo test
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:name")
		c.SetParamNames("name")
		c.SetParamValues("example-gns-id")
		ec.Context = c

		err := declarative.GetHandler(ec)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(rec.Body.String()).To(gomega.Equal("{}\n"))
		gomega.Expect(requestURI).To(gomega.Equal("/v1alpha1/project/default/global-namespaces/example-gns-id"))
	})

	ginkgo.It("should test PutHandler for given gns id", func() {
		ec := declarative.SetupContext(ResourceURI, http.MethodPut, declarative.Paths[ResourceURI].Put)
		gnsJSON := `{
    "metadata": {
        "name": "test"
    },
    "spec": {
        "foo": "bar"
    }
}`

		// setup test http server for backend service calls
		var requestURI string
		var requestBody string
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			requestURI = req.URL.String()
			if b, err := io.ReadAll(req.Body); err == nil {
				requestBody = string(b)
			}
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(`{}`))
		}))
		defer server.Close()
		config.Cfg = &config.Config{BackendService: server.URL}

		// setup echo test
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		ec.Context = c

		err := declarative.PutHandler(ec)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(rec.Body.String()).To(gomega.Equal("{}\n"))
		gomega.Expect(requestBody).To(gomega.Equal("{\"foo\":\"bar\"}"))
		gomega.Expect(requestURI).To(gomega.Equal("/v1alpha1/project/default/global-namespaces/test"))
	})

	ginkgo.It("should test PutHandler for given gns id with empty spec", func() {
		ec := declarative.SetupContext(ResourceURI, http.MethodPut, declarative.Paths[ResourceURI].Put)
		gnsJSON := `{
    "metadata": {
        "name": "test"
    }
}`

		// setup test http server for backend service calls
		var requestURI string
		var requestBody string
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			requestURI = req.URL.String()
			if b, err := io.ReadAll(req.Body); err == nil {
				requestBody = string(b)
			}
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(`{}`))
		}))
		defer server.Close()
		config.Cfg = &config.Config{BackendService: server.URL}

		// setup echo test
		e := echo.New()
		req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(gnsJSON))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		ec.Context = c

		err := declarative.PutHandler(ec)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(rec.Body.String()).To(gomega.Equal("{}\n"))
		gomega.Expect(requestBody).To(gomega.Equal(""))
		gomega.Expect(requestURI).To(gomega.Equal("/v1alpha1/project/default/global-namespaces/test"))
	})

	ginkgo.It("should test DeleteHandler for given gns id", func() {
		ec := declarative.SetupContext(ResourceURI, http.MethodDelete, declarative.Paths[ResourceURI].Delete)

		// setup test http server for backend service calls
		var requestURI string
		server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			requestURI = req.URL.String()
			res.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		config.Cfg = &config.Config{BackendService: server.URL}

		// setup echo test
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:name")
		c.SetParamNames("name")
		c.SetParamValues("example-gns-id")
		ec.Context = c

		err := declarative.DeleteHandler(ec)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		gomega.Expect(requestURI).To(gomega.Equal("/v1alpha1/project/default/global-namespaces/example-gns-id"))
	})

	ginkgo.It("should test buildUrlFromParams method with provided labels", func() {
		config.Cfg.BackendService = ""
		ec := declarative.SetupContext(ResourceURI, http.MethodGet, declarative.Paths[ResourceURI].Get)
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/?labelSelector=projectId=example-id", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:name")
		c.SetParamNames("name")
		c.SetParamValues("example-gns-id")
		ec.Context = c
		url, err := declarative.BuildURLFromParams(ec)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(url).To(gomega.Equal("/v1alpha1/project/example-id/global-namespaces/example-gns-id"))
	})

	ginkgo.It("should test buildUrlFromParams method without labels", func() {
		config.Cfg.BackendService = ""
		ec := declarative.SetupContext(ResourceURI, http.MethodGet, declarative.Paths[ResourceURI].Get)
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/:name")
		c.SetParamNames("name")
		c.SetParamValues("example-gns-id")
		ec.Context = c
		url, err := declarative.BuildURLFromParams(ec)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(url).To(gomega.Equal("/v1alpha1/project/default/global-namespaces/example-gns-id"))
	})

	ginkgo.It("should test buildUrlFromBody method with provided labels", func() {
		config.Cfg.BackendService = ""
		ec := declarative.SetupContext(ResourceURI, http.MethodPut, declarative.Paths[ResourceURI].Put)
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		ec.Context = c
		url, err := declarative.BuildURLFromBody(ec, map[string]interface{}{
			"name": "test",
			"labels": map[string]interface{}{
				"projectId": "example-id",
			},
		})
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(url).To(gomega.Equal("/v1alpha1/project/example-id/global-namespaces/test"))
	})

	ginkgo.It("should test buildUrlFromBody method without labels", func() {
		config.Cfg.BackendService = ""
		ec := declarative.SetupContext(ResourceURI, http.MethodPut, declarative.Paths[ResourceURI].Put)
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		ec.Context = c
		url, err := declarative.BuildURLFromBody(ec, map[string]interface{}{
			"name": "test",
		})
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(url).To(gomega.Equal("/v1alpha1/project/default/global-namespaces/test"))
	})

	ginkgo.It("should test Apis handler", func() {
		echoServer := echoserver.NewEchoServer(config.Cfg, &kubernetes.Clientset{},
			&nexusClient.Clientset{},
			&nexusClient.Clientset{},
		)
		echoServer.RegisterDeclarativeRouter()

		// setup echo test
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/declarative/apis", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/declarative/apis")

		err := declarative.ApisHandler(c)
		gomega.Expect(err).To(gomega.BeNil())
		expectedBody := `{
			"/apis/gns.vmware.org/v1/globalnamespacelists": {
				"GET": {
					"group": "gns.vmware.org",
					"kind": "GlobalNamespaceList",
					"params": [],
					"uri": "/v1alpha1/global-namespaces/test"
				},
				"short": {
					"name": "gns",
					"uri": "/apis/v1/gns"
				}
			},
			"/apis/gns.vmware.org/v1/globalnamespaces": {
				"GET": {
					"group": "gns.vmware.org",
					"kind": "GlobalNamespace",
					"params": ["projectId"],
					"uri": "/v1alpha1/project/{projectId}/global-namespaces"
				},
				"PUT": {
					"group": "gns.vmware.org",
					"kind": "GlobalNamespace",
					"params": ["projectId", "id"],
					"uri": "/v1alpha1/project/{projectId}/global-namespaces/{id}"
				},
				"short": {
					"name": "gns",
					"uri": "/apis/v1/gns"
				},
				"yaml": "apiVersion: gns.vmware.org/v1\nkind: GlobalNamespace\nmetadata:\n  labels:\n
				    projectId: string\n  name: string\nspec:\n  api_discovery_enabled: true\n  
					ca: string\n  ca_type: PreExistingCA\n  color: string\n  description: string\n  
					display_name: string\n  domain_name: string\n  match_conditions:\n  - cluster:\n    
					  match: string\n      type: string\n    namespace:\n      match: string\n      
					  type: string\n    service: object\n  mtls_enforced: true\n  name: string\n  
					  use_shared_gateway: true\n  version: string\n"
			},
			"/apis/gns.vmware.org/v1/globalnamespaces/:name": {
				"DELETE": {
					"group": "gns.vmware.org",
					"kind": "GlobalNamespace",
					"params": ["projectId", "id"],
					"uri": "/v1alpha1/project/{projectId}/global-namespaces/{id}"
				},
				"GET": {
					"group": "gns.vmware.org",
					"kind": "GlobalNamespace",
					"params": ["projectId", "id"],
					"uri": "/v1alpha1/project/{projectId}/global-namespaces/{id}"
				},
				"short": {
					"name": "gns",
					"uri": "/apis/v1/gns/:name"
				}
			}
		}`
		recBodyStr := NormalizeString(rec.Body.String())
		expectedStr := NormalizeString(expectedBody)

		gomega.Expect(recBodyStr).To(gomega.Equal(expectedStr))
	})

	ginkgo.It("should test Apis handler with globalnamespaces.gns.vmware.org crd", func() {
		echoServer := echoserver.NewEchoServer(config.Cfg, &kubernetes.Clientset{},
			&nexusClient.Clientset{}, &nexusClient.Clientset{},
		)
		echoServer.RegisterDeclarativeRouter()

		// setup echo test
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/declarative/apis?crd=globalnamespaces.gns.vmware.org", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/declarative/apis?crd=globalnamespaces.gns.vmware.org")

		err := declarative.ApisHandler(c)
		gomega.Expect(err).To(gomega.BeNil())

		expectedBody := `apiVersion: gns.vmware.org/v1
kind: GlobalNamespace
metadata:
  labels:
    projectId: string
  name: string
spec:
  api_discovery_enabled: true
  ca: string
  ca_type: PreExistingCA
  color: string
  description: string
  display_name: string
  domain_name: string
  match_conditions:
  - cluster:
      match: string
      type: string
    namespace:
      match: string
      type: string
    service: object
  mtls_enforced: true
  name: string
  use_shared_gateway: true
  version: string
`
		gomega.Expect(rec.Body.String()).To(gomega.Equal(expectedBody))
	})

	ginkgo.It("should test Apis handler with non-existent crd", func() {
		echoServer := echoserver.NewEchoServer(config.Cfg, &kubernetes.Clientset{},
			&nexusClient.Clientset{}, &nexusClient.Clientset{},
		)
		echoServer.RegisterDeclarativeRouter()

		// setup echo test
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/declarative/apis?crd=non-existent-crd", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/declarative/apis?crd=non-existent-crd")

		err := declarative.ApisHandler(c)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(rec.Code).To(gomega.Equal(http.StatusNotFound))
	})
})

// normalizeString removes all whitespace characters from the input string.
func NormalizeString(s string) string {
	// Replace all whitespace characters (including \n, \t, etc.) with an empty string
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, "")
}
