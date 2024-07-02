// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

const (
	identifyDiskDrivesPwshScript = `Get-WmiObject -Class Win32_DiskDrive | Select-Object Name, SCSILogicalUnit, Index, SerialNumber | ConvertTo-Json`
	identifyDiskOnlinePwshScript = `Get-Disk -Number %d | Select-Object Number, IsOffline, IsReadOnly | ConvertTo-Json`
	identifyPartitionPwshScript  = `Get-Disk -Number %d | Get-Partition | Select DriveLetter, Size, Type | ConvertTo-Json`
	setDiskOnlinePwshScript      = `Set-Disk -Number %d -IsOffline %s`
	setDiskReadOnlyPwshScript    = `Set-Disk -Number %d -IsReadOnly %s`
)

type diskDrive struct {
	Name            string `json:"Name"`
	SCSILogicalUnit int    `json:"SCSILogicalUnit"`
	Index           int    `json:"Index"`
	SerialNumber    string `json:"SerialNumber"`
}

type diskStatus struct {
	Number   int  `json:"Number"`
	Offline  bool `json:"IsOffline"`
	Readonly bool `json:"IsReadOnly"`
}

type diskPartition struct {
	DriveLetter string `json:"DriveLetter"`
	Size        uint64 `json:"Size"`
	Type        string `json:"Type"`
}

func (d *WindowsDeviceManager) setDiskOnlineState(diskNumber int, online bool) error {
	str := "$true"
	if online {
		str = "$false"
	}
	log.Debug().Int("diskNumber", diskNumber).Bool("online", online).Msg("setting disk online state")
	script := fmt.Sprintf(setDiskOnlinePwshScript, diskNumber, str)
	cmd, err := d.cmdRunner.RunCommand(powershell.Encode(script))
	if err != nil {
		return err
	}
	if cmd.ExitStatus != 0 {
		outErr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to run powershell script: %s", outErr)
	}

	return nil
}

func (d *WindowsDeviceManager) setDiskReadonlyState(diskNumber int, readonly bool) error {
	str := "$true"
	if !readonly {
		str = "$false"
	}
	log.Debug().Int("diskNumber", diskNumber).Bool("readonly", readonly).Msg("setting disk readonly state")
	script := fmt.Sprintf(setDiskReadOnlyPwshScript, diskNumber, str)
	cmd, err := d.cmdRunner.RunCommand(powershell.Encode(script))
	if err != nil {
		return err
	}
	if cmd.ExitStatus != 0 {
		outErr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to run powershell script: %s", outErr)
	}

	return nil
}

func (d *WindowsDeviceManager) identifyDiskStatus(diskNumber int) (*diskStatus, error) {
	cmd, err := d.cmdRunner.RunCommand(powershell.Encode(fmt.Sprintf(identifyDiskOnlinePwshScript, diskNumber)))
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to run powershell script: %s", outErr)
	}

	stdout, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	var status *diskStatus
	err = json.Unmarshal(stdout, &status)
	if err != nil {
		return nil, err
	}

	return status, nil
}

func (d *WindowsDeviceManager) IdentifyDiskDrives() ([]*diskDrive, error) {
	cmd, err := d.cmdRunner.RunCommand(powershell.Encode(identifyDiskDrivesPwshScript))
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		outErr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to run powershell script: %s", outErr)
	}

	stdout, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	var drives []*diskDrive
	err = json.Unmarshal(stdout, &drives)
	if err != nil {
		return nil, err
	}

	return drives, nil
}

func (d *WindowsDeviceManager) identifyPartitions(diskNumber int) ([]*diskPartition, error) {
	script := fmt.Sprintf(identifyPartitionPwshScript, diskNumber)
	cmd, err := d.cmdRunner.RunCommand(powershell.Encode(script))
	if err != nil {
		return nil, err
	}

	if cmd.ExitStatus != 0 {
		outErr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to run powershell script: %s", outErr)
	}

	stdout, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	var partitions []*diskPartition
	err = json.Unmarshal(stdout, &partitions)
	if err != nil {
		// fallback, if only one partition is found, the output is not an array
		var partition *diskPartition
		err = json.Unmarshal(stdout, &partition)
		if err != nil {
			return nil, err
		}
		return []*diskPartition{partition}, nil
	}

	return partitions, nil
}

func filterDiskDrives(drives []*diskDrive, opts map[string]string) (*diskDrive, error) {
	serialNumber := opts[SerialNumberOption]
	lun := opts[LunOption]
	if serialNumber != "" {
		return filterDiskDrivesBySerialNumber(drives, serialNumber)
	}

	lunInt, err := strconv.Atoi(lun)
	if err != nil {
		return nil, err
	}
	return filterDiskDrivesByLun(drives, lunInt)
}

func filterDiskDrivesBySerialNumber(drives []*diskDrive, serialNumber string) (*diskDrive, error) {
	for _, d := range drives {
		if serialNumber == d.SerialNumber {
			log.Debug().Str("serialNumber", serialNumber).Str("name", d.Name).Int("index", d.Index).Msg("found disk drive with matching serial number")
			return d, nil
		}
	}
	return nil, errors.New("no disk drive with matching serial number found")
}

func filterDiskDrivesByLun(drives []*diskDrive, lun int) (*diskDrive, error) {
	for _, d := range drives {
		if lun == d.SCSILogicalUnit {
			log.Debug().Int("lun", lun).Str("name", d.Name).Int("index", d.Index).Msg("found disk drive with matching LUN")
			return d, nil
		}
	}
	return nil, errors.New("no disk drive with matching LUN found")
}

func filterPartitions(partitions []*diskPartition) (*diskPartition, error) {
	allowed := []string{"Basic", "Windows", "IFS"}
	for _, p := range partitions {
		if slices.Contains(allowed, p.Type) && p.DriveLetter != "" {
			return p, nil
		}
	}
	return nil, errors.New("no basic partition with assigned drive letter found")
}
