// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/store"
)

// storeBackend is the subset of store.Store used by the API handlers.
// Defined as an interface so handler logic can be unit-tested without a real DB.
type storeBackend interface {
	OrgNameResolver
	ProjectDeletionChecker

	// Org operations
	ListOrgs(ctx context.Context) ([]*ent.Org, error)
	GetOrg(ctx context.Context, name string) (*ent.Org, error)
	CreateOrg(ctx context.Context, name, description string) (*ent.Org, error)
	UpdateOrg(ctx context.Context, name, description string) (*ent.Org, error)
	DeleteOrg(ctx context.Context, name string) error

	// Project operations
	ListProjects(ctx context.Context, orgNames []string) ([]*ent.Project, error)
	GetProject(ctx context.Context, name string, orgNames []string) (*ent.Project, *ent.Org, error)
	CreateProject(ctx context.Context, orgName, projectName, description string) (*ent.Project, *ent.Org, error)
	UpdateProject(ctx context.Context, name string, orgNames []string, description string) (*ent.Project, *ent.Org, error)
	DeleteProject(ctx context.Context, name string, orgNames []string) error
	GetProjectByID(ctx context.Context, id uuid.UUID) (*ent.Project, *ent.Org, error)

	// Event operations
	GetEventsAfter(ctx context.Context, afterID int64, limit int) ([]*ent.TenancyEvent, error)
	SynthesizeReplayEvents(ctx context.Context) ([]store.ReplayEvent, int64, error)

	// Controller status operations
	UpsertControllerStatus(ctx context.Context, controllerName, resourceType string, resourceID uuid.UUID, status, message string) error
	DeleteControllerStatus(ctx context.Context, controllerName, resourceType string, resourceID uuid.UUID) error

	// Status derivation
	DeriveStatus(ctx context.Context, resourceType string, resourceID uuid.UUID, isDeleted bool, registeredControllers []string) (string, string)
}

// Compile-time assertion: store.Store must satisfy storeBackend.
var _ storeBackend = (*store.Store)(nil)

// Handler provides HTTP handlers for the Tenant Manager REST API.
type Handler struct {
	store         storeBackend
	cfg           *config.Config
	jwtValidator  *JWTValidator // nil when auth is disabled
	internalToken string        // shared secret for internal endpoints; empty = rely on network policy
}

// NewHandler creates a new API handler.
func NewHandler(s storeBackend, cfg *config.Config, jwtValidator *JWTValidator, internalToken string) *Handler {
	return &Handler{store: s, cfg: cfg, jwtValidator: jwtValidator, internalToken: internalToken}
}

// maxBodyBytes is the maximum request body size accepted by mutation endpoints.
const maxBodyBytes = 64 * 1024 // 64 KB

// decodeBody reads and JSON-decodes the request body, enforcing a size limit.
// Returns false and writes a 400 response if the body is invalid or too large.
// An empty body (no Content-Type sent) yields the zero value of dst — callers
// that treat all fields as optional should check for io.EOF themselves; this
// helper treats io.EOF as an empty-but-valid body.
func decodeBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	err := json.NewDecoder(r.Body).Decode(dst)
	if err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// Router returns the chi router with all routes registered.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/v1", func(r chi.Router) {
		// External endpoints -- require auth when configured.
		r.Group(func(r chi.Router) {
			if h.jwtValidator != nil {
				r.Use(AuthnMiddleware(h.jwtValidator, h.store))
				r.Use(AuthzMiddleware())
			}
			r.Use(DeletionCheckMiddleware(h.store, h.store))

			// Org endpoints
			r.Get("/orgs", h.ListOrgs)
			r.Get("/orgs/{name}", h.GetOrg)
			r.Put("/orgs/{name}", h.CreateOrUpdateOrg)
			r.Delete("/orgs/{name}", h.DeleteOrg)
			r.Get("/orgs/{name}/status", h.GetOrgStatus)

			// Project endpoints
			r.Get("/projects", h.ListProjects)
			r.Get("/projects/{name}", h.GetProject)
			r.Put("/projects/{name}", h.CreateOrUpdateProject)
			r.Delete("/projects/{name}", h.DeleteProject)
			r.Get("/projects/{name}/status", h.GetProjectStatus)
		})

		// Internal endpoints (controller-facing) — protected by shared secret.
		// When INTERNAL_AUTH_TOKEN is not set the middleware is a no-op and
		// access must be restricted by Kubernetes NetworkPolicy.
		r.Group(func(r chi.Router) {
			r.Use(InternalAuthMiddleware(h.internalToken))
			r.Get("/events", h.GetEvents)
			r.Put("/status", h.UpdateControllerStatus)
			r.Delete("/status", h.DeleteControllerStatus)
			r.Get("/internal/projects/{id}", h.GetProjectByID)
		})
	})

	// Health
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return r
}

// --- Org handlers ---

func (h *Handler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.store.ListOrgs(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("ListOrgs failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	result := make([]OrgResponse, 0, len(orgs))
	for _, o := range orgs {
		status, msg := h.store.DeriveStatus(r.Context(), "org", o.ID, false, h.cfg.ControllersForResource("org"))
		result = append(result, toOrgResponse(o, status, msg))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetOrg(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	o, err := h.store.GetOrg(r.Context(), name)
	if ent.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "org not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("org", name).Msg("GetOrg failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	status, msg := h.store.DeriveStatus(r.Context(), "org", o.ID, false, h.cfg.ControllersForResource("org"))
	writeJSON(w, http.StatusOK, toOrgResponse(o, status, msg))
}

func (h *Handler) CreateOrUpdateOrg(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var body struct {
		Description string `json:"description"`
	}
	if !decodeBody(w, r, &body) { // description is optional; empty body is valid
		return
	}

	updateIfExists := r.URL.Query().Get("update_if_exists") != "false" // default true

	existing, err := h.store.GetOrg(r.Context(), name)
	if err == nil {
		if !updateIfExists {
			writeError(w, http.StatusConflict, "org already exists")
			return
		}
		// Update existing org.
		updated, err := h.store.UpdateOrg(r.Context(), name, body.Description)
		if err != nil {
			log.Error().Err(err).Str("org", name).Msg("UpdateOrg failed")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		status, msg := h.store.DeriveStatus(r.Context(), "org", updated.ID, false, h.cfg.ControllersForResource("org"))
		writeJSON(w, http.StatusOK, toOrgResponse(updated, status, msg))
		return
	}

	if !ent.IsNotFound(err) {
		log.Error().Err(err).Str("org", name).Msg("GetOrg failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	_ = existing // not found, create new
	o, err := h.store.CreateOrg(r.Context(), name, body.Description)
	if err != nil {
		if ent.IsConstraintError(err) {
			writeError(w, http.StatusConflict, "org already exists")
		} else {
			log.Error().Err(err).Str("org", name).Msg("CreateOrg failed")
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	status, msg := h.store.DeriveStatus(r.Context(), "org", o.ID, false, h.cfg.ControllersForResource("org"))
	writeJSON(w, http.StatusOK, toOrgResponse(o, status, msg))
}

func (h *Handler) DeleteOrg(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.store.DeleteOrg(r.Context(), name); err != nil {
		if ent.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "org not found")
			return
		}
		log.Error().Err(err).Str("org", name).Msg("DeleteOrg failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetOrgStatus(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	o, err := h.store.GetOrgIncludingDeleted(r.Context(), name)
	if ent.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "org not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("org", name).Msg("GetOrgStatus failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	isDeleted := o.DeletedAt != nil
	status, msg := h.store.DeriveStatus(r.Context(), "org", o.ID, isDeleted, h.cfg.ControllersForResource("org"))
	writeJSON(w, http.StatusOK, toOrgResponse(o, status, msg))
}

// --- Project handlers ---

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	orgNames := resolveOrgNames(r)
	projects, err := h.store.ListProjects(r.Context(), orgNames)
	if err != nil {
		log.Error().Err(err).Msg("ListProjects failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	result := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		status, msg := h.store.DeriveStatus(r.Context(), "project", p.ID, false, h.cfg.ControllersForResource("project"))
		orgName := ""
		if p.Edges.Org != nil {
			orgName = p.Edges.Org.Name
		}
		result = append(result, toProjectResponse(p, orgName, status, msg))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	orgNames := resolveOrgNames(r)

	p, o, err := h.store.GetProject(r.Context(), name, orgNames)
	if ent.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		if isAmbiguous(err) {
			writeError(w, http.StatusConflict, "ambiguous project: specify ?org=name")
			return
		}
		log.Error().Err(err).Str("project", name).Msg("GetProject failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	orgName := ""
	if o != nil {
		orgName = o.Name
	}
	status, msg := h.store.DeriveStatus(r.Context(), "project", p.ID, false, h.cfg.ControllersForResource("project"))
	writeJSON(w, http.StatusOK, toProjectResponse(p, orgName, status, msg))
}

func (h *Handler) CreateOrUpdateProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	orgNames := resolveOrgNamesWithDefault(r, h.cfg.DefaultOrgName)

	var body struct {
		Description string `json:"description"`
	}
	if !decodeBody(w, r, &body) { // description is optional; empty body is valid
		return
	}

	updateIfExists := r.URL.Query().Get("update_if_exists") != "false" // default true

	// Try update first.
	existing, o, err := h.store.GetProject(r.Context(), name, orgNames)
	if err == nil {
		if !updateIfExists {
			writeError(w, http.StatusConflict, "project already exists")
			return
		}
		updated, updatedOrg, err := h.store.UpdateProject(r.Context(), name, orgNames, body.Description)
		if err != nil {
			log.Error().Err(err).Str("project", name).Msg("UpdateProject failed")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		orgName := ""
		if updatedOrg != nil {
			orgName = updatedOrg.Name
		}
		status, msg := h.store.DeriveStatus(r.Context(), "project", updated.ID, false, h.cfg.ControllersForResource("project"))
		writeJSON(w, http.StatusOK, toProjectResponse(updated, orgName, status, msg))
		return
	}
	_, _ = existing, o

	if !ent.IsNotFound(err) {
		if isAmbiguous(err) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		log.Error().Err(err).Str("project", name).Msg("GetProject failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Determine which org to create in.
	if len(orgNames) == 0 {
		writeError(w, http.StatusBadRequest, "org must be specified for project creation (use ?org=name)")
		return
	}
	if len(orgNames) > 1 {
		writeError(w, http.StatusConflict, "ambiguous org context: specify ?org=name")
		return
	}

	p, createdOrg, err := h.store.CreateProject(r.Context(), orgNames[0], name, body.Description)
	if err != nil {
		if ent.IsConstraintError(err) {
			writeError(w, http.StatusConflict, "project already exists")
		} else {
			log.Error().Err(err).Str("project", name).Msg("CreateProject failed")
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	orgName := ""
	if createdOrg != nil {
		orgName = createdOrg.Name
	}
	status, msg := h.store.DeriveStatus(r.Context(), "project", p.ID, false, h.cfg.ControllersForResource("project"))
	writeJSON(w, http.StatusOK, toProjectResponse(p, orgName, status, msg))
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	orgNames := resolveOrgNamesWithDefault(r, h.cfg.DefaultOrgName)

	if err := h.store.DeleteProject(r.Context(), name, orgNames); err != nil {
		if ent.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		if isAmbiguous(err) {
			writeError(w, http.StatusConflict, "ambiguous project: specify ?org=name")
			return
		}
		log.Error().Err(err).Str("project", name).Msg("DeleteProject failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetProjectStatus(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	orgNames := resolveOrgNames(r)

	p, o, err := h.store.GetProjectIncludingDeleted(r.Context(), name, orgNames)
	if ent.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		if isAmbiguous(err) {
			writeError(w, http.StatusConflict, "ambiguous project: specify ?org=name")
			return
		}
		log.Error().Err(err).Str("project", name).Msg("GetProjectStatus failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	isDeleted := p.DeletedAt != nil
	orgName := ""
	if o != nil {
		orgName = o.Name
	}
	status, msg := h.store.DeriveStatus(r.Context(), "project", p.ID, isDeleted, h.cfg.ControllersForResource("project"))
	writeJSON(w, http.StatusOK, toProjectResponse(p, orgName, status, msg))
}

// --- Internal project lookup (controller-facing, no auth) ---

// GetProjectByID returns a project by UUID. This is an internal endpoint
// for controller-to-service lookups (e.g., app-deployment-manager resolving
// a project UUID to org/project names). No JWT auth is required; access is
// restricted to in-cluster traffic via network policy.
func (h *Handler) GetProjectByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	p, o, err := h.store.GetProjectByID(r.Context(), id)
	if ent.IsNotFound(err) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		log.Error().Err(err).Str("id", idParam).Msg("GetProjectByID failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	orgName := ""
	if o != nil {
		orgName = o.Name
	}
	status, msg := h.store.DeriveStatus(r.Context(), "project", p.ID, false, h.cfg.ControllersForResource("project"))
	writeJSON(w, http.StatusOK, toProjectResponse(p, orgName, status, msg))
}

// --- Event handlers (internal, controller-facing) ---

func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	controllerName := r.URL.Query().Get("controller")
	if controllerName == "" {
		writeError(w, http.StatusBadRequest, "controller query parameter required")
		return
	}

	replay := r.URL.Query().Get("replay") == "true"

	if replay {
		events, maxID, err := h.store.SynthesizeReplayEvents(r.Context())
		if err != nil {
			log.Error().Err(err).Msg("SynthesizeReplayEvents failed")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		resp := EventsResponse{
			Events:      make([]EventResponse, 0, len(events)),
			LastEventID: maxID,
		}
		for _, e := range events {
			resp.Events = append(resp.Events, EventResponse{
				ID:           0,
				EventType:    e.EventType,
				ResourceType: e.ResourceType,
				ResourceID:   e.ResourceID,
				ResourceName: e.ResourceName,
				OrgID:        e.OrgID,
				OrgName:      e.OrgName,
				FolderID:     e.FolderID,
				CreatedAt:    time.Now(),
			})
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// Incremental polling.
	var afterID int64
	if afterStr := r.URL.Query().Get("after"); afterStr != "" {
		n, err := strconv.ParseInt(afterStr, 10, 64)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid after parameter: must be a non-negative integer")
			return
		}
		afterID = n
	}

	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		n, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil || n <= 0 || n > 1000 {
			writeError(w, http.StatusBadRequest, "invalid limit parameter: must be between 1 and 1000")
			return
		}
		limit = int(n)
	}

	events, err := h.store.GetEventsAfter(r.Context(), afterID, limit)
	if err != nil {
		log.Error().Err(err).Msg("GetEventsAfter failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := EventsResponse{
		Events:      make([]EventResponse, 0, len(events)),
		LastEventID: afterID,
	}
	for _, e := range events {
		resp.Events = append(resp.Events, EventResponse{
			ID:           e.ID,
			EventType:    e.EventType,
			ResourceType: e.ResourceType,
			ResourceID:   e.ResourceID,
			ResourceName: e.ResourceName,
			OrgID:        e.OrgID,
			OrgName:      e.OrgName,
			FolderID:     e.FolderID,
			CreatedAt:    e.CreatedAt,
		})
		resp.LastEventID = e.ID
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Controller status handlers ---

func (h *Handler) UpdateControllerStatus(w http.ResponseWriter, r *http.Request) {
	var body StatusUpdateRequest
	if !decodeBody(w, r, &body) {
		return
	}

	if body.Controller == "" || body.ResourceType == "" || body.ResourceID == uuid.Nil || body.Status == "" {
		writeError(w, http.StatusBadRequest, "controller, resourceType, resourceId, and status are required")
		return
	}

	if err := h.store.UpsertControllerStatus(
		r.Context(), body.Controller, body.ResourceType, body.ResourceID, body.Status, body.Message,
	); err != nil {
		log.Error().Err(err).Msg("UpsertControllerStatus failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteControllerStatus(w http.ResponseWriter, r *http.Request) {
	var body StatusDeleteRequest
	if !decodeBody(w, r, &body) {
		return
	}

	if body.Controller == "" || body.ResourceType == "" || body.ResourceID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "controller, resourceType, and resourceId are required")
		return
	}

	if err := h.store.DeleteControllerStatus(
		r.Context(), body.Controller, body.ResourceType, body.ResourceID,
	); err != nil {
		log.Error().Err(err).Msg("DeleteControllerStatus failed")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// --- Helpers ---

// resolveOrgNames determines the org scope for a request. Priority:
// 1. Explicit ?org= query parameter (validated against JWT org membership).
// 2. Org names extracted from JWT claims by AuthnMiddleware.
// 3. nil if no auth context (internal endpoints, or auth disabled).
func resolveOrgNames(r *http.Request) []string {
	ac := getAuthContext(r)

	if orgParam := r.URL.Query().Get("org"); orgParam != "" {
		// If auth is active, validate that the caller has access to this org.
		if ac != nil {
			// Admin bypass: KC 'admin' role is superuser — allow any org.
			if hasRole(ac.Roles, "admin") {
				return []string{orgParam}
			}
			for _, name := range ac.OrgNames {
				if name == orgParam {
					return []string{orgParam}
				}
			}
			// Caller specified an org they don't have access to.
			// Return empty (not nil) to indicate "no valid org" rather than
			// "unscoped". The handler will get zero results.
			return []string{}
		}
		return []string{orgParam}
	}

	// Use JWT-derived org names if available.
	if ac != nil && len(ac.OrgNames) > 0 {
		return ac.OrgNames
	}

	return nil
}

// resolveOrgNamesWithDefault is like resolveOrgNames but falls back to the
// configured default org when no explicit org context is available. Use this
// for mutation endpoints where an org is required.
//
// Importantly, the fallback only applies when the caller supplied no org
// context at all (nil return from resolveOrgNames).  An empty non-nil slice
// means the caller explicitly specified an org they do not have access to;
// in that case we must NOT fall back to the default org.
func resolveOrgNamesWithDefault(r *http.Request, defaultOrg string) []string {
	names := resolveOrgNames(r)
	if names != nil {
		// Caller provided explicit org context (possibly empty if denied).
		return names
	}
	// No org context at all — fall back to default.
	if defaultOrg != "" {
		return []string{defaultOrg}
	}
	return nil
}

func isAmbiguous(err error) bool {
	return err != nil && err.Error() != "" && len(err.Error()) > 9 && err.Error()[:9] == "ambiguous"
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("failed to write response")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
