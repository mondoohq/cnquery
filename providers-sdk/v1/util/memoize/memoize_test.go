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
)

// TestBasic adopts the code from readme.md into a simple test case
func TestBasic(t *testing.T) {
	expensiveCalls := 0

	// Function tracks how many times its been called
	expensive := func() (interface{}, error) {
		expensiveCalls++
		return expensiveCalls, nil
	}

	cache := NewMemoizer(90*time.Second, 10*time.Minute)

	// First call SHOULD NOT be cached
	result, err, cached := cache.Memoize("key1", expensive)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 1)
	assert.False(t, cached)

	// Second call on same key SHOULD be cached
	result, err, cached = cache.Memoize("key1", expensive)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 1)
	assert.True(t, cached)

	// First call on a new key SHOULD NOT be cached
	result, err, cached = cache.Memoize("key2", expensive)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 2)
	assert.False(t, cached)
}

// TestFailure checks that failed function values are not cached
func TestFailure(t *testing.T) {
	calls := 0

	// This function will fail IFF it has not been called before.
	twoForTheMoney := func() (interface{}, error) {
		calls++

		if calls == 1 {
			return calls, errors.New("Try again")
		} else {
			return calls, nil
		}
	}

	cache := NewMemoizer(90*time.Second, 10*time.Minute)

	// First call should fail, and not be cached
	result, err, cached := cache.Memoize("key1", twoForTheMoney)
	assert.Error(t, err)
	assert.Equal(t, result.(int), 1)
	assert.False(t, cached)

	// Second call should succeed, and not be cached
	result, err, cached = cache.Memoize("key1", twoForTheMoney)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 2)
	assert.False(t, cached)

	// Third call should succeed, and be cached
	result, err, cached = cache.Memoize("key1", twoForTheMoney)
	assert.NoError(t, err)
	assert.Equal(t, result.(int), 2)
	assert.True(t, cached)
}
