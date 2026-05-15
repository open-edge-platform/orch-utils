// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package bootstrap implements the single-tenant initialization flow that
// runs after the default org and project have been created (see
// store.Bootstrap). It mirrors the behavior of the legacy tenancy-init Job
// (orch-utils/tenancy-init): create a tenant-admin user in Keycloak with a
// generated password, persist that password as a Kubernetes Secret, and add
// the user to the org-level admin group(s) and the project-level edge
// groups. Enabled by BOOTSTRAP_TENANT_ADMIN_ENABLED=true.
package bootstrap

import (
	"fmt"
	"os"
	"strings"
)

// Config holds the env-driven settings for the tenant-admin bootstrap step.
// Variable names match the legacy tenancy-init container so operators see
// familiar knobs.
type Config struct {
	Enabled bool

	KeycloakService   string // platform-keycloak
	KeycloakNamespace string // orch-platform
	KeycloakPort      string // 8080
	KeycloakRealm     string // master
	KeycloakAdminUser string // admin

	TenantAdminUser string // tenant-admin

	// Group short names (without the "_" prefix and without the orgUUID /
	// projectUUID prefix that Keycloak-tenant-controller adds at realm-group
	// creation time).
	ProjectAdminGroups []string // e.g. ["Project-Manager-Group"]
	ProjectEdgeGroups  []string // e.g. ["Edge-Manager-Group", ...]

	// Namespace in which the generated tenant-admin password Secret is
	// stored. Defaults to the pod's own namespace.
	SecretNamespace string

	// SecretName is the name of the K8s Secret that stores the generated
	// password. Defaults to "tenant-admin-password".
	SecretName string
}

// LoadConfig reads Config from environment variables, applying the same
// defaults as the legacy tenant-init utility.
func LoadConfig() Config {
	return Config{
		Enabled:            getenvBool("BOOTSTRAP_TENANT_ADMIN_ENABLED", false),
		KeycloakService:    getenv("KEYCLOAK_SERVICE", "platform-keycloak"),
		KeycloakNamespace:  getenv("KEYCLOAK_NAMESPACE", "orch-platform"),
		KeycloakPort:       getenv("KEYCLOAK_PORT", "8080"),
		KeycloakRealm:      getenv("KEYCLOAK_REALM", "master"),
		KeycloakAdminUser:  getenv("KEYCLOAK_ADMIN_USER", "admin"),
		TenantAdminUser:    strings.ToLower(getenv("TENANT_ADMIN_USER", "tenant-admin")),
		ProjectAdminGroups: splitCSV(getenv("PROJECT_ADMIN_GROUPS", "Project-Manager-Group")),
		ProjectEdgeGroups: splitCSV(getenv("PROJECT_EDGE_GROUPS",
			"Edge-Manager-Group,Edge-Onboarding-Group,Edge-Operator-Group,Host-Manager-Group")),
		SecretNamespace: os.Getenv("TENANT_ADMIN_SECRET_NAMESPACE"), // empty -> use pod's own ns
		SecretName:      getenv("TENANT_ADMIN_SECRET_NAME", "tenant-admin-password"),
	}
}

// KeycloakBaseURL returns the in-cluster Keycloak admin URL.
func (c Config) KeycloakBaseURL() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%s",
		c.KeycloakService, c.KeycloakNamespace, c.KeycloakPort)
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getenvBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off", "":
		return false
	}
	return def
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
