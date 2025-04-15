// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"testing"

	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/config"
	"github.com/stretchr/testify/assert"
)

// Test data.
var testConfigYAML = `
server:
  httpPort: "8080"
  healthProbeAddrress: "/health"
  metricsAddress: "/metrics"
  address: "127.0.0.1"
  certPath: "/certs/cert.pem"
  keyPath: "/certs/key.pem"
enableNexusRuntime: true
disableAuthz: false
backendService: "backend-service"
tenancyService: true
tenantApiGwDomain: "example.com"
customNotFoundPage: "/404.html"
`

func createTempConfigFile(t *testing.T, content string) string {
	t.Helper() // Mark this function as a test helper

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpfile.Name()
}

func TestLoadConfig(t *testing.T) {
	tempConfigFile := createTempConfigFile(t, testConfigYAML)
	defer os.Remove(tempConfigFile)

	conf, err := config.LoadConfig(tempConfigFile)
	assert.NoError(t, err)

	expectedServerConfig := config.ServerConfig{
		HTTPPort:            "8080",
		HealthProbeAddrress: "/health",
		MetricsAddress:      "/metrics",
		Address:             "127.0.0.1",
		CertPath:            "/certs/cert.pem",
		KeyPath:             "/certs/key.pem",
	}

	assert.Equal(t, expectedServerConfig, conf.Server)
	assert.True(t, conf.EnableNexusRuntime)
	assert.Equal(t, "backend-service", conf.BackendService)
	assert.True(t, conf.TenancyService)
	assert.Equal(t, "example.com", conf.TenantAPIGwDomain)
	assert.Equal(t, "/404.html", conf.CustomNotFoundPage)
}

func TestLoadConfigMissingFields(t *testing.T) {
	missingFieldsYAML := `
server:
  httpPort: "8080"
  address: "127.0.0.1"
`

	tempConfigFile := createTempConfigFile(t, missingFieldsYAML)
	defer os.Remove(tempConfigFile)

	_, err := config.LoadConfig(tempConfigFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config doesn't contain Server.CertPath")
}

func TestLoadInvalidConfig(t *testing.T) {
	invalidYAML := `
server:
  httpPort: "8080"
  address
`

	tempConfigFile := createTempConfigFile(t, invalidYAML)
	defer os.Remove(tempConfigFile)

	_, err := config.LoadConfig(tempConfigFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}
