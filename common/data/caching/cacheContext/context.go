// Copyright 2016 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cacheContext implements a context.Context wrapper which caches the
// results of Value calls, speeding up subsequent calls for the same key.
//
// Context values are stored as immutable layers in a backwards-referenced list.
// Each lookup traverses the list from tail to head looking for a layer with the
// requested value. As a Context retains more values, this traversal will get
// more expensive. This class can be used for large Contexts to avoid repeatedly
// paying the cost of traversal for the same read-only value.
package cacheContext

import (
	"context"
	"sync"
)

// cacheContext implements a caching context.Context.
type cacheContext struct {
	context.Context

	mu    sync.RWMutex
	cache map[interface{}]interface{}
}

// Wrap wraps the supplied Context in a caching Context. All Value lookups will
// be cached at this level, avoiding the expense of future Context traversals
// for that same key.
func Wrap(c context.Context) context.Context {
	if _, ok := c.(*cacheContext); ok {
		return c
	}
	return &cacheContext{Context: c}
}

func (c *cacheContext) Value(key interface{}) interface{} {
	// Optimistic: the value is cached.
	c.mu.RLock()
	if v, ok := c.cache[key]; ok {
		c.mu.RUnlock()
		return v
	}
	c.mu.RUnlock()

	// Pessimistic: not in cache, load from Context and cache it.
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.cache[key]; ok {
		return v // someone did it already
	}
	v := c.Context.Value(key)
	if c.cache == nil {
		c.cache = make(map[interface{}]interface{}, 1)
	}
	c.cache[key] = v
	return v
}
