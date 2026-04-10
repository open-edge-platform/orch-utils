// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/open-edge-platform/orch-utils/component-status/pkg/config"
)

// StatusHandler handles component status requests.
type StatusHandler struct {
	config *config.Config
}

// NewStatusHandler creates a new status handler.
func NewStatusHandler(cfg *config.Config) *StatusHandler {
	return &StatusHandler{
		config: cfg,
	}
}

// GetStatus handles GET /v1/orchestrator requests.
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	// Expose the Date header so browsers can read orchestrator time
	// from a cross-origin fetch (used by the UI to display the clock).
	w.Header().Set("Access-Control-Expose-Headers", "Date")
	if err := json.NewEncoder(w).Encode(h.config); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HealthCheck handles GET /healthz requests.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	}); err != nil {
		log.Printf("Failed to encode health check response: %v", err)
	}
}

// ReadyCheck handles GET /readyz requests.
func ReadyCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
	}); err != nil {
		log.Printf("Failed to encode ready check response: %v", err)
	}
}
