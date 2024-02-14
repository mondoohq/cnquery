// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azureinstancesnapshot

import (
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/mrn"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/azure/connection/auth"
	"go.mondoo.com/cnquery/v10/providers/azure/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/connection/fs"
	"go.mondoo.com/cnquery/v10/providers/os/connection/local"
	"go.mondoo.com/cnquery/v10/providers/os/connection/snapshot"
	"go.mondoo.com/cnquery/v10/providers/os/detector"
	"go.mondoo.com/cnquery/v10/providers/os/id/azcompute"
	"go.mondoo.com/cnquery/v10/providers/os/id/ids"
)

const (
	SnapshotConnectionType shared.ConnectionType = "azure-snapshot"
	DiskTargetType         string                = "disk"
	SnapshotTargetType     string                = "snapshot"
	InstanceTargetType     string                = "instance"
	SkipCleanup            string                = "skip-snapshot-cleanup"
	SkipSetup              string                = "skip-snapshot-setup"
	Lun                    string                = "lun"
)

// the instance from which we're performing the scan
type azureScannerInstance struct {
	instanceInfo
}

type assetInfo struct {
	assetName  string
	platformId string
}

type scanTarget struct {
	TargetType     string
	Target         string
	ResourceGroup  string
	SubscriptionId string
}

type mountInfo struct {
	deviceName string
}

type mountedDiskInfo struct {
	diskId   string
	diskName string
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
			TargetType:     conf.Options["type"],
			Target:         conf.Options["target"],
			ResourceGroup:  scanner.resourceGroup,
			SubscriptionId: scanner.subscriptionId,
		}, nil
	}
	return scanTarget{
		TargetType:     conf.Options["type"],
		Target:         id.Name,
		ResourceGroup:  id.ResourceGroupName,
		SubscriptionId: id.SubscriptionID,
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
	// local connection is required here to run lsscsi and lsblk to identify where the mounted disk is
	localConn := local.NewConnection(id, conf, asset)

	// 1. check if we run on an azure instance
	scanner, err := determineScannerInstanceInfo(localConn, token)
	if err != nil {
		return nil, err
	}

	// 2. determine the target
	target, err := ParseTarget(conf, scanner)
	if err != nil {
		return nil, err
	}

	sc, err := NewSnapshotCreator(token, scanner.subscriptionId)
	if err != nil {
		return nil, err
	}

	c := &AzureSnapshotConnection{
		opts:            conf.Options,
		snapshotCreator: sc,
		scanner:         *scanner,
		identifier:      conf.PlatformId,
		localConn:       localConn,
	}

	var lun int32
	// 3. we either clone the target disk/snapshot and mount it
	// or we skip the setup and expect the disk to be already attached
	if !c.skipSetup() {
		scsiDevices, err := c.listScsiDevices()
		if err != nil {
			c.Close()
			return nil, err
		}
		lun, err = getAvailableLun(scsiDevices)
		if err != nil {
			c.Close()
			return nil, err
		}
		diskInfo, ai, err := c.setupDiskAndMount(target, lun)
		if err != nil {
			c.Close()
			return nil, err
		}
		asset.Name = ai.assetName
		conf.PlatformId = ai.platformId
		c.mountedDiskInfo = diskInfo
	} else {
		log.Debug().Msg("skipping snapshot setup, expect that disk is already attached")
		if c.opts[Lun] == "" {
			return nil, errors.New("lun is required to hint where the target disk is located")
		}
		lunOpt, err := strconv.Atoi(c.opts[Lun])
		if err != nil {
			return nil, errors.Wrap(err, "could not parse lun")
		}
		lun = int32(lunOpt)
		asset.Name = target.Target
	}

	// 4. once mounted (either by the connection or from the outside), identify the disk by the provided LUN
	mi, err := c.identifyDisk(lun)
	if err != nil {
		c.Close()
		return nil, err
	}
	c.mountInfo = mi

	// 5. mount volume
	shell := []string{"sh", "-c"}
	volumeMounter := snapshot.NewVolumeMounter(shell)
	volumeMounter.VolumeAttachmentLoc = c.mountInfo.deviceName
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

var _ plugin.Closer = (*AzureSnapshotConnection)(nil)

type AzureSnapshotConnection struct {
	*fs.FileSystemConnection
	opts            map[string]string
	volumeMounter   *snapshot.VolumeMounter
	snapshotCreator *snapshotCreator
	scanner         azureScannerInstance
	mountInfo       mountInfo
	// only set if the connection mounts the disk. used for cleanup
	mountedDiskInfo mountedDiskInfo
	identifier      string
	// used on the target VM to run commands, related to finding the target disk by LUN
	localConn *local.LocalConnection
}

func (c *AzureSnapshotConnection) Close() {
	log.Debug().Msg("closing azure snapshot connection")
	if c == nil {
		return
	}

	if c.volumeMounter != nil {
		err := c.volumeMounter.UnmountVolumeFromInstance()
		if err != nil {
			log.Error().Err(err).Msg("unable to unmount volume")
		}
	}
	if c.skipDiskCleanup() {
		log.Debug().Msgf("skipping azure snapshot cleanup, %s flag is set to true", SkipCleanup)
	} else if c.snapshotCreator != nil {
		if c.mountedDiskInfo.diskName != "" {
			err := c.snapshotCreator.detachDisk(c.mountedDiskInfo.diskName, c.scanner.instanceInfo)
			if err != nil {
				log.Error().Err(err).Msg("unable to detach volume")
			}
		}

		if c.mountedDiskInfo.diskName != "" {
			err := c.snapshotCreator.deleteCreatedDisk(c.scanner.resourceGroup, c.mountedDiskInfo.diskName)
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

func (c *AzureSnapshotConnection) skipDiskCleanup() bool {
	return c.opts[SkipCleanup] == "true"
}

func (c *AzureSnapshotConnection) skipSetup() bool {
	return c.opts[SkipSetup] == "true"
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