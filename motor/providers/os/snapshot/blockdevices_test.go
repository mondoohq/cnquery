package snapshot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var RootDevice = blockdevice{Name: "sda", Children: []blockdevice{{Uuid: "1234", FsType: "xfs", Label: "ROOT", Name: "sda1", MountPoint: "/"}}}

func TestGetMatchingBlockEntryByName(t *testing.T) {
	blockEntries := blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "sdx", Children: []blockdevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err := blockEntries.GetBlockEntryByName("/dev/sdx")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sdh1"}, *realFsInfo)

	blockEntries = blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "xvdx", Children: []blockdevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdx")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/xvdh1"}, *realFsInfo)

	blockEntries = blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
		{Name: "xvdh", Children: []blockdevice{{Uuid: "12346", FsType: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/xvdh")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/xvdh1"}, *realFsInfo)

	blockEntries = blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdh")
	require.Error(t, err)

	blockEntries = blockdevices{Blockdevices: []blockdevice{RootDevice}}
	realFsInfo, err = blockEntries.GetBlockEntryByName("/dev/sdh")
	require.Error(t, err)
}

func TestGetNonRootBlockEntry(t *testing.T) {
	blockEntries := blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", FsType: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", FsType: "", Label: "EFI"}}},
	}...)
	realFsInfo, err := blockEntries.GetNonRootBlockEntry()
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/nvmd1n1"}, *realFsInfo)
}

func TestGetRootBlockEntry(t *testing.T) {
	blockEntries := blockdevices{Blockdevices: []blockdevice{RootDevice}}
	realFsInfo, err := blockEntries.GetRootBlockEntry()
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sda1"}, *realFsInfo)
}
