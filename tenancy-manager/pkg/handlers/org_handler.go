// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	orgv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/org.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OrgHandler handles organization-related HTTP requests.
type OrgHandler struct {
	NexusClient *nexus_client.Clientset
	Logger      zerolog.Logger
}

// NewOrgHandler creates a new organization handler.
func NewOrgHandler(nexusClient *nexus_client.Clientset, logger zerolog.Logger) *OrgHandler {
	return &OrgHandler{
		NexusClient: nexusClient,
		Logger:      logger,
	}
}

// CreateOrg creates a new organization.
func (h *OrgHandler) CreateOrg(c echo.Context) error {
	log := h.Logger.With().Str("handler", "CreateOrg").Logger()
	log.Info().Msg("Creating organization")

	var orgObj orgv1.Org
	if err := c.Bind(&orgObj); err != nil {
		log.Error().Msgf("Invalid request body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if orgObj.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Organization name is required")
	}

	// Create org using Nexus client
	tenant := h.NexusClient.TenancyMultiTenancy()
	orgClient, err := tenant.Config().AddOrgs(context.Background(), &orgObj)
	if err != nil {
		log.Error().Msgf("Failed to create org: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create organization")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Organization created successfully",
		"org":     orgClient.Org,
	})
}

// ListOrgs lists all organizations.
func (h *OrgHandler) ListOrgs(c echo.Context) error {
	log := h.Logger.With().Str("handler", "ListOrgs").Logger()
	log.Info().Msg("Listing organizations")

	// Get org list from Kubernetes API
	orgList, err := h.NexusClient.Org().ListOrgs(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Error().Msgf("Failed to list orgs: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list organizations")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": orgList,
		"total": len(orgList),
	})
}

// GetOrg gets a specific organization by ID.
func (h *OrgHandler) GetOrg(c echo.Context) error {
	orgID := c.Param("orgId")
	log := h.Logger.With().Str("handler", "GetOrg").Str("orgId", orgID).Logger()
	log.Info().Msg("Getting organization")

	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Organization ID is required")
	}

	// Get organization by name
	org, err := h.NexusClient.Org().GetOrgByName(context.Background(), orgID)
	if err != nil {
		log.Error().Msgf("Failed to get org: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Organization not found")
	}

	return c.JSON(http.StatusOK, org)
}

// UpdateOrg updates an existing organization.
func (h *OrgHandler) UpdateOrg(c echo.Context) error {
	orgID := c.Param("orgId")
	log := h.Logger.With().Str("handler", "UpdateOrg").Str("orgId", orgID).Logger()
	log.Info().Msg("Updating organization")

	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Organization ID is required")
	}

	var orgObj orgv1.Org
	if err := c.Bind(&orgObj); err != nil {
		log.Error().Msgf("Invalid request body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Get existing organization via nexus client
	existingOrg, err := h.NexusClient.TenancyMultiTenancy().Config().GetOrgs(context.Background(), orgID)
	if err != nil {
		log.Error().Msgf("Failed to get org for update: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Organization not found")
	}

	// Update organization fields from request
	if orgObj.Labels != nil {
		existingOrg.SetLabels(orgObj.Labels)
	}

	// Update in database
	err = existingOrg.Update(context.Background())
	if err != nil {
		log.Error().Msgf("Failed to update org: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update organization")
	}

	return c.JSON(http.StatusOK, existingOrg.Org)
}

// DeleteOrg deletes an organization.
func (h *OrgHandler) DeleteOrg(c echo.Context) error {
	orgID := c.Param("orgId")
	log := h.Logger.With().Str("handler", "DeleteOrg").Str("orgId", orgID).Logger()
	log.Info().Msg("Deleting organization")

	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Organization ID is required")
	}

	// Delete organization
	err := h.NexusClient.TenancyMultiTenancy().Config().DeleteOrgs(context.Background(), orgID)
	if err != nil {
		log.Error().Msgf("Failed to delete org: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Organization not found or could not be deleted")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Organization deleted successfully",
	})
}

// GetOrgStatus gets the status of an organization.
func (h *OrgHandler) GetOrgStatus(c echo.Context) error {
	orgID := c.Param("orgId")
	log := h.Logger.With().Str("handler", "GetOrgStatus").Str("orgId", orgID).Logger()
	log.Info().Msg("Getting organization status")

	if orgID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Organization ID is required")
	}

	// Get organization status
	orgClient, err := h.NexusClient.TenancyMultiTenancy().Config().GetOrgs(context.Background(), orgID)
	if err != nil {
		log.Error().Msgf("Failed to get org: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Organization not found")
	}

	status, err := orgClient.GetOrgStatus(context.Background())
	if err != nil {
		log.Error().Msgf("Failed to get org status: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get organization status")
	}

	return c.JSON(http.StatusOK, status)
}
