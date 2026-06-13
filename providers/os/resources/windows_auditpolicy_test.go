// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/testutils"
)

func TestResource_WindowsAuditPolicy(t *testing.T) {
	// one tester for all subtests, so lookups, list iteration, and the
	// missing-subcategory stub share a runtime and exercise resource caching
	win := testutils.InitTester(testutils.WindowsMock())

	t.Run("list all subcategories", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.length")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(59), res[0].Data.Value)
	})

	t.Run("lookup by English name", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.subcategory('Logon') { success && failure }")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		truthy, found := res[0].Data.IsTruthy()
		assert.True(t, found)
		assert.True(t, truthy)
	})

	t.Run("lookup is case-insensitive", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.subcategory('logon').guid")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "0CCE9215-69AE-11D9-BED3-505054503030", res[0].Data.Value)
	})

	t.Run("lookup by GUID", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.subcategory('0CCE922B-69AE-11D9-BED3-505054503030').name")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Process Creation", res[0].Data.Value)
	})

	t.Run("lookup by braced GUID", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.subcategory('{0CCE922B-69AE-11D9-BED3-505054503030}').name")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Process Creation", res[0].Data.Value)
	})

	t.Run("category from the well-known table", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.subcategory('Credential Validation').category")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Account Logon", res[0].Data.Value)
	})

	t.Run("filter by category", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.where(category == 'Detailed Tracking').length")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, int64(6), res[0].Data.Value)
	})

	t.Run("success and failure derive from the inclusion setting", func(t *testing.T) {
		cases := []struct {
			name             string // its inclusion setting in the recording
			success, failure bool
		}{
			{"System Integrity", true, true},            // "Success and Failure"
			{"Security State Change", true, false},      // "Success"
			{"Security System Extension", false, false}, // "No Auditing"
		}
		for _, tc := range cases {
			res := win.TestQuery(t, "windows.auditPolicy.subcategory('"+tc.name+"').success")
			require.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.Equal(t, tc.success, res[0].Data.Value, "success for %s", tc.name)

			res = win.TestQuery(t, "windows.auditPolicy.subcategory('"+tc.name+"').failure")
			require.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.Equal(t, tc.failure, res[0].Data.Value, "failure for %s", tc.name)
		}
	})

	t.Run("raw settings pass through unchanged", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.subcategory('Logon').inclusionSetting")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Success and Failure", res[0].Data.Value)

		// on an English system the localized name is the English name
		res = win.TestQuery(t, "windows.auditPolicy.subcategory('Logon').localizedName")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Logon", res[0].Data.Value)
	})

	t.Run("missing subcategory audits nothing and fails checks cleanly", func(t *testing.T) {
		res := win.TestQuery(t, "windows.auditPolicy.subcategory('No Such Subcategory').success")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, false, res[0].Data.Value)

		res = win.TestQuery(t, "windows.auditPolicy.subcategory('No Such Subcategory').guid")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Nil(t, res[0].Data.Value)

		res = win.TestQuery(t, "windows.auditPolicy.subcategory('No Such Subcategory') { success && failure }")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		truthy, found := res[0].Data.IsTruthy()
		assert.True(t, found)
		assert.False(t, truthy)

		// a missed lookup must not poison the cache for present subcategories
		res = win.TestQuery(t, "windows.auditPolicy.subcategory('Logon').success")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})
}

// TestResource_WindowsAuditPolicyGerman exercises the resource end-to-end
// against a recording of a German-localized `auditpol /r`: subcategory names
// and inclusion settings arrive localized, while name, category, and the
// success/failure booleans must come out language-stable.
func TestResource_WindowsAuditPolicyGerman(t *testing.T) {
	abs, err := filepath.Abs("testdata/auditpol_windows_de.json")
	require.NoError(t, err)
	de := testutils.InitTester(testutils.RecordingMock(abs))

	// "Richtlinienänderungen überwachen" / "Erfolg" on GUID 0CCE922F
	t.Run("English name resolves on a German system", func(t *testing.T) {
		res := de.TestQuery(t, "windows.auditPolicy.subcategory('Audit Policy Change').success")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)

		res = de.TestQuery(t, "windows.auditPolicy.subcategory('Audit Policy Change').failure")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, false, res[0].Data.Value)

		res = de.TestQuery(t, "windows.auditPolicy.subcategory('Audit Policy Change').localizedName")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Richtlinienänderungen überwachen", res[0].Data.Value)
	})

	t.Run("localized name resolves too", func(t *testing.T) {
		res := de.TestQuery(t, "windows.auditPolicy.subcategory('Richtlinienänderungen überwachen').guid")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "0CCE922F-69AE-11D9-BED3-505054503030", res[0].Data.Value)
	})

	t.Run("names and categories are reported in English", func(t *testing.T) {
		res := de.TestQuery(t, "windows.auditPolicy.subcategory('0CCE9217-69AE-11D9-BED3-505054503030').name")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Account Lockout", res[0].Data.Value)

		res = de.TestQuery(t, "windows.auditPolicy.subcategory('Account Lockout').category")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "Logon/Logoff", res[0].Data.Value)
	})

	// the issue's motivating check shape: "Success and Failure" on a
	// localized system, asserted without GUIDs or props
	t.Run("audit check pattern", func(t *testing.T) {
		res := de.TestQuery(t, "windows.auditPolicy.subcategory('Account Lockout') { success && failure }")
		require.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		truthy, found := res[0].Data.IsTruthy()
		assert.True(t, found)
		assert.True(t, truthy)
	})
}
