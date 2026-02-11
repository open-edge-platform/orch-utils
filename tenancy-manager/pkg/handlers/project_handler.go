// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	projectv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/project.edge-orchestrator.intel.com/v1"
	nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectHandler handles project-related HTTP requests.
type ProjectHandler struct {
	NexusClient *nexus_client.Clientset
	Logger      zerolog.Logger
}

// NewProjectHandler creates a new project handler.
func NewProjectHandler(nexusClient *nexus_client.Clientset, logger zerolog.Logger) *ProjectHandler {
	return &ProjectHandler{
		NexusClient: nexusClient,
		Logger:      logger,
	}
}

// CreateProject creates a new project.
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	log := h.Logger.With().Str("handler", "CreateProject").Logger()
	log.Info().Msg("Creating project")

	var projectObj projectv1.Project
	if err := c.Bind(&projectObj); err != nil {
		log.Error().Msgf("Invalid request body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Validate required fields
	if projectObj.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Project name is required")
	}

	// In Nexus, org/folder hierarchy is managed via labels, not Spec fields
	// Initialize labels if not present
	if projectObj.Labels == nil {
		projectObj.Labels = make(map[string]string)
	}

	// Extract org from labels (required)
	orgId := projectObj.Labels["orgs.org.edge-orchestrator.intel.com"]
	if orgId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "OrgId is required (provide in labels['orgs.org.edge-orchestrator.intel.com'])")
	}

	// Default folderId to "default" if not provided in labels
	folderId := projectObj.Labels["folders.folder.edge-orchestrator.intel.com"]
	if folderId == "" {
		folderId = "default"
		projectObj.Labels["folders.folder.edge-orchestrator.intel.com"] = folderId
	}

	// Create project using Nexus client
	tenant := h.NexusClient.TenancyMultiTenancy()
	projectClient, err := tenant.Config().Orgs(orgId).Folders(folderId).AddProjects(context.Background(), &projectObj)
	if err != nil {
		log.Error().Msgf("Failed to create project: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Project created successfully",
		"project": projectClient.Project,
	})
}

// ListProjects lists all projects, optionally filtered by orgId and folderId.
func (h *ProjectHandler) ListProjects(c echo.Context) error {
	log := h.Logger.With().Str("handler", "ListProjects").Logger()
	log.Info().Msg("Listing projects")

	orgId := c.QueryParam("orgId")
	folderId := c.QueryParam("folderId")

	var projects []*nexus_client.ProjectProject
	var err error

	// Build label selector for filtering
	listOpts := metav1.ListOptions{}
	if orgId != "" || folderId != "" {
		labelParts := []string{}
		if orgId != "" {
			labelParts = append(labelParts, "orgs.org.edge-orchestrator.intel.com="+orgId)
		}
		if folderId != "" {
			labelParts = append(labelParts, "folders.folder.edge-orchestrator.intel.com="+folderId)
		}
		listOpts.LabelSelector = strings.Join(labelParts, ",")
	}

	// List projects with optional filtering via label selector
	projects, err = h.NexusClient.Project().ListProjects(context.Background(), listOpts)
	if err != nil {
		log.Error().Msgf("Failed to list projects: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list projects")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": projects,
		"total": len(projects),
	})
}

// GetProject gets a specific project by ID.
func (h *ProjectHandler) GetProject(c echo.Context) error {
	projectID := c.Param("projectId")
	log := h.Logger.With().Str("handler", "GetProject").Str("projectId", projectID).Logger()
	log.Info().Msg("Getting project")

	if projectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	// Get project by name - need to search across namespaces
	projectList, err := h.NexusClient.Project().ListProjects(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Error().Msgf("Failed to list projects: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get project")
	}

	// Find project by name
	for _, project := range projectList {
		if project.Name == projectID {
			return c.JSON(http.StatusOK, project)
		}
	}

	return echo.NewHTTPError(http.StatusNotFound, "Project not found")
}

// UpdateProject updates an existing project.
func (h *ProjectHandler) UpdateProject(c echo.Context) error {
	projectID := c.Param("projectId")
	log := h.Logger.With().Str("handler", "UpdateProject").Str("projectId", projectID).Logger()
	log.Info().Msg("Updating project")

	if projectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	var projectObj projectv1.Project
	if err := c.Bind(&projectObj); err != nil {
		log.Error().Msgf("Invalid request body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Initialize labels if not present
	if projectObj.Labels == nil {
		projectObj.Labels = make(map[string]string)
	}

	// Get orgId and folderId from labels (required for update)
	orgId := projectObj.Labels["orgs.org.edge-orchestrator.intel.com"]
	folderId := projectObj.Labels["folders.folder.edge-orchestrator.intel.com"]
	if orgId == "" || folderId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "OrgId and FolderId are required in labels for update")
	}

	existingProject, err := h.NexusClient.TenancyMultiTenancy().Config().
		Orgs(orgId).Folders(folderId).
		GetProjects(context.Background(), projectID)
	if err != nil {
		log.Error().Msgf("Failed to get project for update: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	// Update project fields from request
	if projectObj.Labels != nil {
		existingProject.SetLabels(projectObj.Labels)
	}

	// Update in database
	err = existingProject.Update(context.Background())
	if err != nil {
		log.Error().Msgf("Failed to update project: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update project")
	}

	return c.JSON(http.StatusOK, existingProject.Project)
}

// DeleteProject deletes a project.
func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	projectID := c.Param("projectId")
	log := h.Logger.With().Str("handler", "DeleteProject").Str("projectId", projectID).Logger()
	log.Info().Msg("Deleting project")

	if projectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	// Need orgId and folderId from query parameters
	orgId := c.QueryParam("orgId")
	folderId := c.QueryParam("folderId")

	if orgId == "" || folderId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "OrgId and FolderId query parameters are required")
	}

	// Delete project
	err := h.NexusClient.TenancyMultiTenancy().Config().
		Orgs(orgId).Folders(folderId).
		DeleteProjects(context.Background(), projectID)
	if err != nil {
		log.Error().Msgf("Failed to delete project: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Project not found or could not be deleted")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Project deleted successfully",
	})
}

// GetProjectStatus gets the status of a project.
func (h *ProjectHandler) GetProjectStatus(c echo.Context) error {
	projectID := c.Param("projectId")
	log := h.Logger.With().Str("handler", "GetProjectStatus").Str("projectId", projectID).Logger()
	log.Info().Msg("Getting project status")

	if projectID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Project ID is required")
	}

	// Need orgId and folderId from query parameters
	orgId := c.QueryParam("orgId")
	folderId := c.QueryParam("folderId")

	if orgId == "" || folderId == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "OrgId and FolderId query parameters are required")
	}

	// Get project status
	projectClient, err := h.NexusClient.TenancyMultiTenancy().Config().
		Orgs(orgId).Folders(folderId).
		GetProjects(context.Background(), projectID)
	if err != nil {
		log.Error().Msgf("Failed to get project: %v", err)
		return echo.NewHTTPError(http.StatusNotFound, "Project not found")
	}

	status, err := projectClient.GetProjectStatus(context.Background())
	if err != nil {
		log.Error().Msgf("Failed to get project status: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get project status")
	}

	return c.JSON(http.StatusOK, status)
}
