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

//go:generate mockgen -source=./volumemounter.go -destination=./mock_volumemounter.go -package=snapshot
type VolumeMounter interface {
	MountP(dto *MountPartitionDto) (string, error)
	UmountP(partition *PartitionInfo) error
	UnmountVolumeFromInstance() error
	RemoveTempScanDir() error
}

type volumeMounter struct {
	// the tmp dirs we create; serves as the directory we mount the volumes to
	// maps the device name to the directory
	scanDirs  map[string]string
	cmdRunner *LocalCommandRunner
}

func NewVolumeMounter(shell []string) VolumeMounter {
	return &volumeMounter{
		cmdRunner: &LocalCommandRunner{Shell: shell},
		scanDirs:  make(map[string]string),
	}
}

// Mounts a specific partition and returns the directory it was mounted to
func (m *volumeMounter) MountP(dto *MountPartitionDto) (string, error) {
	if dto == nil {
		return "", errors.New("mount device> partition is required")
	}

	partition := dto.PartitionInfo
	if partition == nil {
		return "", errors.New("mount device> partition is required")
	}
	if partition.Name == "" {
		return "", errors.New("mount device> partition name is required")
	}
	if partition.FsType == "" {
		return "", errors.New("mount device> partition fs type is required")
	}

	var dir string
	var err error
	if dto.ScanDir == nil {
		dir, err = m.createScanDir()
		if err != nil {
			return "", err
		}
	} else {
		dir = *dto.ScanDir
	}

	m.scanDirs[partition.key()] = dir

	return dir, m.mountVolume(dto)
}

func (m *volumeMounter) UmountP(partition *PartitionInfo) error {
	if partition == nil {
		return errors.New("unmount device> partition is required")
	}
	if partition.Name == "" {
		return errors.New("unmount device> partition name is required")
	}
	key := partition.key()
	dir, ok := m.scanDirs[key]
	if !ok {
		return errors.New("unmount device> partition not found")
	}
	log.Debug().Str("dir", dir).Str("name", partition.Name).Msg("unmount volume")
	if err := Unmount(dir); err != nil {
		log.Warn().Err(err).Str("dir", dir).Msg("failed to unmount dir")
		return err
	}
	delete(m.scanDirs, key)

	return nil
}

func (m *volumeMounter) createScanDir() (string, error) {
	dir, err := os.MkdirTemp("", "cnspec-scan")
	if err != nil {
		log.Error().Err(err).Msg("error creating directory")
		return "", err
	}
	log.Debug().Str("dir", dir).Msg("created tmp scan dir")
	return dir, nil
}

func (m *volumeMounter) mountVolume(fsInfo *MountPartitionDto) error {
	opts := fsInfo.MountOptions
	if fsInfo.FsType == "xfs" {
		opts = append(opts, "nouuid")
	}
	opts = stringx.DedupStringArray(opts)
	opts = sanitizeOptions(opts)

	scanDir := m.scanDirs[fsInfo.key()]
	log.Debug().Str("fstype", fsInfo.FsType).Str("device", fsInfo.Name).Str("scandir", scanDir).Str("opts", strings.Join(opts, ",")).Msg("mount volume to scan dir")
	return Mount(fsInfo.Name, scanDir, fsInfo.FsType, opts)
}

func sanitizeOptions(options []string) []string {
	sanitized := make([]string, 0)
	for _, opt := range options {
		switch {
		case opt == "defaults":
			continue
		case strings.HasPrefix(opt, "x-systemd"):
			continue
		default:
			sanitized = append(sanitized, opt)
		}
	}
	return sanitized
}

func (m *volumeMounter) UnmountVolumeFromInstance() error {
	if len(m.scanDirs) == 0 {
		log.Warn().Msg("no scan dirs to unmount, skipping")
		return nil
	}

	var errs []error
	for name, dir := range m.scanDirs {
		log.Debug().
			Str("dir", dir).
			Str("name", name).
			Msg("unmount volume")
		if err := Unmount(dir); err != nil {
			log.Warn().
				Str("dir", dir).
				Err(err).Msg("failed to unmount dir")
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (m *volumeMounter) RemoveTempScanDir() error {
	var errs []error
	for name, dir := range m.scanDirs {
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
