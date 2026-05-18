// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/api"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/bootstrap"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/store"
)

func main() {
	configPath := flag.String("config", "/etc/config/config.yaml", "path to config file")
	dbURL := flag.String("database-url", "", "database URL (overrides config)")
	listenAddr := flag.String("listen", "", "listen address (overrides config)")
	logLevel := flag.String("log-level", "info", "log level (debug, info, warn, error)")
	flag.Parse()

	// Logging setup.
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Load config.
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load config file, using defaults")
		cfg = config.DefaultConfig()
	}

	// Apply flag overrides.
	if *dbURL != "" {
		cfg.DatabaseURL = *dbURL
	}
	if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
		cfg.DatabaseURL = envDB
	}
	if *listenAddr != "" {
		cfg.ListenAddr = *listenAddr
	}

	log.Info().
		Str("listen", cfg.ListenAddr).
		Str("database", cfg.RedactedDatabaseURL()).
		Int("org_controllers", len(cfg.Controllers.Org)).
		Int("project_controllers", len(cfg.Controllers.Project)).
		Msg("starting tenant manager")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize store (connects to DB, runs migrations).
	s, err := store.New(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize store")
	}
	defer s.Close()

	// Bootstrap default org/project.
	if err := s.Bootstrap(ctx, cfg.DefaultOrgName, cfg.DefaultProjectName); err != nil {
		log.Fatal().Err(err).Msg("bootstrap failed")
	}

	// Tenant-admin bootstrap (creates Keycloak user + groups for the default
	// org/project when EMF_DEFAULT_TENANCY=true → BOOTSTRAP_TENANT_ADMIN_ENABLED=true).
	// Runs asynchronously: it must wait for keycloak-tenant-controller to
	// report the org/project IDLE, which can take a few minutes after first
	// install. Failure is logged but not fatal — org/project still exist and
	// an operator can re-trigger by restarting the pod.
	bcfg := bootstrap.LoadConfig()
	if bcfg.Enabled && cfg.DefaultOrgName != "" && cfg.DefaultProjectName != "" {
		go func() {
			if err := bootstrap.Run(ctx, s, cfg, bcfg, cfg.DefaultOrgName, cfg.DefaultProjectName); err != nil {
				log.Error().Err(err).Msg("tenant-admin bootstrap failed (non-fatal)")
			}
		}()
	}

	// Initialize JWT validator (nil if OIDC not configured).
	var jwtValidator *api.JWTValidator
	oidcURL := cfg.OIDCServerURL
	if envOIDC := os.Getenv("OIDC_SERVER_URL"); envOIDC != "" {
		oidcURL = envOIDC
	}
	if oidcURL != "" {
		var err error
		jwtValidator, err = api.NewJWTValidator(oidcURL)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to initialize JWT validator")
		}
		log.Info().Str("oidc_url", oidcURL).Msg("JWT authentication enabled")
	} else {
		log.Warn().Msg("OIDC_SERVER_URL not set; JWT authentication disabled")
	}

	// Start cleanup goroutine.
	go runCleanup(ctx, s, cfg)

	// Internal-endpoint auth token (guards /v1/status and /v1/events).
	internalToken := os.Getenv("INTERNAL_AUTH_TOKEN")
	if internalToken == "" {
		log.Warn().Msg("INTERNAL_AUTH_TOKEN not set; internal endpoints (/v1/status, /v1/events) are unauthenticated")
	}

	// Start HTTP server.
	handler := api.NewHandler(s, cfg, jwtValidator, internalToken)
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown: drain HTTP first, then cancel background goroutines.
	// Cancelling the context before Shutdown would abort in-flight DB writes.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info().Msg("shutting down")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("server shutdown error")
		}
		cancel() // cancel background goroutines only after HTTP is drained
	}()

	log.Info().Str("addr", cfg.ListenAddr).Msg("listening")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server failed")
	}
}

func runCleanup(ctx context.Context, s *store.Store, cfg *config.Config) {
	ticker := time.NewTicker(cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := s.CleanupOldEvents(ctx, cfg.EventRetention)
			if err != nil {
				log.Warn().Err(err).Msg("event cleanup failed")
			} else if n > 0 {
				log.Info().Int("count", n).Msg("cleaned up old events")
			}

			if err := s.CleanupHardDelete(ctx, cfg); err != nil {
				log.Warn().Err(err).Msg("hard-delete cleanup failed")
			}
		}
	}
}
