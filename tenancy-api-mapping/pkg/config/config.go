/*
 * Copyright (C) 2025 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions
 * and limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func ReadCRFile(filePath string) ([]byte, error) {
	// Read the file fBytes
	fBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	return fBytes, nil
}

func ParseCRContent(yamlContent []byte) (APIMappingConfig, error) {
	var crConfig APIMappingConfig
	err := yaml.Unmarshal(yamlContent, &crConfig)
	if err != nil {
		return APIMappingConfig{}, err
	}
	return crConfig, nil
}

func GetCrYAMLFilePaths(crDirPath string) ([]string, error) {
	var filePaths []string
	// Read the directory contents
	files, err := os.ReadDir(crDirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	// Iterate over the files
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return nil, fmt.Errorf("error getting file info: %w", err)
		}
		if info.IsDir() {
			continue // Skip directories
		}

		// Check if the file has a .yaml or .yml extension
		if strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml") {
			// Construct the full file path
			filePath := filepath.Join(crDirPath, info.Name())

			// Append the content to the slice
			filePaths = append(filePaths, filePath)
		}
	}
	return filePaths, nil
}
