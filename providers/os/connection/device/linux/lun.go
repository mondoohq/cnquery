// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
)

type scsiDeviceInfo struct {
	// the LUN, e.g. 3
	Lun int
	// where the disk is mounted, e.g. /dev/sda
	VolumePath string
}

type scsiDevices = []scsiDeviceInfo

func (c *LinuxDeviceManager) listScsiDevices() ([]scsiDeviceInfo, error) {
	cmd, err := c.volumeMounter.CmdRunner.RunCommand("lsscsi --brief")
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

// there can be multiple devices mounted at the same LUN.
// the LUN so we need to find all the blocks, mounted at that LUN. then we find the first one
// that has no mounted partitions and use that as the target device. this is a best-effort approach
func findMatchingDeviceByBlock(scsiDevices scsiDevices, blockDevices *snapshot.BlockDevices) (snapshot.BlockDevice, error) {
	matchingBlocks := []snapshot.BlockDevice{}
	for _, device := range scsiDevices {
		for _, block := range blockDevices.BlockDevices {
			devName := "/dev/" + block.Name
			if devName == device.VolumePath {
				matchingBlocks = append(matchingBlocks, block)
			}
		}
	}

	if len(matchingBlocks) == 0 {
		return snapshot.BlockDevice{}, errors.New("no matching blocks found")
	}

	for _, b := range matchingBlocks {
		log.Debug().Str("name", b.Name).Msg("device connection> checking block")
		for _, ch := range b.Children {
			if len(ch.MountPoint) > 0 && ch.MountPoint != "" {
				log.Debug().Str("name", ch.Name).Msg("device connection> has mounted partitons, skipping")
			} else {
				// we found a block that has no mounted partitions
				return b, nil
			}
		}
	}

	return snapshot.BlockDevice{}, errors.New("no matching block found")
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
