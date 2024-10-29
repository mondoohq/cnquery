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
	// the tmp dirs we create; serves as the directory we mount the volumes to
	// maps the device name to the directory
	ScanDirs  map[string]string
	CmdRunner *LocalCommandRunner
}

func NewVolumeMounter(shell []string) *VolumeMounter {
	return &VolumeMounter{
		CmdRunner: &LocalCommandRunner{Shell: shell},
		ScanDirs:  make(map[string]string),
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
	m.ScanDirs[partition.Name] = dir

	return dir, m.mountVolume(partition)
}

func (m *VolumeMounter) createScanDir() (string, error) {
	dir, err := os.MkdirTemp("", "cnspec-scan")
	if err != nil {
		log.Error().Err(err).Msg("error creating directory")
		return "", err
	}
	log.Debug().Str("dir", dir).Msg("created tmp scan dir")
	return dir, nil
}

func (m *VolumeMounter) mountVolume(fsInfo *PartitionInfo) error {
	opts := []string{}
	if fsInfo.FsType == "xfs" {
		opts = append(opts, "nouuid")
	}
	opts = stringx.DedupStringArray(opts)
	scanDir := m.ScanDirs[fsInfo.Name]
	log.Debug().Str("fstype", fsInfo.FsType).Str("device", fsInfo.Name).Str("scandir", scanDir).Str("opts", strings.Join(opts, ",")).Msg("mount volume to scan dir")
	return Mount(fsInfo.Name, scanDir, fsInfo.FsType, opts)
}

func (m *VolumeMounter) UnmountVolumeFromInstance() error {
	if len(m.ScanDirs) == 0 {
		log.Warn().Msg("no scan dirs to unmount, skipping")
		return nil
	}

	var errs []error
	for name, dir := range m.ScanDirs {
		log.Debug().
			Str("dir", dir).
			Str("name", name).
			Msg("unmount volume")
		if err := Unmount(dir); err != nil {
			log.Error().
				Str("dir", dir).
				Err(err).Msg("failed to unmount dir")
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *VolumeMounter) RemoveTempScanDir() error {
	var errs []error
	for name, dir := range m.ScanDirs {
		log.Debug().
			Str("dir", dir).
			Str("name", name).
			Msg("remove created dir")
		if err := os.RemoveAll(dir); err != nil {
			log.Error().Err(err).
				Str("dir", dir).
				Msg("failed to remove dir")
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
