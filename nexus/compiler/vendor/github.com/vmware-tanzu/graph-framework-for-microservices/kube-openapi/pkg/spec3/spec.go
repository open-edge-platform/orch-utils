// Copyright 2021 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package spec3

import (
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
)

// OpenAPI is an object that describes an API and conforms to the OpenAPI Specification.
type OpenAPI struct {
	// Version represents the semantic version number of the OpenAPI Specification that this document uses
	Version string `json:"openapi"`
	// Info provides metadata about the API
	Info *spec.Info `json:"info"`
	// Paths holds the available target and operations for the API
	Paths *Paths `json:"paths,omitempty"`
	// Servers is an array of Server objects which provide connectivity information to a target server
	Servers []*Server `json:"servers,omitempty"`
	// Components hold various schemas for the specification
	Components *Components `json:"components,omitempty"`
	// ExternalDocs holds additional external documentation
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
}
