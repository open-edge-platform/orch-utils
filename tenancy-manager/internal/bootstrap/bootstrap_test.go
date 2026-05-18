// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"strings"
	"testing"
)

func TestGeneratePassword_LengthAndClasses(t *testing.T) {
	for i := 0; i < 50; i++ {
		pw, err := GeneratePassword()
		if err != nil {
			t.Fatalf("GeneratePassword: %v", err)
		}
		if len(pw) != passwordLength {
			t.Fatalf("expected length %d, got %d (%q)", passwordLength, len(pw), pw)
		}
		if !containsAny(pw, uppercaseChars) {
			t.Errorf("missing uppercase: %q", pw)
		}
		if !containsAny(pw, lowercaseChars) {
			t.Errorf("missing lowercase: %q", pw)
		}
		if !containsAny(pw, digitChars) {
			t.Errorf("missing digit: %q", pw)
		}
		if !containsAny(pw, specialChars) {
			t.Errorf("missing special: %q", pw)
		}
	}
}

func TestGeneratePassword_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		pw, err := GeneratePassword()
		if err != nil {
			t.Fatalf("GeneratePassword: %v", err)
		}
		if _, dup := seen[pw]; dup {
			t.Fatalf("duplicate password generated: %q", pw)
		}
		seen[pw] = struct{}{}
	}
}

func containsAny(s, chars string) bool {
	return strings.ContainsAny(s, chars)
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear env to exercise defaults.
	for _, k := range []string{
		"BOOTSTRAP_TENANT_ADMIN_ENABLED", "KEYCLOAK_SERVICE", "KEYCLOAK_NAMESPACE",
		"KEYCLOAK_PORT", "KEYCLOAK_REALM", "KEYCLOAK_ADMIN_USER", "TENANT_ADMIN_USER",
		"PROJECT_ADMIN_GROUPS", "PROJECT_EDGE_GROUPS",
		"TENANT_ADMIN_SECRET_NAMESPACE", "TENANT_ADMIN_SECRET_NAME",
	} {
		t.Setenv(k, "")
	}
	c := LoadConfig()
	if c.Enabled {
		t.Error("expected Enabled=false by default")
	}
	if c.KeycloakService != "platform-keycloak" {
		t.Errorf("KeycloakService default: %s", c.KeycloakService)
	}
	if c.TenantAdminUser != "tenant-admin" {
		t.Errorf("TenantAdminUser default: %s", c.TenantAdminUser)
	}
	wantEdge := []string{"Edge-Manager-Group", "Edge-Onboarding-Group", "Edge-Operator-Group", "Host-Manager-Group"}
	if !equalStrings(c.ProjectEdgeGroups, wantEdge) {
		t.Errorf("ProjectEdgeGroups default = %v, want %v", c.ProjectEdgeGroups, wantEdge)
	}
	if c.SecretName != "tenant-admin-password" {
		t.Errorf("SecretName default: %s", c.SecretName)
	}
}

func TestLoadConfig_EnabledTruthy(t *testing.T) {
	for _, v := range []string{"true", "TRUE", "1", "yes", "on"} {
		t.Setenv("BOOTSTRAP_TENANT_ADMIN_ENABLED", v)
		if !LoadConfig().Enabled {
			t.Errorf("expected Enabled=true for %q", v)
		}
	}
	for _, v := range []string{"false", "0", "no", "off", ""} {
		t.Setenv("BOOTSTRAP_TENANT_ADMIN_ENABLED", v)
		if LoadConfig().Enabled {
			t.Errorf("expected Enabled=false for %q", v)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
