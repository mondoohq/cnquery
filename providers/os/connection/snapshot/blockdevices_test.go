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

var RootDevice = BlockDevice{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}}

func TestGetMatchingBlockEntryByName(t *testing.T) {
	blockEntries := BlockDevices{BlockDevices: []BlockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []BlockDevice{
		{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "sdx", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err := blockEntries.GetBlockEntryByName("/dev/sdx")
	require.Nil(t, err)
	require.Equal(t, fsInfo{FsType: "xfs", Name: "/dev/sdh1"}, *realFsInfo)

	blockEntries = BlockDevices{BlockDevices: []BlockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []BlockDevice{
		{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "xvdx", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdx")
	require.Nil(t, err)
	require.Equal(t, fsInfo{FsType: "xfs", Name: "/dev/xvdh1"}, *realFsInfo)

	blockEntries = BlockDevices{BlockDevices: []BlockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []BlockDevice{
		{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "xvdh", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/xvdh")
	require.Nil(t, err)
	require.Equal(t, fsInfo{FsType: "xfs", Name: "/dev/xvdh1"}, *realFsInfo)

	blockEntries = BlockDevices{BlockDevices: []BlockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []BlockDevice{
		{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdh")
	require.Error(t, err)

	blockEntries = BlockDevices{BlockDevices: []BlockDevice{RootDevice}}
	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdh")
	require.Error(t, err)
}

func TestGetNonRootBlockEntry(t *testing.T) {
	blockEntries := BlockDevices{BlockDevices: []BlockDevice{RootDevice}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []BlockDevice{
		{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)
	realFsInfo, err := blockEntries.GetUnmountedBlockEntry()
	require.Nil(t, err)
	require.Equal(t, fsInfo{FsType: "xfs", Name: "/dev/nvmd1n1"}, *realFsInfo)
}

func TestGetRootBlockEntry(t *testing.T) {
	blockEntries := BlockDevices{BlockDevices: []BlockDevice{RootDevice}}
	realFsInfo, err := blockEntries.GetRootBlockEntry()
	require.Nil(t, err)
	require.Equal(t, fsInfo{FsType: "xfs", Name: "/dev/sda1"}, *realFsInfo)
}

func TestGetRootBlockEntryRhel8(t *testing.T) {
	data, err := os.ReadFile("./testdata/rhel8.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	rootFsInfo, err := blockEntries.GetRootBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{FsType: "xfs", Name: "/dev/sda2"}, *rootFsInfo)

	rootFsInfo, err = blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{FsType: "xfs", Name: "/dev/sdc2"}, *rootFsInfo)
}

func TestGetRootBlockEntryRocky9(t *testing.T) {
	data, err := os.ReadFile("./testdata/rocky9_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	rootFsInfo, err := blockEntries.GetRootBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{FsType: "ext4", Name: "/dev/sda1"}, *rootFsInfo)

	rootFsInfo, err = blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{
		FsType: "xfs",
		Name:   "/dev/rocky-root",
		UUID:   "73976867-73e1-4771-8799-d725d54fa440",
		LVM:    true,
	}, *rootFsInfo)
}

func TestGetBlockEntryByNameRocky9(t *testing.T) {
	data, err := os.ReadFile("./testdata/rocky9_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	fs, err := blockEntries.GetBlockEntryByName("/dev/sdd")
	require.NoError(t, err)
	require.Equal(t, fsInfo{
		FsType: "xfs",
		Name:   "/dev/rocky-root",
		UUID:   "73976867-73e1-4771-8799-d725d54fa440",
		LVM:    true,
	}, *fs)
}

func TestGetRootBlockEntryRhelNoLabels(t *testing.T) {
	data, err := os.ReadFile("./testdata/rhel8_nolabels.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	rootFsInfo, err := blockEntries.GetRootBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{FsType: "xfs", Name: "/dev/sda2"}, *rootFsInfo)

	rootFsInfo, err = blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, fsInfo{FsType: "ext4", Name: "/dev/sdb1"}, *rootFsInfo)
}

func TestAttachedBlockEntry(t *testing.T) {
	data, err := os.ReadFile("./testdata/alma_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.FsType)
	require.True(t, strings.Contains(info.Name, "xvdh"))
}

func TestAttachedBlockEntryAWS(t *testing.T) {
	data, err := os.ReadFile("./testdata/aws_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.FsType)
	require.True(t, strings.Contains(info.Name, "xvdh"))
}

func TestAnotherAttachedBlockEntryAlma(t *testing.T) {
	data, err := os.ReadFile("./testdata/another_alma_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.FsType)
	require.True(t, strings.Contains(info.Name, "nvme1n1"))
}

func TestAttachedBlockEntryOracle(t *testing.T) {
	data, err := os.ReadFile("./testdata/oracle_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "ext4", info.FsType)
	require.True(t, strings.Contains(info.Name, "xvdb"))
}

func TestAttachedBlockEntryRhel(t *testing.T) {
	data, err := os.ReadFile("./testdata/rhel_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.FsType)
	require.True(t, strings.Contains(info.Name, "nvme1n1"))
}

func TestAttachedBlockEntryMultipleMatch(t *testing.T) {
	data, err := os.ReadFile("./testdata/alma9_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.FsType)
	require.True(t, strings.Contains(info.Name, "xvdh4"))
}

func TestAttachedBlockEntryFedora(t *testing.T) {
	data, err := os.ReadFile("./testdata/fedora_attached.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	info, err := blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, "xfs", info.FsType)
	require.True(t, strings.Contains(info.Name, "xvdh4"))
}
