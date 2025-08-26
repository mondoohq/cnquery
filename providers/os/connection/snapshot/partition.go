// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"path"
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

	// if specified, indicates where the root filesystem is present on the partition
	// this is useful for ostree where the root fs is not under the mounted partition directly
	// but is nested in a folder somewhere (e.g. boot.1)
	// e.g. ostree/boot.1.1/fedora-coreos/1f65edba61a143a78be83340f66d3e247e20ec48a539724ca037607c7bdf4942/0
	RootPath string

	// RequestedName is the name of the partition as requested by the user.
	// This might differ from the actual Name if the partition was found using interchangeable names.
	// E.g. this could be '/dev/sdm' while Name is '/dev/xvdm' since we treat [sd]m and [xvd]m the same.
	RequestedName string
}

func (p *Partition) ToMountInput(opts []string, mountDir string) *MountPartitionInput {
	return &MountPartitionInput{
		Partition:    p,
		MountOptions: opts,
		MountDir:     mountDir,
	}
}

func (p *Partition) ToDefaultMountInput() *MountPartitionInput {
	return p.ToMountInput([]string{}, "")
}

type MountedPartition struct {
	Partition *Partition
	// MountPoint is the directory where the partition is mounted
	MountPoint string
	// MountOptions are the mount options
	MountOptions []string
}

// MountPartitionInput is the input for the Mount method
type MountPartitionInput struct {
	Partition    *Partition
	MountOptions []string
	// if specfied, mount the partition at this directory
	MountDir string
}

// Gets the path on the mounted partition where the root filesystem is located.
func (mp *MountedPartition) RootFsPath() string {
	return path.Join(mp.MountPoint, mp.Partition.RootPath)
}

func (entry BlockDevice) isNoBootVolume() bool {
	typ := strings.ToLower(entry.FsType)
	label := strings.ToLower(entry.Label)
	return entry.Uuid != "" && typ != "" && typ != "vfat" && label != "efi" && label != "boot"
}

func (entry BlockDevice) isMounted() bool {
	return entry.MountPoint != ""
}
