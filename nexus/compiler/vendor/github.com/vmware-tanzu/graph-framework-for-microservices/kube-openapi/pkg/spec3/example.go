// Copyright 2021 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package spec3

import (
	"encoding/json"

	"github.com/go-openapi/swag"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
)

// Example https://swagger.io/specification/#example-object

type Example struct {
	spec.Refable
	ExampleProps
	spec.VendorExtensible
}

// MarshalJSON is a custom marshal function that knows how to encode RequestBody as JSON
func (e *Example) MarshalJSON() ([]byte, error) {
	b1, err := json.Marshal(e.Refable)
	if err != nil {
		return nil, err
	}
	b2, err := json.Marshal(e.ExampleProps)
	if err != nil {
		return nil, err
	}
	b3, err := json.Marshal(e.VendorExtensible)
	if err != nil {
		return nil, err
	}
	return swag.ConcatJSON(b1, b2, b3), nil
}

func (e *Example) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &e.Refable); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &e.ExampleProps); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &e.VendorExtensible); err != nil {
		return err
	}
	return nil
}

type ExampleProps struct {
	// Summary holds a short description of the example
	Summary string `json:"summary,omitempty"`
	// Description holds a long description of the example
	Description string `json:"description,omitempty"`
	// Embedded literal example.
	Value interface{} `json:"value,omitempty"`
	// A URL that points to the literal example. This provides the capability to reference examples that cannot easily be included in JSON or YAML documents.
	ExternalValue string `json:"externalValue,omitempty"`
}
