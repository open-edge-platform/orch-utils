// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/cache"
	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/common"
	amcV1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/apimappingconfig.edge-orchestrator.intel.com/v1"
)

var _ = ginkgo.Describe("Cache", func() {
	var (
		APIRemapCache      *cache.Cache[string, common.APIMappingVO]
		GlobalProjectCache *cache.Cache[string, common.Project]
		GlobalOrgCache     *cache.Cache[string, common.Org]
	)

	ginkgo.BeforeEach(func() {
		// Initialize a new cache before each test
		APIRemapCache = cache.NewCache[string, common.APIMappingVO]()
		GlobalProjectCache = cache.NewCache[string, common.Project]()
		GlobalOrgCache = cache.NewCache[string, common.Org]()
	})

	ginkgo.Describe("Set and Get operations", func() {
		ginkgo.Context("when a new key-value pair is added", func() {
			ginkgo.It("should retrieve the value using Get for ApiRemapCache", func() {
				key := "mappingNameKey"
				value := common.APIMappingVO{ServiceURI: "", Backend: amcV1.Backend{Service: "", Port: 8080}}
				APIRemapCache.Set(key, value)

				retrievedValue, ok := APIRemapCache.Get(key)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(retrievedValue).To(gomega.Equal(value))
			})
			ginkgo.It("should retrieve the value using Get for GlobalProjectCache", func() {
				key := "projNameKey"
				value := common.Project{Name: "projectName", UID: "project-uid", Org: common.Org{Name: "orgName", UID: "org-uid"}}
				GlobalProjectCache.Set(key, value)

				retrievedValue, ok := GlobalProjectCache.Get(key)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(retrievedValue).To(gomega.Equal(value))
			})
			ginkgo.It("should retrieve the value using Get for GlobalOrgCache", func() {
				key := "orgNameKey"
				value := common.Org{Name: "orgName", UID: "the-uu-id"}
				GlobalOrgCache.Set(key, value)

				retrievedValue, ok := GlobalOrgCache.Get(key)
				gomega.Expect(ok).To(gomega.BeTrue())
				gomega.Expect(retrievedValue).To(gomega.Equal(value))
			})
		})

		ginkgo.Context("when a key does not exist", func() {
			ginkgo.It("should return false and the zero value", func() {
				key := "nonExistentKey"
				_, ok := GlobalOrgCache.Get(key)
				gomega.Expect(ok).To(gomega.BeFalse())
			})
		})
	})

	ginkgo.Describe("Delete operation", func() {
		ginkgo.Context("when a key exists", func() {
			ginkgo.It("should remove the key-value pair", func() {
				key := "myKey"
				value := common.APIMappingVO{ServiceURI: "", Backend: amcV1.Backend{Service: "", Port: 8080}}
				APIRemapCache.Set(key, value)

				APIRemapCache.Delete(key)
				_, ok := APIRemapCache.Get(key)
				gomega.Expect(ok).To(gomega.BeFalse())
			})
		})

		ginkgo.Context("when a key does not exist", func() {
			ginkgo.It("should not affect the cache", func() {
				nonExistentKey := "nonExistentKey"
				APIRemapCache.Delete(nonExistentKey)

				// Assuming we have a way to check the size of the cache or iterate over items
				// Here we just ensure that calling Delete on a non-existent key does not cause an error
				gomega.Expect(func() { APIRemapCache.Delete(nonExistentKey) }).NotTo(gomega.Panic())
			})
		})
	})
})
