// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azureinstancesnapshot

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

type scsiDeviceInfo struct {
	// the LUN, e.g. 3
	Lun int32
	// where the disk is mounted, e.g. /dev/sda
	VolumePath string
}

type scsiDevices = []scsiDeviceInfo

// TODO: we should combine this with the OS-connection blockDevices struct
type blockDevices struct {
	BlockDevices []blockDevice `json:"blockDevices,omitempty"`
}

type blockDevice struct {
	Name       string        `json:"name,omitempty"`
	FsType     string        `json:"fstype,omitempty"`
	Label      string        `json:"label,omitempty"`
	Uuid       string        `json:"uuid,omitempty"`
	Mountpoint []string      `json:"mountpoints,omitempty"`
	Children   []blockDevice `json:"children,omitempty"`
}

func getAvailableLun(scsiDevices scsiDevices) (int32, error) {
	takenLuns := []int32{}
	for _, d := range scsiDevices {
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
func (c *AzureSnapshotConnection) listScsiDevices() ([]scsiDeviceInfo, error) {
	cmd, err := c.localConn.RunCommand("lsscsi --brief")
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

// https://learn.microsoft.com/en-us/azure/virtual-machines/linux/azure-to-guest-disk-mapping
// for more information. we want to find the LUNs of the data disks and their mount location
func (c *AzureSnapshotConnection) listBlockDevices() (*blockDevices, error) {
	cmd, err := c.localConn.RunCommand("lsblk -f --json")
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
	blockEntries := &blockDevices{}
	if err := json.Unmarshal(data, blockEntries); err != nil {
		return nil, err
	}
	return blockEntries, nil
}

func filterScsiDevices(scsiDevices scsiDevices, lun int32) []scsiDeviceInfo {
	matching := []scsiDeviceInfo{}
	for _, d := range scsiDevices {
		if d.Lun == lun {
			matching = append(matching, d)
		}
	}

	return matching
}

// there can be multiple devices mounted at the same LUN. the Azure API only provides
// the LUN so we need to find all the blocks, mounted at that LUN. then we find the first one
// that has no mounted partitions and use that as the target device. this is a best-effort approach
func findMatchingDeviceByBlock(scsiDevices scsiDevices, blockDevices *blockDevices) (string, error) {
	matchingBlocks := []blockDevice{}
	for _, device := range scsiDevices {
		for _, block := range blockDevices.BlockDevices {
			devName := "/dev/" + block.Name
			if devName == device.VolumePath {
				matchingBlocks = append(matchingBlocks, block)
			}
		}
	}

	if len(matchingBlocks) == 0 {
		return "", errors.New("no matching blocks found")
	}

	var target string
	for _, b := range matchingBlocks {
		log.Debug().Str("name", b.Name).Msg("azure snapshot> checking block")
		mounted := false
		for _, ch := range b.Children {
			if len(ch.Mountpoint) > 0 && ch.Mountpoint[0] != "" {
				log.Debug().Str("name", ch.Name).Msg("azure snapshot> has mounted partitons, skipping")
				mounted = true
			}
			if !mounted {
				target = "/dev/" + b.Name
			}
		}
	}

	return target, nil
}

// parses the output from running 'lsscsi --brief' and gets the device info, the output looks like this:
// [0:0:0:0]    /dev/sda
// [1:0:0:0]    /dev/sdb
func parseLsscsiOutput(output string) (scsiDevices, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	mountedDevices := []scsiDeviceInfo{}
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
		mountedDevices = append(mountedDevices, scsiDeviceInfo{Lun: int32(lunInt), VolumePath: path})
	}

	return mountedDevices, nil
}
