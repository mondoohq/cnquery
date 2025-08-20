// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"strings"
)

type Partition struct {
	// Device name (e.g. /dev/sda1)
	Name string
	// Filesystem type (e.g. ext4)
	FsType string
	// Resolved device name aliases (e.g. /dev/sda1 -> /dev/nvme0n1p1)
	Aliases []string
	// (optional) Label is the partition label
	Label string
	// (optional) UUID is the volume UUID
	Uuid string
	// (optional) PartUuid is the partition UUID
	PartUuid string
}

func (p *Partition) ToMountInput(opts []string, scanDir *string) *MountPartitionInput {
	return &MountPartitionInput{
		Name:         p.Name,
		FsType:       p.FsType,
		Label:        p.Label,
		Uuid:         p.Uuid,
		PartUuid:     p.PartUuid,
		MountOptions: opts,
		ScanDir:      scanDir,
	}
}

func (p *Partition) ToDefaultMountInput() *MountPartitionInput {
	return p.ToMountInput([]string{}, nil)
}

type MountedPartition struct {
	// Device name (e.g. /dev/sda1)
	Name string
	// Filesystem type (e.g. ext4)
	FsType string

	// Resolved device name aliases (e.g. /dev/sda1 -> /dev/nvme0n1p1)
	Aliases []string
	// (optional) Label is the partition label
	Label string
	// (optional) UUID is the volume UUID
	Uuid       string
	MountPoint string
	// (optional) PartUuid is the partition UUID
	PartUuid string
	// (optional) MountOptions are the mount options
	MountOptions []string
}

// MountPartitionInput is the input for the Mount method
type MountPartitionInput struct {
	MountOptions []string
	FsType       string
	Label        string
	PartUuid     string
	Uuid         string
	Name         string
	// Override the scan dir for the mount
	ScanDir *string
}

func (entry BlockDevice) isNoBootVolume() bool {
	typ := strings.ToLower(entry.FsType)
	label := strings.ToLower(entry.Label)
	return entry.Uuid != "" && typ != "" && typ != "vfat" && label != "efi" && label != "boot"
}

func (entry BlockDevice) isMounted() bool {
	if len(entry.MountPoints) == 1 && entry.MountPoints[0] == "" {
		// This is a special case where the partition is not mounted
		return false
	}
	return len(entry.MountPoints) > 0
}
