// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
)

const (
	Tar shared.ConnectionType = "tar"
)

type TarConnection struct {
	id    uint32
	asset *inventory.Asset
}

func NewTarConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*TarConnection, error) {
	// expect unix shell by default
	res := TarConnection{
		id:    id,
		asset: asset,
	}

	panic("Not yet migrated")

	return &res, nil
}

func (p *TarConnection) ID() uint32 {
	return p.id
}

func (p *TarConnection) Name() string {
	return string(Tar)
}

func (p *TarConnection) Type() shared.ConnectionType {
	return Tar
}

func (p *TarConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *TarConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File
}

func (p *TarConnection) RunCommand(command string) (*shared.Command, error) {
	return nil, errors.New("cannot run commands on docker snapshots")
}

func (p *TarConnection) FileSystem() afero.Fs {
	panic("Not yet migrated")
	return nil
}

func (p *TarConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	panic("Not yet migrated")
	return shared.FileInfoDetails{}, nil
}

func (p *TarConnection) Close() {
	// TODO: we need to close all commands and file handles
}
