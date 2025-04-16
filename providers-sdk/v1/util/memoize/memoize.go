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

// Memoizer allows you to memoize function calls. Memoizer is safe for concurrent use by multiple goroutines.
//
// NOTE that callers must `Flush()` to clean GC resources
type Memoizer struct {
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

// NewMemoizer creates a new Memoizer with the configured expiry and cleanup policies.
// If desired, use cache.NoExpiration to cache values forever.
func NewMemoizer(defaultExpiration, cleanupInterval time.Duration) *Memoizer {
	return &Memoizer{
		defaultExpiration: defaultExpiration,
		cleanupInterval:   cleanupInterval,
		group:             singleflight.Group{},
	}
}

// Memoize executes and returns the results of the given function, unless there was a cached value of the same key.
// Only one execution is in-flight for a given key at a time.
// The boolean return value indicates whether v was previously stored.
func (m *Memoizer) Memoize(key string, fn func() (interface{}, error)) (interface{}, error, bool) {
	if m.storage == nil {
		m.init()
	}

	// Check cache
	value, found := m.storage.Get(key)
	if found {
		return value, nil, true
	}

	// Combine memoized function with a cache store
	value, err, _ := m.group.Do(key, func() (interface{}, error) {
		data, innerErr := fn()

		if innerErr == nil {
			m.storage.Set(key, data, cache.DefaultExpiration)
		}

		return data, innerErr
	})
	return value, err, false
}

// init delays the cache initialization to when it is first used
func (m *Memoizer) init() {
	m.lock.Lock()
	m.storage = cache.New(m.defaultExpiration, m.cleanupInterval)
	m.lock.Unlock()
}

// Flush will remove all items from the cache, set `storage` to `nil` and run the garbage collector to
// release infinite goroutines.
//
// @afiune we could explore using this other package instead https://github.com/go-pkgz/expirable-cache
func (m *Memoizer) Flush() {
	if m.storage != nil {
		m.storage.Flush()
		m.storage = nil
		go runtime.GC()
	}
}
