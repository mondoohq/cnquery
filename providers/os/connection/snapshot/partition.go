// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"path"
	"strings"
)

type PartitionInfo struct {
	// Device name (e.g. /dev/sda1)
	Name string
	// Filesystem type (e.g. ext4)
	FsType string

	// Resolved device name aliases (e.g. /dev/sda1 -> /dev/nvme0n1p1)
	Aliases []string
	// (optional) Label is the partition label
	Label string
	// (optional) UUID is the partition UUID
	Uuid string
	// (optional) MountPoint is the partition mount point
	MountPoint string

	// (optional) MountOptions are the mount options
	MountOptions []string
	// (optional) bind adjusts the root for FS connection
	bind string
}

type MountPartitionDto struct {
	*PartitionInfo

	ScanDir *string
}

func (entry BlockDevice) isNoBootVolume() bool {
	typ := strings.ToLower(entry.FsType)
	label := strings.ToLower(entry.Label)
	return entry.Uuid != "" && typ != "" && typ != "vfat" && label != "efi" && label != "boot"
}

func (entry BlockDevice) isNoBootVolumeAndUnmounted() bool {
	return entry.isNoBootVolume() && !entry.isMounted()
}

func (entry BlockDevice) isMounted() bool {
	return entry.MountPoint != ""
}

func (entry PartitionInfo) key() string {
	return strings.Join(append([]string{entry.Name, entry.Uuid}, entry.MountOptions...), "|")
}

func (i PartitionInfo) RootDir() string {
	return path.Join(i.MountPoint, i.bind)
}

func (i PartitionInfo) SetBind(bind string) PartitionInfo {
	i.bind = bind
	return i
}

func (i PartitionInfo) SetBind(bind string) PartitionInfo {
	i.bind = bind
	return i
}
