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

func TestBlockDevicesUnmarshal(t *testing.T) {
	common := `{
   "blockdevices": [
      {"name": "nvme1n1", "size": 8589934592, "fstype": null, "mountpoint": null, "label": null, "uuid": null,
         "children": [
            {"name": "nvme1n1p1", "size": 7515127296, "fstype": "ext4", "mountpoint": null, "label": "cloudimg-rootfs", "uuid": "d84ccd9b-0384-4314-88be-5bd38eb59f30"},
            {"name": "nvme1n1p14", "size": 4194304, "fstype": null, "mountpoint": null, "label": null, "uuid": null},
            {"name": "nvme1n1p15", "size": 111149056, "fstype": "vfat", "mountpoint": null, "label": "UEFI", "uuid": "9601-9938"},
            {"name": "nvme1n1p16", "size": 957350400, "fstype": "ext4", "mountpoint": null, "label": "BOOT", "uuid": "c2032e48-1c8e-4f92-87c6-9db270bf4274"}
         ]
      },
      {"name": "nvme0n1", "size": "8589934592", "fstype": null, "mountpoint": null, "label": null, "uuid": null,
         "children": [
            {"name": "nvme0n1p1", "size": 8578383360, "fstype": "xfs", "mountpoint": "/", "label": "/", "uuid": "804f6603-f3df-4054-8161-50bd9cbd9cf9"},
            {"name": "nvme0n1p128", "size": 10485760, "fstype": "vfat", "mountpoint": "/boot/efi", "label": null, "uuid": "BCB5-3E0E"}
         ]
      }
   ]
}`

	blockEntries := &BlockDevices{}
	err := json.Unmarshal([]byte(common), blockEntries)
	require.NoError(t, err)

	stringer := `{
   "blockdevices": [
      {"name": "nvme1n1", "size": "8589934592", "fstype": null, "mountpoint": null, "label": null, "uuid": null,
         "children": [
            {"name": "nvme1n1p1", "size": "7515127296", "fstype": "ext4", "mountpoint": null, "label": "cloudimg-rootfs", "uuid": "d84ccd9b-0384-4314-88be-5bd38eb59f30"},
            {"name": "nvme1n1p14", "size": "4194304", "fstype": null, "mountpoint": null, "label": null, "uuid": null},
            {"name": "nvme1n1p15", "size": "111149056", "fstype": "vfat", "mountpoint": null, "label": "UEFI", "uuid": "9601-9938"},
            {"name": "nvme1n1p16", "size": "957350400", "fstype": "ext4", "mountpoint": null, "label": "BOOT", "uuid": "c2032e48-1c8e-4f92-87c6-9db270bf4274"}
         ]
      },
      {"name": "nvme0n1", "size": "8589934592", "fstype": null, "mountpoint": null, "label": null, "uuid": null,
         "children": [
            {"name": "nvme0n1p1", "size": "8578383360", "fstype": "xfs", "mountpoint": "/", "label": "/", "uuid": "804f6603-f3df-4054-8161-50bd9cbd9cf9"},
            {"name": "nvme0n1p128", "size": "10485760", "fstype": "vfat", "mountpoint": "/boot/efi", "label": null, "uuid": "BCB5-3E0E"}
         ]
      }
   ]
}`

	blockEntries = &BlockDevices{}
	err = json.Unmarshal([]byte(stringer), blockEntries)
	require.NoError(t, err)
}

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
						{Uuid: "12345", FsType: "xfs", Size: 120, Label: "BOOT", Name: "sde2"},
					},
				},
			},
		}
		partition, err := blockEntries.GetMountablePartitionByDevice("/dev/sde")
		require.Nil(t, err)
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sde1"}, partition)
	})
}

func TestFindDevice(t *testing.T) {
	t.Run("match by exact name", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "sdx", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
			},
		}

		expected := blockEntries.BlockDevices[2]
		res, err := blockEntries.FindDevice("/dev/sdx")
		require.Nil(t, err)
		require.Equal(t, expected, res)
	})

	t.Run("match by alias name", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "sdx", Aliases: []string{"xvdx"}, Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
			},
		}

		expected := blockEntries.BlockDevices[2]
		res, err := blockEntries.FindDevice("/dev/xvdx")
		require.Nil(t, err)
		require.Equal(t, expected, res)
	})

	t.Run("match by interchangeable name", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "xvdc", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
			},
		}

		expected := blockEntries.BlockDevices[2]
		res, err := blockEntries.FindDevice("/dev/sdc")
		require.Nil(t, err)
		require.Equal(t, expected, res)
	})

	t.Run("no match", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "xvdc", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
			},
		}

		_, err := blockEntries.FindDevice("/dev/sdd")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no block device found with name")
	})

	t.Run("multiple matches by trailing letter", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "stc", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "xvdc", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
			},
		}

		expected := blockEntries.BlockDevices[3]
		res, err := blockEntries.FindDevice("/dev/sdc")
		require.Nil(t, err)
		require.Equal(t, expected, res)
	})

	t.Run("perfect match and trailing letter matches", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "sta", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "xvda", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
			},
		}

		expected := blockEntries.BlockDevices[0]
		res, err := blockEntries.FindDevice("/dev/sda")
		require.Nil(t, err)
		require.Equal(t, expected, res)
	})

	t.Run("perfect match and trailing letter matches (scrambled)", func(t *testing.T) {
		blockEntries := BlockDevices{
			BlockDevices: []BlockDevice{
				{Name: "xvda", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "sta", Children: []BlockDevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "nvme0n1", Children: []BlockDevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
				{Name: "sda", Children: []BlockDevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}},
			},
		}

		expected := blockEntries.BlockDevices[3]
		res, err := blockEntries.FindDevice("/dev/sda")
		require.Nil(t, err)
		require.Equal(t, expected, res)
	})
}

func TestGetMountablePartition(t *testing.T) {
	t.Run("no suitable partition (mounted)", func(t *testing.T) {
		block := BlockDevice{
			Name: "sda",
			Children: []BlockDevice{
				{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"},
			},
		}
		_, err := block.GetMountablePartition()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no suitable partitions found")
	})

	t.Run("no suitable partition (no fs type)", func(t *testing.T) {
		block := BlockDevice{
			Name: "sda",
			Children: []BlockDevice{
				{Uuid: "1234", FsType: "", Label: "ROOT", Name: "sda1", MountPoint: ""},
			},
		}
		_, err := block.GetMountablePartition()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no suitable partitions found")
	})

	t.Run("no suitable partition (EFI label)", func(t *testing.T) {
		block := BlockDevice{
			Name: "sda",
			Children: []BlockDevice{
				{Uuid: "1234", FsType: "xfs", Label: "EFI", Name: "sda1", MountPoint: ""},
			},
		}
		_, err := block.GetMountablePartition()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no suitable partitions found")
	})

	t.Run("no suitable partition (vfat fs type)", func(t *testing.T) {
		block := BlockDevice{
			Name: "sda",
			Children: []BlockDevice{
				{Uuid: "1234", FsType: "vfat", Label: "", Name: "sda1", MountPoint: ""},
			},
		}
		_, err := block.GetMountablePartition()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no suitable partitions found")
	})

	t.Run("no suitable partition (boot label)", func(t *testing.T) {
		block := BlockDevice{
			Name: "sda",
			Children: []BlockDevice{
				{Uuid: "1234", FsType: "xfs", Label: "boot", Name: "sda1", MountPoint: ""},
			},
		}
		_, err := block.GetMountablePartition()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no suitable partitions found")
	})

	t.Run("no suitable partition (empty)", func(t *testing.T) {
		block := BlockDevice{
			Name:     "sda",
			Children: []BlockDevice{},
		}
		_, err := block.GetMountablePartition()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no suitable partitions found")
	})

	t.Run("suitable single partition", func(t *testing.T) {
		block := BlockDevice{
			Name: "sde",
			Children: []BlockDevice{
				{Uuid: "12346", FsType: "xfs", Size: 110, Label: "ROOT", Name: "sde1"},
			},
		}
		partition, err := block.GetMountablePartition()
		require.Nil(t, err)
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sde1"}, partition)
	})

	t.Run("largest suitable partition", func(t *testing.T) {
		block := BlockDevice{
			Name: "sda",
			Children: []BlockDevice{
				{Uuid: "12346", FsType: "xfs", Size: 110, Label: "ROOT", Name: "sda1"},
				{Uuid: "12346", FsType: "xfs", Size: 120, Label: "ROOT", Name: "sda2"},
			},
		}
		partition, err := block.GetMountablePartition()
		require.Nil(t, err)
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sda2"}, partition)
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
	require.True(t, strings.Contains(info.Name, "xvdh3"))
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

func TestLongestMatchingSuffix(t *testing.T) {
	requested := "abcde"
	entries := []string{"a", "e", "de"}

	for i, entry := range entries {
		r := LongestMatchingSuffix(requested, entry)
		require.Equal(t, i, r)
	}
}
