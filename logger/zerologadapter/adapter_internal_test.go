// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package zerologadapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertToFields(t *testing.T) {
	t.Run("Valid key-value pairs", func(t *testing.T) {
		input := []any{"key1", "value1", "key2", 42, "key3", true}
		expected := map[string]any{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		}

		result := convertToFields(input...)
		assert.Equal(t, expected, result)
	})

	t.Run("Odd number of elements", func(t *testing.T) {
		input := []any{"key1", "value1", "key2"}
		expected := map[string]any{
			"key1": "value1",
		}

		result := convertToFields(input...)
		assert.Equal(t, expected, result)
	})

	t.Run("Non-string keys are ignored", func(t *testing.T) {
		input := []any{123, "value1", "key2", 42, 3.14, "value3", "key3", true}
		expected := map[string]any{
			"key2": 42,
			"key3": true,
		}

		result := convertToFields(input...)
		assert.Equal(t, expected, result)
	})

	t.Run("Empty input", func(t *testing.T) {
		input := []any{}
		expected := map[string]any{}

		result := convertToFields(input...)
		assert.Equal(t, expected, result)
	})

	t.Run("Nil input", func(t *testing.T) {
		var input []any
		expected := map[string]any{}

		result := convertToFields(input...)
		assert.Equal(t, expected, result)
	})
}
