// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"errors"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
)

const (
	LunOption = "lun"
)

type WindowsDeviceManager struct {
	cmdRunner *snapshot.LocalCommandRunner
	opts      map[string]string
	// indicates if the disk we've targeted has been set to online. we use this to know if we need to put it back offline once we're done
	diskSetToOnline bool
	// if we've set the disk online, we need to know the index to set it back offline
	diskIndex int
}

func NewWindowsDeviceManager(shell []string, opts map[string]string) (*WindowsDeviceManager, error) {
	if err := validateOpts(opts); err != nil {
		return nil, err
	}
	return &WindowsDeviceManager{
		cmdRunner: &snapshot.LocalCommandRunner{Shell: shell},
		opts:      opts,
	}, nil
}

func (d *WindowsDeviceManager) Name() string {
	return "windows"
}

func (d *WindowsDeviceManager) IdentifyMountTargets(opts map[string]string) ([]*snapshot.PartitionInfo, error) {
	lun := opts[LunOption]
	lunInt, err := strconv.Atoi(lun)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("lun", lun).Msg("device connection> identifying mount targets")
	diskDrives, err := d.IdentifyDiskDrives()
	if err != nil {
		return nil, err
	}

	targetDrive, err := filterDiskDrives(diskDrives, lunInt)
	if err != nil {
		return nil, err
	}

	diskOnline, err := d.identifyDiskOnline(targetDrive.Index)
	if err != nil {
		return nil, err
	}
	if diskOnline.IsOffline {
		err = d.setDiskOnlineState(targetDrive.Index, true)
		if err != nil {
			return nil, err
		}
		d.diskSetToOnline = true
		d.diskIndex = targetDrive.Index
	}
	partitions, err := d.identifyPartitions(targetDrive.Index)
	if err != nil {
		return nil, err
	}
	partition, err := filterPartitions(partitions)
	if err != nil {
		return nil, err
	}
	partitionInfo := &snapshot.PartitionInfo{
		Name:   partition.DriveLetter,
		FsType: "Windows",
	}
	return []*snapshot.PartitionInfo{partitionInfo}, nil
}

// validates the options provided to the device manager
func validateOpts(opts map[string]string) error {
	lun := opts[LunOption]
	if lun == "" {
		return errors.New("lun is required for a windows device connection")
	}

	return nil
}

func (d *WindowsDeviceManager) Mount(pi *snapshot.PartitionInfo) (string, error) {
	// note: we do not (yet) do the mounting in windows. for now, we simply return the drive letter
	// as that means the drive is already mounted
	if strings.HasSuffix(pi.Name, ":") {
		return pi.Name, nil
	}
	return pi.Name + ":", nil
}

func (d *WindowsDeviceManager) UnmountAndClose() {
	log.Debug().Msg("closing windows device manager")
	if d.diskSetToOnline {
		err := d.setDiskOnlineState(d.diskIndex, false)
		if err != nil {
			log.Debug().Err(err).Msg("could not set disk offline")
		}
	}
}
