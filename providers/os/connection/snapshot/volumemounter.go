// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"os"
	"strings"

	"go.mondoo.com/cnquery/v11/utils/stringx"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

const (
	NoSetup = "no-setup"
	IsSetup = "is-setup"
)

type VolumeMounter struct {
	// the tmp dir we create; serves as the directory we mount the volume to
	ScanDir string
	// where we tell AWS to attach the volume; it doesn't necessarily get attached there, but we have to reference this same location when detaching
	VolumeAttachmentLoc string
	CmdRunner           *LocalCommandRunner
}

func NewVolumeMounter(shell []string) *VolumeMounter {
	return &VolumeMounter{
		CmdRunner: &LocalCommandRunner{Shell: shell},
	}
}

// Mounts a specific partition and returns the directory it was mounted to
func (m *VolumeMounter) MountP(partition *PartitionInfo) (string, error) {
	if partition == nil {
		return "", errors.New("mount device> partition is required")
	}
	if partition.Name == "" {
		return "", errors.New("mount device> partition name is required")
	}
	if partition.FsType == "" {
		return "", errors.New("mount device> partition fs type is required")
	}
	dir, err := m.createScanDir()
	if err != nil {
		return "", err
	}
	return dir, m.mountVolume(partition)
}

func (m *VolumeMounter) createScanDir() (string, error) {
	dir, err := os.MkdirTemp("", "cnspec-scan")
	if err != nil {
		log.Error().Err(err).Msg("error creating directory")
		return "", err
	}
	m.ScanDir = dir
	log.Debug().Str("dir", dir).Msg("created tmp scan dir")
	return dir, nil
}

// GetDeviceForMounting iterates through all the partitions of the target and returns the first one that matches the filters
// If device is not specified, it will return the first non-mounted, non-boot partition (best-effort guessing)
// E.g. if target is "sda", it will return the first partition of the block device "sda" that satisfies the filters
func (m *VolumeMounter) GetMountablePartition(device string) (*PartitionInfo, error) {
	if device == "" {
		log.Debug().Msg("no device provided, searching for unnamed block device")
	} else {
		log.Debug().Str("device", device).Msg("search for target partition")
	}

	blockDevices, err := m.CmdRunner.GetBlockDevices()
	if err != nil {
		return nil, err
	}
	if device == "" {
		// TODO: i dont know what the difference between GetUnnamedBlockEntry and GetUnmountedBlockEntry is
		// we need to simplify those
		return blockDevices.GetUnnamedBlockEntry()
	}

	d, err := blockDevices.FindDevice(device)
	if err != nil {
		return nil, err
	}
	return d.GetMountablePartition()
}

func (m *VolumeMounter) mountVolume(fsInfo *PartitionInfo) error {
	opts := []string{}
	if fsInfo.FsType == "xfs" {
		opts = append(opts, "nouuid")
	}
	opts = stringx.DedupStringArray(opts)
	log.Debug().Str("fstype", fsInfo.FsType).Str("device", fsInfo.Name).Str("scandir", m.ScanDir).Str("opts", strings.Join(opts, ",")).Msg("mount volume to scan dir")
	return Mount(fsInfo.Name, m.ScanDir, fsInfo.FsType, opts)
}

func (m *VolumeMounter) UnmountVolumeFromInstance() error {
	if m.ScanDir == "" {
		log.Warn().Msg("no scan dir to unmount, skipping")
		return nil
	}
	log.Debug().Str("dir", m.ScanDir).Msg("unmount volume")
	if err := Unmount(m.ScanDir); err != nil {
		log.Error().Err(err).Msg("failed to unmount dir")
		return err
	}
	return nil
}

func (m *VolumeMounter) RemoveTempScanDir() error {
	log.Debug().Str("dir", m.ScanDir).Msg("remove created dir")
	return os.RemoveAll(m.ScanDir)
}
