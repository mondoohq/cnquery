// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypes(t *testing.T) {
	list := []struct {
		T             Type
		ExpectedLabel string
	}{
		{T: Unset, ExpectedLabel: "unset"},
		{T: Any, ExpectedLabel: "any"},
		{T: Nil, ExpectedLabel: "null"},
		{T: Ref, ExpectedLabel: "ref"},
		{T: Bool, ExpectedLabel: "bool"},
		{T: Int, ExpectedLabel: "int"},
		{T: Float, ExpectedLabel: "float"},
		{T: String, ExpectedLabel: "string"},
		{T: Regex, ExpectedLabel: "regex"},
		{T: Time, ExpectedLabel: "time"},
		{T: Dict, ExpectedLabel: "dict"},
		{T: Score, ExpectedLabel: "score"},
		{T: Block, ExpectedLabel: "block"},
		{T: Empty, ExpectedLabel: "empty"},
		{T: Version, ExpectedLabel: "version"},
		{T: IP, ExpectedLabel: "ip"},
		{T: Array(String), ExpectedLabel: "[]string"},
		{T: Map(String, String), ExpectedLabel: "map[string]string"},
		{T: Resource("mockresource"), ExpectedLabel: "mockresource"},
		{T: Function('f', []Type{String, Int}), ExpectedLabel: "func()"},
	}

	for i := range list {
		test := list[i]

		// test for human friendly name
		assert.Equal(t, test.ExpectedLabel, test.T.Label())
	}
}

func TestEmptyType(t *testing.T) {
	empty := Type("")

	t.Run("NotSet returns true for empty type", func(t *testing.T) {
		assert.True(t, empty.NotSet())
	})

	t.Run("Underlying returns NoType for empty type", func(t *testing.T) {
		// This should not panic
		result := empty.Underlying()
		assert.Equal(t, NoType, result)
	})

	t.Run("IsArray returns false for empty type", func(t *testing.T) {
		// This should not panic
		assert.False(t, empty.IsArray())
	})

	t.Run("IsMap returns false for empty type", func(t *testing.T) {
		// This should not panic
		assert.False(t, empty.IsMap())
	})

	t.Run("IsFunction returns false for empty type", func(t *testing.T) {
		// This should not panic
		assert.False(t, empty.IsFunction())
	})

	t.Run("IsResource returns false for empty type", func(t *testing.T) {
		// This should not panic
		assert.False(t, empty.IsResource())
	})

	t.Run("Child returns NoType for empty type", func(t *testing.T) {
		// This should not panic
		result := empty.Child()
		assert.Equal(t, NoType, result)
	})

	t.Run("Label returns EMPTY for empty type", func(t *testing.T) {
		assert.Equal(t, "EMPTY", empty.Label())
	})

	t.Run("Key panics for empty type", func(t *testing.T) {
		require.Panics(t, func() {
			empty.Key()
		})
	})

	t.Run("ResourceName panics for empty type", func(t *testing.T) {
		require.Panics(t, func() {
			empty.ResourceName()
		})
	})
}
