// Copyright 2015 go-swagger maintainers
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var pathItem = PathItem{
	Refable: Refable{Ref: MustCreateRef("Dog")},
	VendorExtensible: VendorExtensible{
		Extensions: map[string]interface{}{
			"x-framework": "go-swagger",
		},
	},
	PathItemProps: PathItemProps{
		Get: &Operation{
			OperationProps: OperationProps{Description: "get operation description"},
		},
		Put: &Operation{
			OperationProps: OperationProps{Description: "put operation description"},
		},
		Post: &Operation{
			OperationProps: OperationProps{Description: "post operation description"},
		},
		Delete: &Operation{
			OperationProps: OperationProps{Description: "delete operation description"},
		},
		Options: &Operation{
			OperationProps: OperationProps{Description: "options operation description"},
		},
		Head: &Operation{
			OperationProps: OperationProps{Description: "head operation description"},
		},
		Patch: &Operation{
			OperationProps: OperationProps{Description: "patch operation description"},
		},
		Parameters: []Parameter{
			{
				ParamProps: ParamProps{In: "path"},
			},
		},
	},
}

const pathItemJSON = `{
	"$ref": "Dog",
	"x-framework": "go-swagger",
	"get": { "description": "get operation description" },
	"put": { "description": "put operation description" },
	"post": { "description": "post operation description" },
	"delete": { "description": "delete operation description" },
	"options": { "description": "options operation description" },
	"head": { "description": "head operation description" },
	"patch": { "description": "patch operation description" },
	"parameters": [{"in":"path"}]
}`

func TestIntegrationPathItem(t *testing.T) {
	var actual PathItem
	if assert.NoError(t, json.Unmarshal([]byte(pathItemJSON), &actual)) {
		assert.EqualValues(t, actual, pathItem)
	}

	assertParsesJSON(t, pathItemJSON, pathItem)
}
