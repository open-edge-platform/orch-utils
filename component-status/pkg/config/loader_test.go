// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	content := `schema-version: "1.0"
orchestrator:
  version: "2026.0"
  features:
    application-orchestration:
      installed: true
    edge-infrastructure-manager:
      installed: true
      inventory:
        installed: true
      device-onboarding:
        installed: false
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test loading the config
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify schema version
	if cfg.SchemaVersion != "1.0" {
		t.Errorf("Expected schema-version '1.0', got '%s'", cfg.SchemaVersion)
	}

	// Verify orchestrator version
	if cfg.Orchestrator.Version != "2026.0" {
		t.Errorf("Expected version '2026.0', got '%s'", cfg.Orchestrator.Version)
	}

	// Verify top-level feature
	appOrch, exists := cfg.Orchestrator.Features["application-orchestration"]
	if !exists {
		t.Error("Expected 'application-orchestration' feature to exist")
	}
	if !appOrch.Installed {
		t.Error("Expected 'application-orchestration' to be installed")
	}

	// Verify nested features
	eim, exists := cfg.Orchestrator.Features["edge-infrastructure-manager"]
	if !exists {
		t.Error("Expected 'edge-infrastructure-manager' feature to exist")
	}
	if !eim.Installed {
		t.Error("Expected 'edge-infrastructure-manager' to be installed")
	}

	inventory, exists := eim.SubFeatures["inventory"]
	if !exists {
		t.Error("Expected 'inventory' sub-feature to exist")
	}
	if !inventory.Installed {
		t.Error("Expected 'inventory' to be installed")
	}

	deviceOnboarding, exists := eim.SubFeatures["device-onboarding"]
	if !exists {
		t.Error("Expected 'device-onboarding' sub-feature to exist")
	}
	if deviceOnboarding.Installed {
		t.Error("Expected 'device-onboarding' to NOT be installed")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/file.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	content := `invalid: yaml: content:`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}

func TestLoadMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing schema-version",
			content: `orchestrator:
  version: "2026.0"
  features: {}
`,
		},
		{
			name: "missing orchestrator.version",
			content: `schema-version: "1.0"
orchestrator:
  features: {}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			_, err = Load(tmpfile.Name())
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

func TestIsFeatureInstalled(t *testing.T) {
	cfg := &Config{
		SchemaVersion: "1.0",
		Orchestrator: Orchestrator{
			Version: "2026.0",
			Features: map[string]Feature{
				"application-orchestration": {
					Installed:   true,
					SubFeatures: map[string]Feature{},
				},
				"edge-infrastructure-manager": {
					Installed: true,
					SubFeatures: map[string]Feature{
						"inventory": {
							Installed:   true,
							SubFeatures: map[string]Feature{},
						},
						"device-onboarding": {
							Installed:   false,
							SubFeatures: map[string]Feature{},
						},
					},
				},
				"cluster-orchestration": {
					Installed:   false,
					SubFeatures: map[string]Feature{},
				},
			},
		},
	}

	tests := []struct {
		name     string
		path     []string
		expected bool
	}{
		{
			name:     "top-level installed feature",
			path:     []string{"application-orchestration"},
			expected: true,
		},
		{
			name:     "top-level not installed feature",
			path:     []string{"cluster-orchestration"},
			expected: false,
		},
		{
			name:     "nested installed feature",
			path:     []string{"edge-infrastructure-manager", "inventory"},
			expected: true,
		},
		{
			name:     "nested not installed feature",
			path:     []string{"edge-infrastructure-manager", "device-onboarding"},
			expected: false,
		},
		{
			name:     "non-existent feature",
			path:     []string{"non-existent"},
			expected: false,
		},
		{
			name:     "non-existent nested feature",
			path:     []string{"edge-infrastructure-manager", "non-existent"},
			expected: false,
		},
		{
			name:     "empty path",
			path:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.IsFeatureInstalled(tt.path...)
			if result != tt.expected {
				t.Errorf("IsFeatureInstalled(%v) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}
