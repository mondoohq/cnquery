// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azureinstancesnapshot

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/mrn"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v9/providers/azure/connection/auth"
	"go.mondoo.com/cnquery/v9/providers/azure/connection/shared"
	"go.mondoo.com/cnquery/v9/providers/os/connection/fs"
	"go.mondoo.com/cnquery/v9/providers/os/connection/local"
	"go.mondoo.com/cnquery/v9/providers/os/connection/snapshot"
	"go.mondoo.com/cnquery/v9/providers/os/detector"
	"go.mondoo.com/cnquery/v9/providers/os/id/azcompute"
	"go.mondoo.com/cnquery/v9/providers/os/id/ids"
)

const (
	SnapshotConnectionType shared.ConnectionType = "azure-snapshot"
	DiskTargetType         string                = "disk"
	SnapshotTargetType     string                = "snapshot"
	InstanceTargetType     string                = "instance"
)

// the instance from which we're performing the scan
type azureScannerInstance struct {
	instanceInfo
}

type scanTarget struct {
	TargetType    string
	Target        string
	ResourceGroup string
}

type mountInfo struct {
	deviceName string
	diskId     string
	diskName   string
}

func determineScannerInstanceInfo(localConn *local.LocalConnection, token azcore.TokenCredential) (*azureScannerInstance, error) {
	pf, detected := detector.DetectOS(localConn)
	if !detected {
		return nil, errors.New("could not detect platform")
	}
	scannerInstanceInfo, err := azcompute.Resolve(localConn, pf)
	if err != nil {
		return nil, errors.Wrap(err, "Azure snapshot provider must run from an Azure VM instance")
	}
	identity, err := scannerInstanceInfo.Identify()
	if err != nil {
		return nil, errors.Wrap(err, "Azure snapshot provider must run from an Azure VM instance")
	}
	instanceID := identity.InstanceID

	// parse the platform id
	// platformid.api.mondoo.app/runtime/azure/subscriptions/f1a2873a-6b27-4097-aa7c-3df51f103e96/resourceGroups/preslav-test-ssh_group/providers/Microsoft.Compute/virtualMachines/preslav-test-ssh
	platformMrn, err := mrn.NewMRN(instanceID)
	if err != nil {
		return nil, err
	}
	subId, err := platformMrn.ResourceID("subscriptions")
	if err != nil {
		return nil, err
	}
	resourceGrp, err := platformMrn.ResourceID("resourceGroups")
	if err != nil {
		return nil, err
	}
	instanceName, err := platformMrn.ResourceID("virtualMachines")
	if err != nil {
		return nil, err
	}

	instanceInfo, err := InstanceInfo(resourceGrp, instanceName, subId, token)
	if err != nil {
		return nil, err
	}
	return &azureScannerInstance{
		instanceInfo: instanceInfo,
	}, nil
}

func ParseTarget(conf *inventory.Config, scanner *azureScannerInstance) (scanTarget, error) {
	target := conf.Options["target"]
	if target == "" {
		return scanTarget{}, errors.New("target is required")
	}
	id, err := arm.ParseResourceID(conf.Options["target"])
	if err != nil {
		log.Debug().Msg("could not parse target as resource id, assuming it's only the resource name")
		return scanTarget{
			TargetType:    conf.Options["type"],
			Target:        conf.Options["target"],
			ResourceGroup: scanner.resourceGroup,
		}, nil
	}
	return scanTarget{
		TargetType:    conf.Options["type"],
		Target:        id.Name,
		ResourceGroup: id.ResourceGroupName,
	}, nil
}

func NewAzureSnapshotConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*AzureSnapshotConnection, error) {
	var cred *vault.Credential
	if len(conf.Credentials) > 0 {
		cred = conf.Credentials[0]
	}
	token, err := auth.GetTokenCredential(cred, conf.Options["tenant-id"], conf.Options["client-id"])
	if err != nil {
		return nil, err
	}
	localConn := local.NewConnection(id, conf, asset)

	// check if we run on an azure instance
	scanner, err := determineScannerInstanceInfo(localConn, token)
	if err != nil {
		return nil, err
	}

	target, err := ParseTarget(conf, scanner)
	if err != nil {
		return nil, err
	}

	// determine the target
	sc, err := NewSnapshotCreator(token, scanner.subscriptionId)
	if err != nil {
		return nil, err
	}

	c := &AzureSnapshotConnection{
		opts:            conf.Options,
		snapshotCreator: sc,
		scanner:         *scanner,
		identifier:      conf.PlatformId,
	}

	// setup disk image so and attach it to the instance
	mi := mountInfo{}

	diskName := "cnspec-" + target.TargetType + "-snapshot-" + time.Now().Format("2006-01-02t15-04-05z00-00")
	switch target.TargetType {
	case InstanceTargetType:
		instanceInfo, err := sc.instanceInfo(target.ResourceGroup, target.Target)
		if err != nil {
			return nil, err
		}
		if instanceInfo.bootDiskId == "" {
			return nil, fmt.Errorf("could not find boot disk for instance %s", target.Target)
		}

		log.Debug().Str("boot disk", instanceInfo.bootDiskId).Msg("found boot disk for instance, cloning")
		disk, err := sc.cloneDisk(instanceInfo.bootDiskId, scanner.resourceGroup, diskName, scanner.location, scanner.vm.Zones)
		if err != nil {
			log.Error().Err(err).Msg("could not complete disk cloning")
			return nil, errors.Wrap(err, "could not complete disk cloning")
		}
		log.Debug().Str("disk", *disk.ID).Msg("cloned disk from instance boot disk")
		mi.diskId = *disk.ID
		mi.diskName = *disk.Name
		asset.Name = instanceInfo.instanceName
		conf.PlatformId = azcompute.MondooAzureInstanceID(*instanceInfo.vm.ID)
	case SnapshotTargetType:
		snapshotInfo, err := sc.snapshotInfo(target.ResourceGroup, target.Target)
		if err != nil {
			return nil, err
		}

		disk, err := sc.createSnapshotDisk(snapshotInfo.snapshotId, scanner.resourceGroup, diskName, scanner.location, scanner.vm.Zones)
		if err != nil {
			log.Error().Err(err).Msg("could not complete snapshot disk creation")
			return nil, errors.Wrap(err, "could not create disk from snapshot")
		}
		log.Debug().Str("disk", *disk.ID).Msg("created disk from snapshot")
		mi.diskId = *disk.ID
		mi.diskName = *disk.Name
		asset.Name = target.Target
		conf.PlatformId = SnapshotPlatformMrn(snapshotInfo.snapshotId)
	case DiskTargetType:
		diskInfo, err := sc.diskInfo(target.ResourceGroup, target.Target)
		if err != nil {
			return nil, err
		}

		disk, err := sc.cloneDisk(diskInfo.diskId, scanner.resourceGroup, diskName, scanner.location, scanner.vm.Zones)
		if err != nil {
			log.Error().Err(err).Msg("could not complete disk cloning")
			return nil, errors.Wrap(err, "could not complete disk cloning")
		}
		log.Debug().Str("disk", *disk.ID).Msg("cloned disk from target disk")
		mi.diskId = *disk.ID
		mi.diskName = *disk.Name
		asset.Name = diskInfo.diskName
		conf.PlatformId = DiskPlatformMrn(diskInfo.diskId)
	default:
		return nil, errors.New("invalid target type")
	}

	// fetch the mounted devices. we want to find an available LUN to mount the disk at
	mountedDevices, err := getMountedDevices(localConn)
	if err != nil {
		return nil, err
	}
	lun, err := scanner.getAvailableLun(mountedDevices)
	if err != nil {
		return nil, err
	}
	err = sc.attachDisk(scanner.instanceInfo, mi.diskName, mi.diskId, lun)
	if err != nil {
		c.Close()
		return nil, err
	}

	// refetch the mounted devices, we now are looking for the specific LUN that we just attached.
	// we don't know from the Azure API where it will be mounted, we need to look it up
	mountedDevices, err = getMountedDevices(localConn)
	if err != nil {
		c.Close()
		return nil, err
	}
	matchingDevice, err := getMatchingDevice(mountedDevices, lun)
	if err != nil {
		c.Close()
		return nil, err
	}
	mi.deviceName = matchingDevice.VolumePath

	// mount volume
	shell := []string{"sh", "-c"}
	volumeMounter := snapshot.NewVolumeMounter(shell)
	volumeMounter.VolumeAttachmentLoc = mi.deviceName
	err = volumeMounter.Mount()
	if err != nil {
		log.Error().Err(err).Msg("unable to complete mount step")
		c.Close()
		return nil, err
	}

	conf.Options["path"] = volumeMounter.ScanDir
	// create and initialize fs provider
	fsConn, err := fs.NewConnection(id, &inventory.Config{
		Path:       volumeMounter.ScanDir,
		PlatformId: conf.PlatformId,
		Options:    conf.Options,
		Type:       conf.Type,
		Record:     conf.Record,
	}, asset)
	if err != nil {
		c.Close()
		return nil, err
	}

	c.FileSystemConnection = fsConn
	c.mountInfo = mi
	c.volumeMounter = volumeMounter

	var ok bool
	asset.IdDetector = []string{ids.IdDetector_Hostname}
	asset.Platform, ok = detector.DetectOS(fsConn)
	if !ok {
		c.Close()
		return nil, errors.New("failed to detect OS")
	}
	asset.Id = conf.Type
	asset.Platform.Kind = c.Kind()
	asset.Platform.Runtime = c.Runtime()
	return c, nil
}

type AzureSnapshotConnection struct {
	*fs.FileSystemConnection
	opts            map[string]string
	volumeMounter   *snapshot.VolumeMounter
	snapshotCreator *snapshotCreator
	scanner         azureScannerInstance
	mountInfo       mountInfo
	identifier      string
}

func (c *AzureSnapshotConnection) Close() {
	log.Debug().Msg("closing azure snapshot connection")
	if c == nil {
		return
	}

	if c.opts != nil {
		if c.opts[snapshot.NoSetup] == "true" {
			return
		}
	}

	if c.volumeMounter != nil {
		err := c.volumeMounter.UnmountVolumeFromInstance()
		if err != nil {
			log.Error().Err(err).Msg("unable to unmount volume")
		}
	}

	if c.snapshotCreator != nil {
		if c.mountInfo.diskName != "" {
			err := c.snapshotCreator.detachDisk(c.mountInfo.diskName, c.scanner.instanceInfo)
			if err != nil {
				log.Error().Err(err).Msg("unable to detach volume")
			}
		}

		if c.mountInfo.diskName != "" {
			err := c.snapshotCreator.deleteCreatedDisk(c.scanner.resourceGroup, c.mountInfo.diskName)
			if err != nil {
				log.Error().Err(err).Msg("could not delete created disk")
			}
		}
	}

	if c.volumeMounter != nil {
		err := c.volumeMounter.RemoveTempScanDir()
		if err != nil {
			log.Error().Err(err).Msg("unable to remove dir")
		}
	}
}

func (c *AzureSnapshotConnection) Kind() string {
	return "api"
}

func (c *AzureSnapshotConnection) Runtime() string {
	return "azure-vm"
}

func (c *AzureSnapshotConnection) Identifier() (string, error) {
	return c.identifier, nil
}

func (c *AzureSnapshotConnection) Type() shared.ConnectionType {
	return SnapshotConnectionType
}

func (c *AzureSnapshotConnection) Config() *inventory.Config {
	return c.FileSystemConnection.Conf
}
