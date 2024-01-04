// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azureinstancesnapshot

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/mrn"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v9/providers/azure/connection/auth"
	"go.mondoo.com/cnquery/v9/providers/azure/connection/shared"
	"go.mondoo.com/cnquery/v9/providers/os/connection"
	"go.mondoo.com/cnquery/v9/providers/os/connection/snapshot"
	"go.mondoo.com/cnquery/v9/providers/os/detector"
	"go.mondoo.com/cnquery/v9/providers/os/id/azcompute"
	"go.mondoo.com/cnquery/v9/providers/os/id/ids"
)

type scanTarget struct {
	TargetType   string
	InstanceName string
	SnapshotName string
}

const (
	SnapshotConnectionType shared.ConnectionType = "azure-snapshot"
)

type deviceInfo struct {
	// the LUN number, e.g. 3
	Lun string
	// where the disk is mounted, e.g. /dev/sda
	VolumePath string
}

// the instance from which we're performing the scan
type azureScannerInstance struct {
	instanceInfo
}

type mountInfo struct {
	deviceName string
	diskId     string
	diskName   string
}

func (a *azureScannerInstance) getAvailableLun(mountedDevices []deviceInfo) (int32, error) {
	takenLuns := []string{}
	for _, d := range mountedDevices {
		takenLuns = append(takenLuns, d.Lun)
	}

	availableLuns := []int32{}
	// the available LUNs are 0-63, so we exclude everything thats in takenLuns
	for i := int32(0); i < 64; i++ {
		exists := false
		for _, d := range takenLuns {
			if d == fmt.Sprintf("%d", i) {
				exists = true
				break
			}
		}
		if !exists {
			availableLuns = append(availableLuns, i)
		} else {
			// log just for visibility
			log.Debug().Int32("LUN", i).Msg("azure snapshot> LUN is taken, skipping")
		}
	}
	if len(availableLuns) == 0 {
		return 0, errors.New("no available LUNs to attach disk to")
	}
	return availableLuns[0], nil
}

// https://learn.microsoft.com/en-us/azure/virtual-machines/linux/azure-to-guest-disk-mapping
// for more information. we want to find the LUNs of the data disks and their mount location
func getMountedDevices(localConn *connection.LocalConnection) ([]deviceInfo, error) {
	cmd, err := localConn.RunCommand("lsscsi --brief")
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
	// output looks like this:
	// [0:0:0:0]    /dev/sda
	// [1:0:0:0]    /dev/sdb
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	strData := string(data)
	lines := strings.Split(strings.TrimSpace(strData), "\n")
	mountedDevices := []deviceInfo{}
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
		mountedDevices = append(mountedDevices, deviceInfo{Lun: lun, VolumePath: path})
	}

	return mountedDevices, nil
}

func getMatchingDevice(mountedDevices []deviceInfo, lun int32) (deviceInfo, error) {
	for _, d := range mountedDevices {
		if d.Lun == fmt.Sprintf("%d", lun) {
			return d, nil
		}
	}
	return deviceInfo{}, errors.New("could not find matching device")
}

func determineScannerInstanceInfo(localConn *connection.LocalConnection, token azcore.TokenCredential) (*azureScannerInstance, error) {
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

func ParseTarget(conf *inventory.Config) scanTarget {
	return scanTarget{
		TargetType:   conf.Options["type"],
		InstanceName: conf.Options["instance-name"],
		SnapshotName: conf.Options["snapshot-name"],
	}
}

func NewAzureSnapshotConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*AzureSnapshotConnection, error) {
	target := ParseTarget(conf)

	var cred *vault.Credential
	if len(conf.Credentials) > 0 {
		cred = conf.Credentials[0]
	}
	token, err := auth.GetTokenCredential(cred, conf.Options["tenant-id"], conf.Options["client-id"])
	if err != nil {
		return nil, err
	}
	localConn := connection.NewLocalConnection(id, conf, asset)

	// check if we run on an azure instance
	scanner, err := determineScannerInstanceInfo(localConn, token)
	if err != nil {
		return nil, err
	}

	// determine the target
	sc, err := NewSnapshotCreator(token, scanner.SubscriptionId)
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
	switch target.TargetType {
	case "instance":
		instanceInfo, err := sc.InstanceInfo(scanner.ResourceGroup, target.InstanceName)
		if err != nil {
			return nil, err
		}
		if instanceInfo.BootDiskId == "" {
			return nil, fmt.Errorf("could not find boot disk for instance %s", target.InstanceName)
		}

		log.Debug().Str("boot disk", instanceInfo.BootDiskId).Msg("found boot disk for instance, cloning")
		disk, err := sc.cloneDisk(instanceInfo.BootDiskId, scanner.ResourceGroup, "cnspec-"+target.InstanceName+"-snapshot-"+time.Now().Format("2006-01-02t15-04-05z00-00"), instanceInfo.Location, scanner.Vm.Zones)
		if err != nil {
			log.Error().Err(err).Msg("could not complete disk cloning")
			return nil, errors.Wrap(err, "could not complete disk cloning")
		}
		log.Debug().Str("disk", *disk.ID).Msg("cloned disk from instance boot disk")
		mi.diskId = *disk.ID
		mi.diskName = *disk.Name
		asset.Name = instanceInfo.InstanceName
		conf.PlatformId = azcompute.MondooAzureInstanceID(*instanceInfo.Vm.ID)
	case "snapshot":
		snapshotInfo, err := sc.SnapshotInfo(scanner.ResourceGroup, target.SnapshotName)
		if err != nil {
			return nil, err
		}

		disk, err := sc.createSnapshotDisk(snapshotInfo.SnapshotId, scanner.ResourceGroup, "cnspec-"+target.InstanceName+"-snapshot-"+time.Now().Format("2006-01-02t15-04-05z00-00"), snapshotInfo.Location, scanner.Vm.Zones)
		if err != nil {
			log.Error().Err(err).Msg("could not complete snapshot disk creation")
			return nil, errors.Wrap(err, "could not create disk from snapshot")
		}
		log.Debug().Str("disk", *disk.ID).Msg("created disk from snapshot")
		mi.diskId = *disk.ID
		mi.diskName = *disk.Name
		asset.Name = target.SnapshotName
		conf.PlatformId = SnapshotPlatformMrn(snapshotInfo.SnapshotId)
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
	fsConn, err := connection.NewFileSystemConnection(id, &inventory.Config{
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
	*connection.FileSystemConnection
	opts            map[string]string
	volumeMounter   *snapshot.VolumeMounter
	snapshotCreator *SnapshotCreator
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

	err := c.volumeMounter.UnmountVolumeFromInstance()
	if err != nil {
		log.Error().Err(err).Msg("unable to unmount volume")
	}

	if c.snapshotCreator != nil {
		err = c.snapshotCreator.detachDisk(c.mountInfo.diskName, c.scanner.instanceInfo)
		if err != nil {
			log.Error().Err(err).Msg("unable to detach volume")
		}

		err = c.snapshotCreator.deleteCreatedDisk(c.scanner.ResourceGroup, c.mountInfo.diskName)
		if err != nil {
			log.Error().Err(err).Msg("could not delete created disk")
		}
	}

	err = c.volumeMounter.RemoveTempScanDir()
	if err != nil {
		log.Error().Err(err).Msg("unable to remove dir")
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
