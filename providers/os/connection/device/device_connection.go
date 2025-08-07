// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package device

import (
	"errors"
	"maps"
	"runtime"
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/device/linux"
	"go.mondoo.com/cnquery/v11/providers/os/connection/device/windows"
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
	"go.mondoo.com/cnquery/v11/utils/stringx"

	"go.mondoo.com/cnquery/v11/providers/os/connection/fs"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/detector"
	"go.mondoo.com/cnquery/v11/providers/os/id"
	"go.mondoo.com/cnquery/v11/providers/os/id/ids"
)

const (
	PlatformIdInject   = "inject-platform-ids"
	KeepMounted        = "keep-mounted"
	SkipAssetDetection = "skip-asset-detection"
)

type DeviceConnection struct {
	// represents the single fs where an asset has been identified
	*fs.FileSystemConnection
	plugin.Connection
	asset         *inventory.Asset
	deviceManager DeviceManager

	MountedDirs []string
	// map of mountpoints to partition infos
	partitions map[string]*snapshot.PartitionInfo

	// whether to keep the devices mounted after the connection is closed
	keepMounted bool
}

func getDeviceManager(conf *inventory.Config) (DeviceManager, error) {
	shell := []string{"sh", "-c"}
	if runtime.GOOS == "darwin" {
		return nil, errors.New("device manager not implemented for darwin")
	}
	if runtime.GOOS == "windows" {
		shell = []string{"powershell", "-c"}
		return windows.NewWindowsDeviceManager(shell, conf.Options)
	}
	return linux.NewLinuxDeviceManager(shell, conf.Options)
}

func NewDeviceConnection(connId uint32, conf *inventory.Config, asset *inventory.Asset) (*DeviceConnection, error) {
	// allow injecting platform ids into the device connection. we cannot always know the asset that's being scanned, e.g.
	// if we can scan an azure VM's disk we should be able to inject the platform ids of the VM
	if platformIDs, ok := conf.Options[PlatformIdInject]; ok {
		platformIds := strings.Split(platformIDs, ",")
		for _, id := range platformIds {
			if !stringx.Contains(asset.PlatformIds, id) {
				log.Debug().Str("platform-id", id).Msg("device connection> injecting platform id")
				asset.PlatformIds = append(asset.PlatformIds, id)
			}
		}
	}

	manager, err := getDeviceManager(conf)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("manager", manager.Name()).Msg("device manager created")

	blocks, err := manager.IdentifyMountTargets(conf.Options)
	if err != nil {
		log.Warn().Err(err).Msg("device connection> unable to identify some mount targets, proceeding with the rest")
	}

	if len(blocks) == 0 {
		return nil, errors.New("device connection> no blocks found, cannot perform a scan")
	}

	deviceConnection := &DeviceConnection{
		Connection:    plugin.NewConnection(connId, asset),
		deviceManager: manager,
		asset:         asset,
	}

	if conf.Options == nil {
		conf.Options = make(map[string]string)
	}
	deviceConnection.keepMounted = conf.Options[KeepMounted] == "true"

	if len(asset.IdDetector) == 0 {
		asset.IdDetector = []string{ids.IdDetector_Hostname, ids.IdDetector_SshHostkey}
	}
	if !stringx.Contains(asset.IdDetector, ids.IdDetector_MachineID) {
		asset.IdDetector = append(asset.IdDetector, ids.IdDetector_MachineID)
	}
	// This detector helps set the `asset.kind` if the connected device is a cloud instance
	if !stringx.Contains(asset.IdDetector, ids.IdDetector_CloudDetect) {
		asset.IdDetector = append(asset.IdDetector, ids.IdDetector_CloudDetect)
	}

	deviceConnection.partitions = make(map[string]*snapshot.PartitionInfo)

	skipAssetDetection := conf.Options[SkipAssetDetection] == "true"

	// first, iterate over all blocks and mount them, if needed.
	for _, block := range blocks {
		logBuilder := log.Debug().
			Str("name", block.Name).
			Str("type", block.FsType).
			Str("mountpoint", block.MountPoint)
		if block.MountPoint == "" {
			logBuilder.Msg("device connection> mounting block device")
			scanDir, err := manager.Mount(block)
			if err != nil {
				log.Error().Err(err).Msg("unable to complete mount step")
				continue
			}
			block.MountPoint = scanDir
		} else {
			logBuilder.Msg("device connection> already mounted block device")
		}
		if !stringx.Contains(deviceConnection.MountedDirs, block.MountPoint) {
			log.Debug().
				Str("name", block.Name).
				Str("mountpoint", block.MountPoint).
				Msg("device connection> adding mountpoint to mounted dirs")
			deviceConnection.MountedDirs = append(deviceConnection.MountedDirs, block.MountPoint)
		}

		deviceConnection.partitions[block.RootDir()] = block
	}

	if skipAssetDetection {
		log.Debug().Msg("device connection> skipping asset detection as requested")
		return deviceConnection, nil
	}

	log.Debug().
		Strs("partitions", slices.Collect(maps.Keys(deviceConnection.partitions))).
		Strs("mountedDirs", deviceConnection.MountedDirs).
		Msg("device connection> mounted partitions, proceeding with asset detection")

	// once everything is mounted, we can try and find the correct partition that holds the OS
	for partition, block := range deviceConnection.partitions {
		if deviceConnection.FileSystemConnection != nil {
			log.Debug().Str("partition", block.Name).Msg("device connection> already found a filesystem connection, skipping")
			continue
		}

		fsConn, err := tryDetectAsset(connId, block, conf, asset)
		if fsConn != nil {
			deviceConnection.FileSystemConnection = fsConn
		}
		if err != nil {
			log.Error().Err(err).Str("partition", partition).Msg("partition did not return an asset, continuing")
		}
	}

	// if none of the blocks returned a platform that we could detect, we return an error
	if asset.Platform == nil {
		log.Debug().Msg("device connection> no platform detected, closing device connection")
		deviceConnection.Close()
		return nil, errors.New("device connection> no platform detected")
	}

	return deviceConnection, nil
}

func (c *DeviceConnection) Close() {
	log.Debug().Msg("closing device connection")
	if c == nil {
		return
	}

	if c.deviceManager != nil && !c.keepMounted {
		c.deviceManager.UnmountAndClose()
	}
}

func (p *DeviceConnection) Name() string {
	return "device"
}

func (p *DeviceConnection) Type() shared.ConnectionType {
	return shared.Type_Device
}

func (p *DeviceConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *DeviceConnection) UpdateAsset(asset *inventory.Asset) {
	p.asset = asset
}

func (p *DeviceConnection) Capabilities() shared.Capabilities {
	return p.FileSystemConnection.Capabilities()
}

func (p *DeviceConnection) RunCommand(command string) (*shared.Command, error) {
	return nil, plugin.ErrRunCommandNotImplemented
}

func (p *DeviceConnection) FileSystem() afero.Fs {
	return p.FileSystemConnection.FileSystem()
}

func (p *DeviceConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return p.FileSystemConnection.FileInfo(path)
}

func (p *DeviceConnection) Conf() *inventory.Config {
	return p.FileSystemConnection.Conf
}

func (p *DeviceConnection) Partitions() map[string]*snapshot.PartitionInfo {
	if p.partitions == nil {
		p.partitions = make(map[string]*snapshot.PartitionInfo)
	}

	return p.partitions
}

// tryDetectAsset tries to detect the OS on a given block device
func tryDetectAsset(connId uint32, partition *snapshot.PartitionInfo, conf *inventory.Config, asset *inventory.Asset) (*fs.FileSystemConnection, error) {
	fsPath := partition.RootDir()

	// create and initialize fs provider
	log.Debug().Str("path", fsPath).Msg("device connection> trying to detect asset")
	conf.Options["path"] = fsPath
	fsConn, err := fs.NewConnection(connId, &inventory.Config{
		Path:       fsPath,
		PlatformId: conf.PlatformId,
		Options:    conf.Options,
		Type:       "fs",
		Record:     conf.Record,
	}, asset)
	if err != nil {
		return nil, err
	}

	p, ok := detector.DetectOS(fsConn)
	if !ok {
		log.Debug().
			Str("partition", partition.Name).
			Msg("device connection> cannot detect os")
		return nil, errors.New("cannot detect os")
	}

	log.Debug().Str("partition", partition.MountPoint).Str("block", partition.Name).Msg("device connection> detected os from mountpoint")
	fingerprint, p, err := id.IdentifyPlatform(fsConn, &plugin.ConnectReq{}, p, asset.IdDetector)
	if err != nil {
		if len(asset.PlatformIds) == 0 {
			log.Debug().Err(err).Msg("device connection> failed to identify platform from device")
			return nil, err
		}
		log.Warn().Err(err).Msg("device connection> cannot detect platform ids, using existing ones")
	}

	if p == nil {
		log.Debug().Str("partition", partition.MountPoint).Str("block", partition.Name).Msg("device connection> no platform detected")
		return nil, errors.New("device connection> no platform detected")
	}

	if asset.Name == "" && fingerprint != nil {
		asset.Name = fingerprint.Name
	}

	if fingerprint != nil {
		asset.PlatformIds = append(asset.PlatformIds, fingerprint.PlatformIDs...)
		asset.IdDetector = fingerprint.ActiveIdDetectors
	}

	asset.Id = conf.Type

	// volumes and partitions without a system on them would return an "unknown" platform
	// we don't want to overwrite the platform if a "proper" one was detected already
	// as well as we want to always have some platform to the asset
	if asset.Platform == nil || !slices.Contains([]string{"", "unknown"}, p.Name) {
		asset.Platform = p
		log.Debug().Str("partition", partition.MountPoint).Str("block", partition.Name).Msg("device connection> platform os from mountpoint")
		return fsConn, nil
	}

	return nil, nil
}
