// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package tdmclient

import (
	"context"
	"fmt"
	"os"

	"github.com/open-edge-platform/orch-library/go/pkg/tenancy"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/keycloak"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/log"
)

// TdmClient is the interface for the tenancy data model client.
type TdmClient interface {
	Init() error
	Stop()
}

type tdmclient struct {
	appName  string
	kcClient keycloak.Client
	cancel   context.CancelFunc
}

// NewMTClient creates a new multi-tenancy client.
func NewMTClient(appName string, kcClient keycloak.Client) TdmClient {
	return &tdmclient{
		appName:  appName,
		kcClient: kcClient,
	}
}

// Init starts the tenancy event poller.
func (tc *tdmclient) Init() error {
	tenantManagerURL := os.Getenv("TENANT_MANAGER_URL")
	if tenantManagerURL == "" {
		tenantManagerURL = "http://tenancy-manager.orch-iam:8080"
	}

	handler := &keycloakHandler{kcClient: tc.kcClient}

	poller := tenancy.NewPoller(tenantManagerURL, tc.appName, handler,
		func(cfg *tenancy.PollerConfig) {
			cfg.OnError = func(err error, msg string) {
				log.Errorf("%s: %v", msg, err)
			}
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	tc.cancel = cancel

	go func() {
		if err := poller.Run(ctx); err != nil && ctx.Err() == nil {
			log.Errorf("poller stopped with error: %v", err)
		}
	}()

	log.Infof("Tenancy poller started for %s", tc.appName)
	return nil
}

// Stop cancels the poller.
func (tc *tdmclient) Stop() {
	if tc.cancel != nil {
		tc.cancel()
	}
}

// keycloakHandler implements tenancy.Handler for the keycloak controller.
type keycloakHandler struct {
	kcClient keycloak.Client
}

func (h *keycloakHandler) HandleEvent(ctx context.Context, event tenancy.Event) error {
	switch {
	case event.ResourceType == "org" && event.EventType == "created":
		return h.handleOrgCreated(event)
	case event.ResourceType == "org" && event.EventType == "deleted":
		return h.handleOrgDeleted(event)
	case event.ResourceType == "project" && event.EventType == "created":
		return h.handleProjectCreated(event)
	case event.ResourceType == "project" && event.EventType == "deleted":
		return h.handleProjectDeleted(event)
	default:
		return nil
	}
}

func (h *keycloakHandler) handleOrgCreated(event tenancy.Event) error {
	log.Infof("Creating org %s in Keycloak", event.ResourceName)
	if err := h.kcClient.CreateOrg(event.ResourceID.String()); err != nil {
		return fmt.Errorf("create org %s in Keycloak: %w", event.ResourceName, err)
	}
	log.Infof("Org %s created in Keycloak", event.ResourceName)
	return nil
}

func (h *keycloakHandler) handleOrgDeleted(event tenancy.Event) error {
	log.Infof("Deleting org %s from Keycloak", event.ResourceName)
	if err := h.kcClient.DeleteOrg(event.ResourceID.String()); err != nil {
		return fmt.Errorf("delete org %s from Keycloak: %w", event.ResourceName, err)
	}
	log.Infof("Org %s deleted from Keycloak", event.ResourceName)
	return nil
}

func (h *keycloakHandler) handleProjectCreated(event tenancy.Event) error {
	orgID := ""
	if event.OrgID != nil {
		orgID = event.OrgID.String()
	}
	log.Infof("Creating project %s (org %s) in Keycloak", event.ResourceName, orgID)
	if err := h.kcClient.CreateProject(orgID, event.ResourceID.String()); err != nil {
		return fmt.Errorf("create project %s in Keycloak: %w", event.ResourceName, err)
	}
	log.Infof("Project %s created in Keycloak", event.ResourceName)
	return nil
}

func (h *keycloakHandler) handleProjectDeleted(event tenancy.Event) error {
	log.Infof("Deleting project %s from Keycloak", event.ResourceName)
	if err := h.kcClient.DeleteProject(event.ResourceID.String()); err != nil {
		return fmt.Errorf("delete project %s from Keycloak: %w", event.ResourceName, err)
	}
	log.Infof("Project %s deleted from Keycloak", event.ResourceName)
	return nil
}
