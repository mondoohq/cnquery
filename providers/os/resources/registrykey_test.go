// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func TestResource_Registrykey(t *testing.T) {
	t.Run("non existent registry key", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\Software\\Policies\\Microsoft\\Windows\\Personalization').exists")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, false, res[0].Data.Value)
	})

	t.Run("registry key path", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System').path")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System", res[0].Data.Value)
	})

	t.Run("existing registry key", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System').exists")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, true, res[0].Data.Value)
	})

	t.Run("registry key properties", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System').properties")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, 24, len(res[0].Data.Value.(map[string]any)))
	})

	t.Run("registry key children", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System').children")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System\\Audit", res[0].Data.Value.([]any)[0])
	})

	t.Run("non-existent registry key - props", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('nope').properties")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, &llx.RawData{Type: types.Map(types.String, types.String)}, res[0].Data)
	})

	t.Run("non-existent registry key - items", func(t *testing.T) {
		res := testWindowsQuery(t, "registrykey('nope').items")
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Nil(t, res[0].Data.Value)
	})

	// A missing registry property must not error when its fields are read or
	// compared — this is what lets policies drop the
	// `switch(x) { case _ != empty: ... default: false }` workaround around
	// registrykey.property(...).data.
	t.Run("missing property does not error on field access or comparison", func(t *testing.T) {
		existPath := "HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Policies\\System"
		queries := []string{
			// missing property on an existing key path
			"registrykey.property(path: '" + existPath + "', name: 'DoesNotExist').exists",
			"registrykey.property(path: '" + existPath + "', name: 'DoesNotExist').data > 0",
			"registrykey.property(path: '" + existPath + "', name: 'DoesNotExist').data > 0 && registrykey.property(path: '" + existPath + "', name: 'DoesNotExist').data <= 30",
			// missing property on a non-existent key path
			"registrykey.property(path: 'HKEY_LOCAL_MACHINE\\Nope\\Nope', name: 'DoesNotExist').data > 0",
		}
		for _, q := range queries {
			t.Run(q, func(t *testing.T) {
				res := testWindowsQuery(t, q)
				assert.NotEmpty(t, res)
				last := res[len(res)-1]
				assert.NoError(t, last.Data.Error)
				assert.Equal(t, false, last.Data.Value)
			})
		}
	})
}

func TestResource_RegistrykeyPerUserHive(t *testing.T) {
	// A per-user read (userSid + ntuserDat) must resolve cleanly even when the
	// hive can't be read (here: the mock has no recording for it). It degrades to
	// "not present" rather than erroring, so callers don't get a false positive.
	t.Run("property in an unreadable user hive does not exist", func(t *testing.T) {
		res := testWindowsQuery(t, `registrykey.property(userSid: 'S-1-5-21-1-2-3-1001', ntuserDat: 'C:\Users\test\NTUSER.DAT', path: 'Software\Policies\Microsoft\Windows\CloudContent', name: 'DisableThirdPartySuggestions').exists`)
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, false, res[0].Data.Value)
	})

	t.Run("path is interpreted relative to the user hive", func(t *testing.T) {
		res := testWindowsQuery(t, `registrykey(userSid: 'S-1-5-21-1-2-3-1001', ntuserDat: 'C:\Users\test\NTUSER.DAT', path: 'Software\Policies').path`)
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, `Software\Policies`, res[0].Data.Value)
	})

	t.Run("userSid is exposed on the key", func(t *testing.T) {
		res := testWindowsQuery(t, `registrykey(userSid: 'S-1-5-21-1-2-3-1001', ntuserDat: 'C:\Users\test\NTUSER.DAT', path: 'Software\Policies').userSid`)
		assert.NotEmpty(t, res)
		assert.Empty(t, res[0].Result().Error)
		assert.Equal(t, "S-1-5-21-1-2-3-1001", res[0].Data.Value)
	})
}
