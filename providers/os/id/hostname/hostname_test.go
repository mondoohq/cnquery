// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hostname_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/detector"
	"go.mondoo.com/cnquery/v11/providers/os/id/hostname"
)

func TestHostnameLinuxEtcHostname(t *testing.T) {
	conn, err := mock.New(0, "./testdata/hostname_arch.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "9be843c4be9f", hostame)
}

func TestHostnameLinux(t *testing.T) {
	conn, err := mock.New(0, "./testdata/hostname_linux.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "abefed34cc9c", hostame)
}

func TestHostnameLinuxFqdn(t *testing.T) {
	conn, err := mock.New(0, "./testdata/hostname_fqdn.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "myhost.example.com", hostame)
}

func TestHostnameLinuxGetentIPv4(t *testing.T) {
	conn, err := mock.New(0, "./testdata/hostname_getent_hosts_ipv4.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "myhost.example.com", hostame)
}

func TestHostnameLinuxGetentIPv6(t *testing.T) {
	conn, err := mock.New(0, "./testdata/hostname_getent_hosts_ipv6.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "myhost.example.com", hostame)
}

func TestHostnameWindows(t *testing.T) {
	conn, err := mock.New(0, "./testdata/hostname_windows.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "WIN-ABCDEFGVHLD", hostame)
}

func TestHostnameMacos(t *testing.T) {
	conn, err := mock.New(0, "./testdata/hostname_macos.toml", &inventory.Asset{})
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	hostame, ok := hostname.Hostname(conn, platform)
	require.True(t, ok)

	assert.Equal(t, "moonshot.local", hostame)
}
