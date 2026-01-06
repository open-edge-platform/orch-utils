// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the component status configuration
type Config struct {
	SchemaVersion string       `yaml:"schema-version" json:"schema-version"`
	Orchestrator  Orchestrator `yaml:"orchestrator" json:"orchestrator"`
}

// Orchestrator represents orchestrator information
type Orchestrator struct {
	Version  string    `yaml:"version" json:"version"`
	Features []Feature `yaml:"features" json:"features"`
}

// Feature represents a feature and its installation status
type Feature struct {
	Name        string    `yaml:"name" json:"name"`
	Version     string    `yaml:"version,omitempty" json:"version,omitempty"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	Status      string    `yaml:"status" json:"status"`
	Features    []Feature `yaml:"features,omitempty" json:"features,omitempty"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if config.SchemaVersion == "" {
		return nil, fmt.Errorf("schema-version is required")
	}

	if config.Orchestrator.Version == "" {
		return nil, fmt.Errorf("orchestrator.version is required")
	}

	return &config, nil
}

// IsFeatureInstalled checks if a feature is installed using hierarchical lookup
// Example: IsFeatureInstalled("observability", "grafana")
func (c *Config) IsFeatureInstalled(featurePath ...string) bool {
	if len(featurePath) == 0 {
		return false
	}

	// Start with top-level features
	features := c.Orchestrator.Features
	
	// Traverse the feature path
	for i, name := range featurePath {
		found := false
		var currentFeature *Feature
		
		// Search for feature by name
		for j := range features {
			if features[j].Name == name {
				currentFeature = &features[j]
				found = true
				break
			}
		}
		
		if !found {
			return false
		}
		
		// If this is the last key in the path, check if enabled
		if i == len(featurePath)-1 {
			return currentFeature.Status == "enabled"
		}
		
		// Move to sub-features for next iteration
		features = currentFeature.Features
	}
	
	return false
}
