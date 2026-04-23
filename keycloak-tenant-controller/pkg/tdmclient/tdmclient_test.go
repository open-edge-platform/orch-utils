// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package tdmclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-edge-platform/orch-library/go/pkg/tenancy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// mockKCClient – test double for keycloak.Client
// ---------------------------------------------------------------------------

type mockKCClient struct {
	createOrgCalls     []string
	deleteOrgCalls     []string
	createProjectCalls []struct{ orgID, projID string }
	deleteProjectCalls []string

	initErr          error
	createOrgErr     error
	deleteOrgErr     error
	createProjectErr error
	deleteProjectErr error
}

func (m *mockKCClient) Init() error { return m.initErr }

func (m *mockKCClient) CreateOrg(orgID string) error {
	m.createOrgCalls = append(m.createOrgCalls, orgID)
	return m.createOrgErr
}

func (m *mockKCClient) DeleteOrg(orgID string) error {
	m.deleteOrgCalls = append(m.deleteOrgCalls, orgID)
	return m.deleteOrgErr
}

func (m *mockKCClient) CreateProject(orgID, projID string) error {
	m.createProjectCalls = append(m.createProjectCalls, struct{ orgID, projID string }{orgID, projID})
	return m.createProjectErr
}

func (m *mockKCClient) DeleteProject(projID string) error {
	m.deleteProjectCalls = append(m.deleteProjectCalls, projID)
	return m.deleteProjectErr
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newTestEvent builds a tenancy.Event with a fixed resource UUID.
func newTestEvent(resourceType, eventType string, orgID *uuid.UUID) tenancy.Event {
	return tenancy.Event{
		ID:           1,
		EventType:    eventType,
		ResourceType: resourceType,
		ResourceID:   uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ResourceName: "test-resource",
		OrgID:        orgID,
	}
}

// emptyEventsServer returns a test HTTP server that satisfies the Tenant
// Manager API contract with empty event lists.
func emptyEventsServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/events", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events":      []interface{}{},
			"lastEventId": 0,
		})
	})
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return httptest.NewServer(mux)
}

// newClient is a test convenience wrapper for NewMTClient that fails the test
// if construction returns an error.
func newClient(t *testing.T, appName string, kc *mockKCClient) TdmClient {
	t.Helper()
	c, err := NewMTClient(appName, kc)
	require.NoError(t, err)
	return c
}

// ---------------------------------------------------------------------------
// NewMTClient construction tests
// ---------------------------------------------------------------------------

func TestNewMTClient_Success(t *testing.T) {
	c, err := NewMTClient("test-app", &mockKCClient{})
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestNewMTClient_EmptyAppName(t *testing.T) {
	_, err := NewMTClient("", &mockKCClient{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "appName")
}

func TestNewMTClient_NilKCClient(t *testing.T) {
	_, err := NewMTClient("test-app", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kcClient")
}

// ---------------------------------------------------------------------------
// validateURL tests
// ---------------------------------------------------------------------------

func TestValidateURL_ValidHTTP(t *testing.T) {
	assert.NoError(t, validateURL("http://tenancy-manager.orch-iam:8080"))
}

func TestValidateURL_ValidHTTPS(t *testing.T) {
	assert.NoError(t, validateURL("https://example.com/path"))
}

func TestValidateURL_InvalidScheme(t *testing.T) {
	err := validateURL("ftp://example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheme")
}

func TestValidateURL_MissingHost(t *testing.T) {
	err := validateURL("http:///no-host")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "host")
}

func TestValidateURL_NotAURL(t *testing.T) {
	err := validateURL("not-a-url")
	require.Error(t, err)
}

func TestValidateURL_Empty(t *testing.T) {
	err := validateURL("")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Init tests
// ---------------------------------------------------------------------------

func TestInit_ValidEnvURL(t *testing.T) {
	srv := emptyEventsServer(t)
	defer srv.Close()
	t.Setenv("TENANT_MANAGER_URL", srv.URL)

	c := newClient(t, "test-app", &mockKCClient{})
	require.NoError(t, c.Init())
	time.Sleep(50 * time.Millisecond)
	c.Stop()
}

func TestInit_DefaultURL_NoEnv(t *testing.T) {
	// Default URL is unreachable; NewPoller accepts it (URL is not dialled
	// at construction time), so Init returns nil and the goroutine retries
	// in the background.
	t.Setenv("TENANT_MANAGER_URL", "")

	c := newClient(t, "test-app", &mockKCClient{})
	require.NoError(t, c.Init())
	c.Stop()
}

func TestInit_InvalidURL(t *testing.T) {
	t.Setenv("TENANT_MANAGER_URL", "not-a-url")

	c := newClient(t, "test-app", &mockKCClient{})
	err := c.Init()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid TENANT_MANAGER_URL")
}

func TestInit_InvalidScheme(t *testing.T) {
	t.Setenv("TENANT_MANAGER_URL", "ftp://example.com")

	c := newClient(t, "test-app", &mockKCClient{})
	err := c.Init()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scheme")
}

func TestInit_Idempotent_StopsOldPoller(t *testing.T) {
	srv := emptyEventsServer(t)
	defer srv.Close()
	t.Setenv("TENANT_MANAGER_URL", srv.URL)

	c := newClient(t, "test-app", &mockKCClient{})
	require.NoError(t, c.Init())
	time.Sleep(30 * time.Millisecond)

	// Second Init must stop the first poller and start a new one without error.
	require.NoError(t, c.Init())
	time.Sleep(30 * time.Millisecond)
	c.Stop()
}

// ---------------------------------------------------------------------------
// Stop tests
// ---------------------------------------------------------------------------

func TestStop_BeforeInit_NoPanic(t *testing.T) {
	tc := &tdmclient{appName: "test-app", kcClient: &mockKCClient{}}
	assert.NotPanics(t, func() { tc.Stop() })
}

func TestStop_MultipleCalls_NoPanic(t *testing.T) {
	srv := emptyEventsServer(t)
	defer srv.Close()
	t.Setenv("TENANT_MANAGER_URL", srv.URL)

	c := newClient(t, "test-app", &mockKCClient{})
	require.NoError(t, c.Init())
	assert.NotPanics(t, func() {
		c.Stop()
		c.Stop()
		c.Stop()
	})
}

func TestStop_CancelsPollerContext(t *testing.T) {
	tc := &tdmclient{appName: "test-app", kcClient: &mockKCClient{}}
	ctx, cancel := context.WithCancel(context.Background())
	tc.cancel = cancel

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()

	tc.Stop()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context was not cancelled by Stop()")
	}
}

// ---------------------------------------------------------------------------
// keycloakHandler.HandleEvent – routing tests
// ---------------------------------------------------------------------------

func TestHandleEvent_OrgCreated(t *testing.T) {
	mock := &mockKCClient{}
	h := &keycloakHandler{kcClient: mock}
	event := newTestEvent(tenancy.ResourceTypeOrg, tenancy.EventTypeCreated, nil)

	require.NoError(t, h.HandleEvent(context.Background(), event))
	require.Len(t, mock.createOrgCalls, 1)
	assert.Equal(t, event.ResourceID.String(), mock.createOrgCalls[0])
}

func TestHandleEvent_OrgCreated_KeycloakError(t *testing.T) {
	mock := &mockKCClient{createOrgErr: errors.New("keycloak unavailable")}
	h := &keycloakHandler{kcClient: mock}

	err := h.HandleEvent(context.Background(), newTestEvent(tenancy.ResourceTypeOrg, tenancy.EventTypeCreated, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create org")
	assert.Contains(t, err.Error(), "keycloak unavailable")
}

func TestHandleEvent_OrgDeleted(t *testing.T) {
	mock := &mockKCClient{}
	h := &keycloakHandler{kcClient: mock}
	event := newTestEvent(tenancy.ResourceTypeOrg, tenancy.EventTypeDeleted, nil)

	require.NoError(t, h.HandleEvent(context.Background(), event))
	require.Len(t, mock.deleteOrgCalls, 1)
	assert.Equal(t, event.ResourceID.String(), mock.deleteOrgCalls[0])
}

func TestHandleEvent_OrgDeleted_KeycloakError(t *testing.T) {
	mock := &mockKCClient{deleteOrgErr: errors.New("keycloak unavailable")}
	h := &keycloakHandler{kcClient: mock}

	err := h.HandleEvent(context.Background(), newTestEvent(tenancy.ResourceTypeOrg, tenancy.EventTypeDeleted, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete org")
}

func TestHandleEvent_ProjectCreated_WithOrgID(t *testing.T) {
	mock := &mockKCClient{}
	h := &keycloakHandler{kcClient: mock}
	orgID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	event := newTestEvent(tenancy.ResourceTypeProject, tenancy.EventTypeCreated, &orgID)

	require.NoError(t, h.HandleEvent(context.Background(), event))
	require.Len(t, mock.createProjectCalls, 1)
	assert.Equal(t, orgID.String(), mock.createProjectCalls[0].orgID)
	assert.Equal(t, event.ResourceID.String(), mock.createProjectCalls[0].projID)
}

func TestHandleEvent_ProjectCreated_NilOrgID(t *testing.T) {
	mock := &mockKCClient{}
	h := &keycloakHandler{kcClient: mock}
	event := newTestEvent(tenancy.ResourceTypeProject, tenancy.EventTypeCreated, nil)

	require.NoError(t, h.HandleEvent(context.Background(), event))
	require.Len(t, mock.createProjectCalls, 1)
	assert.Equal(t, "", mock.createProjectCalls[0].orgID)
	assert.Equal(t, event.ResourceID.String(), mock.createProjectCalls[0].projID)
}

func TestHandleEvent_ProjectCreated_KeycloakError(t *testing.T) {
	mock := &mockKCClient{createProjectErr: errors.New("keycloak unavailable")}
	h := &keycloakHandler{kcClient: mock}

	err := h.HandleEvent(context.Background(), newTestEvent(tenancy.ResourceTypeProject, tenancy.EventTypeCreated, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create project")
}

func TestHandleEvent_ProjectDeleted(t *testing.T) {
	mock := &mockKCClient{}
	h := &keycloakHandler{kcClient: mock}
	event := newTestEvent(tenancy.ResourceTypeProject, tenancy.EventTypeDeleted, nil)

	require.NoError(t, h.HandleEvent(context.Background(), event))
	require.Len(t, mock.deleteProjectCalls, 1)
	assert.Equal(t, event.ResourceID.String(), mock.deleteProjectCalls[0])
}

func TestHandleEvent_ProjectDeleted_KeycloakError(t *testing.T) {
	mock := &mockKCClient{deleteProjectErr: errors.New("keycloak unavailable")}
	h := &keycloakHandler{kcClient: mock}

	err := h.HandleEvent(context.Background(), newTestEvent(tenancy.ResourceTypeProject, tenancy.EventTypeDeleted, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete project")
}

func TestHandleEvent_UnknownEventType_NoError(t *testing.T) {
	mock := &mockKCClient{}
	h := &keycloakHandler{kcClient: mock}

	require.NoError(t, h.HandleEvent(context.Background(), newTestEvent(tenancy.ResourceTypeOrg, "unknown", nil)))
	assert.Empty(t, mock.createOrgCalls)
	assert.Empty(t, mock.deleteOrgCalls)
}

func TestHandleEvent_UnknownResourceType_NoError(t *testing.T) {
	mock := &mockKCClient{}
	h := &keycloakHandler{kcClient: mock}

	require.NoError(t, h.HandleEvent(context.Background(), newTestEvent("unknown-resource", tenancy.EventTypeCreated, nil)))
	assert.Empty(t, mock.createOrgCalls)
	assert.Empty(t, mock.createProjectCalls)
}

// ---------------------------------------------------------------------------
// End-to-end: Init delivers events to the keycloak handler
// ---------------------------------------------------------------------------

func TestInit_DeliveredEvents_OrgCreated(t *testing.T) {
	orgID := uuid.New()
	orgName := "my-org"

	// Serve a single org-created event on the first poll.
	var served bool
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !served {
			served = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"events": []map[string]interface{}{
					{
						"id":           1,
						"eventType":    tenancy.EventTypeCreated,
						"resourceType": tenancy.ResourceTypeOrg,
						"resourceId":   orgID.String(),
						"resourceName": orgName,
						"createdAt":    time.Now().Format(time.RFC3339),
					},
				},
				"lastEventId": 1,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"events": []interface{}{}, "lastEventId": 1})
	})
	mux.HandleFunc("/v1/status", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("TENANT_MANAGER_URL", srv.URL)
	mock := &mockKCClient{}
	c := newClient(t, "test-app", mock)
	require.NoError(t, c.Init())
	defer c.Stop()

	// Wait up to 2 s for the event to be processed.
	require.Eventually(t, func() bool {
		return len(mock.createOrgCalls) == 1
	}, 2*time.Second, 20*time.Millisecond, "expected CreateOrg to be called once")
	assert.Equal(t, orgID.String(), mock.createOrgCalls[0])
}
