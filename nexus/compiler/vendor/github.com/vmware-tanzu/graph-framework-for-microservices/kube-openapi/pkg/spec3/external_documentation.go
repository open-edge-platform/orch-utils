// Copyright 2021 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package spec3

import (
	"encoding/json"

	"github.com/go-openapi/swag"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
)

type ExternalDocumentation struct {
	ExternalDocumentationProps
	spec.VendorExtensible
}

type ExternalDocumentationProps struct {
	// Description is a short description of the target documentation. CommonMark syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty"`
	// URL is the URL for the target documentation.
	URL string `json:"url"`
}

// MarshalJSON is a custom marshal function that knows how to encode Responses as JSON
func (e *ExternalDocumentation) MarshalJSON() ([]byte, error) {
	b1, err := json.Marshal(e.ExternalDocumentationProps)
	if err != nil {
		return nil, err
	}
	b2, err := json.Marshal(e.VendorExtensible)
	if err != nil {
		return nil, err
	}
	return swag.ConcatJSON(b1, b2), nil
}

func (e *ExternalDocumentation) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &e.ExternalDocumentationProps); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &e.VendorExtensible); err != nil {
		return err
	}
	return nil
}
