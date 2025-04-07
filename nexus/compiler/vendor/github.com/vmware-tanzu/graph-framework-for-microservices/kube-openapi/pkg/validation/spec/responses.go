// Copyright 2015 go-swagger maintainers
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/go-openapi/swag"
)

// Responses is a container for the expected responses of an operation.
// The container maps a HTTP response code to the expected response.
// It is not expected from the documentation to necessarily cover all possible HTTP response codes,
// since they may not be known in advance. However, it is expected from the documentation to cover
// a successful operation response and any known errors.
//
// The `default` can be used a default response object for all HTTP codes that are not covered
// individually by the specification.
//
// The `Responses Object` MUST contain at least one response code, and it SHOULD be the response
// for a successful operation call.
//
// For more information: http://goo.gl/8us55a#responsesObject
type Responses struct {
	VendorExtensible
	ResponsesProps
}

// UnmarshalJSON hydrates this items instance with the data from JSON
func (r *Responses) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &r.ResponsesProps); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.VendorExtensible); err != nil {
		return err
	}
	if reflect.DeepEqual(ResponsesProps{}, r.ResponsesProps) {
		r.ResponsesProps = ResponsesProps{}
	}
	return nil
}

// MarshalJSON converts this items object to JSON
func (r Responses) MarshalJSON() ([]byte, error) {
	b1, err := json.Marshal(r.ResponsesProps)
	if err != nil {
		return nil, err
	}
	b2, err := json.Marshal(r.VendorExtensible)
	if err != nil {
		return nil, err
	}
	concated := swag.ConcatJSON(b1, b2)
	return concated, nil
}

// ResponsesProps describes all responses for an operation.
// It tells what is the default response and maps all responses with a
// HTTP status code.
type ResponsesProps struct {
	Default             *Response
	StatusCodeResponses map[int]Response
}

// MarshalJSON marshals responses as JSON
func (r ResponsesProps) MarshalJSON() ([]byte, error) {
	toser := map[string]Response{}
	if r.Default != nil {
		toser["default"] = *r.Default
	}
	for k, v := range r.StatusCodeResponses {
		toser[strconv.Itoa(k)] = v
	}
	return json.Marshal(toser)
}

// UnmarshalJSON unmarshals responses from JSON
func (r *ResponsesProps) UnmarshalJSON(data []byte) error {
	var res map[string]Response
	if err := json.Unmarshal(data, &res); err != nil {
		return nil
	}
	if v, ok := res["default"]; ok {
		r.Default = &v
		delete(res, "default")
	}
	for k, v := range res {
		if nk, err := strconv.Atoi(k); err == nil {
			if r.StatusCodeResponses == nil {
				r.StatusCodeResponses = map[int]Response{}
			}
			r.StatusCodeResponses[nk] = v
		}
	}
	return nil
}
