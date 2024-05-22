// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog/log"
)

type BlockDevices struct {
	BlockDevices []BlockDevice `json:"blockDevices,omitempty"`
}

type BlockDevice struct {
	Name       string        `json:"name,omitempty"`
	FsType     string        `json:"FsType,omitempty"`
	Label      string        `json:"label,omitempty"`
	Uuid       string        `json:"uuid,omitempty"`
	MountPoint string        `json:"mountpoint,omitempty"`
	Children   []BlockDevice `json:"children,omitempty"`
	FsUse      string        `json:"fsuse%,omitempty"`
}

type fsInfo struct {
	Name   string
	FsType string
}

func (cmdRunner *LocalCommandRunner) GetBlockDevices() (*BlockDevices, error) {
	cmd, err := cmdRunner.RunCommand("lsblk -f --json")
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		outErr, err := io.ReadAll(cmd.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to run lsblk: %s", outErr)
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	blockEntries := &BlockDevices{}
	if err := json.Unmarshal(data, blockEntries); err != nil {
		return nil, err
	}
	return blockEntries, nil
}

func (blockEntries BlockDevices) GetRootBlockEntry() (*fsInfo, error) {
	log.Debug().Msg("get root block entry")
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		for i := range d.Children {
			entry := d.Children[i]
			if entry.IsNoBootVolume() {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{Name: devFsName, FsType: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries BlockDevices) GetBlockEntryByName(name string) (*fsInfo, error) {
	log.Debug().Str("name", name).Msg("get matching block entry")
	var secondName string
	if strings.HasPrefix(name, "/dev/sd") {
		// sdh and xvdh are interchangeable
		end := strings.TrimPrefix(name, "/dev/sd")
		secondName = "/dev/xvd" + end
	}
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		fullDeviceName := "/dev/" + d.Name
		if name != fullDeviceName { // check if the device name matches
			if secondName == "" {
				continue
			}
			if secondName != fullDeviceName { // check if the device name matches the second name option (sdh and xvdh are interchangeable)
				continue
			}
		}
		log.Debug().Msg("found match")
		for i := range d.Children {
			entry := d.Children[i]
			if entry.IsNotBootOrRootVolumeAndUnmounted() {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{Name: devFsName, FsType: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries BlockDevices) GetUnnamedBlockEntry() (*fsInfo, error) {
	fsInfo, err := blockEntries.GetUnmountedBlockEntry()
	if err == nil && fsInfo != nil {
		return fsInfo, nil
	} else {
		// if we get here, there was no non-root, non-mounted volume on the instance
		// this is expected in the "no setup" case where we start an instance with the target
		// volume attached and only that volume attached
		fsInfo, err = blockEntries.GetRootBlockEntry()
		if err == nil && fsInfo != nil {
			return fsInfo, nil
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries BlockDevices) GetUnmountedBlockEntry() (*fsInfo, error) {
	log.Debug().Msg("get unmounted block entry")
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		if d.MountPoint != "" { // empty string means it is not mounted
			continue
		}
		if fsinfo := findVolume(d.Children); fsinfo != nil {
			return fsinfo, nil
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func findVolume(children []BlockDevice) *fsInfo {
	var fs *fsInfo
	for i := range children {
		entry := children[i]
		if entry.IsNotBootOrRootVolumeAndUnmounted() {
			// we are NOT searching for the root volume here, so we can exclude the "sda" and "xvda" volumes
			devFsName := "/dev/" + entry.Name
			fs = &fsInfo{Name: devFsName, FsType: entry.FsType}
		}
	}
	return fs
}

func (entry BlockDevice) IsNoBootVolume() bool {
	return entry.Uuid != "" && entry.FsType != "" && entry.FsType != "vfat" && entry.Label != "EFI" && entry.Label != "boot"
}

func (entry BlockDevice) IsRootVolume() bool {
	return strings.Contains(entry.Name, "sda") || strings.Contains(entry.Name, "xvda") || strings.Contains(entry.Name, "nvme0n1")
}

func (entry BlockDevice) IsNotBootOrRootVolumeAndUnmounted() bool {
	return entry.IsNoBootVolume() && entry.MountPoint == "" && !entry.IsRootVolume()
}
