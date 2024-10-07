// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package device

import (
	"errors"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/device/linux"
	"go.mondoo.com/cnquery/v11/providers/os/connection/device/windows"

	"go.mondoo.com/cnquery/v11/providers/os/connection/fs"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/detector"
	"go.mondoo.com/cnquery/v11/providers/os/id"
	"go.mondoo.com/cnquery/v11/providers/os/id/ids"
)

const PlatformIdInject = "inject-platform-ids"

type DeviceConnection struct {
	FsConnections []*fs.FileSystemConnection
	plugin.Connection
	asset         *inventory.Asset
	deviceManager DeviceManager
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
	manager, err := getDeviceManager(conf)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("manager", manager.Name()).Msg("device manager created")

	blocks, err := manager.IdentifyMountTargets(conf.Options)
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, errors.New("internal> no blocks found")
	}

	res := &DeviceConnection{
		Connection:    plugin.NewConnection(connId, asset),
		deviceManager: manager,
		asset:         asset,
	}

	for i := range blocks {
		block := blocks[i]
		log.Debug().Str("name", block.Name).Str("type", block.FsType).Msg("identified partition for mounting")

		scanDir, err := manager.Mount(block)
		if err != nil {
			log.Error().Err(err).Msg("unable to complete mount step")
			res.Close()
			return nil, err
		}
		if conf.Options == nil {
			conf.Options = make(map[string]string)
		}

		conf.Options["path"] = scanDir
		// create and initialize fs provider
		fsConn, err := fs.NewConnection(connId, &inventory.Config{
			Path:       scanDir,
			PlatformId: conf.PlatformId,
			Options:    conf.Options,
			Type:       "fs",
			Record:     conf.Record,
		}, asset)
		if err != nil {
			res.Close()
			return nil, err
		}

		res.FsConnections = append(res.FsConnections, fsConn)

		// allow injecting platform ids into the device connection. we cannot always know the asset that's being scanned, e.g.
		// if we can scan an azure VM's disk we should be able to inject the platform ids of the VM
		if platformIDs, ok := conf.Options[PlatformIdInject]; ok {
			platformIds := strings.Split(platformIDs, ",")
			if len(platformIds) > 0 {
				log.Debug().Strs("platform-ids", platformIds).Msg("device connection> injecting platform ids")
				conf.PlatformId = platformIds[0]
				asset.PlatformIds = append(asset.PlatformIds, platformIds...)
			}
		}

		if asset.Platform != nil {
			log.Debug().Msg("device connection> platform already detected")
			continue
		}

		p, ok := detector.DetectOS(fsConn)
		if !ok {
			log.Debug().
				Str("block", block.Name).
				Msg("device connection> cannot detect os")
			continue
		}
		asset.Platform = p
		asset.IdDetector = []string{ids.IdDetector_Hostname}
		fingerprint, p, err := id.IdentifyPlatform(fsConn, &plugin.ConnectReq{}, asset.Platform, asset.IdDetector)
		if err == nil {
			if asset.Name == "" {
				asset.Name = fingerprint.Name
			}
			asset.PlatformIds = append(asset.PlatformIds, fingerprint.PlatformIDs...)
			asset.IdDetector = fingerprint.ActiveIdDetectors
			asset.Platform = p
			asset.Id = conf.Type
		}
	}

	if asset.Platform == nil {
		res.Close()
		return nil, errors.New("failed to detect OS")
	}

	return res, nil
}

func (c *DeviceConnection) Close() {
	log.Debug().Msg("closing device connection")
	if c == nil {
		return
	}

	if c.deviceManager != nil {
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
	return p.FsConnections[0].Capabilities()
}

func (p *DeviceConnection) RunCommand(command string) (*shared.Command, error) {
	return nil, plugin.ErrRunCommandNotImplemented
}

func (p *DeviceConnection) FileSystem() afero.Fs {
	return p.FsConnections[0].FileSystem()
}

func (p *DeviceConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return p.FsConnections[0].FileInfo(path)
}

func (p *DeviceConnection) Conf() *inventory.Config {
	return p.FsConnections[0].Conf
}
