// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-edge-platform/orch-utils/component-status/pkg/config"
)

func TestGetStatus(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		SchemaVersion: "1.0",
		Orchestrator: config.Orchestrator{
			Version: "2026.0",
			Features: map[string]config.Feature{
				"application-orchestration": {
					Installed:   true,
					SubFeatures: map[string]config.Feature{},
				},
			},
		},
	}

	handler := NewStatusHandler(cfg)

	// Test request
	req, err := http.NewRequest("GET", "/v1/orchestrator", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Response recorder
	rr := httptest.NewRecorder()

	handler.GetStatus(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the Content-Type header
	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
	}

	var response config.Config
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	// Verify response
	if response.SchemaVersion != "1.0" {
		t.Errorf("Expected schema-version '1.0', got '%s'", response.SchemaVersion)
	}
	if response.Orchestrator.Version != "2026.0" {
		t.Errorf("Expected version '2026.0', got '%s'", response.Orchestrator.Version)
	}
}

func TestGetStatusMethodNotAllowed(t *testing.T) {
	cfg := &config.Config{
		SchemaVersion: "1.0",
		Orchestrator: config.Orchestrator{
			Version: "2026.0",
			Features: map[string]config.Feature{},
		},
	}

	handler := NewStatusHandler(cfg)

	// Test POST request (should be rejected)
	req, err := http.NewRequest("POST", "/v1/orchestrator", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.GetStatus(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(HealthCheck).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}

func TestReadyCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/readyz", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(ReadyCheck).ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", response["status"])
	}
}
