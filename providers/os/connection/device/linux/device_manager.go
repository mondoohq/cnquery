// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package linux

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
	"go.mondoo.com/cnquery/v11/providers/os/fs"
	"go.mondoo.com/cnquery/v11/providers/os/resources"
	"k8s.io/utils/ptr"
)

const (
	LunOption                   = "lun"
	LunsOption                  = "luns"
	DeviceName                  = "device-name"
	DeviceNames                 = "device-names"
	MountAllPartitions          = "mount-all-partitions"
	IncludeMounted              = "include-mounted"
	SkipAttemptExpandPartitions = "skip-attempt-expand-partitions"
)

type LinuxDeviceManager struct {
	volumeMounter snapshot.VolumeMounter
	opts          map[string]string
}

func NewLinuxDeviceManager(shell []string, opts map[string]string) (*LinuxDeviceManager, error) {
	if err := validateOpts(opts); err != nil {
		return nil, err
	}

	return &LinuxDeviceManager{
		volumeMounter: snapshot.NewVolumeMounter(shell),
		opts:          opts,
	}, nil
}

func (d *LinuxDeviceManager) Name() string {
	return "linux"
}

func (d *LinuxDeviceManager) IdentifyMountTargets(opts map[string]string) ([]*snapshot.PartitionInfo, error) {
	if err := validateOpts(opts); err != nil {
		return nil, err
	}

	deviceNames := []string{}
	luns, err := getLunsFromOpts(opts)
	if err != nil {
		return nil, err
	}
	for _, l := range luns {
		devices, err := d.identifyDeviceViaLun(l)
		if err != nil {
			return nil, err
		}
		for _, device := range devices {
			deviceNames = append(deviceNames, device.Name)
		}
	}

	if opts[DeviceNames] != "" {
		deviceNames = strings.Split(opts[DeviceNames], ",")
	}
	if opts[DeviceName] != "" {
		deviceNames = append(deviceNames, opts[DeviceName])
	}

	var partitions []*snapshot.PartitionInfo
	var errs []error
	for _, deviceName := range deviceNames {
		partitionsForDevice, err := d.identifyViaDeviceName(deviceName, opts[MountAllPartitions] == "true", opts[IncludeMounted] == "true")
		if err != nil {
			errs = append(errs, err)
			continue
		}
		partitions = append(partitions, partitionsForDevice...)
	}

	if len(partitions) == 0 {
		errs = append(errs, errors.New("no partitions found"))
		return partitions, errors.Join(errs...)
	}

	if opts[SkipAttemptExpandPartitions] == "true" {
		return partitions, errors.Join(errs...)
	}

	partitions, err = d.attemptExpandPartitions(partitions)
	errs = append(errs, err)

	return partitions, errors.Join(errs...)
}

func (d *LinuxDeviceManager) attemptExpandPartitions(partitions []*snapshot.PartitionInfo) ([]*snapshot.PartitionInfo, error) {
	log.Debug().Msg("attempting to expand partitions infos")

	fstabEntries, err := d.hintFSTypes(partitions)
	if err != nil {
		log.Warn().Err(err).Msg("could not find fstab")
		return partitions, nil
	}
	log.Debug().Any("fstab", fstabEntries).
		Msg("fstab entries found")

	partitions, err = d.mountWithFstab(partitions, fstabEntries)
	if err != nil {
		log.Error().Err(err).Msg("unable to mount partitions with fstab")
		d.UnmountAndClose()
		return partitions, nil
	}

	return partitions, nil
}

func (d *LinuxDeviceManager) hintFSTypes(partitions []*snapshot.PartitionInfo) ([]resources.FstabEntry, error) {
	for i := range partitions {
		partition := partitions[i]

		dir, err := d.volumeMounter.MountP(&snapshot.MountPartitionDto{PartitionInfo: partition})
		if err != nil {
			continue
		}
		defer func() {
			if err := d.volumeMounter.UmountP(partition); err != nil {
				log.Warn().Err(err).Str("device", partition.Name).Msg("unable to unmount partition")
			}
		}()

		entries, err := d.attemptFindFstab(dir)
		if err != nil {
			return nil, err
		}
		if entries != nil {
			return entries, nil
		}

		if ok := d.attemptFindOSTree(dir, partition); ok {
			log.Debug().Str("device", partition.Name).Msg("ostree found")
			return nil, nil
		}

	}

	return nil, errors.New("fstab not found")
}

func (d *LinuxDeviceManager) attemptFindFstab(dir string) ([]resources.FstabEntry, error) {
	cmd, err := d.volumeMounter.CmdRunner().RunCommand(fmt.Sprintf("find %s -type f -wholename '*/etc/fstab'", dir))
	if err != nil {
		log.Error().Err(err).Msg("error searching for fstab")
		return nil, nil
	}

	out, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		log.Error().Err(err).Msg("error reading find output")
		return nil, nil
	}

	if l := len(strings.Split(string(out), "\n")); l > 2 {
		log.Debug().Bytes("find", out).Msg("fstab not found, too many results")
		return nil, nil
	} else if l < 2 {
		log.Debug().Bytes("find", out).Msg("fstab not found, no results")
		return nil, nil
	}

	mnt, fstab := path.Split(strings.TrimSpace(string(out)))
	fstabFile, err := afero.ReadFile(
		fs.NewMountedFs(mnt),
		path.Base(fstab))
	if err != nil {
		log.Error().Err(err).Msg("error reading fstab")
		return nil, nil
	}

	return resources.ParseFstab(bytes.NewReader(fstabFile))
}

func (d *LinuxDeviceManager) attemptFindOSTree(dir string, partition *snapshot.PartitionInfo) bool {
	log.Debug().Str("device", partition.Name).Msg("attempting to find ostree")

	info, err := os.Stat(path.Join(dir, "ostree"))
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}

		log.Error().Err(err).Str("device", partition.Name).Msg("unable to stat ostree")
		return false
	}

	if !info.IsDir() {
		log.Warn().Str("device", partition.Name).Msg("ostree is not a directory")
		return false
	}

	entries, err := os.ReadDir(path.Join(dir, "ostree"))
	if err != nil {
		log.Error().Err(err).Str("device", partition.Name).Msg("unable to read ostree directory")
		return false
	}

	entries = slices.DeleteFunc(entries, func(entry os.DirEntry) bool {
		if entry.Type().Type() != os.ModeSymlink {
			return true
		}

		trimmed := strings.TrimPrefix(entry.Name(), "boot.")
		if trimmed == entry.Name() {
			return true
		}

		_, err := strconv.Atoi(trimmed)
		return err != nil
	})

	if len(entries) == 0 {
		log.Debug().Str("device", partition.Name).Msg("no ostree entries")
		return false
	}
	if len(entries) > 1 {
		slices.SortFunc(entries, func(a, b os.DirEntry) int {
			aIndex, _ := strconv.Atoi(strings.TrimPrefix(a.Name(), "boot."))
			bIndex, _ := strconv.Atoi(strings.TrimPrefix(b.Name(), "boot."))
			return aIndex - bIndex
		})
	}

	log.Debug().Str("device", partition.Name).Str("boot1", entries[0].Name()).Msg("found ostree deployment")
	boot1, err := os.Readlink(path.Join(dir, "ostree", entries[0].Name()))
	if err != nil {
		log.Error().Err(err).Str("device", partition.Name).Msg("unable to readlink boot.1")
		return false
	}

	matches, err := filepath.Glob(path.Join(dir, "ostree", boot1, "*", "*", "0"))
	if err != nil {
		log.Error().Err(err).Str("device", partition.Name).Msg("unable to glob ostree")
		return false
	}

	if len(matches) == 0 {
		log.Debug().Str("device", partition.Name).Msg("no ostree matches")
		return false
	} else if len(matches) > 1 {
		log.Warn().Str("device", partition.Name).Msg("multiple ostree matches")
	}

	partition.SetBind(strings.TrimPrefix(matches[0], dir))
	return true
}

func pathDepth(path string) int {
	if path == "/" {
		return 0
	}
	return len(strings.Split(strings.Trim(path, "/"), "/"))
}

// mountWithFstab mounts partitions adjusting the mountpoint and mount options according to the discovered fstab entries
func (d *LinuxDeviceManager) mountWithFstab(partitions []*snapshot.PartitionInfo, entries []resources.FstabEntry) ([]*snapshot.PartitionInfo, error) {
	// sort the entries by the length of the mountpoint, so we can mount the top level partitions first
	sort.Slice(entries, func(i, j int) bool {
		return pathDepth(entries[i].Mountpoint) < pathDepth(entries[j].Mountpoint)
	})

	rootScanDir := ""
	for _, entry := range entries {
		for i := range partitions {
			partition := partitions[i]
			mustAppend := false
			if !cmpPartition2Fstab(partition, entry) {
				continue
			}

			log.Debug().
				Str("device", partition.Name).
				Str("guest-mountpoint", entry.Mountpoint).
				Str("host-mountpouint", partition.MountPoint).
				Msg("partition matches fstab entry")

			// if the partition is already mounted
			if partition.MountPoint != "" {
				mountedWithFstab := strings.HasPrefix(partition.MountPoint, rootScanDir)
				// mounted without fstab consideration, unmount it
				if rootScanDir == "" || !mountedWithFstab {
					log.Debug().Str("device", partition.Name).Msg("partition already mounted")
					if err := d.volumeMounter.UmountP(partition); err != nil {
						log.Error().Err(err).Str("device", partition.Name).Msg("unable to unmount partition")
						continue
					}
					partition.MountPoint = ""
				} else if mountedWithFstab { // mounted with fstab, duplicate the partition (probably a subvolume)
					partitionCopy := *partition
					partition = &partitionCopy
					mustAppend = true
				}
			}

			var scanDir *string
			if rootScanDir != "" {
				scanDir = ptr.To(path.Join(rootScanDir, entry.Mountpoint))
			}
			partition.MountOptions = entry.Options

			log.Debug().Str("device", partition.Name).
				Strs("options", partition.MountOptions).
				Any("scan-dir", scanDir).
				Msg("mounting partition as subvolume")
			mnt, err := d.volumeMounter.MountP(&snapshot.MountPartitionDto{
				PartitionInfo: partition,
				ScanDir:       scanDir,
			})
			if err != nil {
				log.Error().Err(err).Str("device", partition.Name).Msg("unable to mount partition")
				return partitions, err
			}

			partition.MountPoint = mnt
			if entry.Mountpoint == "/" {
				rootScanDir = mnt
			}

			if mustAppend {
				partitions = append(partitions, partition)
			} else {
				partitions[i] = partition
			}

			break // partition matched, no need to check the rest
		}
	}
	return partitions, nil
}

func cmpPartition2Fstab(partition *snapshot.PartitionInfo, entry resources.FstabEntry) bool {
	// Edge case: fstab entry is a symlink to a device mapper device (LVM2)
	if strings.HasPrefix(entry.Device, "/dev/mapper/") {
		return entry.Device == partition.Name
	}

	parts := strings.Split(entry.Device, "=")
	if len(parts) != 2 {
		log.Warn().Str("device", entry.Device).Msg("possibly invalid fstab entry, skipping")
		return false
	}

	if parts[1] == "" {
		log.Warn().Str("device", entry.Device).Msg("possibly invalid fstab entry, skipping")
		return false
	}

	switch parts[0] {
	case "UUID":
		return partition.Uuid == parts[1]
	case "LABEL":
		return partition.Label == parts[1]
	case "PARTUUID":
		return partition.PartUuid == parts[1]
	default:
		log.Warn().Str("device", entry.Device).Msg("couldn't identify fstab device")
		return false
	}
}

func (d *LinuxDeviceManager) Mount(pi *snapshot.PartitionInfo) (string, error) {
	return d.volumeMounter.MountP(&snapshot.MountPartitionDto{PartitionInfo: pi})
}

func (d *LinuxDeviceManager) UnmountAndClose() {
	log.Debug().Msg("closing linux device manager")
	if d == nil {
		return
	}

	if d.volumeMounter != nil {
		err := d.volumeMounter.UnmountVolumeFromInstance()
		if err != nil {
			log.Error().Err(err).Msg("unable to unmount volume")
		}
		err = d.volumeMounter.RemoveTempScanDir()
		if err != nil {
			log.Error().Err(err).Msg("unable to remove dir")
		}
	}
}

// validates the options provided to the device manager
// we cannot have both LUN and device name provided, those are mutually exclusive
func validateOpts(opts map[string]string) error {
	// this is needed only for the validation purposes
	deviceNamesPresent := opts[DeviceName] != "" || opts[DeviceNames] != ""
	lunsPresent := opts[LunOption] != "" || opts[LunsOption] != ""
	mountAll := opts[MountAllPartitions] == "true"

	if deviceNamesPresent && lunsPresent {
		return errors.New("both lun and device names provided")
	}

	if !deviceNamesPresent && !lunsPresent {
		return errors.New("either lun or device names must be provided")
	}

	if !deviceNamesPresent && mountAll {
		return errors.New("mount-all-partitions requires device names")
	}

	return nil
}

func getLunsFromOpts(opts map[string]string) ([]int, error) {
	luns := []int{}
	if opts[LunOption] != "" {
		lun, err := strconv.Atoi(opts[LunOption])
		if err != nil {
			return nil, err
		}
		luns = append(luns, lun)
	}
	if opts[LunsOption] != "" {
		vals := strings.Split(opts[LunsOption], ",")
		for _, l := range vals {
			lun, err := strconv.Atoi(l)
			if err != nil {
				return nil, err
			}
			luns = append(luns, lun)
		}
	}
	return luns, nil
}

func (c *LinuxDeviceManager) identifyDeviceViaLun(lun int) ([]snapshot.BlockDevice, error) {
	scsiDevices, err := c.listScsiDevices()
	if err != nil {
		return nil, err
	}

	// only interested in the scsi devices that match the provided LUN
	filteredScsiDevices := filterScsiDevices(scsiDevices, lun)
	if len(filteredScsiDevices) == 0 {
		return nil, errors.New("no matching scsi devices found")
	}
	devices := []snapshot.BlockDevice{}
	for _, device := range filteredScsiDevices {
		d := snapshot.BlockDevice{
			Name: device.VolumePath,
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (c *LinuxDeviceManager) identifyViaDeviceName(deviceName string, mountAll bool, includeMounted bool) ([]*snapshot.PartitionInfo, error) {
	if deviceName == "" {
		log.Warn().Msg("can't identify partition via device name, no device name provided")
		return []*snapshot.PartitionInfo{}, nil
	}

	blockDevices, err := c.volumeMounter.CmdRunner().GetBlockDevices()
	if err != nil {
		return nil, err
	}

	device, err := blockDevices.FindDevice(deviceName)
	if err != nil {
		return nil, err
	}

	if mountAll {
		log.Debug().Str("device", device.Name).Msg("mounting all partitions")
		return device.GetPartitions(true, includeMounted)
	}

	pi, err := device.GetMountablePartition()
	if err != nil {
		return nil, err
	}
	return []*snapshot.PartitionInfo{pi}, nil
}
