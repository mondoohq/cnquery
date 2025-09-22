// Copyright (c) 2022 Nathaniel Kofalt
// SPDX-License-Identifier: MIT
//
// This implementation was forked from https://github.com/kofalt/go-memoize to ensure we have the latest dependencies
// updated. The original implementation was released under the MIT license. The files stay under the same license.
// We have made some changes to the original implementation to use the github.com/stretchr/testify/assert for assertions.
package memoize

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	// Prevent "goleak: Errors on successful test run: found unexpected goroutines"
	opts := []goleak.Option{
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
	}
	// verify that we are not leaking goroutines
	goleak.VerifyTestMain(m, opts...)
}

// Test that if a new Memoizer is created but NOT used, it doesn't create
// unnecessary resources.
func TestNewMemoizerUnused(t *testing.T) {
	cache := New(time.Second, time.Minute)
	assert.NotNil(t, cache)
	// we do not call Flush on purpose, it is not needed since the cache
	// was never used, and no resources were allocated
}

// TestBasic adopts the code from readme.md into a simple test case
func TestBasic(t *testing.T) {
	expensiveCalls := 0

	// Function tracks how many times its been called
	expensive := func() (any, error) {
		expensiveCalls++
		return expensiveCalls, nil
	}

	cache := New(90*time.Second, 10*time.Minute)

	// First call SHOULD NOT be cached
	result, cached, err := cache.Memoize("key1", expensive)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 1)
	assert.False(t, cached)

	// Second call on same key SHOULD be cached
	result, cached, err = cache.Memoize("key1", expensive)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 1)
	assert.True(t, cached)

	// First call on a new key SHOULD NOT be cached
	result, cached, err = cache.Memoize("key2", expensive)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 2)
	assert.False(t, cached)

	// call flush
	cache.Flush()

	// After flush, expect a new cache, so we call and cache
	result, cached, err = cache.Memoize("key1", expensive)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 3)
	assert.False(t, cached)
	// Second after flush is cached, so it doesn't get called
	result, cached, err = cache.Memoize("key1", expensive)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 3)
	assert.True(t, cached)

	// callers need to flush to remove leftover resources
	cache.Flush()

}

// TestFailure checks that failed function values are not cached
func TestFailure(t *testing.T) {
	calls := 0

	// This function will fail IFF it has not been called before.
	twoForTheMoney := func() (any, error) {
		calls++

		if calls == 1 {
			return calls, errors.New("Try again")
		} else {
			return calls, nil
		}
	}

	cache := New(90*time.Second, 10*time.Minute)

	// First call should fail, and not be cached
	result, cached, err := cache.Memoize("key1", twoForTheMoney)
	assert.Error(t, err)
	assert.Equal(t, result.(int), 1)
	assert.False(t, cached)

	// Second call should succeed, and not be cached
	result, cached, err = cache.Memoize("key1", twoForTheMoney)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 2)
	assert.False(t, cached)

	// Third call should succeed, and be cached
	result, cached, err = cache.Memoize("key1", twoForTheMoney)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 2)
	assert.True(t, cached)

	// callers need to flush to remove leftover resources
	cache.Flush()
}
