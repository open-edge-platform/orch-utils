// Copyright 2021 The Kubernetes Authors.
// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"sync"
)

// HandlerCache represents a lazy cache for generating a byte array
// It is used to lazily marshal OpenAPI v2/v3 and lazily generate the ETag
type HandlerCache struct {
	BuildCache func() ([]byte, error)
	once       sync.Once
	bytes      []byte
	err        error
}

// Get either returns the cached value or calls BuildCache() once before caching and returning
// its results. If BuildCache returns an error, the last valid value for the cache (from prior
// calls to New()) is used instead if possible.
func (c *HandlerCache) Get() ([]byte, error) {
	c.once.Do(func() {
		bytes, err := c.BuildCache()
		// if there is an error updating the cache, there can be situations where
		// c.bytes contains a valid value (carried over from the previous update)
		// but c.err is also not nil; the cache user is expected to check for this
		c.err = err
		if c.err == nil {
			// don't override previous spec if we had an error
			c.bytes = bytes
		}
	})
	return c.bytes, c.err
}

// New creates a new HandlerCache for situations where a cache refresh is needed.
// This function is not thread-safe and should not be called at the same time as Get().
func (c *HandlerCache) New(cacheBuilder func() ([]byte, error)) HandlerCache {
	return HandlerCache{
		bytes:      c.bytes,
		BuildCache: cacheBuilder,
	}
}
