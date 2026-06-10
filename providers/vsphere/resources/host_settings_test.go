// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHostAdvancedSettingInt(t *testing.T) {
	t.Run("numeric value", func(t *testing.T) {
		n, err := parseHostAdvancedSettingInt("Security.AccountLockFailures", "5")
		require.NoError(t, err)
		assert.Equal(t, int64(5), n)
	})

	t.Run("empty value reads as zero", func(t *testing.T) {
		n, err := parseHostAdvancedSettingInt("UserVars.DcuiTimeOut", "")
		require.NoError(t, err)
		assert.Equal(t, int64(0), n)
	})

	t.Run("non-numeric value errors", func(t *testing.T) {
		_, err := parseHostAdvancedSettingInt("Mem.ShareForceSalting", "notanumber")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Mem.ShareForceSalting")
	})
}

func TestParseHostBoolSetting(t *testing.T) {
	assert.True(t, parseHostBoolSetting("true"))
	assert.True(t, parseHostBoolSetting("1"))
	assert.False(t, parseHostBoolSetting("false"))
	assert.False(t, parseHostBoolSetting("0"))
	assert.False(t, parseHostBoolSetting(""))
}
