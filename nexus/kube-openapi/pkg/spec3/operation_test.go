// Copyright 2021 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package spec3_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/spec3"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
)

func TestOperationJSONSerialization(t *testing.T) {
	cases := []struct {
		name           string
		target         *spec3.Operation
		expectedOutput string
	}{
		{
			name: "basic",
			target: &spec3.Operation{
				OperationProps: spec3.OperationProps{
					Tags:        []string{"pet"},
					Summary:     "Updates a pet in the store with form data",
					OperationId: "updatePetWithForm",
					Parameters: []*spec3.Parameter{
						&spec3.Parameter{
							ParameterProps: spec3.ParameterProps{
								Name:        "petId",
								In:          "path",
								Description: "ID of pet that needs to be updated",
								Required:    true,
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type: []string{"string"},
									},
								},
							},
						},
					},
					RequestBody: &spec3.RequestBody{
						RequestBodyProps: spec3.RequestBodyProps{
							Content: map[string]*spec3.MediaType{
								"application/x-www-form-urlencoded": &spec3.MediaType{
									MediaTypeProps: spec3.MediaTypeProps{
										Schema: &spec.Schema{
											SchemaProps: spec.SchemaProps{
												Type: []string{"object"},
												Properties: map[string]spec.Schema{
													"name": spec.Schema{
														SchemaProps: spec.SchemaProps{
															Description: "Updated name of the pet",
															Type:        []string{"string"},
														},
													},
													"status": spec.Schema{
														SchemaProps: spec.SchemaProps{
															Description: "Updated status of the pet",
															Type:        []string{"string"},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					Responses: &spec3.Responses{
						ResponsesProps: spec3.ResponsesProps{
							StatusCodeResponses: map[int]*spec3.Response{
								200: &spec3.Response{
									ResponseProps: spec3.ResponseProps{
										Description: "Pet updated.",
										Content: map[string]*spec3.MediaType{
											"application/json": &spec3.MediaType{},
											"application/xml":  &spec3.MediaType{},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedOutput: `{"tags":["pet"],"summary":"Updates a pet in the store with form data","operationId":"updatePetWithForm","parameters":[{"name":"petId","in":"path","description":"ID of pet that needs to be updated","required":true,"schema":{"type":"string"}}],"requestBody":{"content":{"application/x-www-form-urlencoded":{"schema":{"type":"object","properties":{"name":{"description":"Updated name of the pet","type":"string"},"status":{"description":"Updated status of the pet","type":"string"}}}}}},"responses":{"200":{"description":"Pet updated.","content":{"application/json":{},"application/xml":{}}}}}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rawTarget, err := json.Marshal(tc.target)
			if err != nil {
				t.Fatal(err)
			}
			serializedTarget := string(rawTarget)
			if !cmp.Equal(serializedTarget, tc.expectedOutput) {
				t.Fatalf("diff %s", cmp.Diff(serializedTarget, tc.expectedOutput))
			}
		})
	}
}
