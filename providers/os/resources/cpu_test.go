// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers/os/connection/mock"
)

func TestGetCpuInfoLinuxX64(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./procfs/testdata/cpu-info-x64.toml"))
	require.NoError(t, err)

	info, err := getCpuInfoLinux(conn)
	require.NoError(t, err)

	assert.Equal(t, "Intel", info.Manufacturer)
	assert.Equal(t, "Intel(R) Core(TM) i7-6700K CPU @ 4.00GHz", info.Model)
	assert.Equal(t, int64(2), info.ProcessorCount)
	assert.Equal(t, int64(2), info.Cores)
}

func TestGetCpuInfoMacosAppleSilicon(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			"sysctl -n machdep.cpu.brand_string hw.physicalcpu": {
				Stdout: "Apple M4 Pro\n14\n",
			},
		},
	}))
	require.NoError(t, err)

	info, err := getCpuInfoMacos(conn)
	require.NoError(t, err)

	assert.Equal(t, "Apple", info.Manufacturer)
	assert.Equal(t, "M4 Pro", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	assert.Equal(t, int64(14), info.Cores)
}

func TestGetCpuInfoMacosIntel(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithData(&mock.TomlData{
		Commands: map[string]*mock.Command{
			"sysctl -n machdep.cpu.brand_string hw.physicalcpu": {
				Stdout: "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz\n8\n",
			},
		},
	}))
	require.NoError(t, err)

	info, err := getCpuInfoMacos(conn)
	require.NoError(t, err)

	assert.Equal(t, "Intel", info.Manufacturer)
	assert.Equal(t, "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	assert.Equal(t, int64(8), info.Cores)
}

func TestGetCpuInfoLinuxArm(t *testing.T) {
	conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./procfs/testdata/cpu-info-aarch64.toml"))
	require.NoError(t, err)

	info, err := getCpuInfoLinux(conn)
	require.NoError(t, err)

	assert.Equal(t, "", info.Manufacturer)
	assert.Equal(t, "", info.Model)
	assert.Equal(t, int64(1), info.ProcessorCount)
	// ARM: no CPUCores info, falls back to processor count for cores
	assert.Equal(t, int64(2), info.Cores)
}
