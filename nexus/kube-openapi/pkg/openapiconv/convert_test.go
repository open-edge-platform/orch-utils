// Copyright 2022 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package openapiconv

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/spec3"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
)

func TestConvert(t *testing.T) {

	tcs := []struct {
		groupVersion string
	}{{
		"batch.v1",
	}, {
		"api.v1",
	}, {
		"apiextensions.k8s.io.v1",
	}}

	for _, tc := range tcs {

		spec2JSON, err := ioutil.ReadFile(filepath.Join("testdata_generated_from_k8s/v2_" + tc.groupVersion + ".json"))
		if err != nil {
			t.Fatal(err)
		}
		var swaggerSpec spec.Swagger
		err = json.Unmarshal(spec2JSON, &swaggerSpec)
		if err != nil {
			t.Fatal(err)
		}

		openAPIV2JSONBeforeConversion, err := json.Marshal(swaggerSpec)
		if err != nil {
			t.Fatal(err)
		}

		convertedV3Spec := ConvertV2ToV3(&swaggerSpec)

		openAPIV2JSONAfterConversion, err := json.Marshal(swaggerSpec)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(openAPIV2JSONBeforeConversion, openAPIV2JSONAfterConversion) {
			t.Errorf("Expected OpenAPI V2 to be untouched before and after conversion")
		}

		spec3JSON, err := ioutil.ReadFile(filepath.Join("testdata_generated_from_k8s/v3_" + tc.groupVersion + ".json"))
		if err != nil {
			t.Fatal(err)
		}

		var V3Spec spec3.OpenAPI
		json.Unmarshal(spec3JSON, &V3Spec)
		if !reflect.DeepEqual(V3Spec, *convertedV3Spec) {
			t.Error("Expected specs to be equal")
		}
	}
}
