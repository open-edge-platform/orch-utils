// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds the Tenant Manager configuration.
type Config struct {
	// Database connection URL.
	DatabaseURL string `yaml:"database_url"`

	// HTTP listen address.
	ListenAddr string `yaml:"listen_addr"`

	// Registered controllers, keyed by resource type ("org", "project").
	Controllers ControllersConfig `yaml:"controllers"`

	// Event retention period. Events older than this are cleaned up.
	EventRetention time.Duration `yaml:"event_retention"`

	// Cleanup interval for the background goroutine.
	CleanupInterval time.Duration `yaml:"cleanup_interval"`

	// Default org name for bootstrap.
	DefaultOrgName string `yaml:"default_org_name"`

	// Default project name for bootstrap.
	DefaultProjectName string `yaml:"default_project_name"`

	// OIDC server URL for JWT validation (e.g.,
	// "http://platform-keycloak.orch-platform.svc:8080/realms/master").
	// Also read from OIDC_SERVER_URL env var.
	// When empty, JWT authentication is disabled.
	OIDCServerURL string `yaml:"oidc_server_url"`
}

// ControllersConfig defines which controllers are registered for each
// resource type.
type ControllersConfig struct {
	Org     []string `yaml:"org"`
	Project []string `yaml:"project"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DatabaseURL:        "postgres://localhost:5432/tenancy?sslmode=disable",
		ListenAddr:         ":8080",
		EventRetention:     7 * 24 * time.Hour,
		CleanupInterval:    1 * time.Minute,
		DefaultOrgName:     "default",
		DefaultProjectName: "default",
		Controllers: ControllersConfig{
			Org: []string{
				"keycloak-tenant-controller",
			},
			Project: []string{
				"app-orch-tenant-controller",
				"app-deployment-manager",
				"keycloak-tenant-controller",
				"infra-tenant-controller",
				"cluster-manager",
				"observability-tenant-controller",
				"metadata-broker",
			},
		},
	}
}

// Load reads a YAML config file and merges it with defaults.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ControllersForResource returns the registered controller names for a
// resource type ("org" or "project").
func (c *Config) ControllersForResource(resourceType string) []string {
	switch resourceType {
	case "org":
		return c.Controllers.Org
	case "project":
		return c.Controllers.Project
	default:
		return nil
	}
}
