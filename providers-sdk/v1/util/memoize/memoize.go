// Copyright (c) 2022 Nathaniel Kofalt
// SPDX-License-Identifier: MIT
//
// This implementation was forked from https://github.com/kofalt/go-memoize to ensure we have the latest dependencies
// updated. The original implementation was released under the MIT license. The files stay under the same license.

package memoize

import (
	"runtime"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

const NoExpiration time.Duration = -1

// Memoizer allows you to memoize function calls. Implementations should be safe
// for concurrent use by multiple goroutines.
type Memoizer interface {
	Memoize(key string, fn func() (any, error)) (any, bool, error)
	Flush()
}

// cache implements Memoizer using the `go-cache` package
//
// @afiune we could explore using this other package https://github.com/go-pkgz/expirable-cache
type goCache struct {
	// how long items are cached for
	defaultExpiration time.Duration

	// how often the cleanup process should run
	cleanupInterval time.Duration

	// storage is the underlying cache of memoized results
	storage *cache.Cache

	// group makes sure that only one execution is in-flight for a given cache key
	group singleflight.Group

	// lock is used to delay the underlying cache initialization
	lock sync.Mutex
}

// New creates a new Memoizer with the configured expiry and cleanup policies.
// If desired, use memoize.NoExpiration to cache values forever.
func New(defaultExpiration, cleanupInterval time.Duration) Memoizer {
	return &goCache{
		defaultExpiration: defaultExpiration,
		cleanupInterval:   cleanupInterval,
		group:             singleflight.Group{},
	}
}

// Memoize executes and returns the results of the given function, unless there was a cached value of the same key.
// Only one execution is in-flight for a given key at a time.
// The boolean return value indicates whether v was previously stored.
func (m *goCache) Memoize(key string, fn func() (interface{}, error)) (interface{}, bool, error) {
	m.init()

	// Check cache
	value, found := m.storage.Get(key)
	if found {
		return value, true, nil
	}

	// Combine memoized function with a cache store
	value, err, _ := m.group.Do(key, func() (interface{}, error) {
		data, innerErr := fn()

		if innerErr == nil {
			m.storage.Set(key, data, cache.DefaultExpiration)
		}

		return data, innerErr
	})
	return value, false, err
}

// Flush will remove all items from the cache, set `storage` to `nil` and
// run the garbage collector to release long running goroutines.
func (m *goCache) Flush() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.storage != nil {
		m.storage.Flush()
		m.storage = nil
		go runtime.GC()
	}
}

// init delays the cache initialization to when it is first used
func (m *goCache) init() {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.storage == nil {
		m.storage = cache.New(m.defaultExpiration, m.cleanupInterval)
	}
}
