// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package apiremap_test

import (
	"net/http"
	"strings"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/apiremap"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/cache"
	amcV1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/apimappingconfig.edge-orchestrator.intel.com/v1"
	tenancy_nexus_client "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
)

var _ = ginkgo.Describe("TenancyAPIRemapping", ginkgo.Ordered, func() {
	var (
		apiRemapInput apiremap.Input
		amc           tenancy_nexus_client.ApimappingconfigAPIMappingConfig
	)

	ginkgo.Context("when a valid request URI is provided", func() {
		ginkgo.It("should remap the API request without query params", func() {
			amc = tenancy_nexus_client.ApimappingconfigAPIMappingConfig{}
			amc.APIMappingConfig = &amcV1.APIMappingConfig{
				Spec: amcV1.APIMappingConfigSpec{
					Mappings: []amcV1.Mapping{
						{
							ExternalURI: "/v1/projects/{projectName}/resources",
							ServiceURI:  "/v1/resources",
						},
						{
							ExternalURI: "/v1/projects/{projectName}/resources/{resourceId}",
							ServiceURI:  "/v1/resources/{resourceId}",
						},
					},
					Backend: amcV1.Backend{
						Service: "backend-service",
						Port:    8080,
					},
				},
			}
			apiremap.StoreMappingToCache(&amc)
			input := apiremap.Input{
				RequestURI: "/v1/projects/default/resources/123",
				Headers:    http.Header{},
			}
			output, err := apiremap.TenancyAPIRemapping(input)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output.ServiceURI).To(gomega.Equal("/v1/resources/123"))
			gomega.Expect(output.Backendinfo.SvcName).To(gomega.Equal("backend-service"))
			gomega.Expect(output.Backendinfo.PortStr).To(gomega.Equal(uint32(8080)))
		})
		ginkgo.It("should remap the API request with query params", func() {
			input := apiremap.Input{
				RequestURI: "/v1/projects/default/resources/123?query=abc",
				Headers:    http.Header{},
			}
			output, err := apiremap.TenancyAPIRemapping(input)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output.ServiceURI).To(gomega.Equal("/v1/resources/123?query=abc"))
			gomega.Expect(output.Backendinfo.SvcName).To(gomega.Equal("backend-service"))
			gomega.Expect(output.Backendinfo.PortStr).To(gomega.Equal(uint32(8080)))
		})
	})

	ginkgo.Context("when an invalid request URI is provided", func() {
		ginkgo.It("should return an error", func() {
			apiRemapInput.RequestURI = "/api/invalid/resource"

			_, err := apiremap.TenancyAPIRemapping(apiRemapInput)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.Equal("API mapping not found"))
		})
	})
})

var _ = ginkgo.Describe("URLSpecificityMatching", ginkgo.Ordered, func() {
	ginkgo.Context("when similar URL patterns exist", func() {
		ginkgo.BeforeAll(func() {
			// Initialize with test mappings that have ambiguous patterns
			amc := tenancy_nexus_client.ApimappingconfigAPIMappingConfig{}
			amc.APIMappingConfig = &amcV1.APIMappingConfig{
				Spec: amcV1.APIMappingConfigSpec{
					Mappings: []amcV1.Mapping{
						{
							// More generic pattern - should match last
							ExternalURI: "/v1/projects/{projectName}/compute/hosts/{resourceId}",
							ServiceURI:  "/v2/hosts/{resourceId}",
						},
						{
							// More specific pattern - should match first
							ExternalURI: "/v1/projects/{projectName}/compute/hosts/summary",
							ServiceURI:  "/v2/hosts_summary",
						},
						{
							// Another specific pattern
							ExternalURI: "/v1/projects/{projectName}/compute/hosts/register",
							ServiceURI:  "/v2/hosts/register",
						},
						{
							// Another generic pattern
							ExternalURI: "/v1/projects/{projectName}/compute/hosts/{resourceId}/invalidate",
							ServiceURI:  "/v2/hosts/{resourceId}/invalidate",
						},
					},
					Backend: amcV1.Backend{
						Service: "test-service",
						Port:    8080,
					},
				},
			}
			apiremap.StoreMappingToCache(&amc)
		})

		ginkgo.It("should match specific 'summary' endpoint, not generic {resourceId}", func() {
			input := apiremap.Input{
				RequestURI: "/v1/projects/myproject/compute/hosts/summary",
				Headers:    http.Header{},
			}

			output, err := apiremap.TenancyAPIRemapping(input)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output.ServiceURI).To(gomega.Equal("/v2/hosts_summary"))
			gomega.Expect(output.Backendinfo.SvcName).To(gomega.Equal("test-service"))
			gomega.Expect(output.Backendinfo.PortStr).To(gomega.Equal(uint32(8080)))
		})

		ginkgo.It("should match specific 'register' endpoint, not generic {resourceId}", func() {
			input := apiremap.Input{
				RequestURI: "/v1/projects/myproject/compute/hosts/register",
				Headers:    http.Header{},
			}

			output, err := apiremap.TenancyAPIRemapping(input)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output.ServiceURI).To(gomega.Equal("/v2/hosts/register"))
			gomega.Expect(output.Backendinfo.SvcName).To(gomega.Equal("test-service"))
		})

		ginkgo.It("should match generic {resourceId} pattern for actual resource IDs", func() {
			input := apiremap.Input{
				RequestURI: "/v1/projects/myproject/compute/hosts/host-123",
				Headers:    http.Header{},
			}

			output, err := apiremap.TenancyAPIRemapping(input)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output.ServiceURI).To(gomega.Equal("/v2/hosts/host-123"))
			gomega.Expect(output.PathParams["resourceId"]).To(gomega.Equal("host-123"))
			gomega.Expect(output.PathParams["projectName"]).To(gomega.Equal("myproject"))
		})

		ginkgo.It("should handle nested resource patterns correctly", func() {
			input := apiremap.Input{
				RequestURI: "/v1/projects/myproject/compute/hosts/host-456/invalidate",
				Headers:    http.Header{},
			}

			output, err := apiremap.TenancyAPIRemapping(input)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output.ServiceURI).To(gomega.Equal("/v2/hosts/host-456/invalidate"))
			gomega.Expect(output.PathParams["resourceId"]).To(gomega.Equal("host-456"))
		})

		ginkgo.It("should preserve query parameters in all cases", func() {
			input := apiremap.Input{
				RequestURI: "/v1/projects/myproject/compute/hosts/summary?limit=10&offset=0",
				Headers:    http.Header{},
			}

			output, err := apiremap.TenancyAPIRemapping(input)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(output.ServiceURI).To(gomega.Equal("/v2/hosts_summary?limit=10&offset=0"))
		})
	})
})

var _ = ginkgo.Describe("APIMappingsEventsSubscription", func() {
	ginkgo.Context("when subscribing to API mapping events", func() {
		ginkgo.It("should successfully subscribe to events", func() {
			tnc := tenancy_nexus_client.NewFakeClient()
			gomega.Expect(tnc.TenancyMultiTenancy().Config().IsSubscribed()).To(gomega.BeFalse())

			apiremap.SubscribeToAPIMappingsEvents(tnc)

			// The fake client implementation shows that IsSubscribed() returns true after subscription
			// This is the expected behavior for the fake client used in testing
			gomega.Expect(tnc.TenancyMultiTenancy().Config().IsSubscribed()).To(gomega.BeTrue())
		})
	})
})

// FuzzTenancyAPIRemapping provides fuzz testing for the API remapping functionality.
func FuzzTenancyAPIRemapping(f *testing.F) {
	// Seed corpus
	f.Add("/v1/projects/default/resources/123")
	cache.InitializeCaches()
	f.Fuzz(func(t *testing.T, requestURI string) {
		amc := tenancy_nexus_client.ApimappingconfigAPIMappingConfig{}
		amc.APIMappingConfig = &amcV1.APIMappingConfig{
			Spec: amcV1.APIMappingConfigSpec{
				Mappings: []amcV1.Mapping{
					{
						ExternalURI: "/v1/projects/{projectName}/resources/{resourceId}",
						ServiceURI:  "/v1/resources/{resourceId}",
					},
				},
				Backend: amcV1.Backend{
					Service: "backend-service",
					Port:    8080,
				},
			},
		}
		apiremap.StoreMappingToCache(&amc)
		input := apiremap.Input{
			RequestURI: requestURI,
			Headers:    http.Header{},
		}
		output, err := apiremap.TenancyAPIRemapping(input)
		if err != nil {
			if err.Error() == "API mapping not found" {
				// Expected error case
				t.Logf("expected error for input %v: %v", input, err)
				return
			}
			t.Fatalf("unexpected error: %v", err)
		}
		rid := output.PathParams["resourceId"]

		uriArr := strings.Split(requestURI, "?")
		var trailingStr string
		if len(uriArr) > 1 && strings.Trim(uriArr[1], " ") != "" {
			trailingStr = "?" + uriArr[1]
		}
		expURI := "/v1/resources/" + rid + trailingStr
		if output.ServiceURI != expURI {
			t.Logf("requestURI:%s,\n Output:%+v", requestURI, output)
			t.Errorf("expected %s, got %s", expURI, output.ServiceURI)
		}
		if output.Backendinfo.SvcName != "backend-service" {
			t.Errorf("expected backend-service, got %s", output.Backendinfo.SvcName)
		}
		if output.Backendinfo.PortStr != uint32(8080) {
			t.Errorf("expected 8080, got %d", output.Backendinfo.PortStr)
		}
	})
}
