// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestReadEfiVarBool(t *testing.T) {
	t.Run("enabled variable", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// 4-byte attribute header + 1-byte data (0x01 = enabled)
		err := afero.WriteFile(fs, "/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c",
			[]byte{0x06, 0x00, 0x00, 0x00, 0x01}, 0o444)
		assert.NoError(t, err)

		assert.True(t, readEfiVarBool(fs, "SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"))
	})

	t.Run("disabled variable", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// 4-byte attribute header + 1-byte data (0x00 = disabled)
		err := afero.WriteFile(fs, "/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c",
			[]byte{0x06, 0x00, 0x00, 0x00, 0x00}, 0o444)
		assert.NoError(t, err)

		assert.False(t, readEfiVarBool(fs, "SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"))
	})

	t.Run("missing variable", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		assert.False(t, readEfiVarBool(fs, "SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"))
	})

	t.Run("truncated file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// Only 3 bytes — too short to contain attributes + data
		err := afero.WriteFile(fs, "/sys/firmware/efi/efivars/SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c",
			[]byte{0x06, 0x00, 0x00}, 0o444)
		assert.NoError(t, err)

		assert.False(t, readEfiVarBool(fs, "SecureBoot-8be4df61-93ca-11d2-aa0d-00e098032b8c"))
	})
}
