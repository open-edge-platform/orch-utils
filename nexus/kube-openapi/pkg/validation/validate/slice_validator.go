// Copyright 2015 go-swagger maintainers
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"
	"reflect"

	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/strfmt"
)

type schemaSliceValidator struct {
	Path            string
	In              string
	MaxItems        *int64
	MinItems        *int64
	UniqueItems     bool
	AdditionalItems *spec.SchemaOrBool
	Items           *spec.SchemaOrArray
	Root            interface{}
	KnownFormats    strfmt.Registry
	Options         SchemaValidatorOptions
}

func (s *schemaSliceValidator) SetPath(path string) {
	s.Path = path
}

func (s *schemaSliceValidator) Applies(source interface{}, kind reflect.Kind) bool {
	_, ok := source.(*spec.Schema)
	r := ok && kind == reflect.Slice
	return r
}

func (s *schemaSliceValidator) Validate(data interface{}) *Result {
	result := new(Result)
	if data == nil {
		return result
	}
	val := reflect.ValueOf(data)
	size := val.Len()

	if s.Items != nil && s.Items.Schema != nil {
		validator := NewSchemaValidator(s.Items.Schema, s.Root, s.Path, s.KnownFormats, s.Options.Options()...)
		for i := 0; i < size; i++ {
			validator.SetPath(fmt.Sprintf("%s[%d]", s.Path, i))
			value := val.Index(i)
			result.Merge(validator.Validate(value.Interface()))
		}
	}

	itemsSize := 0
	if s.Items != nil && len(s.Items.Schemas) > 0 {
		itemsSize = len(s.Items.Schemas)
		for i := 0; i < itemsSize; i++ {
			validator := NewSchemaValidator(&s.Items.Schemas[i], s.Root, fmt.Sprintf("%s[%d]", s.Path, i), s.KnownFormats, s.Options.Options()...)
			if val.Len() <= i {
				break
			}
			result.Merge(validator.Validate(val.Index(i).Interface()))
		}
	}
	if s.AdditionalItems != nil && itemsSize < size {
		if s.Items != nil && len(s.Items.Schemas) > 0 && !s.AdditionalItems.Allows {
			result.AddErrors(arrayDoesNotAllowAdditionalItemsMsg())
		}
		if s.AdditionalItems.Schema != nil {
			for i := itemsSize; i < size-itemsSize+1; i++ {
				validator := NewSchemaValidator(s.AdditionalItems.Schema, s.Root, fmt.Sprintf("%s[%d]", s.Path, i), s.KnownFormats, s.Options.Options()...)
				result.Merge(validator.Validate(val.Index(i).Interface()))
			}
		}
	}

	if s.MinItems != nil {
		if err := MinItems(s.Path, s.In, int64(size), *s.MinItems); err != nil {
			result.AddErrors(err)
		}
	}
	if s.MaxItems != nil {
		if err := MaxItems(s.Path, s.In, int64(size), *s.MaxItems); err != nil {
			result.AddErrors(err)
		}
	}
	if s.UniqueItems {
		if err := UniqueItems(s.Path, s.In, val.Interface()); err != nil {
			result.AddErrors(err)
		}
	}
	result.Inc()
	return result
}
