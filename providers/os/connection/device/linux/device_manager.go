// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"strconv"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
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

func (d *LinuxDeviceManager) IdentifyMountTargets(opts map[string]string) ([]*snapshot.PartitionInfo, error) {
	if err := validateOpts(opts); err != nil {
		return nil, err
	}
	if opts[LunOption] != "" {
		lun, err := strconv.Atoi(opts[LunOption])
		if err != nil {
			return nil, err
		}
		pi, err := d.identifyViaLun(lun)
		if err != nil {
			return nil, err
		}
		return []*snapshot.PartitionInfo{pi}, nil
	}

	pi, err := d.identifyViaDeviceName(opts[DeviceName])
	if err != nil {
		return nil, err
	}
	return []*snapshot.PartitionInfo{pi}, nil
}

func (d *LinuxDeviceManager) Mount(pi *snapshot.PartitionInfo) (string, error) {
	return d.volumeMounter.MountP(pi)
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

func (c *LinuxDeviceManager) identifyViaLun(lun int) (*snapshot.PartitionInfo, error) {
	scsiDevices, err := c.listScsiDevices()
	if err != nil {
		return nil, err
	}

	// only interested in the scsi devices that match the provided LUN
	filteredScsiDevices := filterScsiDevices(scsiDevices, lun)
	if len(filteredScsiDevices) == 0 {
		return nil, errors.New("no matching scsi devices found")
	}

	var target string
	// if we have exactly one device present at the LUN we can directly point the volume mounter towards it
	if len(filteredScsiDevices) == 1 {
		target = filteredScsiDevices[0].VolumePath
	} else {
		// we have multiple devices at the same LUN. we find the first non-mounted block devices in that list
		blockDevices, err := c.volumeMounter.CmdRunner.GetBlockDevices()
		if err != nil {
			return nil, err
		}
		target, err = findMatchingDeviceByBlock(filteredScsiDevices, blockDevices)
		if err != nil {
			return nil, err
		}
	}

	return c.volumeMounter.GetDeviceForMounting(target)
}

func (c *LinuxDeviceManager) identifyViaDeviceName(deviceName string) (*snapshot.PartitionInfo, error) {
	// GetDeviceForMounting also supports passing in empty strings, in that case we do a best-effort guess
	return c.volumeMounter.GetDeviceForMounting(deviceName)
}
