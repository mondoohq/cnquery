// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
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
	Size       Size          `json:"size,omitempty"`

	Aliases []string `json:"-"`
}

type Size int64

func (s *Size) UnmarshalJSON(data []byte) error {
	var size any
	if err := json.Unmarshal(data, &size); err != nil {
		return err
	}
	switch size := size.(type) {
	case string:
		isize, err := strconv.Atoi(size)
		*s = Size(isize)
		return err
	case float64:
		*s = Size(size)
	}
	return nil
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
	blockEntries.FindAliases()

	return blockEntries, nil
}

func (blockEntries *BlockDevices) FindAliases() {
	entries, err := os.ReadDir("/dev")
	if err != nil {
		log.Warn().Err(err).Msg("Can't read /dev directory")
		return
	}

process_symlinks:
	for _, entry := range entries {
		if entry.Type().Type() != os.ModeSymlink {
			continue
		}

		path := fmt.Sprintf("/dev/%s", entry.Name())
		target, err := os.Readlink(path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Can't read link target")
			continue
		}

		targetName := strings.TrimPrefix(target, "/dev/")
		for i := range blockEntries.BlockDevices {
			device := blockEntries.BlockDevices[i]
			if targetName == device.Name {
				device.Aliases = append(device.Aliases, path)
				blockEntries.BlockDevices[i] = device
				continue process_symlinks
			}
		}
	}
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
		log.Debug().Str("name", partition.Name).Int64("size", int64(partition.Size)).Msg("checking partition")
		if partition.IsNotBootOrRootVolumeAndUnmounted() {
			log.Debug().Str("name", partition.Name).Msg("found suitable partition")
			partitions = append(partitions, partition)
		}
	}

	if len(partitions) == 0 {
		return nil, fmt.Errorf("no suitable partitions found on device %s", block.Name)
	}

	// sort the candidates by size, so we can pick the largest one
	sortBlockDevicesBySize(partitions)

	// return the largest partition. we can extend this to be a parameter in the future
	devFsName := "/dev/" + partitions[0].Name
	return &PartitionInfo{Name: devFsName, FsType: partitions[0].FsType}, nil
}

// LongestMatchingSuffix returns the length of the longest common suffix of two strings
// and caches the result (lengths of the matching suffix) for future calls with the same string
func LongestMatchingSuffix(lmsCache map[string]int, s1, s2 string) int {
	if v, ok := lmsCache[s2]; ok {
		return v
	}

	n1 := len(s1)
	n2 := len(s2)

	// Start from the end of both strings
	i := 0
	for i < int(math.Min(float64(n1), float64(n2))) && s1[n1-i-1] == s2[n2-i-1] {
		i++
	}

	lmsCache[s2] = i
	return i
}

// Searches for a device by name
func (blockEntries BlockDevices) FindDevice(requested string) (BlockDevice, error) {
	log.Debug().Str("device", requested).Msg("searching for device")

	requestedName := strings.TrimPrefix(requested, "/dev/")

	lmsCache := map[string]int{}
	// LongestMatchingSuffix returns the length of the longest common suffix of requested and provided string

	sorted := false
	devices := blockEntries.BlockDevices

	// Bubble sort the devices by the longest matching suffix
	// Longest matches will be at the beginning of the slice
	for !sorted {
		sorted = true

		for i := 0; i < len(devices)-1; i++ {
			if devices[i].Name == requestedName {
				return blockEntries.BlockDevices[i], nil
			}

			lms := LongestMatchingSuffix(lmsCache, requested, devices[i].Name)
			for _, alias := range devices[i].Aliases {
				aliasLms := LongestMatchingSuffix(map[string]int{}, requested, alias)
				if aliasLms > lms {
					lms = aliasLms
					lmsCache[devices[i].Name] = aliasLms
				}
			}

			if lms < LongestMatchingSuffix(lmsCache, requested, devices[i+1].Name) {
				devices[i], devices[i+1] = devices[i+1], devices[i]
				sorted = false
			}
		}
	}

	// If the first device has matching suffix, return it
	if LongestMatchingSuffix(lmsCache, requested, devices[0].Name) > 0 {
		return devices[0], nil
	}

	log.Debug().
		Str("device", requested).
		Any("checked_names", lmsCache).
		Msg("no device found")

	return BlockDevice{}, fmt.Errorf("no block device found with name %s", requested)
}

// Searches all the partitions in the device and finds one that can be mounted. It must be unmounted, non-boot partition
func (device BlockDevice) GetMountablePartitions(includeAll bool) ([]*PartitionInfo, error) {
	log.Debug().Str("device", device.Name).Msg("get partitions for device")

	blockDevices := device.Children
	// sort the candidates by size, so we can pick the largest one
	sortBlockDevicesBySize(blockDevices)

	filter := func(partition BlockDevice) bool {
		return partition.IsNoBootVolumeAndUnmounted()
	}
	if includeAll {
		filter = func(partition BlockDevice) bool {
			return !partition.IsMounted()
		}
	}

	partitions := []*PartitionInfo{}
	for _, partition := range blockDevices {
		log.Debug().Str("name", partition.Name).Int64("size", int64(partition.Size)).Msg("checking partition")
		if partition.FsType == "" {
			log.Debug().Str("name", partition.Name).Msg("skipping partition without filesystem type")
			continue
		}
		if filter(partition) {
			log.Debug().Str("name", partition.Name).Msg("found suitable partition")
			devFsName := "/dev/" + partition.Name
			partitions = append(partitions, &PartitionInfo{Name: devFsName, FsType: partition.FsType})
		}
	}

	if len(partitions) == 0 {
		return nil, fmt.Errorf("no suitable partitions found on device %s", device.Name)
	}

	return partitions, nil
}

// If multiple partitions meet this criteria, the largest one is returned.
func (device BlockDevice) GetMountablePartition() (*PartitionInfo, error) {
	// return the largest partition. we can extend this to be a parameter in the future
	partitions, err := device.GetMountablePartitions(false)
	if err != nil {
		return nil, err
	}

	return partitions[0], nil
}

func sortBlockDevicesBySize(partitions []BlockDevice) {
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

func (blockEntries BlockDevices) GetDeviceWithUnmountedPartitions() (BlockDevice, error) {
	log.Debug().Msg("get device with unmounted partitions")
	for i := range blockEntries.BlockDevices {
		d := blockEntries.BlockDevices[i]
		log.Debug().Str("name", d.Name).Interface("children", d.Children).Interface("mountpoint", d.MountPoint).Msg("found block device")
		if d.MountPoint != "" { // empty string means it is not mounted
			continue
		}

		return d, nil
	}
	return BlockDevice{}, errors.New("target block device not found on instance")
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
	candidates := []BlockDevice{}
	for i := range children {
		entry := children[i]
		if entry.IsNotBootOrRootVolumeAndUnmounted() {
			// we are NOT searching for the root volume here, so we can exclude the "sda" and "xvda" volumes
			candidates = append(candidates, entry)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sortBlockDevicesBySize(candidates)
	return &PartitionInfo{Name: "/dev/" + candidates[0].Name, FsType: candidates[0].FsType}
}

func (entry BlockDevice) IsNoBootVolume() bool {
	typ := strings.ToLower(entry.FsType)
	label := strings.ToLower(entry.Label)
	return entry.Uuid != "" && typ != "" && typ != "vfat" && label != "efi" && label != "boot"
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
