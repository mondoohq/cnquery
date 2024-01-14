// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hostname_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/detector"
	"go.mondoo.com/cnquery/v10/providers/os/id/hostname"
)

func TestHostnameLinuxEtcHostname(t *testing.T) {
	conn, err := mock.New("./testdata/hostname_arch.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "9be843c4be9f", hostame)
}

func TestHostnameLinux(t *testing.T) {
	conn, err := mock.New("./testdata/hostname_linux.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "abefed34cc9c", hostame)
}

func TestHostnameWindows(t *testing.T) {
	conn, err := mock.New("./testdata/hostname_windows.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "WIN-ABCDEFGVHLD", hostame)
}

func TestHostnameMacos(t *testing.T) {
	conn, err := mock.New("./testdata/hostname_macos.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "moonshot.local", hostame)
}
