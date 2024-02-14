// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azureinstancesnapshot

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers/os/id/azcompute"
)

func (c *AzureSnapshotConnection) identifyDisk(lun int32) (mountInfo, error) {
	scsiDevices, err := c.listScsiDevices()
	if err != nil {
		return mountInfo{}, err
	}

	// only interested in the scsi devices that match the provided LUN
	filteredScsiDevices := filterScsiDevices(scsiDevices, lun)
	if len(filteredScsiDevices) == 0 {
		return mountInfo{}, errors.New("no matching scsi devices found")
	}

	// if we have exactly one device present at the LUN we can directly point the volume mounter towards it
	if len(filteredScsiDevices) == 1 {
		return mountInfo{deviceName: filteredScsiDevices[0].VolumePath}, nil
	}

	blockDevices, err := c.listBlockDevices()
	if err != nil {
		return mountInfo{}, err
	}
	target, err := findMatchingDeviceByBlock(filteredScsiDevices, blockDevices)
	if err != nil {
		return mountInfo{}, err
	}
	return mountInfo{deviceName: target}, nil
}

func (c *AzureSnapshotConnection) setupDiskAndMount(target scanTarget, lun int32) (mountedDiskInfo, assetInfo, error) {
	mi, ai, err := c.setupDisk(target)
	if err != nil {
		return mountedDiskInfo{}, assetInfo{}, err
	}
	err = c.snapshotCreator.attachDisk(c.scanner.instanceInfo, mi.diskName, mi.diskId, lun)
	if err != nil {
		return mountedDiskInfo{}, assetInfo{}, err
	}

	return mi, ai, nil
}

func (c *AzureSnapshotConnection) setupDisk(target scanTarget) (mountedDiskInfo, assetInfo, error) {
	if c.scanner.instanceInfo == nil {
		return mountedDiskInfo{}, assetInfo{}, errors.New("cannot setup disk, instance info not found")
	}
	mi := mountedDiskInfo{}
	ai := assetInfo{}
	h := sha256.New()
	now := time.Now()
	// ensure no name collisions if performing multiple snapshot scans at once
	h.Write([]byte(target.Target))
	h.Write([]byte(target.TargetType))
	h.Write([]byte(target.ResourceGroup))
	h.Write([]byte(target.SubscriptionId))
	h.Write([]byte(now.Format("2006-01-02t15-04-05z00-00")))

	diskHash := fmt.Sprintf("%x", h.Sum(nil))
	diskName := fmt.Sprintf("mondoo-snapshot-%s-%s", diskHash[:8], now.Format("2006-01-02t15-04-05z00-00"))
	switch target.TargetType {
	case InstanceTargetType:
		instanceInfo, err := c.snapshotCreator.instanceInfo(target.ResourceGroup, target.Target)
		if err != nil {
			return mountedDiskInfo{}, assetInfo{}, err
		}
		if instanceInfo.bootDiskId == "" {
			return mountedDiskInfo{}, assetInfo{}, fmt.Errorf("could not find boot disk for instance %s", target.Target)
		}

		log.Debug().Str("boot disk", instanceInfo.bootDiskId).Msg("found boot disk for instance, cloning")
		disk, err := c.snapshotCreator.cloneDisk(instanceInfo.bootDiskId, c.scanner.resourceGroup, diskName, c.scanner.instanceInfo.location, c.scanner.instanceInfo.vm.Zones)
		if err != nil {
			log.Error().Err(err).Msg("could not complete disk cloning")
			return mountedDiskInfo{}, assetInfo{}, errors.Wrap(err, "could not complete disk cloning")
		}
		log.Debug().Str("disk", *disk.ID).Msg("cloned disk from instance boot disk")
		mi.diskId = *disk.ID
		mi.diskName = *disk.Name
		ai.assetName = instanceInfo.instanceName
		ai.platformId = azcompute.MondooAzureInstanceID(*instanceInfo.vm.ID)
	case SnapshotTargetType:
		snapshotInfo, err := c.snapshotCreator.snapshotInfo(target.ResourceGroup, target.Target)
		if err != nil {
			return mountedDiskInfo{}, assetInfo{}, err
		}

		disk, err := c.snapshotCreator.createSnapshotDisk(snapshotInfo.snapshotId, c.scanner.resourceGroup, diskName, c.scanner.instanceInfo.location, c.scanner.instanceInfo.vm.Zones)
		if err != nil {
			log.Error().Err(err).Msg("could not complete snapshot disk creation")
			return mountedDiskInfo{}, assetInfo{}, errors.Wrap(err, "could not create disk from snapshot")
		}
		log.Debug().Str("disk", *disk.ID).Msg("created disk from snapshot")
		mi.diskId = *disk.ID
		mi.diskName = *disk.Name
		ai.assetName = target.Target
		ai.platformId = SnapshotPlatformMrn(snapshotInfo.snapshotId)
	case DiskTargetType:
		diskInfo, err := c.snapshotCreator.diskInfo(target.ResourceGroup, target.Target)
		if err != nil {
			return mountedDiskInfo{}, assetInfo{}, err
		}

		disk, err := c.snapshotCreator.cloneDisk(diskInfo.diskId, c.scanner.resourceGroup, diskName, c.scanner.instanceInfo.location, c.scanner.instanceInfo.vm.Zones)
		if err != nil {
			log.Error().Err(err).Msg("could not complete disk cloning")
			return mountedDiskInfo{}, assetInfo{}, errors.Wrap(err, "could not complete disk cloning")
		}
		log.Debug().Str("disk", *disk.ID).Msg("cloned disk from target disk")
		mi.diskId = *disk.ID
		mi.diskName = *disk.Name
		ai.assetName = diskInfo.diskName
		ai.platformId = DiskPlatformMrn(diskInfo.diskId)
	default:
		return mountedDiskInfo{}, assetInfo{}, errors.New("invalid target type")
	}

	return mi, ai, nil
}
