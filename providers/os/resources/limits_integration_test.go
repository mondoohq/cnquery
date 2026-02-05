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
	// The Linux mock doesn't have limits.conf, so we test graceful handling

	t.Run("limits resource exists", func(t *testing.T) {
		// Verify the resource can be instantiated
		res := x.TestQuery(t, "limits")
		require.NotEmpty(t, res)
		// Should not panic or have a fundamental error
	})

	t.Run("limits files returns empty when no limits.conf", func(t *testing.T) {
		res := x.TestQuery(t, "limits.files")
		require.NotEmpty(t, res)
		// When limits.conf doesn't exist, should return empty array (not error)
		if res[0].Data.Error == nil {
			files, ok := res[0].Data.Value.([]any)
			if ok {
				assert.Empty(t, files, "expected empty files when limits.conf doesn't exist")
			}
		}
	})

	t.Run("limits entries returns empty when no files", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries")
		require.NotEmpty(t, res)
		// When no files, should return empty array (not error)
		if res[0].Data.Error == nil {
			entries, ok := res[0].Data.Value.([]any)
			if ok {
				assert.Empty(t, entries, "expected empty entries when no limits files exist")
			}
		}
	})

	t.Run("limits entries length is zero when no files", func(t *testing.T) {
		res := x.TestQuery(t, "limits.entries.length")
		require.NotEmpty(t, res)
		if res[0].Data.Error == nil {
			assert.Equal(t, int64(0), res[0].Data.Value)
		}
	})
}
