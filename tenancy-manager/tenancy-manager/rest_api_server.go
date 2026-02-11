/*
 * Copyright (C) 2025 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	config_helper "github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/handlers"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

var (
	restAPIAppName = "tenancy-manager-rest-api"
	restAPILog     = logging.GetLogger(restAPIAppName)
)

// RESTAPIServer represents the HTTP REST API server for tenant management.
type RESTAPIServer struct {
	Echo        *echo.Echo
	Config      *config_helper.HTTPServerConfig
	NexusClient *nexus_client.Clientset
}

// NewRESTAPIServer creates a new RESTAPIServer instance.
func NewRESTAPIServer(config *config_helper.HTTPServerConfig, nexusClient *nexus_client.Clientset) *RESTAPIServer {
	e := echo.New()

	// Hide banner
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "ActiveProjectID"},
	}))

	// Request timeout
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Second,
	}))

	server := &RESTAPIServer{
		Echo:        e,
		Config:      config,
		NexusClient: nexusClient,
	}

	server.RegisterRoutes()

	return server
}

// RegisterRoutes registers all API routes.
func (s *RESTAPIServer) RegisterRoutes() {
	restAPILog.Info().Msg("Registering REST API routes")

	// Health check endpoints (no auth required)
	s.Echo.GET("/health", s.healthHandler)
	s.Echo.GET("/readiness", s.readinessHandler)

	// API v1 routes (with authentication)
	v1 := s.Echo.Group("/v1")

	// Apply authentication middleware if AUTH is enabled
	if os.Getenv("ENABLE_AUTH") != "false" {
		v1.Use(s.authenticationMiddleware)
	}

	// Organization routes
	orgs := v1.Group("/orgs")
	orgHandler := handlers.NewOrgHandler(s.NexusClient)
	orgs.POST("", orgHandler.CreateOrg)
	orgs.GET("", orgHandler.ListOrgs)
	orgs.GET("/:orgId", orgHandler.GetOrg)
	orgs.PUT("/:orgId", orgHandler.UpdateOrg)
	orgs.DELETE("/:orgId", orgHandler.DeleteOrg)
	orgs.GET("/:orgId/status", orgHandler.GetOrgStatus)

	// Project routes
	projects := v1.Group("/projects")
	projectHandler := handlers.NewProjectHandler(s.NexusClient)
	projects.POST("", projectHandler.CreateProject)
	projects.GET("", projectHandler.ListProjects)
	projects.GET("/:projectId", projectHandler.GetProject)
	projects.PUT("/:projectId", projectHandler.UpdateProject)
	projects.DELETE("/:projectId", projectHandler.DeleteProject)
	projects.GET("/:projectId/status", projectHandler.GetProjectStatus)

	restAPILog.Info().Msg("REST API routes registered successfully")
}

// authenticationMiddleware handles JWT authentication.
func (s *RESTAPIServer) authenticationMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Extract JWT token from Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			restAPILog.Warn().Msg("Missing authorization header")
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
		}

		// Validate JWT token using orch-library
		oidcServerURL := os.Getenv("OIDC_SERVER_URL")
		if oidcServerURL == "" {
			oidcServerURL = "http://platform-keycloak.orch-platform.svc/realms/master"
		}

		tlsInsecureSkipVerify := os.Getenv("OIDC_TLS_INSECURE_SKIP_VERIFY") == "true"

		authenticator, err := auth.NewOIDCAuthenticator(oidcServerURL, tlsInsecureSkipVerify)
		if err != nil {
			restAPILog.Error().Msgf("Failed to create authenticator: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Authentication service error")
		}

		// Extract token from "Bearer <token>"
		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		// Verify token
		claims, err := authenticator.VerifyToken(c.Request().Context(), token)
		if err != nil {
			restAPILog.Warn().Msgf("Token verification failed: %v", err)
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired token")
		}

		// Store claims in context for downstream handlers
		c.Set("claims", claims)

		return next(c)
	}
}

// healthHandler handles GET /health - Health check endpoint.
func (s *RESTAPIServer) healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "tenancy-manager",
		"version": "1.0.0",
	})
}

// readinessHandler handles GET /readiness - Readiness check endpoint.
func (s *RESTAPIServer) readinessHandler(c echo.Context) error {
	// Check if nexus client is operational
	if s.NexusClient == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status":  "not ready",
			"service": "tenancy-manager",
			"reason":  "nexus client not initialized",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ready",
		"service": "tenancy-manager",
		"version": "1.0.0",
	})
}

// Start starts the HTTP REST API server.
func (s *RESTAPIServer) Start() error {
	address := fmt.Sprintf("%s:%s", s.Config.Host, s.Config.Port)
	restAPILog.Info().Msgf("Starting REST API server on %s", address)

	if s.Config.TLSEnabled {
		if s.Config.CertPath == "" || s.Config.KeyPath == "" {
			return fmt.Errorf("TLS enabled but cert/key paths not configured")
		}
		restAPILog.Info().Msg("Starting TLS server")
		return s.Echo.StartTLS(address, s.Config.CertPath, s.Config.KeyPath)
	}

	restAPILog.Info().Msg("Starting HTTP server")
	if err := s.Echo.Start(address); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP REST API server.
func (s *RESTAPIServer) Shutdown(ctx context.Context) error {
	restAPILog.Info().Msg("Shutting down REST API server")
	return s.Echo.Shutdown(ctx)
}

// startRESTAPIServer starts the REST API server in a goroutine.
func startRESTAPIServer(config *config_helper.HTTPServerConfig, nexusClient *nexus_client.Clientset) {
	if !config.Enabled {
		restAPILog.Info().Msg("REST API server is disabled")
		return
	}

	server := NewRESTAPIServer(config, nexusClient)
	go func() {
		if err := server.Start(); err != nil {
			restAPILog.Fatal().Msgf("REST API server failed: %v", err)
		}
	}()

	restAPILog.Info().Msg("REST API server started successfully")
}
