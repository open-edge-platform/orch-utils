// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-jose/go-jose/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent"
)

// contextKey is a private type for request context keys.
type contextKey string

const authContextKey contextKey = "auth"

// AuthContext holds parsed JWT claims extracted by the authn middleware.
type AuthContext struct {
	Username string
	Roles    []string
	OrgUUIDs []uuid.UUID // unique org UUIDs extracted from role patterns
	OrgNames []string    // resolved org names (from DB lookup)
}

// getAuthContext retrieves the auth context from a request, or nil if absent
// or if the stored value is not an *AuthContext.
func getAuthContext(r *http.Request) *AuthContext {
	ac, _ := r.Context().Value(authContextKey).(*AuthContext)
	return ac
}

// --- JWT Validator ---

// JWTValidator validates JWT tokens against Keycloak JWKS keys.
// Implements the same logic as orch-library/go/pkg/auth/jwt.go without
// pulling in the full orch-library dependency tree.
type JWTValidator struct {
	oidcServerURL string
	publicKeys    map[string][]byte
	mu            sync.RWMutex
	httpClient    *http.Client
}

type oidcProviderJSON struct {
	JWKSURL string `json:"jwks_uri"`
}

// NewJWTValidator creates a validator that fetches JWKS from the given
// OIDC server URL (e.g., "http://keycloak.svc:8080/realms/master").
func NewJWTValidator(oidcServerURL string) (*JWTValidator, error) {
	v := &JWTValidator{
		oidcServerURL: oidcServerURL,
		publicKeys:    make(map[string][]byte),
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
	}

	if strings.ToLower(os.Getenv("OIDC_TLS_INSECURE_SKIP_VERIFY")) == "true" {
		log.Warn().Msg("OIDC TLS certificate verification DISABLED — do not use in production")
		v.httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // opt-in via env var for dev/test
				MinVersion:         tls.VersionTLS12,
			},
		}
	}

	// Initial key fetch; non-fatal if Keycloak isn't ready yet.
	if err := v.refreshKeys(); err != nil {
		log.Warn().Err(err).Msg("initial JWKS key fetch failed; will retry on first request")
	}

	return v, nil
}

// Validate parses and validates a JWT token string, returning the claims.
// Only RS* and PS* (asymmetric) algorithms are accepted. HMAC is intentionally
// rejected to prevent algorithm-confusion attacks where an attacker uses the
// JWKS public key as an HMAC secret.
func (v *JWTValidator) Validate(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		alg := token.Method.Alg()

		if strings.HasPrefix(alg, "RS") || strings.HasPrefix(alg, "PS") {
			keyID, ok := token.Header["kid"]
			if !ok {
				return nil, fmt.Errorf("token missing kid header")
			}
			kid, ok := keyID.(string)
			if !ok {
				return nil, fmt.Errorf("token kid header is not a string")
			}

			pubKeyPEM := v.getKey(kid)
			if pubKeyPEM == nil {
				// Key may have been rotated; refresh and retry.
				if err := v.refreshKeys(); err != nil {
					return nil, fmt.Errorf("failed to refresh JWKS: %w", err)
				}
				pubKeyPEM = v.getKey(kid)
				if pubKeyPEM == nil {
					return nil, fmt.Errorf("unknown key ID: %s", kid)
				}
			}

			return jwt.ParseRSAPublicKeyFromPEM(pubKeyPEM)
		}

		// Reject all other algorithms (including HMAC) explicitly.
		return nil, fmt.Errorf("unsupported signing algorithm %q: only RS*/PS* are accepted", alg)
	})

	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (v *JWTValidator) getKey(kid string) []byte {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.publicKeys[kid]
}

const maxJWKSBodyBytes = 1 << 20 // 1 MB — generous but bounded

func (v *JWTValidator) refreshKeys() error {
	discoveryURL := fmt.Sprintf("%s/.well-known/openid-configuration", v.oidcServerURL)
	resp, err := v.httpClient.Get(discoveryURL)
	if err != nil {
		return fmt.Errorf("OIDC discovery failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OIDC discovery returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSBodyBytes))
	if err != nil {
		return fmt.Errorf("reading OIDC discovery response: %w", err)
	}

	var provider oidcProviderJSON
	if err := json.Unmarshal(body, &provider); err != nil {
		return fmt.Errorf("parsing OIDC discovery: %w", err)
	}

	keysResp, err := v.httpClient.Get(provider.JWKSURL)
	if err != nil {
		return fmt.Errorf("fetching JWKS: %w", err)
	}
	defer keysResp.Body.Close()
	if keysResp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned HTTP %d", keysResp.StatusCode)
	}

	keysBody, err := io.ReadAll(io.LimitReader(keysResp.Body, maxJWKSBodyBytes))
	if err != nil {
		return fmt.Errorf("reading JWKS response: %w", err)
	}

	var jwks jose.JSONWebKeySet
	if err := json.Unmarshal(keysBody, &jwks); err != nil {
		return fmt.Errorf("parsing JWKS: %w", err)
	}

	newKeys := make(map[string][]byte, len(jwks.Keys))
	for _, key := range jwks.Keys {
		der, err := x509.MarshalPKIXPublicKey(key.Key)
		if err != nil {
			return fmt.Errorf("marshaling public key: %w", err)
		}
		newKeys[key.KeyID] = pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: der,
		})
	}

	v.mu.Lock()
	v.publicKeys = newKeys
	v.mu.Unlock()

	log.Info().Int("keys", len(newKeys)).Msg("refreshed JWKS keys")
	return nil
}

// --- Role parsing ---

// Regex patterns matching the Keycloak role format inherited from the
// previous tenancy gateway:
//   {orgId}_project-(read|write|delete)-role
//   {orgId}_{projectId}_(member-role|m)
var (
	// Matches: {orgId}_project-(read|write|delete)-role
	projectRoleRegex = regexp.MustCompile(`^([a-f0-9-]+)_project-(read|write|delete)-role$`)
	// Matches: {orgId}_{projectId}_(member-role|m)
	memberRoleRegex = regexp.MustCompile(`^([a-f0-9-]+)_([a-f0-9-]+)_(member-role|m)$`)
)

// extractOrgUUIDs extracts unique org UUIDs from JWT realm_access.roles.
// Org UUIDs appear as the first segment in org-scoped project and member roles.
func extractOrgUUIDs(roles []string) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	var result []uuid.UUID

	for _, role := range roles {
		var idStr string
		if m := projectRoleRegex.FindStringSubmatch(role); m != nil {
			idStr = m[1]
		} else if m := memberRoleRegex.FindStringSubmatch(role); m != nil {
			idStr = m[1]
		} else {
			continue
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			continue // not a valid UUID, skip
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

// hasRole checks if a literal role string is present in the roles list.
func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

// hasOrgScopedRole checks if any role matching {orgId}_{roleSuffix} exists
// for any of the given org UUIDs.
func hasOrgScopedRole(roles []string, orgUUIDs []uuid.UUID, roleSuffix string) bool {
	for _, orgID := range orgUUIDs {
		target := orgID.String() + "_" + roleSuffix
		if hasRole(roles, target) {
			return true
		}
	}
	return false
}

// hasMemberRole checks if any member-role exists for any of the given org UUIDs.
func hasMemberRole(roles []string, orgUUIDs []uuid.UUID) bool {
	for _, role := range roles {
		if m := memberRoleRegex.FindStringSubmatch(role); m != nil {
			orgID, err := uuid.Parse(m[1])
			if err != nil {
				continue
			}
			for _, allowed := range orgUUIDs {
				if orgID == allowed {
					return true
				}
			}
		}
	}
	return false
}

// --- Middleware ---

// OrgNameResolver looks up org names by UUID. Implemented by store.Store.
type OrgNameResolver interface {
	GetOrgNamesByUUIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// AuthnMiddleware validates the JWT, extracts roles and org context, and
// stores an AuthContext in the request context. Returns 401 on failure.
func AuthnMiddleware(validator *JWTValidator, resolver OrgNameResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, http.StatusUnauthorized, "invalid Authorization header format")
				return
			}

			claims, err := validator.Validate(parts[1])
			if err != nil {
				log.Warn().Err(err).Msg("JWT validation failed")
				writeError(w, http.StatusUnauthorized, "invalid or expired JWT")
				return
			}

			// Extract realm_access.roles.
			roles := extractRolesFromClaims(claims)

			// Extract username.
			username, _ := claims["preferred_username"].(string)

			// Extract org UUIDs from roles.
			orgUUIDs := extractOrgUUIDs(roles)

			// Resolve org UUIDs to names.
			var orgNames []string
			if len(orgUUIDs) > 0 && resolver != nil {
				nameMap, err := resolver.GetOrgNamesByUUIDs(r.Context(), orgUUIDs)
				if err != nil {
					log.Warn().Err(err).Msg("failed to resolve org UUIDs")
					// Continue with empty org names -- authz may still pass for
					// org-level roles which don't need org UUID resolution.
				}
				for _, id := range orgUUIDs {
					if name, ok := nameMap[id]; ok {
						orgNames = append(orgNames, name)
					}
				}
			}

			ac := &AuthContext{
				Username: username,
				Roles:    roles,
				OrgUUIDs: orgUUIDs,
				OrgNames: orgNames,
			}

			ctx := context.WithValue(r.Context(), authContextKey, ac)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthzMiddleware checks the caller's JWT roles against the required role
// for the endpoint. Returns 403 if insufficient permissions.
func AuthzMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac := getAuthContext(r)
			if ac == nil {
				writeError(w, http.StatusUnauthorized, "no auth context")
				return
			}

			path := r.URL.Path
			method := r.Method

			var allowed bool
			// Keycloak built-in "admin" realm role is treated as superuser —
			// grants access to all TM endpoints regardless of method. This allows
			// the Keycloak admin account (used by orch-cli and mage setup targets)
			// to manage orgs/projects without needing per-resource role assignment.
			if hasRole(ac.Roles, "admin") {
				allowed = true
			} else {
				switch {
				case isOrgEndpoint(path):
					allowed = checkOrgAuthz(ac.Roles, method)
				case isProjectEndpoint(path):
					allowed = checkProjectAuthz(ac.Roles, ac.OrgUUIDs, method)
				default:
					// Should not reach here if middleware is applied correctly.
					allowed = false
				}
			}

			if !allowed {
				writeError(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOrgEndpoint returns true for /v1/orgs paths.
func isOrgEndpoint(path string) bool {
	return strings.HasPrefix(path, "/v1/orgs")
}

// isProjectEndpoint returns true for /v1/projects paths.
func isProjectEndpoint(path string) bool {
	return strings.HasPrefix(path, "/v1/projects")
}

// checkOrgAuthz checks if the caller has the required global org role.
func checkOrgAuthz(roles []string, method string) bool {
	switch method {
	case http.MethodGet:
		return hasRole(roles, "org-read-role")
	case http.MethodPut:
		return hasRole(roles, "org-write-role")
	case http.MethodDelete:
		return hasRole(roles, "org-delete-role")
	}
	return false
}

// checkProjectAuthz checks if the caller has an org-scoped project role
// or a member-role fallback for reads.
// Global project-*-role values (without org prefix) are accepted as an admin
// bypass — consistent with the org endpoint model where global org-write-role
// grants access across all orgs. This allows admin tooling (e.g. mage targets)
// to manage projects without needing per-org role assignment.
func checkProjectAuthz(roles []string, orgUUIDs []uuid.UUID, method string) bool {
	switch method {
	case http.MethodGet:
		// Global admin read or org-admin project read.
		if hasRole(roles, "project-read-role") {
			return true
		}
		if hasOrgScopedRole(roles, orgUUIDs, "project-read-role") {
			return true
		}
		// Member-role fallback: project members can read
		return hasMemberRole(roles, orgUUIDs)
	case http.MethodPut:
		// Global admin write or org-scoped write.
		return hasRole(roles, "project-write-role") || hasOrgScopedRole(roles, orgUUIDs, "project-write-role")
	case http.MethodDelete:
		// Global admin delete or org-scoped delete.
		return hasRole(roles, "project-delete-role") || hasOrgScopedRole(roles, orgUUIDs, "project-delete-role")
	}
	return false
}

// --- Deletion check middleware ---

// DeletionCheckMiddleware blocks PUT/DELETE on soft-deleted resources,
// returning 400 Bad Request if the target resource is soft-deleted and the
// request method is not GET.
// This runs after authorization and applies regardless of whether JWT auth
// is enabled.
func DeletionCheckMiddleware(s OrgNameResolver, projectChecker ProjectDeletionChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// GET is always allowed (status polling on deleted resources).
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			name := chi.URLParam(r, "name")
			if name == "" {
				// List endpoints, no specific resource to check.
				next.ServeHTTP(w, r)
				return
			}

			path := r.URL.Path
			switch {
			case isOrgEndpoint(path):
				if isOrgDeleted(r.Context(), projectChecker, name) {
					writeError(w, http.StatusBadRequest,
						"Operation not supported. Requested resource is marked for delete.")
					return
				}
			case isProjectEndpoint(path):
				orgNames := resolveOrgNames(r)
				if isProjectDeleted(r.Context(), projectChecker, name, orgNames) {
					writeError(w, http.StatusBadRequest,
						"Operation not supported. Requested resource is marked for delete.")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ProjectDeletionChecker is implemented by store.Store and provides the
// methods needed by the deletion check middleware.
type ProjectDeletionChecker interface {
	GetOrgIncludingDeleted(ctx context.Context, name string) (*ent.Org, error)
	GetProjectIncludingDeleted(ctx context.Context, name string, orgNames []string) (*ent.Project, *ent.Org, error)
}

func isOrgDeleted(ctx context.Context, checker ProjectDeletionChecker, name string) bool {
	o, err := checker.GetOrgIncludingDeleted(ctx, name)
	return err == nil && o.DeletedAt != nil
}

func isProjectDeleted(ctx context.Context, checker ProjectDeletionChecker, name string, orgNames []string) bool {
	p, _, err := checker.GetProjectIncludingDeleted(ctx, name, orgNames)
	return err == nil && p.DeletedAt != nil
}

// InternalAuthMiddleware protects internal controller-facing endpoints
// (/v1/status, /v1/events) with a shared secret token. When token is
// non-empty, every request must carry a matching X-Internal-Token header.
// When token is empty the middleware is a no-op (rely on network policy).
func InternalAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token != "" && r.Header.Get("X-Internal-Token") != token {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractRolesFromClaims extracts realm_access.roles from JWT claims.
func extractRolesFromClaims(claims jwt.MapClaims) []string {
	realmAccess, ok := claims["realm_access"].(map[string]interface{})
	if !ok {
		return nil
	}
	rolesInterface, ok := realmAccess["roles"].([]interface{})
	if !ok {
		return nil
	}
	roles := make([]string, 0, len(rolesInterface))
	for _, ri := range rolesInterface {
		if s, ok := ri.(string); ok {
			roles = append(roles, s)
		}
	}
	return roles
}
