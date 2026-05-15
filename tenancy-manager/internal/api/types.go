// SPDX-FileCopyrightText: 2026 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"time"

	"github.com/google/uuid"

	"github.com/open-edge-platform/orch-utils/tenancy-manager/internal/ent"
)

// --- Request types ---

// StatusUpdateRequest is the body for PUT /v1/status.
type StatusUpdateRequest struct {
	Controller   string    `json:"controller"`
	ResourceType string    `json:"resourceType"`
	ResourceID   uuid.UUID `json:"resourceId"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
}

// StatusDeleteRequest is the body for DELETE /v1/status.
type StatusDeleteRequest struct {
	Controller   string    `json:"controller"`
	ResourceType string    `json:"resourceType"`
	ResourceID   uuid.UUID `json:"resourceId"`
}

// --- Response types ---

// OrgResponse matches the current CLI/UI expected format.
type OrgResponse struct {
	Name   string        `json:"name"`
	Spec   OrgSpec       `json:"spec"`
	Status OrgStatusWrap `json:"status"`
}

type OrgSpec struct {
	Description string `json:"description"`
}

type OrgStatusWrap struct {
	OrgStatus StatusDetail `json:"orgStatus"`
}

type StatusDetail struct {
	StatusIndicator string `json:"statusIndicator"`
	Message         string `json:"message"`
	TimeStamp       int64  `json:"timeStamp"`
	UID             string `json:"uID"`
}

// ProjectResponse matches the current CLI/UI expected format.
type ProjectResponse struct {
	Name    string            `json:"name"`
	OrgName string            `json:"orgName"`
	Spec    ProjectSpec       `json:"spec"`
	Status  ProjectStatusWrap `json:"status"`
}

type ProjectSpec struct {
	Description string `json:"description"`
}

type ProjectStatusWrap struct {
	ProjectStatus StatusDetail `json:"projectStatus"`
}

// EventsResponse is the response for GET /v1/events.
type EventsResponse struct {
	Events      []EventResponse `json:"events"`
	LastEventID int64           `json:"lastEventId"`
}

// EventResponse is a single event in the events list.
type EventResponse struct {
	ID           int64      `json:"id"`
	EventType    string     `json:"eventType"`
	ResourceType string     `json:"resourceType"`
	ResourceID   uuid.UUID  `json:"resourceId"`
	ResourceName string     `json:"resourceName"`
	OrgID        *uuid.UUID `json:"orgId"`
	OrgName      *string    `json:"orgName"`
	FolderID     *uuid.UUID `json:"folderId"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// --- Conversion helpers ---

func toOrgResponse(o *ent.Org, status, message string) OrgResponse {
	return OrgResponse{
		Name: o.Name,
		Spec: OrgSpec{Description: o.Description},
		Status: OrgStatusWrap{
			OrgStatus: StatusDetail{
				StatusIndicator: status,
				Message:         message,
				TimeStamp:       o.UpdatedAt.Unix(),
				UID:             o.ID.String(),
			},
		},
	}
}

func toProjectResponse(p *ent.Project, orgName, status, message string) ProjectResponse {
	return ProjectResponse{
		Name:    p.Name,
		OrgName: orgName,
		Spec:    ProjectSpec{Description: p.Description},
		Status: ProjectStatusWrap{
			ProjectStatus: StatusDetail{
				StatusIndicator: status,
				Message:         message,
				TimeStamp:       p.UpdatedAt.Unix(),
				UID:             p.ID.String(),
			},
		},
	}
}
