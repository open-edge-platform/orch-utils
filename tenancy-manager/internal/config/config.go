// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"net/url"
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
		DefaultOrgName:     "",
		DefaultProjectName: "",
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
// After loading the file, it checks for standard PostgreSQL environment
// variables (PGHOST, PGUSER, PGPASSWORD, PGDATABASE, PGPORT) and
// constructs a database URL from them if present, overriding the file value.
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
	// Standard PG* env vars (from mounted postgresql secret) override config.
	if dbURL := databaseURLFromEnv(); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}
	return cfg, nil
}

// databaseURLFromEnv constructs a Postgres connection URL from standard
// PG* environment variables. Returns "" if the required vars aren't set.
func databaseURLFromEnv() string {
	host := os.Getenv("PGHOST")
	user := os.Getenv("PGUSER")
	if host == "" || user == "" {
		return ""
	}
	password := os.Getenv("PGPASSWORD")
	dbName := os.Getenv("PGDATABASE")
	if dbName == "" {
		dbName = "tenancy"
	}
	port := os.Getenv("PGPORT")
	if port == "" {
		port = "5432"
	}
	sslmode := os.Getenv("PGSSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbName, sslmode)
}

// RedactedDatabaseURL returns the database URL with the password replaced by
// "***", safe for use in log messages and error output.
func (c *Config) RedactedDatabaseURL() string {
	u, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return "[unparseable database URL]"
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "***")
	}
	return u.String()
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
