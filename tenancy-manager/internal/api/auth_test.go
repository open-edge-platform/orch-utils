// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent"
)

// --- Mock implementations ---

type mockOrgNameResolver struct {
	names map[uuid.UUID]string
	err   error
}

func (m *mockOrgNameResolver) GetOrgNamesByUUIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[uuid.UUID]string)
	for _, id := range ids {
		if name, ok := m.names[id]; ok {
			result[id] = name
		}
	}
	return result, nil
}

type mockDeletionChecker struct {
	orgs     map[string]*ent.Org
	projects map[string]*ent.Project
}

func (m *mockDeletionChecker) GetOrgIncludingDeleted(_ context.Context, name string) (*ent.Org, error) {
	if o, ok := m.orgs[name]; ok {
		return o, nil
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockDeletionChecker) GetProjectIncludingDeleted(_ context.Context, name string, _ []string) (*ent.Project, *ent.Org, error) {
	if p, ok := m.projects[name]; ok {
		return p, nil, nil
	}
	return nil, nil, &ent.NotFoundError{}
}

func (m *mockDeletionChecker) GetOrgNamesByUUIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	return nil, nil
}

// --- extractRolesFromClaims tests ---

func TestExtractRolesFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   int
	}{
		{
			name:   "nil claims",
			claims: jwt.MapClaims{},
			want:   0,
		},
		{
			name: "no realm_access",
			claims: jwt.MapClaims{
				"sub": "user1",
			},
			want: 0,
		},
		{
			name: "realm_access not a map",
			claims: jwt.MapClaims{
				"realm_access": "not-a-map",
			},
			want: 0,
		},
		{
			name: "roles not a slice",
			claims: jwt.MapClaims{
				"realm_access": map[string]interface{}{
					"roles": "not-a-slice",
				},
			},
			want: 0,
		},
		{
			name: "valid roles",
			claims: jwt.MapClaims{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{
						"org-read-role",
						"org-write-role",
						"some-other-role",
					},
				},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles := extractRolesFromClaims(tt.claims)
			if len(roles) != tt.want {
				t.Errorf("extractRolesFromClaims() got %d roles, want %d", len(roles), tt.want)
			}
		})
	}
}

// --- extractOrgUUIDs tests ---

func TestExtractOrgUUIDs(t *testing.T) {
	orgID1 := uuid.New()
	orgID2 := uuid.New()
	projectID := uuid.New()

	tests := []struct {
		name  string
		roles []string
		want  int
	}{
		{
			name:  "empty roles",
			roles: nil,
			want:  0,
		},
		{
			name:  "no org-scoped roles",
			roles: []string{"org-read-role", "org-write-role"},
			want:  0,
		},
		{
			name: "project-read-role extracts org UUID",
			roles: []string{
				orgID1.String() + "_project-read-role",
			},
			want: 1,
		},
		{
			name: "member-role extracts org UUID",
			roles: []string{
				orgID1.String() + "_" + projectID.String() + "_member-role",
			},
			want: 1,
		},
		{
			name: "member-role short form",
			roles: []string{
				orgID1.String() + "_" + projectID.String() + "_m",
			},
			want: 1,
		},
		{
			name: "deduplicates org UUIDs",
			roles: []string{
				orgID1.String() + "_project-read-role",
				orgID1.String() + "_project-write-role",
				orgID1.String() + "_" + projectID.String() + "_member-role",
			},
			want: 1,
		},
		{
			name: "multiple org UUIDs",
			roles: []string{
				orgID1.String() + "_project-read-role",
				orgID2.String() + "_project-write-role",
			},
			want: 2,
		},
		{
			name: "invalid UUID is skipped",
			roles: []string{
				"not-a-uuid_project-read-role",
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := extractOrgUUIDs(tt.roles)
			if len(ids) != tt.want {
				t.Errorf("extractOrgUUIDs() got %d UUIDs, want %d", len(ids), tt.want)
			}
		})
	}
}

// --- hasRole tests ---

func TestHasRole(t *testing.T) {
	roles := []string{"org-read-role", "org-write-role"}
	if !hasRole(roles, "org-read-role") {
		t.Error("hasRole should find org-read-role")
	}
	if hasRole(roles, "org-delete-role") {
		t.Error("hasRole should not find org-delete-role")
	}
	if hasRole(nil, "org-read-role") {
		t.Error("hasRole should return false for nil roles")
	}
}

// --- hasOrgScopedRole tests ---

func TestHasOrgScopedRole(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	roles := []string{
		orgID.String() + "_project-read-role",
		orgID.String() + "_project-write-role",
	}

	if !hasOrgScopedRole(roles, []uuid.UUID{orgID}, "project-read-role") {
		t.Error("should find project-read-role for matching org")
	}
	if hasOrgScopedRole(roles, []uuid.UUID{otherOrgID}, "project-read-role") {
		t.Error("should not find project-read-role for non-matching org")
	}
	if hasOrgScopedRole(roles, []uuid.UUID{orgID}, "project-delete-role") {
		t.Error("should not find project-delete-role")
	}
}

// --- hasMemberRole tests ---

func TestHasMemberRole(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	projectID := uuid.New()

	roles := []string{
		orgID.String() + "_" + projectID.String() + "_member-role",
	}

	if !hasMemberRole(roles, []uuid.UUID{orgID}) {
		t.Error("should find member-role for matching org")
	}
	if hasMemberRole(roles, []uuid.UUID{otherOrgID}) {
		t.Error("should not find member-role for non-matching org")
	}
	if hasMemberRole(nil, []uuid.UUID{orgID}) {
		t.Error("should return false for nil roles")
	}
}

// --- checkOrgAuthz tests ---

func TestCheckOrgAuthz(t *testing.T) {
	tests := []struct {
		name   string
		roles  []string
		method string
		want   bool
	}{
		{"GET with read role", []string{"org-read-role"}, http.MethodGet, true},
		{"GET without read role", []string{"org-write-role"}, http.MethodGet, false},
		{"PUT with write role", []string{"org-write-role"}, http.MethodPut, true},
		{"PUT without write role", []string{"org-read-role"}, http.MethodPut, false},
		{"DELETE with delete role", []string{"org-delete-role"}, http.MethodDelete, true},
		{"DELETE without delete role", []string{"org-read-role"}, http.MethodDelete, false},
		{"POST is denied", []string{"org-read-role", "org-write-role", "org-delete-role"}, http.MethodPost, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkOrgAuthz(tt.roles, tt.method)
			if got != tt.want {
				t.Errorf("checkOrgAuthz(%v, %s) = %v, want %v", tt.roles, tt.method, got, tt.want)
			}
		})
	}
}

// --- checkProjectAuthz tests ---

func TestCheckProjectAuthz(t *testing.T) {
	orgID := uuid.New()
	projectID := uuid.New()

	tests := []struct {
		name     string
		roles    []string
		orgUUIDs []uuid.UUID
		method   string
		want     bool
	}{
		{
			"GET with project-read-role",
			[]string{orgID.String() + "_project-read-role"},
			[]uuid.UUID{orgID},
			http.MethodGet,
			true,
		},
		{
			"GET with member-role fallback",
			[]string{orgID.String() + "_" + projectID.String() + "_member-role"},
			[]uuid.UUID{orgID},
			http.MethodGet,
			true,
		},
		{
			"GET without any role",
			[]string{"org-read-role"},
			[]uuid.UUID{orgID},
			http.MethodGet,
			false,
		},
		{
			"PUT with project-write-role",
			[]string{orgID.String() + "_project-write-role"},
			[]uuid.UUID{orgID},
			http.MethodPut,
			true,
		},
		{
			"PUT with member-role only (denied)",
			[]string{orgID.String() + "_" + projectID.String() + "_member-role"},
			[]uuid.UUID{orgID},
			http.MethodPut,
			false,
		},
		{
			"DELETE with project-delete-role",
			[]string{orgID.String() + "_project-delete-role"},
			[]uuid.UUID{orgID},
			http.MethodDelete,
			true,
		},
		{
			"DELETE without delete role",
			[]string{orgID.String() + "_project-read-role"},
			[]uuid.UUID{orgID},
			http.MethodDelete,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkProjectAuthz(tt.roles, tt.orgUUIDs, tt.method)
			if got != tt.want {
				t.Errorf("checkProjectAuthz() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- AuthzMiddleware tests ---

func TestAuthzMiddleware(t *testing.T) {
	orgID := uuid.New()

	handler := AuthzMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		path       string
		method     string
		authCtx    *AuthContext
		wantStatus int
	}{
		{
			"no auth context returns 401",
			"/v1/orgs/test-org",
			http.MethodGet,
			nil,
			http.StatusUnauthorized,
		},
		{
			"org GET with org-read-role allowed",
			"/v1/orgs/test-org",
			http.MethodGet,
			&AuthContext{Roles: []string{"org-read-role"}},
			http.StatusOK,
		},
		{
			"org PUT without write role denied",
			"/v1/orgs/test-org",
			http.MethodPut,
			&AuthContext{Roles: []string{"org-read-role"}},
			http.StatusForbidden,
		},
		{
			"project GET with member-role allowed",
			"/v1/projects/test-proj",
			http.MethodGet,
			&AuthContext{
				Roles:    []string{orgID.String() + "_" + uuid.New().String() + "_member-role"},
				OrgUUIDs: []uuid.UUID{orgID},
			},
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.authCtx != nil {
				ctx := context.WithValue(req.Context(), authContextKey, tt.authCtx)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- DeletionCheckMiddleware tests ---

func TestDeletionCheckMiddleware(t *testing.T) {
	now := time.Now()
	activeOrg := &ent.Org{Name: "active-org"}
	deletedOrg := &ent.Org{Name: "deleted-org", DeletedAt: &now}
	activeProject := &ent.Project{Name: "active-proj"}
	deletedProject := &ent.Project{Name: "deleted-proj", DeletedAt: &now}

	checker := &mockDeletionChecker{
		orgs: map[string]*ent.Org{
			"active-org":  activeOrg,
			"deleted-org": deletedOrg,
		},
		projects: map[string]*ent.Project{
			"active-proj":  activeProject,
			"deleted-proj": deletedProject,
		},
	}

	mw := DeletionCheckMiddleware(checker, checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		path       string
		urlParam   string
		method     string
		wantStatus int
	}{
		{
			"GET on deleted org is allowed",
			"/v1/orgs/deleted-org",
			"deleted-org",
			http.MethodGet,
			http.StatusOK,
		},
		{
			"PUT on deleted org is blocked",
			"/v1/orgs/deleted-org",
			"deleted-org",
			http.MethodPut,
			http.StatusBadRequest,
		},
		{
			"DELETE on deleted org is blocked",
			"/v1/orgs/deleted-org",
			"deleted-org",
			http.MethodDelete,
			http.StatusBadRequest,
		},
		{
			"PUT on active org is allowed",
			"/v1/orgs/active-org",
			"active-org",
			http.MethodPut,
			http.StatusOK,
		},
		{
			"PUT on nonexistent org passes through",
			"/v1/orgs/missing-org",
			"missing-org",
			http.MethodPut,
			http.StatusOK,
		},
		{
			"GET on deleted project is allowed",
			"/v1/projects/deleted-proj",
			"deleted-proj",
			http.MethodGet,
			http.StatusOK,
		},
		{
			"PUT on deleted project is blocked",
			"/v1/projects/deleted-proj",
			"deleted-proj",
			http.MethodPut,
			http.StatusBadRequest,
		},
		{
			"PUT on active project is allowed",
			"/v1/projects/active-proj",
			"active-proj",
			http.MethodPut,
			http.StatusOK,
		},
		{
			"no name param passes through",
			"/v1/orgs",
			"",
			http.MethodGet,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)

			// Set up chi URL param context.
			rctx := chi.NewRouteContext()
			if tt.urlParam != "" {
				rctx.URLParams.Add("name", tt.urlParam)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// --- resolveOrgNames tests ---

func TestResolveOrgNames(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		authCtx  *AuthContext
		wantLen  int
		wantNil  bool
		wantOrgs []string
	}{
		{
			"no auth, no query param",
			"",
			nil,
			0,
			true,
			nil,
		},
		{
			"no auth, explicit org param",
			"org=my-org",
			nil,
			1,
			false,
			[]string{"my-org"},
		},
		{
			"auth context with org names, no query",
			"",
			&AuthContext{OrgNames: []string{"org-a", "org-b"}},
			2,
			false,
			[]string{"org-a", "org-b"},
		},
		{
			"auth context, explicit org param matching",
			"org=org-a",
			&AuthContext{OrgNames: []string{"org-a", "org-b"}},
			1,
			false,
			[]string{"org-a"},
		},
		{
			"auth context, explicit org param not matching",
			"org=org-c",
			&AuthContext{OrgNames: []string{"org-a", "org-b"}},
			0,
			false, // empty slice, not nil
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/projects"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.authCtx != nil {
				ctx := context.WithValue(req.Context(), authContextKey, tt.authCtx)
				req = req.WithContext(ctx)
			}

			got := resolveOrgNames(req)

			if tt.wantNil && got != nil {
				t.Errorf("resolveOrgNames() = %v, want nil", got)
				return
			}
			if !tt.wantNil && got == nil {
				t.Errorf("resolveOrgNames() = nil, want non-nil")
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("resolveOrgNames() len = %d, want %d", len(got), tt.wantLen)
				return
			}
			for i, want := range tt.wantOrgs {
				if got[i] != want {
					t.Errorf("resolveOrgNames()[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

// --- resolveOrgNamesWithDefault tests ---

func TestResolveOrgNamesWithDefault(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		authCtx    *AuthContext
		defaultOrg string
		wantLen    int
		wantNil    bool
		wantOrgs   []string
	}{
		{
			"no auth, no query — falls back to default",
			"",
			nil,
			"default-org",
			1,
			false,
			[]string{"default-org"},
		},
		{
			"auth context with orgs, no query — uses JWT orgs, not default",
			"",
			&AuthContext{OrgNames: []string{"jwt-org"}},
			"default-org",
			1,
			false,
			[]string{"jwt-org"},
		},
		{
			"auth context, denied org — does NOT fall back to default",
			"org=forbidden-org",
			&AuthContext{OrgNames: []string{"allowed-org"}},
			"default-org",
			0,
			false, // empty slice, not nil — means "access denied"
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/projects"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			if tt.authCtx != nil {
				ctx := context.WithValue(req.Context(), authContextKey, tt.authCtx)
				req = req.WithContext(ctx)
			}

			got := resolveOrgNamesWithDefault(req, tt.defaultOrg)

			if tt.wantNil && got != nil {
				t.Errorf("resolveOrgNamesWithDefault() = %v, want nil", got)
				return
			}
			if !tt.wantNil && got == nil {
				t.Errorf("resolveOrgNamesWithDefault() = nil, want non-nil")
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("resolveOrgNamesWithDefault() len = %d, want %d", len(got), tt.wantLen)
				return
			}
			for i, want := range tt.wantOrgs {
				if got[i] != want {
					t.Errorf("resolveOrgNamesWithDefault()[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

// --- isOrgEndpoint / isProjectEndpoint tests ---

func TestEndpointDetection(t *testing.T) {
	if !isOrgEndpoint("/v1/orgs") {
		t.Error("/v1/orgs should be an org endpoint")
	}
	if !isOrgEndpoint("/v1/orgs/my-org") {
		t.Error("/v1/orgs/my-org should be an org endpoint")
	}
	if !isOrgEndpoint("/v1/orgs/my-org/status") {
		t.Error("/v1/orgs/my-org/status should be an org endpoint")
	}
	if isOrgEndpoint("/v1/projects/my-proj") {
		t.Error("/v1/projects/my-proj should not be an org endpoint")
	}

	if !isProjectEndpoint("/v1/projects") {
		t.Error("/v1/projects should be a project endpoint")
	}
	if !isProjectEndpoint("/v1/projects/my-proj") {
		t.Error("/v1/projects/my-proj should be a project endpoint")
	}
	if isProjectEndpoint("/v1/orgs/my-org") {
		t.Error("/v1/orgs/my-org should not be a project endpoint")
	}
}
