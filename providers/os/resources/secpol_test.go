// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
