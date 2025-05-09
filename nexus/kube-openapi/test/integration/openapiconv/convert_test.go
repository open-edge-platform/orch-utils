// Copyright 2022 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package openapiconv

import (
	"encoding/json"
	"log"
	"reflect"
	"testing"

	builderv2 "github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/builder"
	builderv3 "github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/builder3"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/openapiconv"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/pkg/validation/spec"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/test/integration/pkg/generated"
	"github.com/vmware-tanzu/graph-framework-for-microservices/kube-openapi/test/integration/testutil"
)

func TestConvertGolden(t *testing.T) {
	// Generate the definition names from the map keys returned
	// from GetOpenAPIDefinitions. Anonymous function returning empty
	// Ref is not used.
	var defNames []string
	for name, _ := range generated.GetOpenAPIDefinitions(func(name string) spec.Ref {
		return spec.Ref{}
	}) {
		defNames = append(defNames, name)
	}

	// Create a minimal builder config, then call the builder with the definition names.
	config := testutil.CreateOpenAPIBuilderConfig()
	config.GetDefinitions = generated.GetOpenAPIDefinitions
	// Build the Paths using a simple WebService for the final spec
	openapiv2, serr := builderv2.BuildOpenAPISpec(testutil.CreateWebServices(false), config)
	if serr != nil {
		log.Fatalf("ERROR: %s", serr.Error())
	}

	openAPIV2JSONBeforeConversion, err := json.Marshal(openapiv2)
	if err != nil {
		t.Fatal(err)
	}
	openapiv3, serr := builderv3.BuildOpenAPISpec(testutil.CreateWebServices(false), config)
	if serr != nil {
		log.Fatalf("ERROR: %s", serr.Error())
	}

	convertedOpenAPIV3 := openapiconv.ConvertV2ToV3(openapiv2)
	if err != nil {
		t.Fatal(err)
	}
	openAPIV2JSONAfterConversion, err := json.Marshal(openapiv2)
	if !reflect.DeepEqual(openAPIV2JSONBeforeConversion, openAPIV2JSONAfterConversion) {
		t.Errorf("Expected OpenAPI V2 to be untouched before and after conversion")
	}

	if !reflect.DeepEqual(openapiv3, convertedOpenAPIV3) {
		t.Errorf("Expected converted OpenAPI to be equal, %v, %v", openapiv3, convertedOpenAPIV3)
	}
}
