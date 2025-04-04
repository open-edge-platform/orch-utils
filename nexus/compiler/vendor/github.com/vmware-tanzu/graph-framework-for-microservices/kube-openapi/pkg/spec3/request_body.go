// Copyright 2021 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package spec3

import (
	"encoding/json"

	"github.com/go-openapi/swag"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
)

// RequestBody describes a single request body, more at https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#requestBodyObject
//
// Note that this struct is actually a thin wrapper around RequestBodyProps to make it referable and extensible
type RequestBody struct {
	spec.Refable
	RequestBodyProps
	spec.VendorExtensible
}

// MarshalJSON is a custom marshal function that knows how to encode RequestBody as JSON
func (r *RequestBody) MarshalJSON() ([]byte, error) {
	b1, err := json.Marshal(r.Refable)
	if err != nil {
		return nil, err
	}
	b2, err := json.Marshal(r.RequestBodyProps)
	if err != nil {
		return nil, err
	}
	b3, err := json.Marshal(r.VendorExtensible)
	if err != nil {
		return nil, err
	}
	return swag.ConcatJSON(b1, b2, b3), nil
}

func (r *RequestBody) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &r.Refable); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.RequestBodyProps); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.VendorExtensible); err != nil {
		return err
	}
	return nil
}

// RequestBodyProps describes a single request body, more at https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#requestBodyObject
type RequestBodyProps struct {
	// Description holds a brief description of the request body
	Description string `json:"description,omitempty"`
	// Content is the content of the request body. The key is a media type or media type range and the value describes it
	Content map[string]*MediaType `json:"content,omitempty"`
	// Required determines if the request body is required in the request
	Required bool `json:"required,omitempty"`
}
