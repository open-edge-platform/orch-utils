// Copyright 2022 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"reflect"

	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/schemamutation"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
)

// wrapRefs wraps OpenAPI V3 Schema refs that contain sibling elements.
// AllOf is used to wrap the Ref to prevent references from having sibling elements
// Please see https://github.com/kubernetes/kubernetes/issues/106387#issuecomment-967640388
func WrapRefs(schema *spec.Schema) *spec.Schema {
	walker := schemamutation.Walker{
		SchemaCallback: func(schema *spec.Schema) *spec.Schema {
			orig := schema
			clone := func() {
				if orig == schema {
					schema = new(spec.Schema)
					*schema = *orig
				}
			}
			if schema.Ref.String() != "" && !reflect.DeepEqual(*schema, spec.Schema{SchemaProps: spec.SchemaProps{Ref: schema.Ref}}) {
				clone()
				refSchema := new(spec.Schema)
				refSchema.Ref = schema.Ref
				schema.Ref = spec.Ref{}
				schema.AllOf = []spec.Schema{*refSchema}
			}
			return schema
		},
		RefCallback: schemamutation.RefCallbackNoop,
	}
	return walker.WalkSchema(schema)
}
