// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
)

type GcpConnection struct {
	id       uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
	// Add custom connection fields here
}

func NewGcpConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*GcpConnection, error) {
	conn := &GcpConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}

	// initialize your connection here

	return conn, nil
}

func (c *GcpConnection) Name() string {
	return "gcp"
}

func (c *GcpConnection) ID() uint32 {
	return c.id
}

func (c *GcpConnection) Asset() *inventory.Asset {
	return c.asset
}

