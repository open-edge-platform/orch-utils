// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"

	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent/controllerstatus"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent/folder"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent/org"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent/project"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent/tenancyevent"
)

// Store provides transactional tenancy operations on top of the Ent client.
type Store struct {
	client *ent.Client
	cfg    *config.Config
}

// New creates a Store, connecting to Postgres and running auto-migration.
func New(ctx context.Context, cfg *config.Config) (*Store, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(drv))

	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{client: client, cfg: cfg}, nil
}

// Client returns the underlying Ent client (for use in tests).
func (s *Store) Client() *ent.Client {
	return s.client
}

// Close closes the Ent client.
func (s *Store) Close() error {
	return s.client.Close()
}

// --- Org operations ---

// CreateOrg creates an org with a default folder, emitting a created event.
func (s *Store) CreateOrg(ctx context.Context, name, description string) (*ent.Org, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}

	o, err := tx.Org.Create().
		SetName(name).
		SetDescription(description).
		Save(ctx)
	if err != nil {
		return nil, rollback(tx, err)
	}

	_, err = tx.Folder.Create().
		SetName("default").
		SetOrg(o).
		Save(ctx)
	if err != nil {
		return nil, rollback(tx, err)
	}

	_, err = tx.TenancyEvent.Create().
		SetEventType("created").
		SetResourceType("org").
		SetResourceID(o.ID).
		SetResourceName(o.Name).
		Save(ctx)
	if err != nil {
		return nil, rollback(tx, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return o, nil
}

// GetOrgNamesByUUIDs resolves a list of org UUIDs to their names.
// Returns a map of UUID->name for active orgs that were found.
// Missing or deleted orgs are silently omitted.
func (s *Store) GetOrgNamesByUUIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	orgs, err := s.client.Org.Query().
		Where(org.IDIn(ids...), org.DeletedAtIsNil()).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying orgs by UUID: %w", err)
	}
	result := make(map[uuid.UUID]string, len(orgs))
	for _, o := range orgs {
		result[o.ID] = o.Name
	}
	return result, nil
}

// GetOrg returns an active org by name.
func (s *Store) GetOrg(ctx context.Context, name string) (*ent.Org, error) {
	return s.client.Org.Query().
		Where(org.Name(name), org.DeletedAtIsNil()).
		Only(ctx)
}

// ListOrgs returns all active orgs.
func (s *Store) ListOrgs(ctx context.Context) ([]*ent.Org, error) {
	return s.client.Org.Query().
		Where(org.DeletedAtIsNil()).
		Order(ent.Asc(org.FieldName)).
		All(ctx)
}

// UpdateOrg updates an active org's description.
func (s *Store) UpdateOrg(ctx context.Context, name, description string) (*ent.Org, error) {
	o, err := s.GetOrg(ctx, name)
	if err != nil {
		return nil, err
	}
	return o.Update().SetDescription(description).Save(ctx)
}

// DeleteOrg soft-deletes an org and all its projects, emitting events.
// The soft-delete uses a compare-and-set style UPDATE guarded by
// deleted_at IS NULL to prevent duplicate events from concurrent requests.
func (s *Store) DeleteOrg(ctx context.Context, name string) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}

	o, err := tx.Org.Query().
		Where(org.Name(name), org.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return rollback(tx, err)
	}

	// Soft-delete all active projects under this org, emitting events for each.
	// The WHERE deleted_at IS NULL guard ensures each project is only soft-deleted
	// once even if a concurrent request is in flight.
	activeProjects, err := tx.Project.Query().
		Where(project.HasOrgWith(org.ID(o.ID)), project.DeletedAtIsNil()).
		WithFolder().
		All(ctx)
	if err != nil {
		return rollback(tx, err)
	}

	now := time.Now()
	for _, p := range activeProjects {
		n, err := tx.Project.Update().
			Where(project.ID(p.ID), project.DeletedAtIsNil()).
			SetDeletedAt(now).
			Save(ctx)
		if err != nil {
			return rollback(tx, err)
		}
		if n == 0 {
			// Another transaction already soft-deleted this project.
			continue
		}
		ev := tx.TenancyEvent.Create().
			SetEventType("deleted").
			SetResourceType("project").
			SetResourceID(p.ID).
			SetResourceName(p.Name).
			SetOrgID(o.ID).
			SetOrgName(o.Name)
		if p.Edges.Folder != nil {
			ev = ev.SetFolderID(p.Edges.Folder.ID)
		}
		if _, err := ev.Save(ctx); err != nil {
			return rollback(tx, err)
		}
	}

	// Soft-delete the org itself with the same compare-and-set guard.
	n, err := tx.Org.Update().
		Where(org.ID(o.ID), org.DeletedAtIsNil()).
		SetDeletedAt(now).
		Save(ctx)
	if err != nil {
		return rollback(tx, err)
	}
	if n == 0 {
		// Another transaction already soft-deleted this org.
		return rollback(tx, &ent.NotFoundError{})
	}

	_, err = tx.TenancyEvent.Create().
		SetEventType("deleted").
		SetResourceType("org").
		SetResourceID(o.ID).
		SetResourceName(o.Name).
		Save(ctx)
	if err != nil {
		return rollback(tx, err)
	}

	return tx.Commit()
}

// GetOrgIncludingDeleted returns an org by name regardless of soft-delete state.
func (s *Store) GetOrgIncludingDeleted(ctx context.Context, name string) (*ent.Org, error) {
	return s.client.Org.Query().
		Where(org.Name(name)).
		Only(ctx)
}

// --- Project operations ---

// CreateProject creates a project under the given org's default folder,
// emitting a created event.
func (s *Store) CreateProject(ctx context.Context, orgName, projectName, description string) (*ent.Project, *ent.Org, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, nil, err
	}

	o, err := tx.Org.Query().
		Where(org.Name(orgName), org.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		return nil, nil, rollback(tx, err)
	}

	f, err := tx.Folder.Query().
		Where(folder.Name("default"), folder.HasOrgWith(org.ID(o.ID))).
		Only(ctx)
	if err != nil {
		return nil, nil, rollback(tx, err)
	}

	p, err := tx.Project.Create().
		SetName(projectName).
		SetDescription(description).
		SetFolder(f).
		SetOrg(o).
		Save(ctx)
	if err != nil {
		return nil, nil, rollback(tx, err)
	}

	_, err = tx.TenancyEvent.Create().
		SetEventType("created").
		SetResourceType("project").
		SetResourceID(p.ID).
		SetResourceName(p.Name).
		SetOrgID(o.ID).
		SetOrgName(o.Name).
		SetFolderID(f.ID).
		Save(ctx)
	if err != nil {
		return nil, nil, rollback(tx, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return p, o, nil
}

// GetProjectByID returns an active project by UUID.
func (s *Store) GetProjectByID(ctx context.Context, id uuid.UUID) (*ent.Project, *ent.Org, error) {
	p, err := s.client.Project.Query().
		Where(project.ID(id), project.DeletedAtIsNil()).
		WithOrg().
		Only(ctx)
	if err != nil {
		return nil, nil, err
	}
	return p, p.Edges.Org, nil
}

// GetProject returns an active project by name within the specified orgs.
// If orgNames is empty, searches all orgs. Returns the project and its org.
func (s *Store) GetProject(ctx context.Context, name string, orgNames []string) (*ent.Project, *ent.Org, error) {
	q := s.client.Project.Query().
		Where(project.Name(name), project.DeletedAtIsNil()).
		WithOrg()

	if len(orgNames) > 0 {
		q = q.Where(project.HasOrgWith(org.NameIn(orgNames...), org.DeletedAtIsNil()))
	}

	projects, err := q.All(ctx)
	if err != nil {
		return nil, nil, err
	}

	if len(projects) == 0 {
		return nil, nil, &ent.NotFoundError{}
	}
	if len(projects) > 1 {
		return nil, nil, fmt.Errorf("ambiguous project name %q exists in multiple orgs", name)
	}

	p := projects[0]
	return p, p.Edges.Org, nil
}

// ListProjects returns active projects, optionally filtered by org name.
func (s *Store) ListProjects(ctx context.Context, orgNames []string) ([]*ent.Project, error) {
	q := s.client.Project.Query().
		Where(project.DeletedAtIsNil()).
		WithOrg().
		Order(ent.Asc(project.FieldName))

	if len(orgNames) > 0 {
		q = q.Where(project.HasOrgWith(org.NameIn(orgNames...), org.DeletedAtIsNil()))
	}

	return q.All(ctx)
}

// UpdateProject updates a project's description.
func (s *Store) UpdateProject(ctx context.Context, name string, orgNames []string, description string) (*ent.Project, *ent.Org, error) {
	p, o, err := s.GetProject(ctx, name, orgNames)
	if err != nil {
		return nil, nil, err
	}
	updated, err := p.Update().SetDescription(description).Save(ctx)
	if err != nil {
		return nil, nil, err
	}
	return updated, o, nil
}

// DeleteProject soft-deletes a project, emitting a deleted event.
// Uses a compare-and-set style UPDATE guarded by deleted_at IS NULL to
// prevent duplicate events from concurrent requests.
func (s *Store) DeleteProject(ctx context.Context, name string, orgNames []string) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}

	// Query inside the transaction so the row is consistent.
	q := tx.Project.Query().
		Where(project.Name(name), project.DeletedAtIsNil()).
		WithOrg().
		WithFolder()
	if len(orgNames) > 0 {
		q = q.Where(project.HasOrgWith(org.NameIn(orgNames...), org.DeletedAtIsNil()))
	}
	projects, err := q.All(ctx)
	if err != nil {
		return rollback(tx, err)
	}
	if len(projects) == 0 {
		return rollback(tx, &ent.NotFoundError{})
	}
	if len(projects) > 1 {
		return rollback(tx, fmt.Errorf("ambiguous project name %q exists in multiple orgs", name))
	}

	p := projects[0]
	o := p.Edges.Org

	now := time.Now()
	n, err := tx.Project.Update().
		Where(project.ID(p.ID), project.DeletedAtIsNil()).
		SetDeletedAt(now).
		Save(ctx)
	if err != nil {
		return rollback(tx, err)
	}
	if n == 0 {
		// Another transaction already soft-deleted this project.
		return rollback(tx, &ent.NotFoundError{})
	}

	ev := tx.TenancyEvent.Create().
		SetEventType("deleted").
		SetResourceType("project").
		SetResourceID(p.ID).
		SetResourceName(p.Name).
		SetOrgID(o.ID).
		SetOrgName(o.Name)
	if p.Edges.Folder != nil {
		ev = ev.SetFolderID(p.Edges.Folder.ID)
	}
	if _, err := ev.Save(ctx); err != nil {
		return rollback(tx, err)
	}

	return tx.Commit()
}

// GetProjectIncludingDeleted returns a project by name regardless of state.
func (s *Store) GetProjectIncludingDeleted(ctx context.Context, name string, orgNames []string) (*ent.Project, *ent.Org, error) {
	q := s.client.Project.Query().
		Where(project.Name(name)).
		WithOrg()

	if len(orgNames) > 0 {
		q = q.Where(project.HasOrgWith(org.NameIn(orgNames...)))
	}

	projects, err := q.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(projects) == 0 {
		return nil, nil, &ent.NotFoundError{}
	}
	if len(projects) > 1 {
		return nil, nil, fmt.Errorf("ambiguous project name %q exists in multiple orgs", name)
	}
	p := projects[0]
	return p, p.Edges.Org, nil
}

// --- Event operations ---

// GetEventsAfter returns events with ID > afterID, up to limit.
func (s *Store) GetEventsAfter(ctx context.Context, afterID int64, limit int) ([]*ent.TenancyEvent, error) {
	return s.client.TenancyEvent.Query().
		Where(tenancyevent.IDGT(afterID)).
		Order(ent.Asc(tenancyevent.FieldID)).
		Limit(limit).
		All(ctx)
}

// MaxEventID returns the current maximum event ID.
func (s *Store) MaxEventID(ctx context.Context) (int64, error) {
	e, err := s.client.TenancyEvent.Query().
		Order(ent.Desc(tenancyevent.FieldID)).
		Limit(1).
		Only(ctx)
	if ent.IsNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return e.ID, nil
}

// SynthesizeReplayEvents generates events from current DB state for a
// controller replay. All queries run in a single read transaction for
// snapshot consistency. Returns synthesized events and the current max event ID.
func (s *Store) SynthesizeReplayEvents(ctx context.Context) ([]ReplayEvent, int64, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin replay transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // read-only tx, rollback is cleanup

	// Max event ID (snapshot boundary).
	var maxID int64
	e, err := tx.TenancyEvent.Query().
		Order(ent.Desc(tenancyevent.FieldID)).
		Limit(1).
		Only(ctx)
	if ent.IsNotFound(err) {
		maxID = 0
	} else if err != nil {
		return nil, 0, err
	} else {
		maxID = e.ID
	}

	var events []ReplayEvent

	// Active orgs -> created events
	orgs, err := tx.Org.Query().
		Where(org.DeletedAtIsNil()).
		Order(ent.Asc(org.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, o := range orgs {
		events = append(events, ReplayEvent{
			EventType:    "created",
			ResourceType: "org",
			ResourceID:   o.ID,
			ResourceName: o.Name,
		})
	}

	// Active projects -> created events
	activeProjects, err := tx.Project.Query().
		Where(project.DeletedAtIsNil()).
		WithOrg().
		WithFolder().
		Order(ent.Asc(project.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, p := range activeProjects {
		ev := ReplayEvent{
			EventType:    "created",
			ResourceType: "project",
			ResourceID:   p.ID,
			ResourceName: p.Name,
		}
		if p.Edges.Org != nil {
			ev.OrgID = &p.Edges.Org.ID
			ev.OrgName = &p.Edges.Org.Name
		}
		if p.Edges.Folder != nil {
			ev.FolderID = &p.Edges.Folder.ID
		}
		events = append(events, ev)
	}

	// Soft-deleted projects -> deleted events
	deletedProjects, err := tx.Project.Query().
		Where(project.DeletedAtNotNil()).
		WithOrg().
		WithFolder().
		Order(ent.Asc(project.FieldDeletedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, p := range deletedProjects {
		ev := ReplayEvent{
			EventType:    "deleted",
			ResourceType: "project",
			ResourceID:   p.ID,
			ResourceName: p.Name,
		}
		if p.Edges.Org != nil {
			ev.OrgID = &p.Edges.Org.ID
			ev.OrgName = &p.Edges.Org.Name
		}
		if p.Edges.Folder != nil {
			ev.FolderID = &p.Edges.Folder.ID
		}
		events = append(events, ev)
	}

	// Soft-deleted orgs -> deleted events
	deletedOrgs, err := tx.Org.Query().
		Where(org.DeletedAtNotNil()).
		Order(ent.Asc(org.FieldDeletedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, o := range deletedOrgs {
		events = append(events, ReplayEvent{
			EventType:    "deleted",
			ResourceType: "org",
			ResourceID:   o.ID,
			ResourceName: o.Name,
		})
	}

	return events, maxID, nil
}

// ReplayEvent is a synthesized event for controller replay.
type ReplayEvent struct {
	EventType    string
	ResourceType string
	ResourceID   uuid.UUID
	ResourceName string
	OrgID        *uuid.UUID
	OrgName      *string
	FolderID     *uuid.UUID
}

// --- Controller status operations ---

// UpsertControllerStatus creates or updates a controller's status for a resource.
// Uses update-first with a create fallback. If the create hits a unique
// constraint (concurrent insert race), it retries the update.
func (s *Store) UpsertControllerStatus(ctx context.Context, controllerName, resourceType string, resourceID uuid.UUID, status, message string) error {
	// Try update first (most common path in steady state).
	n, err := s.client.ControllerStatus.Update().
		Where(
			controllerstatus.ControllerName(controllerName),
			controllerstatus.ResourceType(resourceType),
			controllerstatus.ResourceID(resourceID),
		).
		SetStatus(controllerstatus.Status(status)).
		SetMessage(message).
		Save(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	// Row doesn't exist yet, create it.
	err = s.client.ControllerStatus.Create().
		SetControllerName(controllerName).
		SetResourceType(resourceType).
		SetResourceID(resourceID).
		SetStatus(controllerstatus.Status(status)).
		SetMessage(message).
		Exec(ctx)
	if err == nil {
		return nil
	}

	// If the create failed due to a concurrent insert (unique constraint
	// violation), retry the update. This closes the TOCTOU window.
	n, err2 := s.client.ControllerStatus.Update().
		Where(
			controllerstatus.ControllerName(controllerName),
			controllerstatus.ResourceType(resourceType),
			controllerstatus.ResourceID(resourceID),
		).
		SetStatus(controllerstatus.Status(status)).
		SetMessage(message).
		Save(ctx)
	if err2 != nil {
		return err // return the original create error
	}
	if n > 0 {
		return nil
	}
	return err // original create error if row still doesn't exist
}

// DeleteControllerStatus removes a controller's status row for a resource.
func (s *Store) DeleteControllerStatus(ctx context.Context, controllerName, resourceType string, resourceID uuid.UUID) error {
	_, err := s.client.ControllerStatus.Delete().
		Where(
			controllerstatus.ControllerName(controllerName),
			controllerstatus.ResourceType(resourceType),
			controllerstatus.ResourceID(resourceID),
		).
		Exec(ctx)
	return err
}

// GetControllerStatuses returns all status rows for a resource.
func (s *Store) GetControllerStatuses(ctx context.Context, resourceType string, resourceID uuid.UUID) ([]*ent.ControllerStatus, error) {
	return s.client.ControllerStatus.Query().
		Where(
			controllerstatus.ResourceType(resourceType),
			controllerstatus.ResourceID(resourceID),
		).
		All(ctx)
}

// DeriveStatus computes the overall status for a resource given its
// controller_statuses rows and the registered controller set.
func (s *Store) DeriveStatus(ctx context.Context, resourceType string, resourceID uuid.UUID, isDeleted bool, registeredControllers []string) (string, string) {
	statuses, err := s.GetControllerStatuses(ctx, resourceType, resourceID)
	if err != nil {
		return "STATUS_INDICATION_ERROR", err.Error()
	}

	return DeriveStatusFromRows(statuses, isDeleted, registeredControllers)
}

// DeriveStatusFromRows computes status from controller_statuses rows.
func DeriveStatusFromRows(statuses []*ent.ControllerStatus, isDeleted bool, registeredControllers []string) (string, string) {
	if len(registeredControllers) == 0 {
		if isDeleted {
			return "STATUS_INDICATION_DELETED", ""
		}
		return "STATUS_INDICATION_IDLE", ""
	}

	statusMap := make(map[string]*ent.ControllerStatus, len(statuses))
	for _, s := range statuses {
		statusMap[s.ControllerName] = s
	}

	if isDeleted {
		// For deleted resources: no rows from registered controllers = DELETED
		hasAnyRow := false
		for _, c := range registeredControllers {
			if _, ok := statusMap[c]; ok {
				hasAnyRow = true
				break
			}
		}
		if !hasAnyRow {
			return "STATUS_INDICATION_DELETED", ""
		}

		for _, c := range registeredControllers {
			cs, ok := statusMap[c]
			if !ok {
				// Controller finished and deleted its row.
				continue
			}
			if cs.Status == controllerstatus.StatusError {
				return "STATUS_INDICATION_ERROR", cs.Message
			}
		}
		// Any remaining rows (in_progress or completed) = still working.
		return "STATUS_INDICATION_IN_PROGRESS", ""
	}

	// Active resources.
	for _, c := range registeredControllers {
		cs, ok := statusMap[c]
		if !ok {
			return "STATUS_INDICATION_IN_PROGRESS", ""
		}
		if cs.Status == controllerstatus.StatusError {
			return "STATUS_INDICATION_ERROR", cs.Message
		}
		if cs.Status == controllerstatus.StatusInProgress {
			return "STATUS_INDICATION_IN_PROGRESS", ""
		}
	}
	return "STATUS_INDICATION_IDLE", ""
}

// --- Cleanup operations ---

// CleanupOldEvents removes events older than the retention period.
func (s *Store) CleanupOldEvents(ctx context.Context, retention time.Duration) (int, error) {
	cutoff := time.Now().Add(-retention)
	return s.client.TenancyEvent.Delete().
		Where(tenancyevent.CreatedAtLT(cutoff)).
		Exec(ctx)
}

// CleanupHardDelete hard-deletes soft-deleted resources that have no
// controller_statuses rows from registered controllers.
func (s *Store) CleanupHardDelete(ctx context.Context, cfg *config.Config) error {
	// Hard-delete soft-deleted projects with no remaining statuses.
	deletedProjects, err := s.client.Project.Query().
		Where(project.DeletedAtNotNil()).
		All(ctx)
	if err != nil {
		return err
	}

	projectControllers := cfg.ControllersForResource("project")
	for _, p := range deletedProjects {
		if s.hasNoRegisteredStatuses(ctx, "project", p.ID, projectControllers) {
			s.cleanOrphanedStatuses(ctx, "project", p.ID, projectControllers)
			if err := s.client.Project.DeleteOneID(p.ID).Exec(ctx); err != nil {
				log.Warn().Err(err).Str("project", p.Name).Msg("failed to hard-delete project")
			} else {
				log.Info().Str("project", p.Name).Msg("hard-deleted project")
			}
		}
	}

	// Hard-delete soft-deleted orgs with no remaining statuses (and no projects).
	deletedOrgs, err := s.client.Org.Query().
		Where(org.DeletedAtNotNil()).
		All(ctx)
	if err != nil {
		return err
	}

	orgControllers := cfg.ControllersForResource("org")
	for _, o := range deletedOrgs {
		// Check no projects remain (including soft-deleted).
		projectCount, err := s.client.Project.Query().
			Where(project.HasOrgWith(org.ID(o.ID))).
			Count(ctx)
		if err != nil || projectCount > 0 {
			continue
		}

		if s.hasNoRegisteredStatuses(ctx, "org", o.ID, orgControllers) {
			s.cleanOrphanedStatuses(ctx, "org", o.ID, orgControllers)
			// Delete the default folder first.
			_, _ = s.client.Folder.Delete().
				Where(folder.HasOrgWith(org.ID(o.ID))).
				Exec(ctx)
			if err := s.client.Org.DeleteOneID(o.ID).Exec(ctx); err != nil {
				log.Warn().Err(err).Str("org", o.Name).Msg("failed to hard-delete org")
			} else {
				log.Info().Str("org", o.Name).Msg("hard-deleted org")
			}
		}
	}

	return nil
}

// hasNoRegisteredStatuses checks if no registered controllers have status rows.
func (s *Store) hasNoRegisteredStatuses(ctx context.Context, resourceType string, resourceID uuid.UUID, controllers []string) bool {
	if len(controllers) == 0 {
		return true
	}
	count, err := s.client.ControllerStatus.Query().
		Where(
			controllerstatus.ResourceType(resourceType),
			controllerstatus.ResourceID(resourceID),
			controllerstatus.ControllerNameIn(controllers...),
		).
		Count(ctx)
	return err == nil && count == 0
}

// cleanOrphanedStatuses removes status rows from controllers not in the
// registered set.
func (s *Store) cleanOrphanedStatuses(ctx context.Context, resourceType string, resourceID uuid.UUID, controllers []string) {
	if len(controllers) == 0 {
		return
	}
	_, _ = s.client.ControllerStatus.Delete().
		Where(
			controllerstatus.ResourceType(resourceType),
			controllerstatus.ResourceID(resourceID),
			controllerstatus.ControllerNameNotIn(controllers...),
		).
		Exec(ctx)
}

// --- Bootstrap ---

// Bootstrap creates the default org and project if they don't exist.
func (s *Store) Bootstrap(ctx context.Context, orgName, projectName string) error {
	_, err := s.GetOrg(ctx, orgName)
	if err == nil {
		log.Info().Str("org", orgName).Msg("default org already exists, skipping bootstrap")
		return nil
	}
	if !ent.IsNotFound(err) {
		return err
	}

	log.Info().Str("org", orgName).Msg("creating default org")
	_, err = s.CreateOrg(ctx, orgName, "Default organization")
	if err != nil {
		return fmt.Errorf("bootstrap org: %w", err)
	}

	log.Info().Str("project", projectName).Msg("creating default project")
	_, _, err = s.CreateProject(ctx, orgName, projectName, "Default project")
	if err != nil {
		return fmt.Errorf("bootstrap project: %w", err)
	}

	return nil
}

func rollback(tx *ent.Tx, err error) error {
	if rerr := tx.Rollback(); rerr != nil {
		return fmt.Errorf("%w: %v", err, rerr)
	}
	return err
}
