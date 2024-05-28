// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

type BlockDevices struct {
	BlockDevices []BlockDevice `json:"blockDevices,omitempty"`
}

type BlockDevice struct {
	Name       string        `json:"name,omitempty"`
	FsType     string        `json:"fstype,omitempty"`
	Label      string        `json:"label,omitempty"`
	Uuid       string        `json:"uuid,omitempty"`
	MountPoint string        `json:"mountpoint,omitempty"`
	Children   []BlockDevice `json:"children,omitempty"`
	Size       int           `json:"size,omitempty"`
}

type PartitionInfo struct {
	Name   string
	FsType string
}

func (cmdRunner *LocalCommandRunner) GetBlockDevices() (*BlockDevices, error) {
	cmd, err := cmdRunner.RunCommand("lsblk -bo NAME,SIZE,FSTYPE,MOUNTPOINT,LABEL,UUID --json")
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

func (blockEntries BlockDevices) GetRootBlockEntry() (*PartitionInfo, error) {
	log.Debug().Msg("get root block entry")
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		for i := range d.Children {
			entry := d.Children[i]
			if entry.IsNoBootVolume() {
				devFsName := "/dev/" + entry.Name
				return &PartitionInfo{Name: devFsName, FsType: entry.FsType}, nil
			}
		}
	}
	return nil, errors.New("target volume not found on instance")
}

// Searches all the partitions in the target device and finds one that can be mounted. It must be unmounted, non-boot partition
// If multiple partitions meet this criteria, the largest one is returned.
// Deprecated: Use GetMountablePartition instead
func (blockEntries BlockDevices) GetMountablePartitionByDevice(device string) (*PartitionInfo, error) {
	log.Debug().Str("device", device).Msg("get partitions for device")
	var block BlockDevice
	partitions := []BlockDevice{}
	var secondName string
	if strings.HasPrefix(device, "/dev/sd") {
		// sdh and xvdh are interchangeable
		end := strings.TrimPrefix(device, "/dev/sd")
		secondName = "/dev/xvd" + end
	}
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		fullDeviceName := "/dev/" + d.Name
		if device != fullDeviceName { // check if the device name matches
			if secondName == "" {
				continue
			}
			if secondName != fullDeviceName { // check if the device name matches the second name option (sdh and xvdh are interchangeable)
				continue
			}
		}
		log.Debug().Str("name", d.Name).Msg("found matching device")
		block = d
		break
	}
	if len(block.Name) == 0 {
		return nil, fmt.Errorf("no block device found with name %s", device)
	}

	for _, partition := range block.Children {
		log.Debug().Str("name", partition.Name).Int("size", partition.Size).Msg("checking partition")
		if partition.IsNotBootOrRootVolumeAndUnmounted() {
			partitions = append(partitions, partition)
		}
	}

	if len(partitions) == 0 {
		return nil, fmt.Errorf("no suitable partitions found on device %s", block.Name)
	}

	// sort the candidates by size, so we can pick the largest one
	sortPartitionsBySize(partitions)

	// return the largest partition. we can extend this to be a parameter in the future
	devFsName := "/dev/" + partitions[0].Name
	return &PartitionInfo{Name: devFsName, FsType: partitions[0].FsType}, nil
}

// Searches for a device by name
func (blockEntries BlockDevices) FindDevice(name string) (BlockDevice, error) {
	log.Debug().Str("device", name).Msg("searching for device")
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
			if secondName == "" || secondName != fullDeviceName {
				continue
			}
		}
		log.Debug().Str("name", d.Name).Msg("found matching device")
		return d, nil
	}

	return BlockDevice{}, fmt.Errorf("no block device found with name %s", name)
}

// Searches all the partitions in the device and finds one that can be mounted. It must be unmounted, non-boot partition
// If multiple partitions meet this criteria, the largest one is returned.
func (device BlockDevice) GetMountablePartition() (*PartitionInfo, error) {
	log.Debug().Str("device", device.Name).Msg("get partitions for device")
	partitions := []BlockDevice{}
	for _, partition := range device.Children {
		log.Debug().Str("name", partition.Name).Int("size", partition.Size).Msg("checking partition")
		if partition.IsNoBootVolumeAndUnmounted() {
			partitions = append(partitions, partition)
		}
	}

	if len(partitions) == 0 {
		return nil, fmt.Errorf("no suitable partitions found on device %s", device.Name)
	}

	// sort the candidates by size, so we can pick the largest one
	sortPartitionsBySize(partitions)

	// return the largest partition. we can extend this to be a parameter in the future
	devFsName := "/dev/" + partitions[0].Name
	return &PartitionInfo{Name: devFsName, FsType: partitions[0].FsType}, nil
}

func sortPartitionsBySize(partitions []BlockDevice) {
	sort.Slice(partitions, func(i, j int) bool {
		return partitions[i].Size > partitions[j].Size
	})
}

func (blockEntries BlockDevices) GetUnnamedBlockEntry() (*PartitionInfo, error) {
	pInfo, err := blockEntries.GetUnmountedBlockEntry()
	if err == nil && pInfo != nil {
		return pInfo, nil
	} else {
		// if we get here, there was no non-root, non-mounted volume on the instance
		// this is expected in the "no setup" case where we start an instance with the target
		// volume attached and only that volume attached
		pInfo, err = blockEntries.GetRootBlockEntry()
		if err == nil && pInfo != nil {
			return pInfo, nil
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func (blockEntries BlockDevices) GetUnmountedBlockEntry() (*PartitionInfo, error) {
	log.Debug().Msg("get unmounted block entry")
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		if d.MountPoint != "" { // empty string means it is not mounted
			continue
		}
		if pInfo := findVolume(d.Children); pInfo != nil {
			return pInfo, nil
		}
	}
	return nil, errors.New("target volume not found on instance")
}

func findVolume(children []BlockDevice) *PartitionInfo {
	var fs *PartitionInfo
	for i := range children {
		entry := children[i]
		if entry.IsNotBootOrRootVolumeAndUnmounted() {
			// we are NOT searching for the root volume here, so we can exclude the "sda" and "xvda" volumes
			devFsName := "/dev/" + entry.Name
			fs = &PartitionInfo{Name: devFsName, FsType: entry.FsType}
		}
	}
	return fs
}

func (entry BlockDevice) IsNoBootVolume() bool {
	return entry.Uuid != "" && entry.FsType != "" && entry.FsType != "vfat" && entry.Label != "EFI" && entry.Label != "boot"
}

func (entry BlockDevice) IsNoBootVolumeAndUnmounted() bool {
	return entry.IsNoBootVolume() && !entry.IsMounted()
}

func (entry BlockDevice) IsRootVolume() bool {
	return strings.Contains(entry.Name, "sda") || strings.Contains(entry.Name, "xvda") || strings.Contains(entry.Name, "nvme0n1")
}

func (entry BlockDevice) IsNotBootOrRootVolumeAndUnmounted() bool {
	return entry.IsNoBootVolumeAndUnmounted() && !entry.IsRootVolume()
}

func (entry BlockDevice) IsMounted() bool {
	return entry.MountPoint != ""
}
