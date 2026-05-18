// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/config"
	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent"
)

// StoreAPI is the slice of tenancy-manager's *store.Store that the
// bootstrap flow depends on. Defined as an interface to keep the package
// testable without a real Postgres-backed store.
type StoreAPI interface {
	GetOrg(ctx context.Context, name string) (*ent.Org, error)
	GetProject(ctx context.Context, name string, orgNames []string) (*ent.Project, *ent.Org, error)
	DeriveStatus(ctx context.Context, resourceType string, resourceID uuid.UUID, isDeleted bool, registeredControllers []string) (string, string)
}

const (
	statusIdle       = "STATUS_INDICATION_IDLE"
	resourceTypeOrg  = "org"
	resourceTypeProj = "project"

	defaultStatusTimeout = 5 * time.Minute
	defaultStatusPoll    = 5 * time.Second
)

// Run performs the tenant-admin bootstrap once. Idempotent: safe to call
// across pod restarts. Returns nil on success.
//
// It expects the default org/project to already exist in the store (created
// by store.Bootstrap). It then waits for keycloak-tenant-controller to
// report the org and project IDLE (i.e., realm groups are provisioned)
// before creating the tenant-admin user, persisting the password Secret,
// and adding the user to org-level admin group(s) and project-level edge
// groups.
func Run(
	ctx context.Context,
	s StoreAPI,
	cfg *config.Config,
	bcfg Config,
	orgName, projectName string,
) error {
	if !bcfg.Enabled {
		log.Info().Msg("tenant-admin bootstrap disabled (BOOTSTRAP_TENANT_ADMIN_ENABLED!=true)")
		return nil
	}
	if orgName == "" || projectName == "" {
		log.Info().Msg("tenant-admin bootstrap skipped (default org/project not configured)")
		return nil
	}

	log.Info().
		Str("org", orgName).
		Str("project", projectName).
		Str("user", bcfg.TenantAdminUser).
		Msg("starting tenant-admin bootstrap")

	// 1. Resolve org / project UUIDs.
	o, err := s.GetOrg(ctx, orgName)
	if err != nil {
		return fmt.Errorf("get org %q: %w", orgName, err)
	}
	p, _, err := s.GetProject(ctx, projectName, []string{orgName})
	if err != nil {
		return fmt.Errorf("get project %q: %w", projectName, err)
	}

	// 2. Wait for org + project to reach IDLE (keycloak-tenant-controller
	//    has provisioned the realm groups).
	if err := waitForIdle(ctx, s, resourceTypeOrg, o.ID, cfg.ControllersForResource(resourceTypeOrg)); err != nil {
		return fmt.Errorf("wait for org IDLE: %w", err)
	}
	if err := waitForIdle(ctx, s, resourceTypeProj, p.ID, cfg.ControllersForResource(resourceTypeProj)); err != nil {
		return fmt.Errorf("wait for project IDLE: %w", err)
	}

	// 3. Build clients (k8s for secrets, Keycloak for user/groups).
	kube, err := NewInClusterKubeClient()
	if err != nil {
		return fmt.Errorf("k8s client: %w", err)
	}

	keycloakAdminPass, ok, err := kube.GetSecretValue(ctx, bcfg.KeycloakNamespace, bcfg.KeycloakService, "admin-password")
	if err != nil {
		return fmt.Errorf("read keycloak admin secret: %w", err)
	}
	if !ok || keycloakAdminPass == "" {
		return fmt.Errorf("keycloak admin-password not found in %s/%s", bcfg.KeycloakNamespace, bcfg.KeycloakService)
	}

	kc, err := NewKeycloakClient(ctx, bcfg.KeycloakBaseURL(), bcfg.KeycloakRealm, bcfg.KeycloakAdminUser, keycloakAdminPass)
	if err != nil {
		return fmt.Errorf("keycloak login: %w", err)
	}

	// 4. Determine namespace for the tenant-admin password secret.
	secretNS := bcfg.SecretNamespace
	if secretNS == "" {
		secretNS = kube.DefaultNamespace()
	}

	// 5. Ensure user exists; manage password and Secret idempotently.
	userID, password, err := ensureUserAndPassword(ctx, kc, kube, bcfg, orgName, secretNS)
	if err != nil {
		return err
	}
	_ = password // intentionally not logged

	// 6. Add user to org-level admin groups: "<orgUUID>_<group>".
	if err := addUserToGroups(ctx, kc, userID, o.ID.String(), bcfg.ProjectAdminGroups); err != nil {
		return fmt.Errorf("add org admin groups: %w", err)
	}

	// 7. Add user to project-level edge groups: "<projectUUID>_<group>".
	if err := addUserToGroups(ctx, kc, userID, p.ID.String(), bcfg.ProjectEdgeGroups); err != nil {
		return fmt.Errorf("add project edge groups: %w", err)
	}

	log.Info().
		Str("user", bcfg.TenantAdminUser).
		Str("org", orgName).
		Str("project", projectName).
		Msg("tenant-admin bootstrap complete")
	return nil
}

// ensureUserAndPassword creates the tenant-admin user if missing, generates
// or reuses a password, and persists it as a Kubernetes Secret. If the
// Secret already contains a password, that value is reused (so restarts
// don't invalidate the password an operator has already retrieved).
func ensureUserAndPassword(
	ctx context.Context,
	kc *KeycloakClient,
	kube *KubeClient,
	bcfg Config,
	orgName, secretNS string,
) (userID, password string, err error) {
	// Look up any existing user.
	existing, err := kc.FindUserByUsername(ctx, bcfg.TenantAdminUser)
	if err != nil {
		return "", "", fmt.Errorf("lookup user: %w", err)
	}

	// Check existing Secret for a previously stored password.
	storedPass, hasStored, err := kube.GetSecretValue(ctx, secretNS, bcfg.SecretName, "admin-password")
	if err != nil {
		return "", "", fmt.Errorf("read existing password secret: %w", err)
	}

	switch {
	case existing != nil && hasStored:
		log.Info().Str("user", bcfg.TenantAdminUser).Msg("tenant-admin user and password secret already present")
		return existing.ID, storedPass, nil

	case existing == nil:
		// Create user.
		u := KCUser{
			Username:      bcfg.TenantAdminUser,
			Email:         fmt.Sprintf("%s@%s.com", bcfg.TenantAdminUser, orgName),
			FirstName:     bcfg.TenantAdminUser,
			LastName:      bcfg.TenantAdminUser,
			Enabled:       true,
			EmailVerified: true,
		}
		id, err := kc.CreateUser(ctx, u)
		if err != nil {
			return "", "", fmt.Errorf("create user: %w", err)
		}
		userID = id
	default:
		userID = existing.ID
	}

	// Decide which password to set.
	if hasStored {
		password = storedPass
	} else {
		password, err = GeneratePassword()
		if err != nil {
			return "", "", fmt.Errorf("generate password: %w", err)
		}
	}

	if err := kc.SetPassword(ctx, userID, password); err != nil {
		return "", "", fmt.Errorf("set password: %w", err)
	}

	if !hasStored {
		labels := map[string]string{
			"app":      "tenant-init",
			"org":      orgName,
			"username": bcfg.TenantAdminUser,
		}
		data := map[string]string{"admin-password": password}
		if err := kube.CreateOpaqueSecret(ctx, secretNS, bcfg.SecretName, labels, data); err != nil {
			return "", "", fmt.Errorf("create password secret: %w", err)
		}
		log.Info().
			Str("namespace", secretNS).
			Str("name", bcfg.SecretName).
			Msg("created tenant-admin password secret")
	}

	return userID, password, nil
}

func addUserToGroups(ctx context.Context, kc *KeycloakClient, userID, uuidPrefix string, groupShortNames []string) error {
	for _, name := range groupShortNames {
		fullName := uuidPrefix + "_" + name
		g, err := kc.GetGroupByPath(ctx, fullName)
		if err != nil {
			return fmt.Errorf("get group %q: %w", fullName, err)
		}
		if err := kc.AddUserToGroup(ctx, userID, g.ID); err != nil {
			return fmt.Errorf("add user to group %q: %w", fullName, err)
		}
		log.Info().Str("group", fullName).Str("user_id", userID).Msg("added user to keycloak group")
	}
	return nil
}

// waitForIdle polls store.DeriveStatus until the resource is IDLE or the
// context deadline / timeout is hit.
func waitForIdle(ctx context.Context, s StoreAPI, resourceType string, id uuid.UUID, controllers []string) error {
	deadline := time.Now().Add(defaultStatusTimeout)
	ticker := time.NewTicker(defaultStatusPoll)
	defer ticker.Stop()

	for {
		status, msg := s.DeriveStatus(ctx, resourceType, id, false, controllers)
		log.Debug().
			Str("resource_type", resourceType).
			Str("id", id.String()).
			Str("status", status).
			Str("message", msg).
			Msg("polling for IDLE")
		if status == statusIdle {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s %s not IDLE after %s (status=%s, msg=%q)", resourceType, id, defaultStatusTimeout, status, msg)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
