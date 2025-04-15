// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package combined_test

import (
	"encoding/json"

	"github.com/getkin/kin-openapi/openapi3"
	yamlv1 "github.com/ghodss/yaml"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/model"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/api"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/combined"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/declarative"
	"github.com/vmware-tanzu/graph-framework-for-microservices/nexus/nexus"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = ginkgo.Describe("Combined OpenAPI tests", ginkgo.Ordered, func() {
	ginkgo.It("should setup and load openapi file", func() {
		err := declarative.Load(spec)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(declarative.Paths).To(gomega.HaveKey(URI))
		gomega.Expect(declarative.Paths).To(gomega.HaveKey(ResourceURI))
	})

	ginkgo.It("should create new datamodel", func() {
		gomega.Expect(api.Schemas).To(gomega.BeEmpty())
		api.New("vmware.org")
		gomega.Expect(api.Schemas["vmware.org"].Info.Title).To(gomega.Equal("Nexus API GW APIs"))

		unstructuredObj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"spec": map[string]interface{}{
					"title": "VMWare Datamodel",
				},
			},
		}

		model.ConstructDatamodel(model.Upsert, "vmware2.org", &unstructuredObj)
		api.New("vmware2.org")
		gomega.Expect(api.Schemas["vmware2.org"].Info.Title).To(gomega.Equal("VMWare Datamodel"))
	})

	ginkgo.It("should add custom description to node", func() {
		restURI := nexus.RestURIs{
			Uri:     "/leader/{orgchart.Leader}",
			Methods: nexus.DefaultHTTPMethodsResponses,
		}

		crdJSON, err := yamlv1.YAMLToJSON([]byte(crdExample))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		var crd apiextensionsv1.CustomResourceDefinition
		err = json.Unmarshal(crdJSON, &crd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		model.ConstructMapCRDTypeToNode(model.Upsert, "leaders.orgchart.vmware.org", "orgchart.Leader",
			[]string{"roots.orgchart.vmware.org"}, nil, nil, false, "my custom description", false)
		model.ConstructMapURIToCRDType(model.Upsert, "leaders.orgchart.vmware.org", []nexus.RestURIs{restURI})

		model.ConstructMapCRDTypeToSpec(model.Upsert, "leaders.orgchart.vmware.org", crd.Spec)
		api.New("vmware.org")
		api.AddPath(restURI, "vmware.org")
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].
			Get.Parameters[0].Value.Name).
			To(gomega.Equal("orgchart.Leader"))
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].
			Get.Parameters[0].Value.Description).
			To(gomega.Equal("my custom description"))
	})

	ginkgo.It("should combine openapi specs", func() {
		schema := combined.Specs()

		pathItem := schema.Paths.Find("/leader/{orgchart.Leader}")
		gomega.Expect(pathItem).ToNot(gomega.BeNil())

		pathItem = schema.Paths.Find("/v1alpha1/project/{projectId}/global-namespaces")
		gomega.Expect(pathItem).ToNot(gomega.BeNil())
	})

	ginkgo.It("should combine openapi specs with additional components", func() {
		s := api.Schemas["vmware.org"]
		s.Components.SecuritySchemes = openapi3.SecuritySchemes{
			"BasicAuth": {
				Value: &openapi3.SecurityScheme{
					Type:   "http",
					Scheme: "basic",
				},
			},
		}
		s.Components.Examples = openapi3.Examples{
			"example": {
				Value: &openapi3.Example{},
			},
		}
		s.Components.Links = openapi3.Links{
			"example": {
				Value: &openapi3.Link{},
			},
		}
		s.Components.Callbacks = openapi3.Callbacks{
			"example": {
				Value: &openapi3.Callback{},
			},
		}
		s.Components.Headers = openapi3.Headers{
			"example": {
				Value: &openapi3.Header{},
			},
		}
		s.Components.Parameters = openapi3.ParametersMap{
			"example": &openapi3.ParameterRef{
				Value: &openapi3.Parameter{Name: "example"},
			},
		}
		api.Schemas["vmware.org"] = s

		schema := combined.Specs()

		pathItem := schema.Paths.Find("/leader/{orgchart.Leader}")
		gomega.Expect(pathItem).ToNot(gomega.BeNil())

		pathItem = schema.Paths.Find("/v1alpha1/project/{projectId}/global-namespaces")
		gomega.Expect(pathItem).ToNot(gomega.BeNil())
	})
})
