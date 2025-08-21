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
}

func (p *Partition) ToMountInput(opts []string, mountDir string) *MountPartitionInput {
	return &MountPartitionInput{
		Name:         p.Name,
		FsType:       p.FsType,
		Label:        p.Label,
		Uuid:         p.Uuid,
		PartUuid:     p.PartUuid,
		RootPath:     p.RootPath,
		MountOptions: opts,
		MountDir:     mountDir,
	}
}

func (p *Partition) ToDefaultMountInput() *MountPartitionInput {
	return p.ToMountInput([]string{}, "")
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

	// if specified, indicates where the root filesystem is present on the partition
	// this is useful for ostree where the root fs is not under the mounted partition directly
	// but is nested in a folder somewhere (e.g. boot.1)
	// e.g. ostree/boot.1.1/fedora-coreos/1f65edba61a143a78be83340f66d3e247e20ec48a539724ca037607c7bdf4942/0
	rootPath string
}

// MountPartitionInput is the input for the Mount method
type MountPartitionInput struct {
	MountOptions []string
	FsType       string
	Label        string
	PartUuid     string
	Uuid         string
	Name         string
	// if specfied, mount the partition at this directory
	MountDir string
	// if specified, indicates where the root filesystem is present on the partition
	// this is useful for ostree where the root fs is not under the mounted partition directly
	// but is nested in a folder somewhere (e.g. boot.1)
	// e.g. ostree/boot.1.1/fedora-coreos/1f65edba61a143a78be83340f66d3e247e20ec48a539724ca037607c7bdf4942/0
	RootPath string
}

// Gets the path on the mounted partition where the root filesystem is located.
func (mp *MountedPartition) RootFsPath() string {
	return path.Join(mp.MountPoint, mp.rootPath)
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
