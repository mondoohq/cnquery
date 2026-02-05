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
	// The Linux mock (arch.json) includes limits.conf test data
	// Note: limits.d files require command execution which isn't mocked,
	// so we only test with the main limits.conf file

	t.Run("limits resource exists", func(t *testing.T) {
		res := x.TestQuery(t, "limits")
		require.NotEmpty(t, res)
	})

	t.Run("limits files returns config files", func(t *testing.T) {
		res := x.TestQuery(t, "limits.files.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// Should have at least 1 file (limits.conf)
		assert.GreaterOrEqual(t, res[0].Data.Value.(int64), int64(1))
	})

	t.Run("limits entries are parsed", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// 6 entries from limits.conf
		assert.Equal(t, int64(6), res[0].Data.Value)
	})

	t.Run("limits entry fields - domain", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries[0].domain")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "*", res[0].Data.Value)
	})

	t.Run("limits entry fields - type", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries[0].type")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "soft", res[0].Data.Value)
	})

	t.Run("limits entry fields - item", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries[0].item")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "core", res[0].Data.Value)
	})

	t.Run("limits entry fields - value", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries[0].value")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "0", res[0].Data.Value)
	})

	t.Run("limits entry fields - file", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries[0].file")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, "/etc/security/limits.conf", res[0].Data.Value)
	})

	t.Run("limits entry fields - lineNumber", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries[0].lineNumber")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// First entry is on line 5 (after comments)
		assert.Equal(t, int64(5), res[0].Data.Value)
	})

	t.Run("filter limits by item nofile", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.where(item == \"nofile\").length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// 3 nofile entries: soft, hard, and root's
		assert.Equal(t, int64(3), res[0].Data.Value)
	})

	t.Run("filter limits by domain root", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.where(domain == \"root\").length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("filter limits by type soft", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.where(type == \"soft\").length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// 3 soft entries: core, nofile, nproc
		assert.Equal(t, int64(3), res[0].Data.Value)
	})

	t.Run("filter limits by group domain", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.where(domain == \"@admin\").length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("filter limits by both type", func(t *testing.T) {
		// Test entries with - type (both soft and hard)
		res := x.TestQuery(t, "limits.entries.where(type == \"-\").length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// root - nofile = 1
		assert.Equal(t, int64(1), res[0].Data.Value)
	})

	t.Run("limits entries map domains", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.map(domain)")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		domains, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.Contains(t, domains, "*")
		assert.Contains(t, domains, "root")
		assert.Contains(t, domains, "@admin")
	})

	t.Run("limits entries map items", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.map(item)")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		items, ok := res[0].Data.Value.([]any)
		require.True(t, ok)
		assert.Contains(t, items, "core")
		assert.Contains(t, items, "nofile")
		assert.Contains(t, items, "nproc")
	})

	t.Run("limits with unlimited value", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.where(value == \"unlimited\").length")
		require.NotEmpty(t, res)
		require.NoError(t, res[0].Data.Error)
		// 2 unlimited: hard core and @admin nproc
		assert.Equal(t, int64(2), res[0].Data.Value)
	})
}
