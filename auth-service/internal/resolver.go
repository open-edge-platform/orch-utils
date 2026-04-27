// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var (
	projectPathRegex = regexp.MustCompile(`^/v[0-9]+/projects/([^/]+)/`)
	tmHTTPClient     = &http.Client{Timeout: 5 * time.Second}
)

// NewProjectResolverHandler returns an HTTP handler for use as a Traefik
// ForwardAuth endpoint at /resolveproject. It:
//  1. Validates the caller's Bearer JWT.
//  2. Extracts the project name from X-Forwarded-Uri (the original request URL
//     that Traefik forwards before any path-rewriting middleware runs).
//  3. Calls the Tenant Manager to resolve the project name to its UUID,
//     forwarding the caller's Authorization header so TM can enforce RBAC.
//  4. Returns 200 with Activeprojectid / ActiveProjectID response headers.
//     Traefik's authResponseHeaders config copies these into the upstream request
//     so cluster-manager, alerting-monitor, and metadata-broker receive the UUID.
func NewProjectResolverHandler(keyset jwk.Set, tenancyManagerURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Validate JWT.
		if _, err := jwt.ParseRequest(r,
			jwt.WithHeaderKey("Authorization"),
			jwt.WithKeySet(keyset, jws.WithRequireKid(false), jws.WithInferAlgorithmFromKey(true)),
			jwt.WithVerify(true),
			jwt.WithValidate(true),
		); err != nil {
			log.Printf("resolveproject: invalid token: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// 2. Extract project name from the original request URI.
		//    Traefik sets X-Forwarded-Uri to the path before any path-rewriting
		//    middlewares; we rely on this to find the project name even when
		//    strip-project-prefix or rewrite-* middlewares are also in the chain.
		forwardedURI := r.Header.Get("X-Forwarded-Uri")
		if forwardedURI == "" {
			// Fallback for direct calls (tests, health checks).
			forwardedURI = r.URL.RequestURI()
		}
		matches := projectPathRegex.FindStringSubmatch(forwardedURI)
		if len(matches) < 2 {
			log.Printf("resolveproject: no project name in URI %q", forwardedURI)
			http.Error(w, "no project name in request path", http.StatusBadRequest)
			return
		}
		projectName := matches[1]

		// 3. Resolve project name → UUID via Tenant Manager.
		projectUUID, err := resolveProjectUUID(r.Context(), projectName, r.Header.Get("Authorization"), tenancyManagerURL)
		if err != nil {
			log.Printf("resolveproject: failed to resolve project %q: %v", projectName, err)
			http.Error(w, fmt.Sprintf("project %q not found", projectName), http.StatusNotFound)
			return
		}

		// 4. Return UUID in response headers. Traefik copies authResponseHeaders
		//    (Activeprojectid, ActiveProjectID) into the upstream request.
		w.Header().Set("Activeprojectid", projectUUID)
		w.Header().Set("ActiveProjectID", projectUUID)
		w.WriteHeader(http.StatusOK)
	}
}

// resolveProjectUUID calls GET /v1/projects/{name} on the Tenant Manager and
// returns the project UUID. The caller's Authorization header is forwarded so
// that the Tenant Manager can enforce project-level RBAC.
func resolveProjectUUID(ctx context.Context, projectName, authHeader, tenancyManagerURL string) (string, error) {
	reqURL := fmt.Sprintf("%s/v1/projects/%s", tenancyManagerURL, url.PathEscape(projectName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil) //nolint:gosec // tenancyManagerURL is a configured service URL; projectName is PathEscaped
	if err != nil {
		return "", fmt.Errorf("build TM request: %w", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := tmHTTPClient.Do(req) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("call tenancy manager: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return "", fmt.Errorf("project %q not found", projectName)
	case http.StatusOK:
		// handled below
	default:
		return "", fmt.Errorf("tenancy manager returned status %d", resp.StatusCode)
	}

	var result struct {
		Status struct {
			ProjectStatus struct {
				UID string `json:"uID"`
			} `json:"projectStatus"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode TM response: %w", err)
	}
	uid := result.Status.ProjectStatus.UID
	if uid == "" {
		return "", fmt.Errorf("tenancy manager returned empty UUID for project %q", projectName)
	}
	return uid, nil
}
