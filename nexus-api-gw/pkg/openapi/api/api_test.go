// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"encoding/json"
	"net/http"

	yamlv1 "github.com/ghodss/yaml"
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/model"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/openapi/api"
	"github.com/vmware-tanzu/graph-framework-for-microservices/nexus/nexus"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = ginkgo.Describe("OpenAPI tests", ginkgo.Ordered, func() {
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
		gomega.Expect(api.Schemas["vmware.org"].
			Paths[restURI.Uri].Get.Parameters[0].Value.Name).
			To(gomega.Equal("orgchart.Leader"))
		gomega.Expect(api.Schemas["vmware.org"].
			Paths[restURI.Uri].Get.Parameters[0].Value.Description).
			To(gomega.Equal("my custom description"))
	})

	ginkgo.It("should add default description to node if custom is not present", func() {
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
			[]string{}, nil, nil, false, "", false)
		model.ConstructMapURIToCRDType(model.Upsert, "leaders.orgchart.vmware.org", []nexus.RestURIs{restURI})

		model.ConstructMapCRDTypeToSpec(model.Upsert, "leaders.orgchart.vmware.org", crd.Spec)
		api.New("vmware.org")
		api.AddPath(restURI, "vmware.org")
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].Get.Parameters[0].Value.Name).
			To(gomega.Equal("orgchart.Leader"))
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].Get.Parameters[0].Value.Description).
			To(gomega.Equal("Name of the orgchart.Leader node"))
	})

	ginkgo.It("should add list endpoint", func() {
		restURI := nexus.RestURIs{
			Uri:     "/leaders",
			Methods: nexus.HTTPListResponse,
		}

		crdJSON, err := yamlv1.YAMLToJSON([]byte(crdExample))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		var crd apiextensionsv1.CustomResourceDefinition
		err = json.Unmarshal(crdJSON, &crd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		model.ConstructMapCRDTypeToNode(model.Upsert, "leaders.orgchart.vmware.org", "orgchart.Leader",
			[]string{}, nil, nil, false, "", false)
		model.ConstructMapURIToCRDType(model.Upsert, "leaders.orgchart.vmware.org", []nexus.RestURIs{restURI})

		model.ConstructMapCRDTypeToSpec(model.Upsert, "leaders.orgchart.vmware.org", crd.Spec)
		api.New("vmware.org")
		api.AddPath(restURI, "vmware.org")
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].Get).To(gomega.Not(gomega.BeNil()))
	})

	ginkgo.It("should add PATCH endpoint", func() {
		restURI := nexus.RestURIs{
			Uri: "/leaders",
			Methods: nexus.HTTPMethodsResponses{
				http.MethodPatch: nexus.HTTPCodesResponse{
					http.StatusOK: nexus.HTTPResponse{Description: http.StatusText(http.StatusOK)},
				},
			},
		}

		crdJSON, err := yamlv1.YAMLToJSON([]byte(crdExample))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		var crd apiextensionsv1.CustomResourceDefinition
		err = json.Unmarshal(crdJSON, &crd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		model.ConstructMapCRDTypeToNode(model.Upsert, "leaders.orgchart.vmware.org", "orgchart.Leader",
			[]string{}, nil, nil, false, "", false)
		model.ConstructMapURIToCRDType(model.Upsert, "leaders.orgchart.vmware.org", []nexus.RestURIs{restURI})

		model.ConstructMapCRDTypeToSpec(model.Upsert, "leaders.orgchart.vmware.org", crd.Spec)
		api.New("vmware.org")
		api.AddPath(restURI, "vmware.org")
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].Patch).To(gomega.Not(gomega.BeNil()))
	})

	ginkgo.It("should add GET, PUT and PATCH status endpoints", func() {
		statusURI := "/leader/status"
		restURI := nexus.RestURIs{
			Uri: statusURI,
			Methods: nexus.HTTPMethodsResponses{
				http.MethodGet: nexus.DefaultHTTPGETResponses,
				http.MethodPut: nexus.DefaultHTTPPUTResponses,
				http.MethodPatch: nexus.HTTPCodesResponse{
					http.StatusOK: nexus.HTTPResponse{Description: http.StatusText(http.StatusOK)},
				},
			},
		}

		urisMap := map[string]model.RestURIInfo{
			statusURI: {
				TypeOfURI: model.StatusURI,
			},
		}
		model.ConstructMapURIToURIInfo(model.Upsert, urisMap)

		crdJSON, err := yamlv1.YAMLToJSON([]byte(crdExample))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		var crd apiextensionsv1.CustomResourceDefinition
		err = json.Unmarshal(crdJSON, &crd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		model.ConstructMapCRDTypeToNode(model.Upsert, "leaders.orgchart.vmware.org", "orgchart.Leader",
			[]string{}, nil, nil, false, "", false)
		model.ConstructMapURIToCRDType(model.Upsert, "leaders.orgchart.vmware.org", []nexus.RestURIs{restURI})

		model.ConstructMapCRDTypeToSpec(model.Upsert, "leaders.orgchart.vmware.org", crd.Spec)
		api.New("vmware.org")
		api.AddPath(restURI, "vmware.org")
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].Get).To(gomega.Not(gomega.BeNil()))
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].Put).To(gomega.Not(gomega.BeNil()))
		gomega.Expect(api.Schemas["vmware.org"].Paths[restURI.Uri].Patch).To(gomega.Not(gomega.BeNil()))
	})

	ginkgo.It("should test Recreate func", func() {
		restURI := nexus.RestURIs{
			Uri:     "/leaders",
			Methods: nexus.HTTPListResponse,
		}

		crdJSON, err := yamlv1.YAMLToJSON([]byte(crdExample))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		var crd apiextensionsv1.CustomResourceDefinition
		err = json.Unmarshal(crdJSON, &crd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		model.ConstructMapCRDTypeToNode(model.Upsert, "leaders.orgchart.vmware.org", "orgchart.Leader",
			[]string{}, nil, nil, false, "", false)
		model.ConstructMapURIToCRDType(model.Upsert, "leaders.orgchart.vmware.org", []nexus.RestURIs{restURI})

		model.ConstructMapCRDTypeToSpec(model.Upsert, "leaders.orgchart.vmware.org", crd.Spec)
		model.ConstructMapCRDTypeToRestUris(model.Upsert, "leaders.orgchart.vmware.org", nexus.RestAPISpec{
			Uris: []nexus.RestURIs{
				restURI,
			},
		})
		api.Recreate()
		gomega.Expect(api.Schemas).To(gomega.HaveKey("vmware.org"))
		gomega.Expect(api.Schemas["vmware.org"].Components.Responses).To(gomega.HaveKey("Listorgchart.Leader"))
	})

	ginkgo.It("should test update notification for new crd", func() {
		restURI := nexus.RestURIs{
			Uri:     "/leaders",
			Methods: nexus.HTTPListResponse,
		}

		crdJSON, err := yamlv1.YAMLToJSON([]byte(crdExample))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		var crd apiextensionsv1.CustomResourceDefinition
		err = json.Unmarshal(crdJSON, &crd)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		model.ConstructMapCRDTypeToNode(model.Upsert, "leaders.orgchart.vmware.org", "orgchart.Leader",
			[]string{}, nil, nil, false, "", false)
		model.ConstructMapURIToCRDType(model.Upsert, "leaders.orgchart.vmware.org", []nexus.RestURIs{restURI})

		model.ConstructMapCRDTypeToSpec(model.Upsert, "leaders.orgchart.vmware.org", crd.Spec)

		// uri `/oldLeaders` added to the cache on add/update request
		model.ConstructMapCRDTypeToRestUris(model.Upsert, "leaders.orgchart.vmware.org", nexus.RestAPISpec{
			Uris: []nexus.RestURIs{
				{
					Uri:     "/oldLeaders",
					Methods: nexus.HTTPListResponse,
				},
			},
		})

		// On the subsequent request, modified to `/leaders`
		// in the nexus annotation and the cache will be updated with the new URI's.
		model.ConstructMapCRDTypeToRestUris(model.Upsert, "leaders.orgchart.vmware.org", nexus.RestAPISpec{
			Uris: []nexus.RestURIs{
				restURI,
			},
		})

		// should contain only updated URI's not the older URI's
		uris, ok := model.GetRestUris("leaders.orgchart.vmware.org")
		gomega.Expect(ok).Should(gomega.BeTrue())
		gomega.Expect(len(uris)).To(gomega.Equal(1))
		gomega.Expect(uris[0].Uri).To(gomega.Equal("/leaders"))

		api.Recreate()

		unstructuredObj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"spec": map[string]interface{}{
					"title": "VMWare Datamodel",
				},
			},
		}

		go api.DatamodelUpdateNotification()
		model.ConstructDatamodel(model.Delete, "vmware.org", &unstructuredObj)

		gomega.Eventually(func() bool {
			return api.Schemas["vmware.org"].Info.Title == "VMWare Datamodel"
		})
	})
})
