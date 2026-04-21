// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package tdmclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/open-edge-platform/orch-library/go/pkg/tenancy"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/keycloak"
	"github.com/open-edge-platform/orch-utils/keycloak-tenant-controller/pkg/log"
)

// defaultTenantManagerURL is used when TENANT_MANAGER_URL is not set.
const defaultTenantManagerURL = "http://tenancy-manager.orch-iam:8080"

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

// NewMTClient creates a new multi-tenancy client. Returns an error if
// appName or kcClient are invalid.
func NewMTClient(appName string, kcClient keycloak.Client) (TdmClient, error) {
	if appName == "" {
		return nil, errors.New("appName must not be empty")
	}
	if kcClient == nil {
		return nil, errors.New("kcClient must not be nil")
	}
	return &tdmclient{
		appName:  appName,
		kcClient: kcClient,
	}, nil
}

// Init starts the tenancy event poller. It is idempotent: calling Init
// again stops any previously running poller before starting a new one.
func (tc *tdmclient) Init() error {
	tenantManagerURL := os.Getenv("TENANT_MANAGER_URL")
	if tenantManagerURL == "" {
		tenantManagerURL = defaultTenantManagerURL
	}

	if err := validateURL(tenantManagerURL); err != nil {
		return fmt.Errorf("invalid TENANT_MANAGER_URL %q: %w", tenantManagerURL, err)
	}

	// Stop any previously running poller before creating a new one.
	if tc.cancel != nil {
		tc.cancel()
		tc.cancel = nil
	}

	handler := &keycloakHandler{kcClient: tc.kcClient}

	poller, err := tenancy.NewPoller(tenantManagerURL, tc.appName, handler,
		func(cfg *tenancy.PollerConfig) {
			cfg.OnError = func(err error, msg string) {
				log.Errorf("%s: %v", msg, err)
			}
		},
	)
	if err != nil {
		return fmt.Errorf("create tenancy poller: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	tc.cancel = cancel

	go func() {
		if err := poller.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Errorf("poller stopped unexpectedly (controller=%s): %v", tc.appName, err)
		}
	}()

	log.Infof("Tenancy poller started for %s", tc.appName)
	return nil
}

// Stop shuts down the tenancy event poller. Safe to call multiple times
// and before Init.
func (tc *tdmclient) Stop() {
	if tc.cancel != nil {
		tc.cancel()
		tc.cancel = nil
	}
}

// validateURL returns an error if rawURL is not a valid absolute HTTP/HTTPS URL.
func validateURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("URL must have a host")
	}
	return nil
}

// keycloakHandler implements tenancy.Handler for the keycloak controller.
type keycloakHandler struct {
	kcClient keycloak.Client
}

func (h *keycloakHandler) HandleEvent(_ context.Context, event tenancy.Event) error {
	switch {
	case event.ResourceType == tenancy.ResourceTypeOrg && event.EventType == tenancy.EventTypeCreated:
		return h.handleOrgCreated(event)
	case event.ResourceType == tenancy.ResourceTypeOrg && event.EventType == tenancy.EventTypeDeleted:
		return h.handleOrgDeleted(event)
	case event.ResourceType == tenancy.ResourceTypeProject && event.EventType == tenancy.EventTypeCreated:
		return h.handleProjectCreated(event)
	case event.ResourceType == tenancy.ResourceTypeProject && event.EventType == tenancy.EventTypeDeleted:
		return h.handleProjectDeleted(event)
	default:
		log.Infof("ignoring unrecognised event: type=%s resource=%s id=%s",
			event.EventType, event.ResourceType, event.ResourceID)
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
