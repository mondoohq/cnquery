// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package smbios

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v10/providers/os/detector"
)

func TestManagerCentos(t *testing.T) {
	conn, err := mock.New("./testdata/centos.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	mm, err := ResolveManager(conn, platform)
	require.NoError(t, err)
	biosInfo, err := mm.Info()
	require.NoError(t, err)
	assert.Equal(t, &SmBiosInfo{
		BIOS: BiosInfo{
			Vendor:      "innotek GmbH",
			Version:     "VirtualBox",
			ReleaseDate: "12/01/2006",
		},
		SysInfo: SysInfo{
			Vendor:       "innotek GmbH",
			Model:        "VirtualBox",
			Version:      "1.2",
			SerialNumber: "0",
			UUID:         "64f118d3-0060-4a4c-bf1f-a11d655c4d6f",
			Familiy:      "Virtual Machine",
			SKU:          "",
		},
		BaseBoardInfo: BaseBoardInfo{
			Vendor:       "Oracle Corporation",
			Model:        "VirtualBox",
			Version:      "1.2",
			SerialNumber: "0",
			AssetTag:     "",
		},
		ChassisInfo: ChassisInfo{
			Vendor:       "Oracle Corporation",
			Model:        "",
			Version:      "",
			SerialNumber: "",
			AssetTag:     "",
			Type:         "1",
		},
	}, biosInfo)
}

func TestManagerMacos(t *testing.T) {
	conn, err := mock.New("./testdata/macos.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	mm, err := ResolveManager(conn, platform)
	require.NoError(t, err)
	biosInfo, err := mm.Info()
	require.NoError(t, err)
	assert.Equal(t, &SmBiosInfo{
		BIOS: BiosInfo{
			Vendor:      "Apple Inc.",
			Version:     "170.0.0.0.0",
			ReleaseDate: "06/17/2019",
		},
		SysInfo: SysInfo{
			Vendor:       "Apple Inc.",
			Model:        "iMac17,1",
			Version:      "1.0",
			SerialNumber: "DAAAA111AA11",
			UUID:         "e126775d-2368-4f51-9863-76d5df0c8108",
			Familiy:      "",
			SKU:          "",
		},
		BaseBoardInfo: BaseBoardInfo{
			Vendor:       "Apple Inc.",
			Model:        "Mac-A111A1117AA1AA1A",
			Version:      "",
			SerialNumber: "DAAAA111AA11",
			AssetTag:     "",
		},
		ChassisInfo: ChassisInfo{
			Vendor:       "Apple Inc.",
			Model:        "",
			Version:      "Mac-A111A1117AA1AA1A",
			SerialNumber: "DAAAA111AA11",
			AssetTag:     "",
			Type:         "Laptop",
		},
	}, biosInfo)
}

func TestManagerWindows(t *testing.T) {
	conn, err := mock.New("./testdata/windows.toml", nil)
	require.NoError(t, err)
	platform, ok := detector.DetectOS(conn)
	require.True(t, ok)

	mm, err := ResolveManager(conn, platform)
	require.NoError(t, err)
	biosInfo, err := mm.Info()
	require.NoError(t, err)
	assert.Equal(t, &SmBiosInfo{
		BIOS: BiosInfo{
			Vendor:      "VMware, Inc.",
			Version:     "VMW71.00V.16722896.B64.2008100651",
			ReleaseDate: "20200810000000.000000+000",
		},
		SysInfo: SysInfo{
			Vendor:       "VMware, Inc.",
			Model:        "VMware7,1",
			Version:      "None",
			SerialNumber: "",
			UUID:         "16BD4D56-6B98-23F9-493C-F6B14E7CFC0B",
			Familiy:      "",
			SKU:          "",
		},
		BaseBoardInfo: BaseBoardInfo{
			Vendor:       "Intel Corporation",
			Model:        "440BX Desktop Reference Platform",
			Version:      "None",
			SerialNumber: "None",
			AssetTag:     "",
		},
		ChassisInfo: ChassisInfo{
			Vendor:       "No Enclosure",
			Model:        "",
			Version:      "N/A",
			SerialNumber: "None",
			AssetTag:     "",
			Type:         "",
		},
	}, biosInfo)
}
