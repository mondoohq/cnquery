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
	LunOption          = "lun"
	DeviceName         = "device-name"
	MountAllPartitions = "mount-all-partitions"
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

	partitions, err := d.identifyViaDeviceName(opts[DeviceName], opts[MountAllPartitions] == "true")
	if err != nil {
		return nil, err
	}
	return partitions, nil
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
	mountAll := opts[MountAllPartitions] == "true"
	if lun != "" && deviceName != "" {
		return errors.New("both lun and device name provided")
	}
	if deviceName == "" && mountAll {
		return errors.New("mount-all-partitions requires a device name")
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

func (c *LinuxDeviceManager) identifyViaDeviceName(deviceName string, mountAll bool) ([]*snapshot.PartitionInfo, error) {
	blockDevices, err := c.volumeMounter.CmdRunner.GetBlockDevices()
	if err != nil {
		return nil, err
	}

	// if we don't have a device name we can just return the first non-boot, non-mounted partition.
	// this is a best-guess approach
	if deviceName == "" {
		// TODO: we should rename/simplify this method
		pi, err := blockDevices.GetUnnamedBlockEntry()
		if err != nil {
			return nil, err
		}
		return []*snapshot.PartitionInfo{pi}, nil
	}

	// if we have a specific device we're looking for we can just ask only for that
	device, err := blockDevices.FindDevice(deviceName)
	if err != nil {
		return nil, err
	}

	if mountAll {
		log.Debug().Str("device", device.Name).Msg("mounting all partitions")
		return device.GetMountablePartitions(true)
	}

	pi, err := device.GetMountablePartition()
	if err != nil {
		return nil, err
	}
	return []*snapshot.PartitionInfo{pi}, nil
}
