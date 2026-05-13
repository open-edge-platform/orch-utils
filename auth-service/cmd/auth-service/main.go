// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/open-edge-platform/orch-utils/auth-service/internal"
)

const port = ":8080"

func main() {
	var jwksURL, rolesFile, otcURL, tenancyManagerURL string
	flag.StringVar(&jwksURL, "jwksURL",
		"http://platform-keycloak.orch-platform.svc:8080/realms/master/protocol/openid-connect/certs",
		"jwksURL endpoint contains public key for input token validation")
	flag.StringVar(&rolesFile, "rolesFile", "",
		"roles file holds a list of roles (one per line) that grant access to Auth Service")
	flag.StringVar(&otcURL, "otc-url",
		"observability-tenant-controller.orch-platform.svc.cluster.local:50051",
		"set observability tenant controller URL")
	flag.StringVar(&tenancyManagerURL, "tenancy-manager-url",
		"http://tenancy-manager.orch-iam.svc:8080",
		"Tenant Manager REST URL used by /resolveproject to look up project UUIDs")

	flag.Parse()
	if rolesFile == "" {
		log.Panic("Missing required -rolesFile flag")
	}
	log.Printf("Using jwksURL: %s", jwksURL)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c := jwk.NewCache(ctx)
	if err := c.Register(jwksURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		log.Panicf("Failed to register jwks URL %s: %v", jwksURL, err)
	}
	// obtain initial keyset
	keySet, err := c.Get(ctx, jwksURL)
	if err != nil {
		log.Panicf("Failed to fetch keyset from jwks URL %s: %v", jwksURL, err)
	}

	// Read file defining access roles
	content, err := os.ReadFile(rolesFile)
	if err != nil {
		log.Panicf("Failed to read roles file: %v", err)
	}
	roles := strings.Split(string(content), "\n")
	roleStore := internal.NewRoleStore(roles)

	// If templates are available, connect to tenant controller and fetch project updates
	if roleStore.HasTemplatesAvailable() {
		log.Printf("Templates available, will connect to tenant controller at: %s", otcURL)
		tenantControllerConn, err := grpc.NewClient(otcURL,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.DefaultConfig}),
		)
		if err != nil {
			log.Panicf("Failed to connect to tenant controller: %v", err)
		}
		defer tenantControllerConn.Close()

		go roleStore.FetchProjectUpdates(ctx, tenantControllerConn)
	}

	http.HandleFunc("/verifyall", internal.NewHandler(keySet, roleStore))
	http.HandleFunc("/resolveproject", internal.NewProjectResolverHandler(keySet, tenancyManagerURL))
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("success"))
	})

	log.Printf("Starting auth-service listening on port: %v", port)
	server := &http.Server{
		Addr:              port,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Panicf("Error while serving: %v", err)
		}
	}()

	<-ctx.Done()
	log.Print("Shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = server.Shutdown(shutdownCtx)
	if err != nil {
		log.Printf("Auth Service HTTP server shutdown error: %v", err)
	}
}
