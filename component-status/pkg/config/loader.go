// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Component status configuration
type Config struct {
	SchemaVersion string       `yaml:"schema-version" json:"schema-version"`
	Orchestrator  Orchestrator `yaml:"orchestrator" json:"orchestrator"`
}

// Orchestrator information
type Orchestrator struct {
	Version  string             `yaml:"version" json:"version"`
	Features map[string]Feature `yaml:"features" json:"features"`
}

// Feature and its installation status
type Feature struct {
	Installed   bool               `yaml:"installed" json:"installed"`
	SubFeatures map[string]Feature `yaml:",inline" json:"-"`
}

// MarshalJSON implements custom JSON marshaling for Feature
func (f Feature) MarshalJSON() ([]byte, error) {
	// Create a map to hold all fields
	result := make(map[string]interface{})
	result["installed"] = f.Installed
	
	// Add all subfeatures directly to the result (flatten)
	for key, value := range f.SubFeatures {
		result[key] = value
	}
	
	return json.Marshal(result)
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

// IsFeatureInstalled checks if a feature is installed
func (c *Config) IsFeatureInstalled(featurePath ...string) bool {
if len(featurePath) == 0 {
return false
}

// Start with top-level features
features := c.Orchestrator.Features

// Traverse the feature path
for i, name := range featurePath {
feature, found := features[name]
if !found {
return false
}

// Check if last key - return installed status
if i == len(featurePath)-1 {
return feature.Installed
}

// Move to sub-features for next iteration
features = feature.SubFeatures
}

return false
}
