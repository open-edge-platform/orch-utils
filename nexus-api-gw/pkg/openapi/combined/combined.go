// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package combined

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/config"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/api"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/declarative"
)

func Specs() openapi3.T {
	newSchema := initializeSchema()

	nexusSchemas := api.Schemas

	for _, schema := range nexusSchemas {
		mergePaths(&newSchema, schema)
		mergeComponents(&newSchema, schema)
	}

	return newSchema
}

func initializeSchema() openapi3.T {
	return openapi3.T{
		OpenAPI:    "3.0.0",
		Components: declarative.Schema.Components,
		Info:       declarative.Schema.Info,
		Paths:      declarative.Schema.Paths,
		Security:   declarative.Schema.Security,
		Servers: openapi3.Servers{
			&openapi3.Server{
				URL: config.Cfg.TenantAPIGwDomain + "/tsm/",
			},
			&openapi3.Server{
				URL: config.Cfg.TenantAPIGwDomain + "/local/v1/",
			},
			&openapi3.Server{
				URL: "http://localhost:3000/v1/",
			},
			&openapi3.Server{
				URL: "http://localhost:3000/",
			},
		},
		Tags:         declarative.Schema.Tags,
		ExternalDocs: declarative.Schema.ExternalDocs,
	}
}

func mergePaths(newSchema *openapi3.T, schema openapi3.T) {
	if newSchema.Paths == nil {
		newSchema.Paths = openapi3.Paths{}
	}
	for k, v := range schema.Paths {
		newSchema.Paths[k] = v
	}
}

func mergeComponents(newSchema *openapi3.T, schema openapi3.T) {
	mergeSchemas(&newSchema.Components.Schemas, schema.Components.Schemas)
	mergeParameters(&newSchema.Components.Parameters, schema.Components.Parameters)
	mergeHeaders(&newSchema.Components.Headers, schema.Components.Headers)
	mergeRequestBodies(&newSchema.Components.RequestBodies, schema.Components.RequestBodies)
	mergeResponses(&newSchema.Components.Responses, schema.Components.Responses)
	mergeSecuritySchemes(&newSchema.Components.SecuritySchemes, schema.Components.SecuritySchemes)
	mergeExamples(&newSchema.Components.Examples, schema.Components.Examples)
	mergeLinks(&newSchema.Components.Links, schema.Components.Links)
	mergeCallbacks(&newSchema.Components.Callbacks, schema.Components.Callbacks)
}

func mergeSchemas(dest *openapi3.Schemas, src openapi3.Schemas) {
	if *dest == nil {
		*dest = openapi3.Schemas{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}

func mergeParameters(dest *openapi3.ParametersMap, src openapi3.ParametersMap) {
	if *dest == nil {
		*dest = openapi3.ParametersMap{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}

func mergeHeaders(dest *openapi3.Headers, src openapi3.Headers) {
	if *dest == nil {
		*dest = openapi3.Headers{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}

func mergeRequestBodies(dest *openapi3.RequestBodies, src openapi3.RequestBodies) {
	if *dest == nil {
		*dest = openapi3.RequestBodies{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}

func mergeResponses(dest *openapi3.Responses, src openapi3.Responses) {
	if *dest == nil {
		*dest = openapi3.Responses{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}

func mergeSecuritySchemes(dest *openapi3.SecuritySchemes, src openapi3.SecuritySchemes) {
	if *dest == nil {
		*dest = openapi3.SecuritySchemes{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}

func mergeExamples(dest *openapi3.Examples, src openapi3.Examples) {
	if *dest == nil {
		*dest = openapi3.Examples{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}

func mergeLinks(dest *openapi3.Links, src openapi3.Links) {
	if *dest == nil {
		*dest = openapi3.Links{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}

func mergeCallbacks(dest *openapi3.Callbacks, src openapi3.Callbacks) {
	if *dest == nil {
		*dest = openapi3.Callbacks{}
	}
	for k, v := range src {
		(*dest)[k] = v
	}
}
