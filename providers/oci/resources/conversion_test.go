// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringValue(t *testing.T) {
	t.Run("nil returns empty string", func(t *testing.T) {
		assert.Equal(t, "", stringValue(nil))
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		s := "hello"
		assert.Equal(t, "hello", stringValue(&s))
	})

	t.Run("empty string pointer returns empty string", func(t *testing.T) {
		s := ""
		assert.Equal(t, "", stringValue(&s))
	})
}

func TestBoolValue(t *testing.T) {
	t.Run("nil returns false", func(t *testing.T) {
		assert.False(t, boolValue(nil))
	})

	t.Run("true pointer returns true", func(t *testing.T) {
		b := true
		assert.True(t, boolValue(&b))
	})

	t.Run("false pointer returns false", func(t *testing.T) {
		b := false
		assert.False(t, boolValue(&b))
	})
}

func TestInt64Value(t *testing.T) {
	t.Run("nil returns 0", func(t *testing.T) {
		assert.Equal(t, int64(0), int64Value(nil))
	})

	t.Run("non-nil returns value", func(t *testing.T) {
		i := int64(42)
		assert.Equal(t, int64(42), int64Value(&i))
	})

	t.Run("zero pointer returns 0", func(t *testing.T) {
		i := int64(0)
		assert.Equal(t, int64(0), int64Value(&i))
	})
}

func TestIntValue(t *testing.T) {
	t.Run("nil returns 0", func(t *testing.T) {
		assert.Equal(t, int64(0), intValue(nil))
	})

	t.Run("non-nil returns value as int64", func(t *testing.T) {
		i := 42
		assert.Equal(t, int64(42), intValue(&i))
	})
}
