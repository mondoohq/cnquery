// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

type NmapConnection struct {
	plugin.Connection
	Conf     *inventory.Config
	asset    *inventory.Asset
	// Add custom connection fields here
}

func NewNmapConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*NmapConnection, error) {
	conn := &NmapConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:  conf,
		asset: asset,
	}

	// initialize your connection here

	return conn, nil
}

func (c *NmapConnection) Name() string {
	return "nmap"
}

func (c *NmapConnection) Asset() *inventory.Asset {
	return c.asset
}

