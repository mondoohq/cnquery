// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"encoding/json"
	"fmt"
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

func (m *VolumeMounter) Mount() error {
	err := m.createScanDir()
	if err != nil {
		return err
	}
	// we should consider dropping this if VolumeAttachmentLoc is set. we need to also add FsType but
	// otherwise that means we're listing the devices twice
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

func (m *VolumeMounter) createScanDir() error {
	dir, err := os.MkdirTemp("", "cnspec-scan")
	if err != nil {
		log.Error().Err(err).Msg("error creating directory")
		return err
	}
	m.ScanDir = dir
	log.Debug().Str("dir", dir).Msg("created tmp scan dir")
	return nil
}

func (m *VolumeMounter) getFsInfo() (*fsInfo, error) {
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
	var fsInfo *fsInfo
	if m.opts[NoSetup] == "true" {
		// this means we didn't attach the volume to the instance
		// so we need to make a best effort guess
		return blockEntries.GetUnnamedBlockEntry()
	}

	fsInfo, err = blockEntries.GetBlockEntryByName(m.VolumeAttachmentLoc)
	if err == nil && fsInfo != nil {
		return fsInfo, nil
	}

	// if we get here, we couldn't find a fs loaded at the expected location
	// AWS does not guarantee this, so that's expected. fallback to find non-boot and non-mounted volume
	fsInfo, err = blockEntries.GetUnmountedBlockEntry()
	if err == nil && fsInfo != nil {
		return fsInfo, nil
	}
	return nil, err
}

func (m *VolumeMounter) mountVolume(fsInfo *fsInfo) error {
	opts := []string{}
	if fsInfo.FsType == "xfs" {
		opts = append(opts, "nouuid")
	}

	if fsInfo.LVM {
		log.Debug().
			Str("name", fsInfo.Name).
			Str("uuid", fsInfo.UUID).
			Msg("logical volume detected, getting block device")
		cmd, err := m.CmdRunner.RunCommand(fmt.Sprintf("sudo blkid --uuid %s", fsInfo.UUID))
		if err != nil {
			log.Debug().Err(err).Msg("unable to detect block device from logical volume")
			return err
		}
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			return err
		}
		fsInfo.Name = strings.Trim(string(data), "\t\n")
		log.Debug().
			Str("device", fsInfo.Name).
			Msg("block device found, setting as new device name")
	}

	opts = stringx.DedupStringArray(opts)
	log.Debug().
		Str("fstype", fsInfo.FsType).
		Str("device", fsInfo.Name).
		Str("scandir", m.ScanDir).
		Str("opts", strings.Join(opts, ",")).
		Msg("mount volume to scan dir")
	return Mount(fsInfo.Name, m.ScanDir, fsInfo.FsType, opts)
}

func (m *VolumeMounter) UnmountVolumeFromInstance() error {
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
