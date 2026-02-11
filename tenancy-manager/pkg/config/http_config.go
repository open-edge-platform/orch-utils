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
	"os"
)

// HTTPServerConfig contains configuration for the HTTP REST API server.
type HTTPServerConfig struct {
	Enabled    bool
	Host       string
	Port       string
	TLSEnabled bool
	CertPath   string
	KeyPath    string
}

// GetHTTPServerConfig returns HTTP server configuration from environment variables.
func GetHTTPServerConfig() *HTTPServerConfig {
	return &HTTPServerConfig{
		Enabled:    getEnvOrDefault("HTTP_SERVER_ENABLED", "true") == "true",
		Host:       getEnvOrDefault("HTTP_SERVER_HOST", "0.0.0.0"),
		Port:       getEnvOrDefault("HTTP_SERVER_PORT", "8080"),
		TLSEnabled: getEnvOrDefault("HTTP_TLS_ENABLED", "false") == "true",
		CertPath:   getEnvOrDefault("HTTP_CERT_PATH", ""),
		KeyPath:    getEnvOrDefault("HTTP_KEY_PATH", ""),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
