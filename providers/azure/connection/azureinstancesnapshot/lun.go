// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azureinstancesnapshot

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/providers/os/connection/local"
)

type deviceInfo struct {
	// the LUN number, e.g. 3
	Lun int32
	// where the disk is mounted, e.g. /dev/sda
	VolumePath string
}

func (a *azureScannerInstance) getAvailableLun(mountedDevices []deviceInfo) (int32, error) {
	takenLuns := []int32{}
	for _, d := range mountedDevices {
		takenLuns = append(takenLuns, d.Lun)
	}

	availableLuns := []int32{}
	// the available LUNs are 0-63, so we exclude everything thats in takenLuns
	for i := int32(0); i < 64; i++ {
		exists := false
		for _, d := range takenLuns {
			if d == i {
				exists = true
				break
			}
		}
		if exists {
			// log just for visibility
			log.Debug().Int32("LUN", i).Msg("azure snapshot> LUN is taken, skipping")
		} else {
			availableLuns = append(availableLuns, i)
		}
	}
	if len(availableLuns) == 0 {
		return 0, errors.New("no available LUNs to attach disk to")
	}
	return availableLuns[0], nil
}

// https://learn.microsoft.com/en-us/azure/virtual-machines/linux/azure-to-guest-disk-mapping
// for more information. we want to find the LUNs of the data disks and their mount location
func getMountedDevices(localConn *local.LocalConnection) ([]deviceInfo, error) {
	cmd, err := localConn.RunCommand("lsscsi --brief")
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to list logical unit numbers: %s", outErr)
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	output := string(data)
	return parseLsscsiOutput(output)
}

func getMatchingDevice(mountedDevices []deviceInfo, lun int32) (deviceInfo, error) {
	for _, d := range mountedDevices {
		if d.Lun == lun {
			return d, nil
		}
	}
	return deviceInfo{}, errors.New("could not find matching device")
}

// parses the output from running 'lsscsi --brief' and gets the device info, the output looks like this:
// [0:0:0:0]    /dev/sda
// [1:0:0:0]    /dev/sdb
func parseLsscsiOutput(output string) ([]deviceInfo, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	mountedDevices := []deviceInfo{}
	for _, line := range lines {
		log.Debug().Str("line", line).Msg("azure snapshot> parsing lsscsi output")
		if line == "" {
			continue
		}
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid lsscsi output: %s", line)
		}
		lunInfo := parts[0]
		path := parts[1]
		// trim the [], turning [1:0:0:0] into 1:0:0:0
		trimLun := strings.Trim(lunInfo, "[]")
		splitLun := strings.Split(trimLun, ":")
		// the LUN is the last one
		lun := splitLun[len(splitLun)-1]
		lunInt, err := strconv.Atoi(lun)
		if err != nil {
			return nil, err
		}
		mountedDevices = append(mountedDevices, deviceInfo{Lun: int32(lunInt), VolumePath: path})
	}

	return mountedDevices, nil
}
