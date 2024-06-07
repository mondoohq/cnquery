// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
)

const (
	identifyDiskDrivesPwshScript = `Get-WmiObject -Class Win32_DiskDrive | Select-Object Name,SCSILogicalUnit,Index | ConvertTo-Json`
	identifyPartitionPwshScript  = `Get-Disk -Number %d | Get-Partition | Select DriveLetter, Size, Type | ConvertTo-Json`
)

type diskDrive struct {
	Name            string `json:"Name"`
	SCSILogicalUnit int    `json:"SCSILogicalUnit"`
	Index           int    `json:"Index"`
}

type diskPartition struct {
	DriveLetter string `json:"DriveLetter"`
	Size        uint64 `json:"Size"`
	Type        string `json:"Type"`
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
		return nil, err
	}

	return partitions, nil
}

func filterDiskDrives(drives []*diskDrive, lun int) (*diskDrive, error) {
	for _, d := range drives {
		if lun == d.SCSILogicalUnit {
			return d, nil
		}
	}
	return nil, errors.New("no disk drive with matching LUN found")
}

func filterPartitions(partitions []*diskPartition) (*diskPartition, error) {
	allowed := []string{"Basic", "Windows"}
	for _, p := range partitions {
		if slices.Contains(allowed, p.Type) && p.DriveLetter != "" {
			return p, nil
		}
	}
	return nil, errors.New("no basic partition with assigned drive letter found")
}
