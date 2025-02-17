// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type scsiDeviceInfo struct {
	// the LUN, e.g. 3
	Lun int
	// where the disk is mounted, e.g. /dev/sda
	VolumePath string
}

type scsiDevices = []scsiDeviceInfo

func (c *LinuxDeviceManager) listScsiDevices() ([]scsiDeviceInfo, error) {
	cmd, err := c.cmdRunner.RunCommand("lsscsi --brief")
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

func filterScsiDevices(scsiDevices scsiDevices, lun int) []scsiDeviceInfo {
	matching := []scsiDeviceInfo{}
	for _, d := range scsiDevices {
		if d.Lun == lun {
			matching = append(matching, d)
		}
	}

	return matching
}

// parses the output from running 'lsscsi --brief' and gets the device info, the output looks like this:
// [0:0:0:0]    /dev/sda
// [1:0:0:0]    /dev/sdb
func parseLsscsiOutput(output string) (scsiDevices, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	mountedDevices := []scsiDeviceInfo{}
	for _, line := range lines {
		log.Debug().Str("line", line).Msg("device connection> parsing lsscsi output")
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
		mountedDevices = append(mountedDevices, scsiDeviceInfo{Lun: lunInt, VolumePath: path})
	}

	return mountedDevices, nil
}
