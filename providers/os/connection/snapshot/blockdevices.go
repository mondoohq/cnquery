// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package snapshot

import (
	"encoding/json"
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

	blockEntries.findAliases()

	return blockEntries, nil
}

func (blockEntries *BlockDevices) findAliases() {
	paths := []string{"/dev", "/dev/disk/by-id"}
	for _, path := range paths {
		entries, err := os.ReadDir(path)
		if err != nil {
			log.Warn().Err(err).Msgf("Can't read %s directory", path)
			return
		}

		for _, entry := range entries {
			if entry.Type().Type() != os.ModeSymlink {
				continue
			}

			path := fmt.Sprintf("%s/%s", path, entry.Name())
			target, err := os.Readlink(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("Can't read link target")
				continue
			}

			parts := strings.Split(target, "/")
			if len(parts) == 0 {
				continue
			}
			target = parts[len(parts)-1]
			blockEntries.findAlias(target, path)
		}
	}
}

// Searches for a device by name
func (blockEntries BlockDevices) FindDevice(requested string) (BlockDevice, error) {
	log.Debug().Str("device", requested).Msg("searching for device")

	devices := blockEntries.BlockDevices
	if len(devices) == 0 {
		return BlockDevice{}, fmt.Errorf("no block devices found")
	}

	requestedName := strings.TrimPrefix(requested, "/dev/")
	lmsCache := map[string]int{}
	bestMatch := struct {
		Device BlockDevice
		Lms    int
	}{
		Device: BlockDevice{},
		Lms:    0,
	}

	for _, d := range devices {
		log.Debug().
			Str("name", d.Name).
			Strs("aliases", d.Aliases).
			Msg("checking device")
		if d.Name == requestedName {
			return d, nil
		}

		lms := longestMatchingSuffix(requested, d.Name)
		for _, alias := range d.Aliases {
			aliasLms := longestMatchingSuffix(requested, alias)
			if aliasLms > lms {
				lms = aliasLms
				lmsCache[d.Name] = aliasLms
			}
		}

		if lms > bestMatch.Lms {
			bestMatch.Device = d
			bestMatch.Lms = lms
		}
	}

	if bestMatch.Lms > 0 {
		return bestMatch.Device, nil
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
		if partition.FsType == "" {
			log.Debug().Str("name", partition.Name).Msg("skipping partition without filesystem type")
			return false
		}
		if includeAll {
			return !partition.isMounted()
		}

		return partition.isNoBootVolumeAndUnmounted()
	}

	partitions := []*PartitionInfo{}
	for _, partition := range blockDevices {
		log.Debug().Str("name", partition.Name).Int64("size", int64(partition.Size)).Msg("checking partition")
		if filter(partition) {
			log.Debug().Str("name", partition.Name).Msg("found suitable partition")
			devFsName := "/dev/" + partition.Name
			partitions = append(partitions, &PartitionInfo{
				Name: devFsName, FsType: partition.FsType,
				Label: partition.Label, Uuid: partition.Uuid,
			})
		} else {
			log.Debug().
				Str("name", partition.Name).
				Str("fs_type", partition.FsType).
				Str("mountpoint", partition.MountPoint).
				Msg("skipping partition, because the filter did not match")
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

func (blockEntries *BlockDevices) findAlias(alias, path string) {
	for i := range blockEntries.BlockDevices {
		device := blockEntries.BlockDevices[i]
		if alias == device.Name {
			log.Debug().
				Str("alias", alias).
				Str("path", path).
				Str("name", device.Name).
				Msg("found alias")
			device.Aliases = append(device.Aliases, path)
			blockEntries.BlockDevices[i] = device
			return
		}
	}
}

// longestMatchingSuffix returns the length of the longest common suffix of two strings
// and caches the result (lengths of the matching suffix) for future calls with the same string
func longestMatchingSuffix(s1, s2 string) int {
	n1 := len(s1)
	n2 := len(s2)

	// Start from the end of both strings
	i := 0
	for i < int(math.Min(float64(n1), float64(n2))) && s1[n1-i-1] == s2[n2-i-1] {
		i++
	}

	return i
}
