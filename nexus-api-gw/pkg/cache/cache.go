// Copyright (C) 2025 Intel Corporation
// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"sort"
	"strings"
	"sync"

	"github.com/open-edge-platform/orch-utils/nexus-api-gw/pkg/common"
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
	sortedCache        []struct {
		ExternalURI string
		ServiceURI  string
		Backend     amcV1.Backend
	}
	sortedCacheMutex sync.RWMutex
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

// getParameterCount counts the number of path parameters in a URL template
func getParameterCount(template string) int {
	count := 0
	parts := strings.Split(template, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			count++
		}
	}
	return count
}

// GetSortedAPIRemapCache returns pre-sorted cache entries (no per-request sorting)
func GetSortedAPIRemapCache() []struct {
	ExternalURI string
	ServiceURI  string
	Backend     amcV1.Backend
} {
	sortedCacheMutex.RLock()
	defer sortedCacheMutex.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make([]struct {
		ExternalURI string
		ServiceURI  string
		Backend     amcV1.Backend
	}, len(sortedCache))
	copy(result, sortedCache)
	return result
}

// RefreshSortedCache rebuilds and sorts the cache (called only when cache changes)
func RefreshSortedCache() {
	sortedCacheMutex.Lock()
	defer sortedCacheMutex.Unlock()
	
	// Get all entries
	var entries []struct {
		ExternalURI string
		ServiceURI  string
		Backend     amcV1.Backend
	}
	
	APIRemapCache.store.Range(func(key, value interface{}) bool {
		keyInString, ok := key.(string)
		if !ok {
			return true
		}
		val, ok := value.(common.APIMappingVO)
		if !ok {
			return true
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
	
	// Sort by specificity (fewer parameters = higher priority)
	sort.Slice(entries, func(i, j int) bool {
		return getParameterCount(entries[i].ExternalURI) < getParameterCount(entries[j].ExternalURI)
	})
	
	sortedCache = entries
}
