// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
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
