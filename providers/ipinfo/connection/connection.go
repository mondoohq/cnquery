// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
)

type IpinfoConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	client interface{} // ipinfo client will be stored here
}

func NewIpinfoConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*IpinfoConnection, error) {
	conn := &IpinfoConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// initialize your connection here

	return conn, nil
}

func (c *IpinfoConnection) Name() string {
	return "ipinfo"
}

func (c *IpinfoConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *IpinfoConnection) Client() interface{} {
	return c.client
}

func (c *IpinfoConnection) SetClient(client interface{}) {
	c.client = client
}
