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

func TestGetMountablePartitionByDevice(t *testing.T) {
	t.Run("match by exact name", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "sdx", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
			},
		}

		partition, err := blockEntries.GetMountablePartitionByDevice("/dev/sdx")
		require.Nil(t, err)
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sdh1"}, partition)
	})
	t.Run("match by interchangeable name", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "xvdx", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
			},
		}

		partition, err := blockEntries.GetMountablePartitionByDevice("/dev/sdx")
		require.Nil(t, err)
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/xvdh1"}, partition)
	})

	t.Run("no match by device name", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{
					Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}},
				},
			},
		}
		_, err := blockEntries.GetMountablePartitionByDevice("/dev/sdh")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no block device found with name")
	})

	t.Run("no suitable partition", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{
					Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}},
				},
			},
		}
		_, err := blockEntries.GetMountablePartitionByDevice("/dev/sda")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no suitable partitions found")
	})

	t.Run("return biggest partition", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{
					Name: "sde",
					Children: []BlockDevice{
						{Uuid: "12346", FsType: "xfs", Size: 110, Label: "ROOT", Name: "sde1"},
						{Uuid: "12345", FsType: "xfs", Size: 120, Label: "ROOT", Name: "sde2"},
					},
				},
			},
		}
		partition, err := blockEntries.GetMountablePartitionByDevice("/dev/sde")
		require.Nil(t, err)
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sde2"}, partition)
	})

	t.Run("ignore boot partition (EFI label)", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{
					Name: "sde",
					Children: []BlockDevice{
						{Uuid: "12346", FsType: "xfs", Size: 110, Label: "ROOT", Name: "sde1"},
						{Uuid: "12345", FsType: "xfs", Size: 120, Label: "EFI", Name: "sde2"},
					},
				},
			},
		}
		partition, err := blockEntries.GetMountablePartitionByDevice("/dev/sde")
		require.Nil(t, err)
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sde1"}, partition)
	})

	t.Run("ignore boot partition (boot label)", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{
					Name: "sde",
					Children: []BlockDevice{
						{Uuid: "12346", FsType: "xfs", Size: 110, Label: "ROOT", Name: "sde1"},
						{Uuid: "12345", FsType: "xfs", Size: 120, Label: "boot", Name: "sde2"},
					},
				},
			},
		}
		partition, err := blockEntries.GetMountablePartitionByDevice("/dev/sde")
		require.Nil(t, err)
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sde1"}, partition)
	})
}

func TestGetNonRootBlockEntry(t *testing.T) {
	blockEntries := BlockDevices{BlockDevices: []BlockDevice{
		{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
	}}
	blockEntries.BlockDevices = append(blockEntries.BlockDevices, []BlockDevice{
		{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)
	realPartitionInfo, err := blockEntries.GetUnmountedBlockEntry()
	require.Nil(t, err)
	require.Equal(t, PartitionInfo{FsType: "xfs", Name: "/dev/nvmd1n1"}, *realPartitionInfo)
}

func TestGetRootBlockEntry(t *testing.T) {
	blockEntries := BlockDevices{BlockDevices: []BlockDevice{
		{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
	}}
	realPartitionInfo, err := blockEntries.GetRootBlockEntry()
	require.Nil(t, err)
	require.Equal(t, PartitionInfo{FsType: "xfs", Name: "/dev/sda1"}, *realPartitionInfo)
}

func TestGetRootBlockEntryRhel8(t *testing.T) {
	data, err := os.ReadFile("./testdata/rhel8.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	rootPartitionInfo, err := blockEntries.GetRootBlockEntry()
	require.NoError(t, err)
	require.Equal(t, PartitionInfo{FsType: "xfs", Name: "/dev/sda2"}, *rootPartitionInfo)

	rootPartitionInfo, err = blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, PartitionInfo{FsType: "xfs", Name: "/dev/sdc2"}, *rootPartitionInfo)
}

func TestGetRootBlockEntryRhelNoLabels(t *testing.T) {
	data, err := os.ReadFile("./testdata/rhel8_nolabels.json")
	require.NoError(t, err)

	blockEntries := BlockDevices{}
	err = json.Unmarshal(data, &blockEntries)
	require.NoError(t, err)

	rootPartitionInfo, err := blockEntries.GetRootBlockEntry()
	require.NoError(t, err)
	require.Equal(t, PartitionInfo{FsType: "xfs", Name: "/dev/sda2"}, *rootPartitionInfo)

	rootPartitionInfo, err = blockEntries.GetUnnamedBlockEntry()
	require.NoError(t, err)
	require.Equal(t, PartitionInfo{FsType: "ext4", Name: "/dev/sdb1"}, *rootPartitionInfo)
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
