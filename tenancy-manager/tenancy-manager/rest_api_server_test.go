// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	httpconfig "github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/config"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func TestRESTAPIServer(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "REST API Server Suite")
}

var _ = ginkgo.Describe("RESTAPIServer", func() {
	var (
		server      *RESTAPIServer
		nexusClient *nexus_client.Clientset
		config      *httpconfig.HTTPServerConfig
		logger      zerolog.Logger
	)

	ginkgo.BeforeEach(func() {
		nexusClient = nexus_client.NewFakeClient()
		nexusClient.DynamicClient = fake.NewSimpleDynamicClient(runtime.NewScheme())
		logger = zerolog.Nop()
		config = &httpconfig.HTTPServerConfig{
			Enabled:    true,
			Host:       "localhost",
			Port:       "8080",
			TLSEnabled: false,
		}
	})

	ginkgo.Context("NewRESTAPIServer", func() {
		ginkgo.When("creating a new server", func() {
			ginkgo.It("should initialize successfully", func() {
				server = NewRESTAPIServer(nexusClient, config, logger, false)

				gomega.Expect(server).NotTo(gomega.BeNil())
				gomega.Expect(server.echo).NotTo(gomega.BeNil())
				gomega.Expect(server.config).To(gomega.Equal(config))
				gomega.Expect(server.nexusClient).To(gomega.Equal(nexusClient))
			})
		})

		ginkgo.When("authentication is enabled", func() {
			ginkgo.It("should create server with auth enabled", func() {
				server = NewRESTAPIServer(nexusClient, config, logger, true)

				gomega.Expect(server).NotTo(gomega.BeNil())
				gomega.Expect(server.authEnabled).To(gomega.BeTrue())
			})
		})

		ginkgo.When("authentication is disabled", func() {
			ginkgo.It("should create server with auth disabled", func() {
				server = NewRESTAPIServer(nexusClient, config, logger, false)

				gomega.Expect(server).NotTo(gomega.BeNil())
				gomega.Expect(server.authEnabled).To(gomega.BeFalse())
			})
		})
	})

	ginkgo.Context("RegisterRoutes", func() {
		ginkgo.BeforeEach(func() {
			server = NewRESTAPIServer(nexusClient, config, logger, false)
		})

		ginkgo.When("routes are registered", func() {
			ginkgo.It("should have health endpoint", func() {
				server.RegisterRoutes()

				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				rec := httptest.NewRecorder()
				server.echo.ServeHTTP(rec, req)

				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
			})

			ginkgo.It("should have readiness endpoint", func() {
				server.RegisterRoutes()

				req := httptest.NewRequest(http.MethodGet, "/ready", nil)
				rec := httptest.NewRecorder()
				server.echo.ServeHTTP(rec, req)

				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
			})
		})
	})

	ginkgo.Context("Health Endpoint", func() {
		ginkgo.BeforeEach(func() {
			server = NewRESTAPIServer(nexusClient, config, logger, false)
			server.RegisterRoutes()
		})

		ginkgo.When("health check is requested", func() {
			ginkgo.It("should return healthy status", func() {
				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				rec := httptest.NewRecorder()
				server.echo.ServeHTTP(rec, req)

				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
				gomega.Expect(rec.Body.String()).To(gomega.ContainSubstring("healthy"))
			})
		})
	})

	ginkgo.Context("Readiness Endpoint", func() {
		ginkgo.BeforeEach(func() {
			server = NewRESTAPIServer(nexusClient, config, logger, false)
			server.RegisterRoutes()
		})

		ginkgo.When("readiness check is requested", func() {
			ginkgo.It("should return ready status", func() {
				req := httptest.NewRequest(http.MethodGet, "/ready", nil)
				rec := httptest.NewRecorder()
				server.echo.ServeHTTP(rec, req)

				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
				gomega.Expect(rec.Body.String()).To(gomega.ContainSubstring("ready"))
			})
		})
	})

	ginkgo.Context("Middleware", func() {
		ginkgo.BeforeEach(func() {
			server = NewRESTAPIServer(nexusClient, config, logger, false)
			server.RegisterRoutes()
		})

		ginkgo.When("request is made", func() {
			ginkgo.It("should have CORS middleware applied", func() {
				req := httptest.NewRequest(http.MethodOptions, "/health", nil)
				req.Header.Set("Origin", "http://localhost:3000")
				req.Header.Set("Access-Control-Request-Method", "GET")
				rec := httptest.NewRecorder()
				server.echo.ServeHTTP(rec, req)

				// CORS should add appropriate headers
				gomega.Expect(rec.Header().Get("Access-Control-Allow-Origin")).NotTo(gomega.BeEmpty())
			})

			ginkgo.It("should handle timeout middleware", func() {
				// Add a test route that takes longer than timeout
				server.echo.GET("/slow", func(c echo.Context) error {
					time.Sleep(100 * time.Millisecond)
					return c.JSON(http.StatusOK, map[string]string{"message": "slow"})
				})

				req := httptest.NewRequest(http.MethodGet, "/slow", nil)
				rec := httptest.NewRecorder()
				server.echo.ServeHTTP(rec, req)

				// Should complete without timeout for short durations
				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusOK))
			})
		})
	})

	ginkgo.Context("Start and Shutdown", func() {
		ginkgo.BeforeEach(func() {
			config.Port = "0" // Use random available port
			server = NewRESTAPIServer(nexusClient, config, logger, false)
			server.RegisterRoutes()
		})

		ginkgo.When("server starts and stops", func() {
			ginkgo.It("should start and shutdown gracefully", func() {
				// Start server in background
				go func() {
					_ = server.Start()
				}()

				// Give it a moment to start
				time.Sleep(100 * time.Millisecond)

				// Shutdown gracefully
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := server.Shutdown(ctx)

				gomega.Expect(err).NotTo(gomega.HaveOccurred())
			})
		})
	})

	ginkgo.Context("TLS Configuration", func() {
		ginkgo.When("TLS is enabled", func() {
			ginkgo.It("should recognize TLS configuration", func() {
				tlsConfig := &httpconfig.HTTPServerConfig{
					Enabled:    true,
					Host:       "localhost",
					Port:       "8443",
					TLSEnabled: true,
				}
				server = NewRESTAPIServer(nexusClient, tlsConfig, logger, false)

				gomega.Expect(server.config.TLSEnabled).To(gomega.BeTrue())
			})
		})

		ginkgo.When("TLS is disabled", func() {
			ginkgo.It("should use HTTP", func() {
				server = NewRESTAPIServer(nexusClient, config, logger, false)

				gomega.Expect(server.config.TLSEnabled).To(gomega.BeFalse())
			})
		})
	})

	ginkgo.Context("Error Handling", func() {
		ginkgo.BeforeEach(func() {
			server = NewRESTAPIServer(nexusClient, config, logger, false)
			server.RegisterRoutes()
		})

		ginkgo.When("404 route is accessed", func() {
			ginkgo.It("should return not found", func() {
				req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
				rec := httptest.NewRecorder()
				server.echo.ServeHTTP(rec, req)

				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusNotFound))
			})
		})

		ginkgo.When("method not allowed", func() {
			ginkgo.It("should return method not allowed", func() {
				req := httptest.NewRequest(http.MethodPost, "/health", nil)
				rec := httptest.NewRecorder()
				server.echo.ServeHTTP(rec, req)

				gomega.Expect(rec.Code).To(gomega.Equal(http.StatusMethodNotAllowed))
			})
		})
	})
})
