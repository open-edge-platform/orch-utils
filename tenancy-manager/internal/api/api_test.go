// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/store"
)

// ── Mock store ────────────────────────────────────────────────────────────────

// mockStore implements storeBackend for handler unit tests.
// Each field controls the return value for the corresponding operation.
type mockStore struct {
	// Org
	orgs       []*ent.Org
	orgByName  map[string]*ent.Org
	createOrgF func(name, desc string) (*ent.Org, error)
	updateOrgF func(name, desc string) (*ent.Org, error)
	deleteOrgF func(name string) error

	// Project
	projects       []*ent.Project
	projectByName  map[string]*ent.Project
	projectByID    map[uuid.UUID]*ent.Project
	createProjectF func(org, name, desc string) (*ent.Project, *ent.Org, error)
	updateProjectF func(name string, orgs []string, desc string) (*ent.Project, *ent.Org, error)
	deleteProjectF func(name string, orgs []string) error

	// Events
	eventsAfter     []*ent.TenancyEvent
	eventsAfterErr  error
	replayEvents    []store.ReplayEvent
	replayEventsErr error

	// Controller status
	upsertStatusErr error
	deleteStatusErr error
}

func (m *mockStore) GetOrgNamesByUUIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string, len(ids))
	for _, id := range ids {
		out[id] = id.String()
	}
	return out, nil
}

func (m *mockStore) GetOrgIncludingDeleted(_ context.Context, name string) (*ent.Org, error) {
	if o, ok := m.orgByName[name]; ok {
		return o, nil
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockStore) GetProjectIncludingDeleted(_ context.Context, name string, _ []string) (*ent.Project, *ent.Org, error) {
	if p, ok := m.projectByName[name]; ok {
		return p, nil, nil
	}
	return nil, nil, &ent.NotFoundError{}
}

func (m *mockStore) ListOrgs(_ context.Context) ([]*ent.Org, error) {
	return m.orgs, nil
}

func (m *mockStore) GetOrg(_ context.Context, name string) (*ent.Org, error) {
	if o, ok := m.orgByName[name]; ok {
		return o, nil
	}
	return nil, &ent.NotFoundError{}
}

func (m *mockStore) CreateOrg(_ context.Context, name, desc string) (*ent.Org, error) {
	if m.createOrgF != nil {
		return m.createOrgF(name, desc)
	}
	o := &ent.Org{Name: name}
	return o, nil
}

func (m *mockStore) UpdateOrg(_ context.Context, name, desc string) (*ent.Org, error) {
	if m.updateOrgF != nil {
		return m.updateOrgF(name, desc)
	}
	o := &ent.Org{Name: name}
	return o, nil
}

func (m *mockStore) DeleteOrg(_ context.Context, name string) error {
	if m.deleteOrgF != nil {
		return m.deleteOrgF(name)
	}
	return nil
}

func (m *mockStore) ListProjects(_ context.Context, _ []string) ([]*ent.Project, error) {
	return m.projects, nil
}

func (m *mockStore) GetProject(_ context.Context, name string, _ []string) (*ent.Project, *ent.Org, error) {
	if p, ok := m.projectByName[name]; ok {
		return p, nil, nil
	}
	return nil, nil, &ent.NotFoundError{}
}

func (m *mockStore) CreateProject(_ context.Context, org, name, desc string) (*ent.Project, *ent.Org, error) {
	if m.createProjectF != nil {
		return m.createProjectF(org, name, desc)
	}
	return &ent.Project{Name: name}, &ent.Org{Name: org}, nil
}

func (m *mockStore) UpdateProject(_ context.Context, name string, orgs []string, desc string) (*ent.Project, *ent.Org, error) {
	if m.updateProjectF != nil {
		return m.updateProjectF(name, orgs, desc)
	}
	return &ent.Project{Name: name}, nil, nil
}

func (m *mockStore) DeleteProject(_ context.Context, name string, orgs []string) error {
	if m.deleteProjectF != nil {
		return m.deleteProjectF(name, orgs)
	}
	return nil
}

func (m *mockStore) GetProjectByID(_ context.Context, id uuid.UUID) (*ent.Project, *ent.Org, error) {
	if p, ok := m.projectByID[id]; ok {
		return p, nil, nil
	}
	return nil, nil, &ent.NotFoundError{}
}

func (m *mockStore) GetEventsAfter(_ context.Context, _ int64, _ int) ([]*ent.TenancyEvent, error) {
	return m.eventsAfter, m.eventsAfterErr
}

func (m *mockStore) SynthesizeReplayEvents(_ context.Context) ([]store.ReplayEvent, int64, error) {
	return m.replayEvents, 0, m.replayEventsErr
}

func (m *mockStore) UpsertControllerStatus(_ context.Context, _, _ string, _ uuid.UUID, _, _ string) error {
	return m.upsertStatusErr
}

func (m *mockStore) DeleteControllerStatus(_ context.Context, _, _ string, _ uuid.UUID) error {
	return m.deleteStatusErr
}

func (m *mockStore) DeriveStatus(_ context.Context, _ string, _ uuid.UUID, _ bool, _ []string) (string, string) {
	return "Active", ""
}

// ── Test helpers ──────────────────────────────────────────────────────────────

// newTestHandler returns a Handler wired to the given mockStore with auth disabled.
func newTestHandler(ms *mockStore) *Handler {
	cfg := &config.Config{}
	return NewHandler(ms, cfg, nil, "")
}

// do fires a request against the handler's router and returns the response recorder.
func do(h *Handler, method, path string, body string) *httptest.ResponseRecorder {
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reqBody)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.Router().ServeHTTP(w, req)
	return w
}

// ── Healthz ───────────────────────────────────────────────────────────────────

func TestHealthz(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodGet, "/healthz", "")
	if w.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d, want 200", w.Code)
	}
}

// ── Body size limit ───────────────────────────────────────────────────────────

func TestBodySizeLimit(t *testing.T) {
	h := newTestHandler(&mockStore{})
	bigBody := strings.Repeat("x", maxBodyBytes+1)
	// Wrap in a JSON string so it parses as valid JSON of a large size.
	payload := fmt.Sprintf(`{"description":%q}`, bigBody)
	w := do(h, http.MethodPut, "/v1/orgs/test-org", payload)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT /v1/orgs/test-org with oversized body = %d, want 400", w.Code)
	}
}

func TestBodyInvalidJSON(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodPut, "/v1/orgs/test-org", `{not valid json}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT /v1/orgs/test-org with invalid JSON = %d, want 400", w.Code)
	}
}

// ── GetEvents param validation ────────────────────────────────────────────────

func TestGetEvents_ParamValidation(t *testing.T) {
	h := newTestHandler(&mockStore{})

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"missing controller param", "", http.StatusBadRequest},
		{"non-numeric after param", "controller=x&after=notanumber", http.StatusBadRequest},
		{"negative after param", "controller=x&after=-1", http.StatusBadRequest},
		{"limit zero", "controller=x&limit=0", http.StatusBadRequest},
		{"limit too large", "controller=x&limit=9999", http.StatusBadRequest},
		{"limit non-numeric", "controller=x&limit=abc", http.StatusBadRequest},
		{"valid minimal", "controller=x", http.StatusOK},
		{"valid with after", "controller=x&after=10&limit=50", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/v1/events"
			if tt.query != "" {
				url += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			h.Router().ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("GET %s = %d, want %d", url, w.Code, tt.wantStatus)
			}
		})
	}
}

// ── writeError body sanitisation ──────────────────────────────────────────────

func TestErrorBodyContainsNoInternalDetails(t *testing.T) {
	ms := &mockStore{} // empty map → any name lookup returns NotFoundError
	h := newTestHandler(ms)

	w := do(h, http.MethodGet, "/v1/orgs/does-not-exist", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
	body := w.Body.String()
	for _, forbidden := range []string{"ent:", "pq:", "sql:", "postgres"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Errorf("404 body leaks internal detail %q: %s", forbidden, body)
		}
	}
}

// ── Org handlers ─────────────────────────────────────────────────────────────

func TestListOrgs(t *testing.T) {
	ms := &mockStore{
		orgs: []*ent.Org{{Name: "org-a"}, {Name: "org-b"}},
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodGet, "/v1/orgs", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /v1/orgs = %d, want 200", w.Code)
	}
	var got []OrgResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d orgs, want 2", len(got))
	}
}

func TestGetOrg_NotFound(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodGet, "/v1/orgs/missing", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /v1/orgs/missing = %d, want 404", w.Code)
	}
}

func TestGetOrg_OK(t *testing.T) {
	ms := &mockStore{
		orgByName: map[string]*ent.Org{"my-org": {Name: "my-org"}},
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodGet, "/v1/orgs/my-org", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /v1/orgs/my-org = %d, want 200", w.Code)
	}
	var got OrgResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "my-org" {
		t.Errorf("got name %q, want %q", got.Name, "my-org")
	}
}

func TestCreateOrg_Created(t *testing.T) {
	ms := &mockStore{} // empty orgByName → org does not exist → create path
	h := newTestHandler(ms)
	w := do(h, http.MethodPut, "/v1/orgs/new-org", `{"description":"test"}`)
	if w.Code != http.StatusOK {
		t.Errorf("PUT /v1/orgs/new-org = %d, want 200", w.Code)
	}
}

func TestUpdateOrg_Updated(t *testing.T) {
	ms := &mockStore{
		orgByName: map[string]*ent.Org{"existing-org": {Name: "existing-org"}},
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodPut, "/v1/orgs/existing-org", `{"description":"updated"}`)
	if w.Code != http.StatusOK {
		t.Errorf("PUT /v1/orgs/existing-org (update) = %d, want 200", w.Code)
	}
}

func TestDeleteOrg_NotFound(t *testing.T) {
	ms := &mockStore{
		deleteOrgF: func(_ string) error { return &ent.NotFoundError{} },
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodDelete, "/v1/orgs/missing", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE /v1/orgs/missing = %d, want 404", w.Code)
	}
}

func TestDeleteOrg_OK(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodDelete, "/v1/orgs/my-org", "")
	if w.Code != http.StatusOK {
		t.Errorf("DELETE /v1/orgs/my-org = %d, want 200", w.Code)
	}
}

// ── Project handlers ──────────────────────────────────────────────────────────

func TestListProjects(t *testing.T) {
	ms := &mockStore{
		projects: []*ent.Project{{Name: "proj-a"}, {Name: "proj-b"}},
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodGet, "/v1/projects", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /v1/projects = %d, want 200", w.Code)
	}
	var got []ProjectResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d projects, want 2", len(got))
	}
}

func TestGetProject_NotFound(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodGet, "/v1/projects/missing", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /v1/projects/missing = %d, want 404", w.Code)
	}
}

func TestGetProject_OK(t *testing.T) {
	ms := &mockStore{
		projectByName: map[string]*ent.Project{"my-proj": {Name: "my-proj"}},
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodGet, "/v1/projects/my-proj", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /v1/projects/my-proj = %d, want 200", w.Code)
	}
}

func TestCreateProject_NoOrg(t *testing.T) {
	// Project doesn't exist + no ?org= param → 400
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodPut, "/v1/projects/new-proj", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT /v1/projects/new-proj without org = %d, want 400", w.Code)
	}
}

func TestCreateProject_WithOrg(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodPut, "/v1/projects/new-proj?org=my-org", `{}`)
	if w.Code != http.StatusOK {
		t.Errorf("PUT /v1/projects/new-proj?org=my-org = %d, want 200", w.Code)
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	ms := &mockStore{
		deleteProjectF: func(_ string, _ []string) error { return &ent.NotFoundError{} },
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodDelete, "/v1/projects/missing?org=some-org", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("DELETE /v1/projects/missing = %d, want 404", w.Code)
	}
}

// ── GetProjectByID ────────────────────────────────────────────────────────────

func TestGetProjectByID_InvalidUUID(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodGet, "/v1/internal/projects/not-a-uuid", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("GET /v1/internal/projects/not-a-uuid = %d, want 400", w.Code)
	}
}

func TestGetProjectByID_NotFound(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodGet, "/v1/internal/projects/"+uuid.New().String(), "")
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /v1/internal/projects/<valid-uuid> (not found) = %d, want 404", w.Code)
	}
}

func TestGetProjectByID_OK(t *testing.T) {
	id := uuid.New()
	ms := &mockStore{
		projectByID: map[uuid.UUID]*ent.Project{id: {Name: "found-proj"}},
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodGet, "/v1/internal/projects/"+id.String(), "")
	if w.Code != http.StatusOK {
		t.Errorf("GET /v1/internal/projects/%s = %d, want 200", id, w.Code)
	}
}

// ── InternalAuthMiddleware ─────────────────────────────────────────────────────

func TestInternalAuth_NoTokenConfigured_AllowsAll(t *testing.T) {
	// When internalToken is empty, all requests pass through.
	h := NewHandler(&mockStore{}, &config.Config{}, nil, "")
	w := do(h, http.MethodGet, "/v1/events?controller=x", "")
	if w.Code != http.StatusOK {
		t.Errorf("no configured token: GET /v1/events = %d, want 200", w.Code)
	}
}

func TestInternalAuth_MissingHeader_Returns401(t *testing.T) {
	h := NewHandler(&mockStore{}, &config.Config{}, nil, "secret-token")
	w := do(h, http.MethodGet, "/v1/events?controller=x", "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing auth header: GET /v1/events = %d, want 401", w.Code)
	}
}

func TestInternalAuth_WrongToken_Returns401(t *testing.T) {
	h := NewHandler(&mockStore{}, &config.Config{}, nil, "secret-token")
	req := httptest.NewRequest(http.MethodGet, "/v1/events?controller=x", nil)
	req.Header.Set("X-Internal-Token", "wrong-token")
	w := httptest.NewRecorder()
	h.Router().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: GET /v1/events = %d, want 401", w.Code)
	}
}

func TestInternalAuth_CorrectToken_Passes(t *testing.T) {
	h := NewHandler(&mockStore{}, &config.Config{}, nil, "secret-token")
	req := httptest.NewRequest(http.MethodGet, "/v1/events?controller=x", nil)
	req.Header.Set("X-Internal-Token", "secret-token")
	w := httptest.NewRecorder()
	h.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("correct token: GET /v1/events = %d, want 200", w.Code)
	}
}

// ── UpdateControllerStatus ────────────────────────────────────────────────────

func TestUpdateControllerStatus_MissingFields(t *testing.T) {
	h := newTestHandler(&mockStore{})
	// Missing resourceId — uuid.Nil will fail the non-nil check.
	body := `{"controller":"ctrl","resourceType":"org","status":"Active"}`
	w := do(h, http.MethodPut, "/v1/status", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("PUT /v1/status missing resourceId = %d, want 400", w.Code)
	}
}

func TestUpdateControllerStatus_OK(t *testing.T) {
	h := newTestHandler(&mockStore{})
	body := fmt.Sprintf(`{"controller":"ctrl","resourceType":"org","resourceId":%q,"status":"Active"}`,
		uuid.New().String())
	w := do(h, http.MethodPut, "/v1/status", body)
	if w.Code != http.StatusOK {
		t.Errorf("PUT /v1/status valid = %d, want 200", w.Code)
	}
}

// ── DeleteControllerStatus ────────────────────────────────────────────────────

func TestDeleteControllerStatus_MissingFields(t *testing.T) {
	h := newTestHandler(&mockStore{})
	body := `{"controller":"ctrl","resourceType":"org"}`
	w := do(h, http.MethodDelete, "/v1/status", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("DELETE /v1/status missing resourceId = %d, want 400", w.Code)
	}
}

func TestDeleteControllerStatus_OK(t *testing.T) {
	h := newTestHandler(&mockStore{})
	body := fmt.Sprintf(`{"controller":"ctrl","resourceType":"org","resourceId":%q}`,
		uuid.New().String())
	w := do(h, http.MethodDelete, "/v1/status", body)
	if w.Code != http.StatusOK {
		t.Errorf("DELETE /v1/status valid = %d, want 200", w.Code)
	}
}

// ── Response shape ────────────────────────────────────────────────────────────

func TestOrgResponseShape(t *testing.T) {
	ms := &mockStore{
		orgByName: map[string]*ent.Org{"shape-org": {Name: "shape-org"}},
	}
	h := newTestHandler(ms)
	w := do(h, http.MethodGet, "/v1/orgs/shape-org", "")
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	// Must be valid JSON with a "name" field — not a raw DB struct.
	var got map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if got["name"] != "shape-org" {
		t.Errorf("response[name] = %v, want %q", got["name"], "shape-org")
	}
}

func TestErrorResponseShape(t *testing.T) {
	h := newTestHandler(&mockStore{})
	w := do(h, http.MethodGet, "/v1/orgs/missing", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
	// Error body must be {"error":"..."}, not a plain string or DB dump.
	var got map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("error response is not valid JSON: %v", err)
	}
	if _, ok := got["error"]; !ok {
		t.Errorf("error response missing 'error' field: %v", got)
	}
}

// Ensure mockStore satisfies the interface (compile-time check).
var _ storeBackend = (*mockStore)(nil)
