// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/open-edge-platform/orch-utils/aws-sm-proxy/internal"
)

func main() {
	var region string
	flag.StringVar(&region, "region", "", "AWS region")
	flag.Parse()

	if region == "" {
		fmt.Println("Missing required -region flag")
		os.Exit(1)
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		fmt.Printf("not able to setup aws config: %v", err)
		os.Exit(1)
	}

	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		log.Printf("https proxy value is: %s", proxy)
		log.Printf("no proxy value is: %s", os.Getenv("NO_PROXY"))
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}

	svc := secretsmanager.NewFromConfig(cfg)

	http.HandleFunc("/aws-secret", internal.NewProxyAWSHandler(svc))
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hihi"))
	})
	const port = ":8080"
	log.Printf("starting secrets-manager proxy listening on port %s", port)

	server := &http.Server{
		Addr:              port,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       15 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("error serving: %v", err)
			os.Exit(1)
		}
	}
}
