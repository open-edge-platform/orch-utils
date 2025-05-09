// Copyright 2015 go-swagger maintainers
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"reflect"
	"regexp"

	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/errors"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/strfmt"
)

type objectValidator struct {
	Path                 string
	In                   string
	MaxProperties        *int64
	MinProperties        *int64
	Required             []string
	Properties           map[string]spec.Schema
	AdditionalProperties *spec.SchemaOrBool
	PatternProperties    map[string]spec.Schema
	Root                 interface{}
	KnownFormats         strfmt.Registry
	Options              SchemaValidatorOptions
}

func (o *objectValidator) SetPath(path string) {
	o.Path = path
}

func (o *objectValidator) Applies(source interface{}, kind reflect.Kind) bool {
	// TODO: this should also work for structs
	// there is a problem in the type validator where it will be unhappy about null values
	// so that requires more testing
	r := reflect.TypeOf(source) == specSchemaType && (kind == reflect.Map || kind == reflect.Struct)
	debugLog("object validator for %q applies %t for %T (kind: %v)\n", o.Path, r, source, kind)
	return r
}

func (o *objectValidator) Validate(data interface{}) *Result {
	val := data.(map[string]interface{})
	// TODO: guard against nil data
	numKeys := int64(len(val))

	if o.MinProperties != nil && numKeys < *o.MinProperties {
		return errorHelp.sErr(errors.TooFewProperties(o.Path, o.In, *o.MinProperties, numKeys))
	}
	if o.MaxProperties != nil && numKeys > *o.MaxProperties {
		return errorHelp.sErr(errors.TooManyProperties(o.Path, o.In, *o.MaxProperties, numKeys))
	}

	res := new(Result)

	// check validity of field names
	if o.AdditionalProperties != nil && !o.AdditionalProperties.Allows {
		// Case: additionalProperties: false
		for k := range val {
			_, regularProperty := o.Properties[k]
			matched := false

			for pk := range o.PatternProperties {
				if matches, _ := regexp.MatchString(pk, k); matches {
					matched = true
					break
				}
			}

			if !regularProperty && !matched {
				// Special properties "$schema" and "id" are ignored
				res.AddErrors(errors.PropertyNotAllowed(o.Path, o.In, k))
			}
		}
	} else {
		// Cases: no additionalProperties (implying: true), or additionalProperties: true, or additionalProperties: { <<schema>> }
		for key, value := range val {
			_, regularProperty := o.Properties[key]

			// Validates property against "patternProperties" if applicable
			// BUG(fredbi): succeededOnce is always false

			// NOTE: how about regular properties which do not match patternProperties?
			matched, succeededOnce, _ := o.validatePatternProperty(key, value, res)

			if !(regularProperty || matched || succeededOnce) {

				// Cases: properties which are not regular properties and have not been matched by the PatternProperties validator
				if o.AdditionalProperties != nil && o.AdditionalProperties.Schema != nil {
					// AdditionalProperties as Schema
					res.Merge(NewSchemaValidator(o.AdditionalProperties.Schema, o.Root, o.Path+"."+key, o.KnownFormats, o.Options.Options()...).Validate(value))
				} else if regularProperty && !(matched || succeededOnce) {
					// TODO: this is dead code since regularProperty=false here
					res.AddErrors(errors.FailedAllPatternProperties(o.Path, o.In, key))
				}
			}
		}
		// Valid cases: additionalProperties: true or undefined
	}

	createdFromDefaults := map[string]bool{}

	// Property types:
	// - regular Property
	for pName, pSchema := range o.Properties {
		rName := pName
		if o.Path != "" {
			rName = o.Path + "." + pName
		}

		// Recursively validates each property against its schema
		if v, ok := val[pName]; ok {
			r := NewSchemaValidator(&pSchema, o.Root, rName, o.KnownFormats, o.Options.Options()...).Validate(v)
			res.Merge(r)
		}
	}

	// Check required properties
	if len(o.Required) > 0 {
		for _, k := range o.Required {
			if _, ok := val[k]; !ok && !createdFromDefaults[k] {
				res.AddErrors(errors.Required(o.Path+"."+k, o.In))
				continue
			}
		}
	}

	// Check patternProperties
	// TODO: it looks like we have done that twice in many cases
	for key, value := range val {
		_, regularProperty := o.Properties[key]
		matched, _ /*succeededOnce*/, patterns := o.validatePatternProperty(key, value, res)
		if !regularProperty && (matched /*|| succeededOnce*/) {
			for _, pName := range patterns {
				if v, ok := o.PatternProperties[pName]; ok {
					res.Merge(NewSchemaValidator(&v, o.Root, o.Path+"."+key, o.KnownFormats, o.Options.Options()...).Validate(value))
				}
			}
		}
	}
	return res
}

// TODO: succeededOnce is not used anywhere
func (o *objectValidator) validatePatternProperty(key string, value interface{}, result *Result) (bool, bool, []string) {
	matched := false
	succeededOnce := false
	var patterns []string

	for k, schema := range o.PatternProperties {
		sch := schema
		if match, _ := regexp.MatchString(k, key); match {
			patterns = append(patterns, k)
			matched = true
			validator := NewSchemaValidator(&sch, o.Root, o.Path+"."+key, o.KnownFormats, o.Options.Options()...)

			res := validator.Validate(value)
			result.Merge(res)
		}
	}

	// BUG(fredbi): can't get to here. Should remove dead code (commented out).

	//if succeededOnce {
	//	result.Inc()
	//}

	return matched, succeededOnce, patterns
}
