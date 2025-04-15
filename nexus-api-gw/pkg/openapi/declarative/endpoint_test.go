// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package declarative_test

import (
	"net/http"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/openapi/declarative"
)

var _ = ginkgo.Describe("OpenAPI tests", func() {
	ginkgo.It("should setup and load openapi file", func() {
		err := declarative.Load(spec)
		gomega.Expect(err).To(gomega.BeNil())

		gomega.Expect(declarative.Paths).To(gomega.HaveKey(URI))
		gomega.Expect(declarative.Paths).To(gomega.HaveKey(ResourceURI))
	})

	ginkgo.It("should get extension value for kind and group", func() {
		kind := declarative.GetExtensionVal(declarative.Paths[URI].Get, declarative.NexusKindName)
		gomega.Expect(kind).To(gomega.Equal("GlobalNamespace"))

		group := declarative.GetExtensionVal(declarative.Paths[URI].Get, declarative.NexusGroupName)
		gomega.Expect(group).To(gomega.Equal("gns.vmware.org"))

		list := declarative.GetExtensionVal(declarative.Paths[ListURI].Get, declarative.NexusListEndpoint)
		gomega.Expect(list).To(gomega.Equal("true"))
	})

	ginkgo.It("should setup context for resource list operation", func() {
		ec := declarative.SetupContext(URI, http.MethodGet, declarative.Paths[URI].Get)

		expectedEc := declarative.EndpointContext{
			Context:      nil,
			SpecURI:      URI,
			Method:       http.MethodGet,
			KindName:     "GlobalNamespace",
			ResourceName: "globalnamespaces",
			GroupName:    "gns.vmware.org",
			CrdName:      "globalnamespaces.gns.vmware.org",
			Params:       [][]string{{"{projectId}", "projectId"}},
			Identifier:   "",
			Single:       false,
			ShortName:    "gns",
			ShortURI:     "/apis/v1/gns",
			URI:          "/apis/gns.vmware.org/v1/globalnamespaces",
		}

		gomega.Expect(ec).To(gomega.Equal(&expectedEc))
	})

	ginkgo.It("should setup context for resource get operation", func() {
		ec := declarative.SetupContext(ResourceURI, http.MethodGet, declarative.Paths[ResourceURI].Get)

		expectedEc := declarative.EndpointContext{
			Context:      nil,
			SpecURI:      ResourceURI,
			Method:       http.MethodGet,
			KindName:     "GlobalNamespace",
			ResourceName: "globalnamespaces",
			GroupName:    "gns.vmware.org",
			CrdName:      "globalnamespaces.gns.vmware.org",
			Params:       [][]string{{"{projectId}", "projectId"}, {"{id}", "id"}},
			Identifier:   "id",
			Single:       true,
			URI:          "/apis/gns.vmware.org/v1/globalnamespaces/:name",
			ShortName:    "gns",
			ShortURI:     "/apis/v1/gns/:name",
		}

		gomega.Expect(ec).To(gomega.Equal(&expectedEc))
	})

	ginkgo.It("should check if resource get operation have an array response", func() {
		isArray := declarative.IsArrayResponse(declarative.Paths[URI].Get)
		gomega.Expect(isArray).To(gomega.BeTrue())
	})

	ginkgo.It("should fail on nil operation when checking if response is array", func() {
		isArray := declarative.IsArrayResponse(nil)
		gomega.Expect(isArray).To(gomega.BeFalse())
	})
})
