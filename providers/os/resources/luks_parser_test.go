// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "luks", name))
	require.NoError(t, err)
	return string(data)
}

func TestParseLuksDump_LUKS1(t *testing.T) {
	out := loadTestdata(t, "luks1_dump.txt")
	d, err := parseLuksDump(out)
	require.NoError(t, err)

	assert.Equal(t, 1, d.Version)
	assert.Equal(t, "fd44f17a-0b3b-44c0-a4a4-6bea30b3c6bf", d.UUID)
	assert.Equal(t, "", d.Label)
	assert.Equal(t, "", d.Subsystem)
	assert.Equal(t, 512, d.MasterKeyBits)
	assert.Equal(t, 4096, d.PayloadOffset)

	assert.Equal(t, "aes", d.Cipher.Name)
	assert.Equal(t, "xts-plain64", d.Cipher.Mode)
	assert.Equal(t, "aes-xts-plain64", d.Cipher.Spec)
	assert.Equal(t, 512, d.Cipher.KeySize)
	assert.Equal(t, "sha256", d.Cipher.Hash)

	require.Len(t, d.Keyslots, 8)

	slot0 := d.Keyslots[0]
	assert.Equal(t, 0, slot0.Index)
	assert.Equal(t, "ENABLED", slot0.State)
	assert.Equal(t, "pbkdf2", slot0.KDF)
	assert.Equal(t, "sha256", slot0.Hash)
	assert.Equal(t, 623013, slot0.Iterations)
	assert.Equal(t, 8, slot0.KeyMaterialOffset)
	assert.Equal(t, 4000, slot0.Stripes)

	slot1 := d.Keyslots[1]
	assert.Equal(t, 1, slot1.Index)
	assert.Equal(t, "DISABLED", slot1.State)
	assert.Equal(t, 0, slot1.Iterations)
	assert.Equal(t, 0, slot1.Stripes)

	slot2 := d.Keyslots[2]
	assert.Equal(t, 2, slot2.Index)
	assert.Equal(t, "ENABLED", slot2.State)
	assert.Equal(t, 312456, slot2.Iterations)
	assert.Equal(t, 1008, slot2.KeyMaterialOffset)

	assert.Empty(t, d.Tokens)
}

func TestParseLuksDump_LUKS2(t *testing.T) {
	out := loadTestdata(t, "luks2_dump.txt")
	d, err := parseLuksDump(out)
	require.NoError(t, err)

	assert.Equal(t, 2, d.Version)
	assert.Equal(t, "7a3f1c4e-8d2b-4a9c-9e1f-5b6c7d8e9f01", d.UUID)
	assert.Equal(t, "root", d.Label)
	assert.Equal(t, "", d.Subsystem)

	require.Len(t, d.Keyslots, 2)

	slot0 := d.Keyslots[0]
	assert.Equal(t, 0, slot0.Index)
	assert.Equal(t, "active", slot0.State)
	assert.Equal(t, "argon2id", slot0.KDF)
	assert.Equal(t, 7, slot0.Time)
	assert.Equal(t, 1048576, slot0.Memory)
	assert.Equal(t, 4, slot0.Parallel)
	assert.Equal(t, 4000, slot0.Stripes)
	assert.Equal(t, "", slot0.Hash, "argon2 keyslots don't carry a KDF hash")
	assert.Equal(t, 32768/512, slot0.KeyMaterialOffset)
	assert.Equal(t, 0, slot0.Iterations, "argon2 slots use time cost, not iterations")

	slot1 := d.Keyslots[1]
	assert.Equal(t, 1, slot1.Index)
	assert.Equal(t, "pbkdf2", slot1.KDF)
	assert.Equal(t, "sha512", slot1.Hash)
	assert.Equal(t, 1879041, slot1.Iterations)
	assert.Equal(t, 0, slot1.Time)

	require.Len(t, d.Tokens, 1)
	token := d.Tokens[0]
	assert.Equal(t, int64(0), token["id"])
	assert.Equal(t, "systemd-tpm2", token["type"])
	keyslots, ok := token["keyslots"].([]int64)
	require.True(t, ok)
	assert.Equal(t, []int64{1}, keyslots)
}

func TestParseLuksDump_MissingVersion(t *testing.T) {
	_, err := parseLuksDump("not a luks dump\n")
	assert.Error(t, err)
}

func TestParseLuksFstype(t *testing.T) {
	assert.True(t, isLuksFstype("crypto_LUKS"))
	assert.False(t, isLuksFstype("ext4"))
	assert.False(t, isLuksFstype(""))
}
