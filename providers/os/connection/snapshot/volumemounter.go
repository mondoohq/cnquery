// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"encoding/json"
	"io"
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
	opts                map[string]string
	CmdRunner           *LocalCommandRunner
}

func NewVolumeMounter(shell []string) *VolumeMounter {
	return &VolumeMounter{
		CmdRunner: &LocalCommandRunner{Shell: shell},
	}
}

// we should try and migrate this towards MountP where we specifically mount a target partition.
// the detection of the partition should be done in the caller (separately from Mount)
func (m *VolumeMounter) Mount() error {
	_, err := m.createScanDir()
	if err != nil {
		return err
	}
	fsInfo, err := m.getFsInfo()
	if err != nil {
		return err
	}
	if fsInfo == nil {
		return errors.New("unable to find target volume on instance")
	}
	log.Debug().Str("device name", fsInfo.Name).Msg("found target volume")
	return m.mountVolume(fsInfo)
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

func (m *VolumeMounter) getFsInfo() (*PartitionInfo, error) {
	log.Debug().Str("volume attachment loc", m.VolumeAttachmentLoc).Msg("search for target volume")

	// TODO: replace with mql query once version with lsblk resource is released
	// TODO: only use sudo if we are not root
	cmd, err := m.CmdRunner.RunCommand("sudo lsblk -f --json")
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	blockEntries := BlockDevices{}
	if err := json.Unmarshal(data, &blockEntries); err != nil {
		return nil, err
	}
	var fsInfo *PartitionInfo
	if m.opts[NoSetup] == "true" {
		// this means we didn't attach the volume to the instance
		// so we need to make a best effort guess
		return blockEntries.GetUnnamedBlockEntry()
	}

	fsInfo, err = blockEntries.GetMountablePartitionByDevice(m.VolumeAttachmentLoc)
	if err == nil && fsInfo != nil {
		return fsInfo, nil
	} else {
		// if we get here, we couldn't find a fs loaded at the expected location
		// AWS does not guarantee this, so that's expected. fallback to find non-boot and non-mounted volume
		fsInfo, err = blockEntries.GetUnmountedBlockEntry()
		if err == nil && fsInfo != nil {
			return fsInfo, nil
		}
	}
	return nil, err
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
	return blockDevices.GetMountablePartitionByDevice(device)
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
