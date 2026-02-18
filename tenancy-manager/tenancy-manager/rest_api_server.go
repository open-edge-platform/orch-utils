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
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	config_helper "github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/pkg/handlers"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/rs/zerolog"
)

var (
	restAPIAppName = "tenancy-manager-rest-api"
	restAPILog     = logging.GetLogger(restAPIAppName)
)

// RESTAPIServer represents the HTTP REST API server for tenant management.
type RESTAPIServer struct {
	echo        *echo.Echo
	config      *config_helper.HTTPServerConfig
	nexusClient *nexus_client.Clientset
	logger      zerolog.Logger
	authEnabled bool
}

// NewRESTAPIServer creates a new RESTAPIServer instance.
func NewRESTAPIServer(nexusClient *nexus_client.Clientset, config *config_helper.HTTPServerConfig, logger zerolog.Logger, authEnabled bool) *RESTAPIServer {
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

	return &RESTAPIServer{
		echo:        e,
		config:      config,
		nexusClient: nexusClient,
		logger:      logger,
		authEnabled: authEnabled,
	}
}

// RegisterRoutes registers all API routes.
func (s *RESTAPIServer) RegisterRoutes() {
	restAPILog.Info().Msg("Registering REST API routes")

	// Health check endpoints (no auth required)
	s.echo.GET("/health", s.healthHandler)
	s.echo.GET("/ready", s.readinessHandler)

	// API v1 routes (with optional authentication)
	v1 := s.echo.Group("/v1")

	if s.authEnabled {
		v1.Use(s.authenticationMiddleware)
	}

	// Organization routes
	orgs := v1.Group("/orgs")
	orgHandler := handlers.NewOrgHandler(s.nexusClient, s.logger)
	orgs.POST("", orgHandler.CreateOrg)
	orgs.GET("", orgHandler.ListOrgs)
	orgs.GET("/:orgId", orgHandler.GetOrg)
	orgs.PUT("/:orgId", orgHandler.UpdateOrg)
	orgs.DELETE("/:orgId", orgHandler.DeleteOrg)
	orgs.GET("/:orgId/status", orgHandler.GetOrgStatus)

	// Project routes
	projects := v1.Group("/projects")
	projectHandler := handlers.NewProjectHandler(s.nexusClient, s.logger)
	projects.POST("", projectHandler.CreateProject)
	projects.GET("", projectHandler.ListProjects)
	projects.GET("/:projectId", projectHandler.GetProject)
	projects.PUT("/:projectId", projectHandler.UpdateProject)
	projects.DELETE("/:projectId", projectHandler.DeleteProject)
	projects.GET("/:projectId/status", projectHandler.GetProjectStatus)

	restAPILog.Info().Msg("REST API routes registered successfully")
}

// authenticationMiddleware handles JWT authentication using the JWKS from the OIDC provider.
// Note: JWT validation is also performed by Traefik's validate-jwt middleware at the ingress level.
// This provides defense-in-depth for direct service access.
func (s *RESTAPIServer) authenticationMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Extract JWT token from Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			restAPILog.Warn().Msg("Missing authorization header")
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
		}

		// Extract token from "Bearer <token>"
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			restAPILog.Warn().Msg("Invalid authorization header format; expected 'Bearer <token>'")
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
		}

		// Validate JWT using the JwtAuthenticator (reads OIDC_SERVER_URL env var for JWKS)
		jwtAuth := new(auth.JwtAuthenticator)
		claims, err := jwtAuth.ParseAndValidate(token)
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
	})
}

// readinessHandler handles GET /ready - Readiness check endpoint.
func (s *RESTAPIServer) readinessHandler(c echo.Context) error {
	if s.nexusClient == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status":  "not ready",
			"service": "tenancy-manager",
			"reason":  "nexus client not initialized",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ready",
		"service": "tenancy-manager",
	})
}

// Start starts the HTTP REST API server.
func (s *RESTAPIServer) Start() error {
	address := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)
	restAPILog.Info().Msgf("Starting REST API server on %s", address)

	if s.config.TLSEnabled {
		if s.config.CertPath == "" || s.config.KeyPath == "" {
			return fmt.Errorf("TLS enabled but cert/key paths not configured")
		}
		restAPILog.Info().Msg("Starting TLS server")
		return s.echo.StartTLS(address, s.config.CertPath, s.config.KeyPath)
	}

	restAPILog.Info().Msg("Starting HTTP server")
	if err := s.echo.Start(address); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP REST API server.
func (s *RESTAPIServer) Shutdown(ctx context.Context) error {
	restAPILog.Info().Msg("Shutting down REST API server")
	return s.echo.Shutdown(ctx)
}

// startRESTAPIServer starts the REST API server in a goroutine.
func startRESTAPIServer(config *config_helper.HTTPServerConfig, nexusClient *nexus_client.Clientset, authEnabled bool) {
	if !config.Enabled {
		restAPILog.Info().Msg("REST API server is disabled")
		return
	}

	server := NewRESTAPIServer(nexusClient, config, restAPILog.Logger, authEnabled)
	server.RegisterRoutes()
	go func() {
		if err := server.Start(); err != nil {
			restAPILog.Fatal().Msgf("REST API server failed: %v", err)
		}
	}()

	restAPILog.Info().Msg("REST API server started successfully")
}
