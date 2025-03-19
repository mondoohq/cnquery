// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
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
	devices, err := c.listScsiDevicesFromSys()
	if err == nil {
		return devices, nil
	}

	log.Warn().Err(err).Msg("failed to list scsi devices from sys, trying lsscsi")
	return c.listScsiDevicesFromCLI()
}

func (c *LinuxDeviceManager) listScsiDevicesFromSys() ([]scsiDeviceInfo, error) {
	blocks, err := os.ReadDir("/sys/block/")
	if err != nil {
		return nil, err
	}
	scsiDevices := []scsiDeviceInfo{}
	for _, block := range blocks {
		if block.Type() != os.ModeSymlink {
			continue
		}

		entry, err := os.Readlink(path.Join("/sys/block/", block.Name()))
		if err != nil {
			log.Warn().Err(err).Str("block", block.Name()).Msg("failed to readlink")
			continue
		}
		entry, err = filepath.Abs(path.Join("/sys/block/", entry))
		if err != nil {
			log.Warn().Err(err).Str("block", block.Name()).Msg("failed to get absolute path")
			continue
		}

		device, err := parseScsiDevicePath(entry)
		if err != nil {
			log.Debug().Err(err).Str("block", block.Name()).Msg("failed to parse device path")
			continue
		}
		scsiDevices = append(scsiDevices, device)
	}

	return scsiDevices, nil
}

func (c *LinuxDeviceManager) listScsiDevicesFromCLI() ([]scsiDeviceInfo, error) {
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

// parses the device path from the sysfs block device path, e.g. /sys/devices/pci0000:00/0000:00:14.0/usb1/1-1/1-1:1.0/host2/target2:0:0/2:0:0:0/block/sdb
// and returns the LUN and the path to the device, e.g. 0 and /dev/sdb
func parseScsiDevicePath(entry string) (res scsiDeviceInfo, err error) {
	chunks := strings.Split(entry, "/")
	if len(chunks) < 5 {
		return res, fmt.Errorf("unexpected entry: %s", entry)
	}

	if chunks[len(chunks)-2] != "block" {
		return res, fmt.Errorf("unexpected entry, expected block: %s", entry)
	}

	res.VolumePath = path.Join("/dev", chunks[len(chunks)-1])
	hbtl := strings.Split(chunks[len(chunks)-3], ":")
	if len(hbtl) != 4 {
		return res, fmt.Errorf("unexpected entry, expected 4 fields for H:B:T:L: %s", entry)
	}

	res.Lun, err = strconv.Atoi(hbtl[3])
	if err != nil {
		return res, fmt.Errorf("unexpected entry, expected integer for LUN: %s", entry)
	}

	return res, nil
}
