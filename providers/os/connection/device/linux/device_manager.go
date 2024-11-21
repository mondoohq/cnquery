// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
)

const (
	LunOption          = "lun"
	DeviceName         = "device-name"
	DeviceNames        = "device-names"
	MountAllPartitions = "mount-all-partitions"
	IncludeMounted     = "include-mounted"
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

	var deviceNames []string
	if opts[DeviceNames] != "" {
		deviceNames = strings.Split(opts[DeviceNames], ",")
	}
	if opts[DeviceName] != "" {
		deviceNames = append(deviceNames, opts[DeviceName])
	}

	var partitions []*snapshot.PartitionInfo
	var errs []error
	for _, deviceName := range deviceNames {
		partitionsForDevice, err := d.identifyViaDeviceName(deviceName, opts[MountAllPartitions] == "true", opts[IncludeMounted] == "true")
		if err != nil {
			errs = append(errs, err)
			continue
		}
		partitions = append(partitions, partitionsForDevice...)
	}

	return partitions, errors.Join(errs...)
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

	// this is needed only for the validation purposes
	deviceNames := opts[DeviceNames] + opts[DeviceName]

	mountAll := opts[MountAllPartitions] == "true"
	if lun != "" && deviceNames != "" {
		return errors.New("both lun and device names provided")
	}

	if lun == "" && deviceNames == "" {
		return errors.New("either lun or device names must be provided")
	}

	if deviceNames == "" && mountAll {
		return errors.New("mount-all-partitions requires device names")
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
	blockDevices, err := c.volumeMounter.CmdRunner.GetBlockDevices()
	if err != nil {
		return nil, err
	}
	var device snapshot.BlockDevice
	var deviceErr error
	// if we have exactly one device present at the LUN we can directly search for it
	if len(filteredScsiDevices) == 1 {
		devicePath := filteredScsiDevices[0].VolumePath
		device, deviceErr = blockDevices.FindDevice(devicePath)
	} else {
		device, deviceErr = findMatchingDeviceByBlock(filteredScsiDevices, blockDevices)
	}
	if deviceErr != nil {
		return nil, deviceErr
	}
	return device.GetMountablePartition()
}

func (c *LinuxDeviceManager) identifyViaDeviceName(deviceName string, mountAll bool, includeMounted bool) ([]*snapshot.PartitionInfo, error) {
	blockDevices, err := c.volumeMounter.CmdRunner.GetBlockDevices()
	if err != nil {
		return nil, err
	}

	device, err := blockDevices.FindDevice(deviceName)
	if err != nil {
		return nil, err
	}

	if mountAll {
		log.Debug().Str("device", device.Name).Msg("mounting all partitions")
		return device.GetMountablePartitions(true, includeMounted)
	}

	pi, err := device.GetMountablePartition()
	if err != nil {
		return nil, err
	}
	return []*snapshot.PartitionInfo{pi}, nil
}
