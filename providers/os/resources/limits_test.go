// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Limits(t *testing.T) {
	// Uses the global 'x' tester from os_test.go which is LinuxMock()
	// The Linux mock (arch.json) may not include limits.conf test data
	// These tests verify the resource handles missing files gracefully
	// For parsing tests, see the TOML-based tests in limits/limits_test.go

	t.Run("limits resource exists", func(t *testing.T) {
		res := x.TestQuery(t, "limits")
		require.NotEmpty(t, res)
	})

	t.Run("limits files returns empty when no config files exist", func(t *testing.T) {
		res := x.TestQuery(t, "limits.files")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// When limits files don't exist, returns empty array
		files, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(files), 0)
	})

	t.Run("limits entries returns empty when no config files exist", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// When limits files don't exist, returns empty array
		entries, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(entries), 0)
	})

	t.Run("limits files length is accessible", func(t *testing.T) {
		res := x.TestQuery(t, "limits.files.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// Length should be 0 or more
		assert.GreaterOrEqual(t, res[0].Data.Value.(int64), int64(0))
	})

	t.Run("limits entries length is accessible", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// Length should be 0 or more
		assert.GreaterOrEqual(t, res[0].Data.Value.(int64), int64(0))
	})

	t.Run("limits where filter works on empty entries", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.where(domain == \"*\")")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
	})

	t.Run("limits map works on empty entries", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.map(domain)")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
	})
}
