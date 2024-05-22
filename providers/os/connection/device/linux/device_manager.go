// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/connection/device/shared"
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
)

const (
	LunOption  = "lun"
	DeviceName = "device-name"
)

type LinuxDeviceManager struct {
	volumeMounter *snapshot.VolumeMounter
	opts          map[string]string
}

func NewLinuxDeviceManager(shell []string, opts map[string]string) (*LinuxDeviceManager, error) {
	if err := validateOpts(opts); err != nil {
		return nil, err
	}

	return &LinuxDeviceManager{
		volumeMounter: snapshot.NewVolumeMounter(shell),
		opts:          opts,
	}, nil
}

func (d *LinuxDeviceManager) Name() string {
	return "linux"
}

func (d *LinuxDeviceManager) IdentifyBlock(opts map[string]string) (shared.MountInfo, error) {
	if err := validateOpts(opts); err != nil {
		return shared.MountInfo{}, err
	}
	if opts[LunOption] != "" {
		lun, err := strconv.Atoi(opts[LunOption])
		if err != nil {
			return shared.MountInfo{}, err
		}
		return d.identifyViaLun(lun)
	}

	return d.identifyViaDeviceName(opts[DeviceName])
}

func (d *LinuxDeviceManager) Mount() (string, error) {
	// TODO: we should make the volume mounter return the scan dir from Mount()
	err := d.volumeMounter.Mount()
	if err != nil {
		return "", err
	}
	return d.volumeMounter.ScanDir, nil
}

func (d *LinuxDeviceManager) UnmountAndClose() {
	log.Debug().Msg("closing linux device manager")
	if d == nil {
		return
	}

	if d.volumeMounter != nil {
		err := d.volumeMounter.UnmountVolumeFromInstance()
		if err != nil {
			log.Error().Err(err).Msg("unable to unmount volume")
		}
		err = d.volumeMounter.RemoveTempScanDir()
		if err != nil {
			log.Error().Err(err).Msg("unable to remove dir")
		}
	}
}

// validates the options provided to the device manager
// we cannot have both LUN and device name provided, those are mutually exclusive
func validateOpts(opts map[string]string) error {
	lun := opts[LunOption]
	deviceName := opts[DeviceName]
	if lun != "" && deviceName != "" {
		return errors.New("both lun and device name provided")
	}

	return nil
}

func (c *LinuxDeviceManager) identifyViaLun(lun int) (shared.MountInfo, error) {
	scsiDevices, err := c.listScsiDevices()
	if err != nil {
		return shared.MountInfo{}, err
	}

	// only interested in the scsi devices that match the provided LUN
	filteredScsiDevices := filterScsiDevices(scsiDevices, lun)
	if len(filteredScsiDevices) == 0 {
		return shared.MountInfo{}, errors.New("no matching scsi devices found")
	}

	// if we have exactly one device present at the LUN we can directly point the volume mounter towards it
	if len(filteredScsiDevices) == 1 {
		return shared.MountInfo{DeviceName: filteredScsiDevices[0].VolumePath}, nil
	}

	// we have multiple devices at the same LUN. we find the first non-mounted block devices in that list
	blockDevices, err := c.volumeMounter.CmdRunner.GetBlockDevices()
	if err != nil {
		return shared.MountInfo{}, err
	}
	target, err := findMatchingDeviceByBlock(filteredScsiDevices, blockDevices)
	if err != nil {
		return shared.MountInfo{}, err
	}
	c.volumeMounter.VolumeAttachmentLoc = target
	return shared.MountInfo{DeviceName: target}, nil
}

func (c *LinuxDeviceManager) identifyViaDeviceName(deviceName string) (shared.MountInfo, error) {
	blockDevices, err := c.volumeMounter.CmdRunner.GetBlockDevices()
	if err != nil {
		return shared.MountInfo{}, err
	}
	// this is a best-effort approach, we try to find the first unmounted block device as we don't have the device name
	if deviceName == "" {
		fsInfo, err := blockDevices.GetUnmountedBlockEntry()
		if err != nil {
			return shared.MountInfo{}, err
		}
		c.volumeMounter.VolumeAttachmentLoc = deviceName
		return shared.MountInfo{DeviceName: fsInfo.Name}, nil
	}

	fsInfo, err := blockDevices.GetBlockEntryByName(deviceName)
	if err != nil {
		return shared.MountInfo{}, err
	}
	c.volumeMounter.VolumeAttachmentLoc = deviceName
	return shared.MountInfo{DeviceName: fsInfo.Name}, nil
}
