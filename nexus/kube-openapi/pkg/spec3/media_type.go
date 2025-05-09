// Copyright 2021 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package spec3

import (
	"encoding/json"

	"github.com/go-openapi/swag"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
)

// MediaType a struct that allows you to specify content format, more at https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#mediaTypeObject
//
// Note that this struct is actually a thin wrapper around MediaTypeProps to make it referable and extensible
type MediaType struct {
	MediaTypeProps
	spec.VendorExtensible
}

// MarshalJSON is a custom marshal function that knows how to encode MediaType as JSON
func (m *MediaType) MarshalJSON() ([]byte, error) {
	b1, err := json.Marshal(m.MediaTypeProps)
	if err != nil {
		return nil, err
	}
	b2, err := json.Marshal(m.VendorExtensible)
	if err != nil {
		return nil, err
	}
	return swag.ConcatJSON(b1, b2), nil
}

func (m *MediaType) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &m.MediaTypeProps); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &m.VendorExtensible); err != nil {
		return err
	}
	return nil
}

// MediaTypeProps a struct that allows you to specify content format, more at https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#mediaTypeObject
type MediaTypeProps struct {
	// Schema holds the schema defining the type used for the media type
	Schema *spec.Schema `json:"schema,omitempty"`
	// Example of the media type
	Example interface{} `json:"example,omitempty"`
	// Examples of the media type. Each example object should match the media type and specific schema if present
	Examples map[string]*Example `json:"examples,omitempty"`
	// A map between a property name and its encoding information. The key, being the property name, MUST exist in the schema as a property. The encoding object SHALL only apply to requestBody objects when the media type is multipart or application/x-www-form-urlencoded
	Encoding map[string]*Encoding `json:"encoding,omitempty"`
}
