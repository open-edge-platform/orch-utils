// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package echoserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/client"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/server/echoserver"
	nc "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	k8sFake "k8s.io/client-go/kubernetes/fake"
)

func setupServer() (*echoserver.EchoServer, chan struct{}) {
	model.URIToCRDType = URIToCRDType
	model.CrdTypeToNodeInfo = CrdTypeToNodeInfo

	stopCh := make(chan struct{})

	// Setup dynamic port allocation
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Get the dynamically allocated port
	addr := listener.Addr()
	if addr == nil {
		panic("failed to get listener address")
	}
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		panic("failed to cast listener address to TCPAddr")
	}
	port := tcpAddr.Port

	// Set the server configuration to use the dynamic port
	config.Cfg = &config.Config{
		Server: config.ServerConfig{
			HTTPPort: fmt.Sprintf("%d", port),
		},
	}
	k8sClient := k8sFake.NewSimpleClientset()
	nexusClient := nc.NewFakeClient()
	nexusClient.DynamicClient = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), GRVToListKind)
	client.Client = fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), GRVToListKind)

	serverObj := echoserver.InitEcho(stopCh, config.Cfg, k8sClient, nexusClient, nexusClient)
	serverObj.Authenticator = &MockAuthenticator{}
	serverObj.TenancyNexusClient = nexusClient

	return serverObj, stopCh
}

func teardownServer(serverObj *echoserver.EchoServer, stopCh chan struct{}) {
	err := serverObj.Echo.Shutdown(context.Background())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	close(stopCh)
}

var _ = ginkgo.Describe("EchoServer Handlers", ginkgo.Ordered, func() {
	ginkgo.Context("GetHandler Tests", ginkgo.Ordered, func() {
		ginkgo.When("Org does not exist", ginkgo.Ordered, func() {
			ginkgo.It("should fail to Get the org", func() {
				serverObj, stopCh := setupServer()
				defer teardownServer(serverObj, stopCh)

				rec := httptest.NewRecorder()
				c := serverObj.Echo.NewContext(httptest.NewRequest(http.MethodGet, "/", http.NoBody), rec)
				nc := &echoserver.NexusContext{
					Context:  c,
					NexusURI: "/v1/orgs/{org.Org}",
				}
				c.SetParamNames("org.Org")
				c.SetParamValues("getHandlerOrg1")
				err := serverObj.GetHandler(nc)

				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})

		ginkgo.When("Org exists", ginkgo.Ordered, func() {
			ginkgo.It("should GET the org successfully", func() {
				serverObj, stopCh := setupServer()
				defer teardownServer(serverObj, stopCh)

				// create org:
				orgUnstructObj, err := client.Client.Resource(constructOrgGVR()).
					Create(context.Background(),
						constructUnstructuredOrg("18a8a4294ab1ac866a53b7b5fe35421875af8be5"), metav1.CreateOptions{})
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(orgUnstructObj).NotTo(gomega.BeNil())

				// get org
				rec := httptest.NewRecorder()
				c := serverObj.Echo.NewContext(httptest.NewRequest(http.MethodGet, "/", http.NoBody), rec)
				c.SetParamNames("org.Org")
				c.SetParamValues("getHandlerOrg1")

				nc := &echoserver.NexusContext{
					Context:  c,
					NexusURI: "/v1/orgs/{org.Org}",
				}

				gomega.Eventually(func() int {
					err := serverObj.GetHandler(nc)
					if err != nil {
						log.Error().Err(err).Msg("GetHandler failed")
					}
					gomega.Expect(err).To(gomega.BeNil())
					log.Info().Msgf("get orgs Response code: %d\n", rec.Code)
					log.Info().Msgf("get orgs Response body: %s\n", rec.Body.String())
					return rec.Code
				}).WithTimeout(10 * time.Second).WithPolling(2 * time.Second).Should(gomega.Equal(http.StatusOK))
			})
		})
	})

	ginkgo.Context("PutHandler Tests", ginkgo.Ordered, func() {
		ginkgo.When("Org does not exist", ginkgo.Ordered, func() {
			ginkgo.It("Create project, should fail", func() {
				serverObj, stopCh := setupServer()
				defer teardownServer(serverObj, stopCh)

				rec := httptest.NewRecorder()
				gnsJSON := `{"description": "desc for project"}`
				req := httptest.NewRequest(http.MethodPut, "/v1/projects/{project.Project}", strings.NewReader(gnsJSON))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

				c := serverObj.Echo.NewContext(req, rec)
				nc := &echoserver.NexusContext{
					Context:  c,
					NexusURI: "/v1/projects/{project.Project}",
				}
				c.SetParamNames("project.Project")
				c.SetParamValues("proj1HashedName")

				err := serverObj.PutHandler(nc)
				gomega.Expect(err).To(gomega.BeNil())
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusConflict))
				var response map[string]interface{}
				err = json.Unmarshal(rec.Body.Bytes(), &response)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
			})
		})
	})

	ginkgo.Context("ListHandler Tests", ginkgo.Ordered, func() {
		ginkgo.It("List org should pass", func() {
			serverObj, stopCh := setupServer()
			defer teardownServer(serverObj, stopCh)

			rec := httptest.NewRecorder()
			c := serverObj.Echo.NewContext(httptest.NewRequest(http.MethodGet, "/v1/orgs", http.NoBody), rec)
			nc := &echoserver.NexusContext{
				Context:  c,
				NexusURI: "/v1/orgs",
			}

			err := serverObj.ListHandler(nc)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		})

		ginkgo.It("List project should pass", func() {
			serverObj, stopCh := setupServer()
			defer teardownServer(serverObj, stopCh)

			rec := httptest.NewRecorder()
			c := serverObj.Echo.NewContext(httptest.NewRequest(http.MethodGet, "/v1/projects", http.NoBody), rec)
			nc := &echoserver.NexusContext{
				Context:  c,
				NexusURI: "/v1/projects",
			}
			err := serverObj.ListHandler(nc)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
		})
	})
})
