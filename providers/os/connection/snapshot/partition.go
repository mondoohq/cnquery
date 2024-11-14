// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"strings"
)

type PartitionInfo struct {
	Name   string
	FsType string

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
