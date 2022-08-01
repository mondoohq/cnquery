package awsec2ebs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var RootDevice = blockdevice{Name: "sda", Children: []blockdevice{{Uuid: "1234", Fstype: "xfs", Label: "ROOT", Name: "sda1", Mountpoint: "/"}}}

func TestGetMatchingBlockEntryByName(t *testing.T) {
	blockEntries := blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", Fstype: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", Fstype: "", Label: "EFI"}}},
		{Name: "sdx", Children: []blockdevice{{Uuid: "12346", Fstype: "xfs", Label: "ROOT", Name: "sdh1"}, {Uuid: "12345", Fstype: "", Label: "EFI"}}},
	}...)

	realFsInfo, err := getMatchingBlockEntryByName(blockEntries, "/dev/sdx")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sdh1"}, *realFsInfo)

	blockEntries = blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", Fstype: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", Fstype: "", Label: "EFI"}}},
		{Name: "xvdx", Children: []blockdevice{{Uuid: "12346", Fstype: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", Fstype: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = getMatchingBlockEntryByName(blockEntries, "/dev/sdx")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/xvdh1"}, *realFsInfo)

	blockEntries = blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", Fstype: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", Fstype: "", Label: "EFI"}}},
		{Name: "xvdh", Children: []blockdevice{{Uuid: "12346", Fstype: "xfs", Label: "ROOT", Name: "xvdh1"}, {Uuid: "12345", Fstype: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = getMatchingBlockEntryByName(blockEntries, "/dev/xvdh")
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/xvdh1"}, *realFsInfo)

	blockEntries = blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", Fstype: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", Fstype: "", Label: "EFI"}}},
	}...)

	realFsInfo, err = getMatchingBlockEntryByName(blockEntries, "/dev/sdh")
	require.Error(t, err)

	blockEntries = blockdevices{Blockdevices: []blockdevice{RootDevice}}
	realFsInfo, err = getMatchingBlockEntryByName(blockEntries, "/dev/sdh")
	require.Error(t, err)
}

func TestGetNonRootBlockEntry(t *testing.T) {
	blockEntries := blockdevices{Blockdevices: []blockdevice{RootDevice}}
	blockEntries.Blockdevices = append(blockEntries.Blockdevices, []blockdevice{
		{Name: "nvme0n1", Children: []blockdevice{{Uuid: "12345", Fstype: "xfs", Label: "ROOT", Name: "nvmd1n1"}, {Uuid: "12345", Fstype: "", Label: "EFI"}}},
	}...)
	realFsInfo, err := getNonRootBlockEntry(blockEntries)
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/nvmd1n1"}, *realFsInfo)
}

func TestGetRootBlockEntry(t *testing.T) {
	blockEntries := blockdevices{Blockdevices: []blockdevice{RootDevice}}
	realFsInfo, err := getRootBlockEntry(blockEntries)
	require.Nil(t, err)
	require.Equal(t, fsInfo{fstype: "xfs", name: "/dev/sda1"}, *realFsInfo)
}
