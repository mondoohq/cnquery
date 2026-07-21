// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSmartScreenSettings_Configured(t *testing.T) {
	data, err := os.ReadFile("testdata/smartscreen_configured.json")
	require.NoError(t, err)

	s, err := ParseSmartScreenSettings(data)
	require.NoError(t, err)

	assert.True(t, s.ExplorerEnabled())
	assert.Equal(t, "Block", s.ShellSmartScreenLevel)
	assert.True(t, s.EdgeEnabled())
	assert.True(t, s.EdgePuaEnabled())
	assert.True(t, s.EdgePreventOverrideEnabled())
	assert.True(t, s.EdgePreventOverrideForFilesEnabled())
	assert.True(t, s.StoreAppsEnabled())
}

func TestParseSmartScreenSettings_NotConfigured(t *testing.T) {
	data, err := os.ReadFile("testdata/smartscreen_notconfigured.json")
	require.NoError(t, err)

	s, err := ParseSmartScreenSettings(data)
	require.NoError(t, err)

	// A missing (null) value resolves to its disabled default, never an error.
	assert.False(t, s.ExplorerEnabled())
	assert.Equal(t, "", s.ShellSmartScreenLevel)
	assert.False(t, s.EdgeEnabled())
	assert.False(t, s.EdgePuaEnabled())
	assert.False(t, s.EdgePreventOverrideEnabled())
	assert.False(t, s.EdgePreventOverrideForFilesEnabled())
	assert.False(t, s.StoreAppsEnabled())
}

// A DWORD explicitly set to 0 is distinct from an unconfigured value but both
// resolve to "not enabled".
func TestParseSmartScreenSettings_ExplicitZero(t *testing.T) {
	s, err := ParseSmartScreenSettings([]byte(`{"EnableSmartScreen":0,"ShellSmartScreenLevel":"Warn"}`))
	require.NoError(t, err)
	assert.False(t, s.ExplorerEnabled())
	require.NotNil(t, s.EnableSmartScreen)
	assert.Equal(t, int64(0), *s.EnableSmartScreen)
	assert.Equal(t, "Warn", s.ShellSmartScreenLevel)
}

// A surface may be configured while others are not (e.g. Explorer on, Edge off).
func TestParseSmartScreenSettings_Mixed(t *testing.T) {
	s, err := ParseSmartScreenSettings([]byte(`{"EnableSmartScreen":1,"ShellSmartScreenLevel":"Block","EdgeSmartScreenEnabled":0}`))
	require.NoError(t, err)
	assert.True(t, s.ExplorerEnabled())
	assert.Equal(t, "Block", s.ShellSmartScreenLevel)
	assert.False(t, s.EdgeEnabled())
	// untouched surfaces are unconfigured, not enabled
	assert.Nil(t, s.StoreAppsEnableWebContentEvaluation)
	assert.False(t, s.StoreAppsEnabled())
}

func TestParseSmartScreenSettings_Malformed(t *testing.T) {
	_, err := ParseSmartScreenSettings([]byte("not json"))
	require.Error(t, err)
}

// enabledFlag only treats an explicit 1 as enabled; any other value (including
// an unexpected non-1 DWORD) is "not enabled".
func TestSmartScreenEnabledFlag(t *testing.T) {
	one := int64(1)
	two := int64(2)
	zero := int64(0)
	assert.True(t, enabledFlag(&one))
	assert.False(t, enabledFlag(&two))
	assert.False(t, enabledFlag(&zero))
	assert.False(t, enabledFlag(nil))
}
