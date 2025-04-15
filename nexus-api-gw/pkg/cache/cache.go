// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"sync"

	"github.com/open-edge-platform/orch-utils/nexus-api-gateway/pkg/common"
	amcV1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/apimappingconfig.edge-orchestrator.intel.com/v1"
)

// Cache is a generic type that provides a thread-safe way to store and retrieve values.
type Cache[K comparable, V any] struct {
	store sync.Map
}

// Set adds a key-value pair to the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.store.Store(key, value)
}

// Get retrieves a value from the cache based on the key.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	v, ok := c.store.Load(key)
	if !ok {
		var zeroValue V
		return zeroValue, false
	}
	val, ok := v.(V)
	if !ok {
		var zeroValue V
		return zeroValue, false
	}
	return val, true
}

// Delete a key-value pair.
func (c *Cache[K, V]) Delete(key string) {
	c.store.Delete(key)
}

// Global cache instances.
var (
	APIRemapCache      *Cache[string, common.APIMappingVO]
	GlobalProjectCache *Cache[string, common.Project]
	GlobalOrgCache     *Cache[string, common.Org]
)

var GlobaltenancyCache *Cache[string, common.APIMappingVO]

// Initialize the global cache instances.
func InitializeCaches() {
	APIRemapCache = NewCache[string, common.APIMappingVO]()
	GlobalProjectCache = NewCache[string, common.Project]()
	GlobalOrgCache = NewCache[string, common.Org]()
}

// NewCache creates a new instance of Cache.
func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{}
}

func GetAllAPIRemapCache() []struct {
	ExternalURI string
	ServiceURI  string
	Backend     amcV1.Backend
} {
	var entries []struct {
		ExternalURI string
		ServiceURI  string
		Backend     amcV1.Backend
	}
	APIRemapCache.store.Range(func(_, _ any) bool {
		return true
	})
	APIRemapCache.store.Range(func(key, value interface{}) bool {
		keyInString, ok := key.(string)
		if !ok {
			return false
		}
		val, ok := value.(common.APIMappingVO)
		if !ok {
			return false
		}
		entries = append(entries, struct {
			ExternalURI string
			ServiceURI  string
			Backend     amcV1.Backend
		}{
			ExternalURI: keyInString,
			ServiceURI:  val.ServiceURI,
			Backend:     val.Backend,
		})
		return true
	})
	return entries
}
