// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
)

type PlatformConnection struct {
	runtime string
	id      uint32
	asset   *inventory.Asset
}

func NewPlatformConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) *PlatformConnection {
	res := PlatformConnection{
		id:    id,
		asset: asset,
	}
	return &res
}

func (p *PlatformConnection) ID() uint32 {
	return p.id
}

func (p *PlatformConnection) Name() string {
	return "platform"
}

func (p *PlatformConnection) Type() shared.ConnectionType {
	return "platform"
}

func (p *PlatformConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *PlatformConnection) Capabilities() shared.Capabilities {
	return 0
}

func (p *PlatformConnection) RunCommand(command string) (*shared.Command, error) {
	return nil, nil
}

func (p *PlatformConnection) FileSystem() afero.Fs {
	return nil
}

func (p *PlatformConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, nil
}

func (p *PlatformConnection) Close() {
}
