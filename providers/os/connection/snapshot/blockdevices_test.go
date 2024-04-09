// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var RootDevice = blockDevice{Name: "sda", Children: []blockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}}

func TestGetMatchingBlockEntryByName(t *testing.T) {
	blockEntries := blockDevices{BlockDevices: []blockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []blockDevice{
		{Name: "nvme0n1", Children: []blockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "sdx", Children: []blockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err := blockEntries.GetBlockEntryByName("/dev/sdx")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sdh1"}, *realFsInfo)

	blockEntries = blockDevices{BlockDevices: []blockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []blockDevice{
		{Name: "nvme0n1", Children: []blockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "xvdx", Children: []blockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdx")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/xvdh1"}, *realFsInfo)

	blockEntries = blockDevices{BlockDevices: []blockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []blockDevice{
		{Name: "nvme0n1", Children: []blockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "xvdh", Children: []blockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/xvdh")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/xvdh1"}, *realFsInfo)

	blockEntries = blockDevices{BlockDevices: []blockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []blockDevice{
		{Name: "nvme0n1", Children: []blockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdh")
	require.Error(t, err)

	blockEntries = blockDevices{BlockDevices: []blockDevice{RootDevice}}
	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdh")
	require.Error(t, err)
}

func TestGetNonRootBlockEntry(t *testing.T) {
	blockEntries := blockDevices{BlockDevices: []blockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []blockDevice{
		{Name: "nvme0n1", Children: []blockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)
	realFsInfo, err := blockEntries.GetUnmountedBlockEntry()
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/nvmd1n1"}, *realFsInfo)
}

func TestGetRootBlockEntry(t *testing.T) {
	blockEntries := blockDevices{BlockDevices: []blockDevice{RootDevice}}
	realFsInfo, err := blockEntries.GetRootBlockEntry()
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sda1"}, *realFsInfo)
}

func TestGetRootBlockEntryRhel8(t *testing.T) {
	data, err := os.ReadFile("./testdata/rhel8.json")
	require.NoError(t, err)

	blockEntries := blockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	rootFsInfo, err := blockEntries.GetRootBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sda2"}, *rootFsInfo)

	rootFsInfo, err = blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sdc2"}, *rootFsInfo)
}

func TestGetRootBlockEntryRhelNoLabels(t *testing.T) {
	data, err := os.ReadFile("./testdata/rhel8_nolabels.json")
	require.NoError(t, err)

	blockEntries := blockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	rootFsInfo, err := blockEntries.GetRootBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sda2"}, *rootFsInfo)

	rootFsInfo, err = blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{fstype: "ext4", name: "/dev/sdb1"}, *rootFsInfo)
}

func TestAttachedBlockEntry(t *testing.T) {
	data, err := os.ReadFile("./testdata/alma_attached.json")
	require.NoError(t, err)

	blockEntries := blockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.fstype)
	require.True(t, strings.Contains(info.name, "xvdh"))
}

func TestAttachedBlockEntryAWS(t *testing.T) {
	data, err := os.ReadFile("./testdata/aws_attached.json")
	require.NoError(t, err)

	blockEntries := blockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.fstype)
	require.True(t, strings.Contains(info.name, "xvdh"))
}

func TestAnotherAttachedBlockEntryAlma(t *testing.T) {
	data, err := os.ReadFile("./testdata/another_alma_attached.json")
	require.NoError(t, err)

	blockEntries := blockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.fstype)
	require.True(t, strings.Contains(info.name, "nvme1n1"))
}

func TestAttachedBlockEntryOracle(t *testing.T) {
	data, err := os.ReadFile("./testdata/oracle_attached.json")
	require.NoError(t, err)

	blockEntries := blockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "ext4", info.fstype)
	require.True(t, strings.Contains(info.name, "xvdb"))
}

func TestAttachedBlockEntryRhel(t *testing.T) {
	data, err := os.ReadFile("./testdata/rhel_attached.json")
	require.NoError(t, err)

	blockEntries := blockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.fstype)
	require.True(t, strings.Contains(info.name, "nvme1n1"))
}
