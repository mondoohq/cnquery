// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package filteropts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCsvSliceOpt(t *testing.T) {
	t.Run("parses comma-separated values correctly", func(t *testing.T) {
		opts := map[string]string{
			"key": "value1,value2,value3",
		}
		result := ParseCsvSliceOpt(opts, "key")
		expected := []string{"value1", "value2", "value3"}
		require.Equal(t, expected, result)
	})

	t.Run("returns empty slice for missing key", func(t *testing.T) {
		opts := map[string]string{}
		result := ParseCsvSliceOpt(opts, "key")
		expected := []string{}
		require.Equal(t, expected, result)
	})

	t.Run("returns empty slice for empty value", func(t *testing.T) {
		opts := map[string]string{
			"key": "",
		}
		result := ParseCsvSliceOpt(opts, "key")
		expected := []string{}
		require.Equal(t, expected, result)
	})
}
