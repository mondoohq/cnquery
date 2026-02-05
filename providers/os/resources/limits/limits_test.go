// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package limits_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v12/providers/os/resources"
)

func TestLimitsParser_MainConfig(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	f, err := conn.FileSystem().Open("/etc/security/limits.conf")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)

	entries := resources.ParseLimitsLines("/etc/security/limits.conf", string(content))

	require.Len(t, entries, 6)

	// First entry: * soft core 0
	assert.Equal(t, "/etc/security/limits.conf", entries[0].File)
	assert.Equal(t, 5, entries[0].LineNumber)
	assert.Equal(t, "*", entries[0].Domain)
	assert.Equal(t, "soft", entries[0].Type)
	assert.Equal(t, "core", entries[0].Item)
	assert.Equal(t, "0", entries[0].Value)

	// Second entry: * hard core unlimited
	assert.Equal(t, 6, entries[1].LineNumber)
	assert.Equal(t, "*", entries[1].Domain)
	assert.Equal(t, "hard", entries[1].Type)
	assert.Equal(t, "core", entries[1].Item)
	assert.Equal(t, "unlimited", entries[1].Value)

	// Third entry: * soft nofile 65536
	assert.Equal(t, "nofile", entries[2].Item)
	assert.Equal(t, "65536", entries[2].Value)

	// Fifth entry: @admin soft nproc unlimited
	assert.Equal(t, "@admin", entries[4].Domain)
	assert.Equal(t, "soft", entries[4].Type)
	assert.Equal(t, "nproc", entries[4].Item)
	assert.Equal(t, "unlimited", entries[4].Value)

	// Sixth entry: root - nofile 1000000
	assert.Equal(t, "root", entries[5].Domain)
	assert.Equal(t, "-", entries[5].Type)
	assert.Equal(t, "nofile", entries[5].Item)
	assert.Equal(t, "1000000", entries[5].Value)
}

func TestLimitsParser_CustomConfig(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	f, err := conn.FileSystem().Open("/etc/security/limits.d/99-custom.conf")
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)

	entries := resources.ParseLimitsLines("/etc/security/limits.d/99-custom.conf", string(content))

	require.Len(t, entries, 3)

	// First entry: @developers - nofile 100000
	assert.Equal(t, "/etc/security/limits.d/99-custom.conf", entries[0].File)
	assert.Equal(t, 2, entries[0].LineNumber)
	assert.Equal(t, "@developers", entries[0].Domain)
	assert.Equal(t, "-", entries[0].Type)
	assert.Equal(t, "nofile", entries[0].Item)
	assert.Equal(t, "100000", entries[0].Value)

	// Second entry: postgres soft nproc 65536
	assert.Equal(t, "postgres", entries[1].Domain)
	assert.Equal(t, "soft", entries[1].Type)
	assert.Equal(t, "nproc", entries[1].Item)
	assert.Equal(t, "65536", entries[1].Value)

	// Third entry: postgres hard nproc 65536
	assert.Equal(t, "postgres", entries[2].Domain)
	assert.Equal(t, "hard", entries[2].Type)
}

func TestLimitsParser_DirectoryExists(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
	require.NoError(t, err)

	stat, err := conn.FileSystem().Stat("/etc/security/limits.d")
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
}
