// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSecureBoot(t *testing.T) {
	t.Run("UEFI host with Secure Boot enabled", func(t *testing.T) {
		input := `{ "Efi": true, "Enabled": true, "SetupMode": false }`
		status, err := ParseSecureBoot(strings.NewReader(input))
		require.NoError(t, err)
		assert.True(t, status.Efi)
		assert.True(t, status.Enabled)
		assert.False(t, status.SetupMode)
	})

	t.Run("UEFI host with Secure Boot disabled", func(t *testing.T) {
		input := `{ "Efi": true, "Enabled": false, "SetupMode": true }`
		status, err := ParseSecureBoot(strings.NewReader(input))
		require.NoError(t, err)
		assert.True(t, status.Efi)
		assert.False(t, status.Enabled)
		assert.True(t, status.SetupMode)
	})

	t.Run("legacy BIOS host", func(t *testing.T) {
		input := `{ "Efi": false, "Enabled": false, "SetupMode": false }`
		status, err := ParseSecureBoot(strings.NewReader(input))
		require.NoError(t, err)
		assert.False(t, status.Efi)
		assert.False(t, status.Enabled)
		assert.False(t, status.SetupMode)
	})

	t.Run("empty output is treated as a non-UEFI host", func(t *testing.T) {
		status, err := ParseSecureBoot(strings.NewReader("  \n"))
		require.NoError(t, err)
		assert.False(t, status.Efi)
		assert.False(t, status.Enabled)
		assert.False(t, status.SetupMode)
	})
}
