// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"encoding/json"
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
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sde1", Uuid: "12346", Label: "ROOT"}, partition)
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
		require.Equal(t, &PartitionInfo{FsType: "xfs", Name: "/dev/sda2", Uuid: "12346", Label: "ROOT"}, partition)
	})
}

func TestGetMountablePartitions(t *testing.T) {
	t.Run("get all non-mounted partitions", func(t *testing.T) {
		block := BlockDevice{
			Name: "sda",
			Children: []BlockDevice{
				// already mounted
				{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"},
				{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "sda2", MountPoint: ""},
				{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sda3", MountPoint: ""},
				// no fs type
				{Uuid: "12347", FsType: "", Label: "ROOT", Name: "sda4", MountPoint: ""},
			},
		}
		parts, err := block.GetMountablePartitions(true, false)
		require.NoError(t, err)
		expected := []*PartitionInfo{
			{Name: "/dev/sda2", FsType: "xfs", Uuid: "12345", Label: "ROOT"},
			{Name: "/dev/sda3", FsType: "xfs", Uuid: "12346", Label: "ROOT"},
		}
		require.ElementsMatch(t, expected, parts)
	})

	t.Run("get all non-mounted partitions (unpartitioned)", func(t *testing.T) {
		block := BlockDevice{
			Name:   "sda",
			FsType: "xfs",
			Label:  "ROOT",
			Uuid:   "1234",
		}
		parts, err := block.GetMountablePartitions(true, false)
		require.NoError(t, err)
		expected := []*PartitionInfo{
			{Name: "/dev/sda", FsType: "xfs", Uuid: "1234", Label: "ROOT"},
		}
		require.ElementsMatch(t, expected, parts)
	})
}

func TestLongestMatchingSuffix(t *testing.T) {
	requested := "abcde"
	entries := []string{"a", "e", "de"}

	for i, entry := range entries {
		r := longestMatchingSuffix(requested, entry)
		require.Equal(t, i, r)
	}
}
