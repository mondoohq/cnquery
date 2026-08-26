// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/testutils"
)

func TestResource_Secpol(t *testing.T) {
	t.Run("list systemaccess", func(t *testing.T) {
		res := testWindowsQuery(t, "secpol.systemaccess")
		assert.NotEmpty(t, res)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "secpol.systemaccess['PasswordHistorySize']")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "0", res[0].Data.Value)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "secpol.privilegerights['SeNetworkLogonRight']")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, []any{
			"S-1-1-0",
			"S-1-5-32-544",
			"S-1-5-32-545",
			"S-1-5-32-551",
		}, res[0].Data.Value)
	})

	t.Run("test a specific secpol systemaccess entry", func(t *testing.T) {
		res := testWindowsQuery(t, "secpol.privilegerights['SeNetworkLogonRight'] == ['S-1-1-0', 'S-1-5-32-544', 'S-1-5-32-545', 'S-1-5-32-551']")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[1].Result().Error)
		assert.Equal(t, true, res[1].Data.Value)
	})

	// A privilege right that is not assigned to anyone simply does not appear
	// in the policy, so secpol.privilegerights['SeMissing'] resolves to a typed
	// null array. Calling assertion methods on it must fail cleanly (return a
	// graceful false) rather than erroring the whole check — this is what lets
	// policies drop the `switch(x) { case _ != empty: ... default: false }`
	// workaround that previously guarded against the error.
	t.Run("missing privilege right does not error on assertion methods", func(t *testing.T) {
		queries := []string{
			"secpol.privilegerights['SeMissingRight'].contains('S-1-5-32-544')",
			"secpol.privilegerights['SeMissingRight'].any(_ == 'S-1-5-32-544')",
			"secpol.privilegerights['SeMissingRight'].all(_ == 'S-1-5-32-544')",
			"secpol.privilegerights['SeMissingRight'].none(_ == 'S-1-5-32-544')",
			"secpol.privilegerights['SeMissingRight'].one(_ == 'S-1-5-32-544')",
		}
		for _, q := range queries {
			t.Run(q, func(t *testing.T) {
				res := testWindowsQuery(t, q)
				assert.NotEmpty(t, res)
				last := res[len(res)-1]
				// no error, and the check fails gracefully (false)
				assert.NoError(t, last.Data.Error)
				assert.Equal(t, false, last.Data.Value)
			})
		}
	})
}

// TestResource_SecpolGerman covers a German host, where secedit names the
// principals of a user right instead of reporting their SIDs.
func TestResource_SecpolGerman(t *testing.T) {
	abs, err := filepath.Abs("testdata/secpol_windows_de.json")
	require.NoError(t, err)
	de := testutils.InitTester(testutils.RecordingMock(abs))

	res := de.TestQuery(t, "secpol.privilegerights['SeDenyNetworkLogonRight']")
	require.NotEmpty(t, res)
	assert.Empty(t, res[0].Result().Error)
	assert.Equal(t, []any{"S-1-5-32-544", "S-1-5-32-546"}, res[0].Data.Value)
}

// TestResource_SecpolSidLookupKilled pins the blast radius of a failing SID
// lookup: every field that does not need it keeps answering, which is most of
// what a Windows benchmark reads out of secpol.
func TestResource_SecpolSidLookupKilled(t *testing.T) {
	abs, err := filepath.Abs("testdata/secpol_windows_de_lookup_killed.json")
	require.NoError(t, err)
	de := testutils.InitTester(testutils.RecordingMock(abs))

	for _, q := range []string{
		"secpol.systemaccess['PasswordHistorySize']",
		"secpol.systemaccess['LockoutBadCount']",
		"secpol.eventaudit['AuditLogonEvents']",
		`secpol.registryvalues['MACHINE\System\CurrentControlSet\Control\Lsa\FullPrivilegeAuditing']`,
	} {
		t.Run(q, func(t *testing.T) {
			res := de.TestQuery(t, q)
			require.NotEmpty(t, res)
			assert.Empty(t, res[0].Result().Error)
			assert.NotEmpty(t, res[0].Data.Value)
		})
	}

	// the field that does need it reports the failure rather than a list the
	// unresolved names were quietly dropped from
	res := de.TestQuery(t, "secpol.privilegerights['SeDenyNetworkLogonRight']")
	require.NotEmpty(t, res)
	assert.Contains(t, res[0].Result().Error, "could not resolve privilege right account names")
}
