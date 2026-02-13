// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kernel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
)

func TestSysctlDebian(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/debian.toml"))
	require.NoError(t, err)

	c, err := mock.RunCommand("/sbin/sysctl -a")
	require.NoError(t, err)

	entries, err := ParseSysctl(c.Stdout, "=")
	require.NoError(t, err)

	assert.Equal(t, 32, len(entries))
	assert.Equal(t, "10000", entries["net.ipv4.conf.all.igmpv2_unsolicited_report_interval"])
}

func TestSysctlMacos(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/osx.toml"))
	require.NoError(t, err)

	c, err := mock.RunCommand("sysctl -a")
	require.NoError(t, err)

	entries, err := ParseSysctl(c.Stdout, ":")
	require.NoError(t, err)

	assert.Equal(t, 17, len(entries))
	assert.Equal(t, "1024", entries["net.inet6.ip6.neighborgcthresh"])
}

func TestSysctlFreebsd14(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/freebsd14.toml"))
	require.NoError(t, err)

	c, err := mock.RunCommand("sysctl -a")
	require.NoError(t, err)

	entries, err := ParseSysctl(c.Stdout, ":")
	require.NoError(t, err)

	assert.Equal(t, 20, len(entries))
	assert.Equal(t, "1", entries["security.bsd.unprivileged_mlock"])
}

func TestSysctlFreebsd15(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/freebsd15.toml"))
	require.NoError(t, err)

	c, err := mock.RunCommand("sysctl -a")
	require.NoError(t, err)

	entries, err := ParseSysctl(c.Stdout, ":")
	require.NoError(t, err)

	assert.Equal(t, 19, len(entries))
	assert.Equal(t, "15.0-BETA4", entries["kern.osrelease"])
}

func TestSysctlOpenBSD(t *testing.T) {
	mock, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/openbsd77.toml"))
	require.NoError(t, err)

	c, err := mock.RunCommand("sysctl -a")
	require.NoError(t, err)

	entries, err := ParseSysctl(c.Stdout, "=")
	require.NoError(t, err)

	assert.Equal(t, 29, len(entries))
	assert.Equal(t, "OpenBSD", entries["kern.ostype"])
}
