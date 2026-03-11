// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slicesx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSubsetOf(t *testing.T) {
	t.Run("equal sets", func(t *testing.T) {
		assert.True(t, IsSubsetOf([]string{"a", "b"}, []string{"a", "b"}))
	})

	t.Run("proper subset", func(t *testing.T) {
		assert.True(t, IsSubsetOf([]string{"a"}, []string{"a", "b", "c"}))
	})

	t.Run("not a subset", func(t *testing.T) {
		assert.False(t, IsSubsetOf([]string{"a", "d"}, []string{"a", "b", "c"}))
	})

	t.Run("sub longer than super", func(t *testing.T) {
		assert.False(t, IsSubsetOf([]string{"a", "b", "c"}, []string{"a"}))
	})

	t.Run("empty sub is always a subset", func(t *testing.T) {
		assert.True(t, IsSubsetOf([]string{}, []string{"a", "b"}))
	})

	t.Run("both empty", func(t *testing.T) {
		assert.True(t, IsSubsetOf([]string{}, []string{}))
	})

	t.Run("works with ints", func(t *testing.T) {
		assert.True(t, IsSubsetOf([]int{1, 2}, []int{1, 2, 3}))
		assert.False(t, IsSubsetOf([]int{1, 4}, []int{1, 2, 3}))
	})
}
