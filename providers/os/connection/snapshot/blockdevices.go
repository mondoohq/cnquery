// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
)

type blockDevices struct {
	BlockDevices []blockDevice `json:"blockDevices,omitempty"`
}

type blockDevice struct {
	Name       string        `json:"name,omitempty"`
	FsType     string        `json:"fstype,omitempty"`
	Label      string        `json:"label,omitempty"`
	Uuid       string        `json:"uuid,omitempty"`
	MountPoint string        `json:"mountpoint,omitempty"`
	Children   []blockDevice `json:"children,omitempty"`
}

type fsInfo struct {
	name   string
	fstype string
}

func (blockEntries blockDevices) GetRootBlockEntry() (*fsInfo, error) {
	log.Debug().Msg("get root block entry")
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		for i := range d.Children {
			entry := d.Children[i]
			if entry.IsNoBootVolume() {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries blockDevices) GetBlockEntryByName(name string) (*fsInfo, error) {
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
			if entry.IsNoBootVolumeAndUnmounted() {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries blockDevices) GetUnnamedBlockEntry() (*fsInfo, error) {
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

func (blockEntries blockDevices) GetUnmountedBlockEntry() (*fsInfo, error) {
	log.Debug().Msg("get unmounted block entry")
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		if d.MountPoint != "" { // empty string means it is not mounted
			continue
		}
		for i := range d.Children {
			entry := d.Children[i]
			if entry.IsNoBootVolumeAndUnmounted() {
				devFsName := "/dev/" + entry.Name
				return &fsInfo{name: devFsName, fstype: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (entry blockDevice) IsNoBootVolume() bool {
	return entry.Uuid != "" && entry.FsType != "" && entry.FsType != "vfat" && entry.Label != "EFI" && entry.Label != "boot" && entry.Label != "/"
}

func (entry blockDevice) IsNoBootVolumeAndUnmounted() bool {
	return entry.IsNoBootVolume() && entry.MountPoint == ""
}
