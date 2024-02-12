// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

type HostConnection struct {
	id       uint32
	parentId *uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
}

func NewHostConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) *HostConnection {
	conn := &HostConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}
	if len(asset.Connections) > 0 && asset.Connections[0].ParentConnectionId > 0 {
		conn.parentId = &asset.Connections[0].ParentConnectionId
	}
	return conn
}

func (h *HostConnection) Name() string {
	return "host"
}

func (h *HostConnection) ID() uint32 {
	return h.id
}

func (c *HostConnection) ParentID() *uint32 {
	return c.parentId
}

func (p *HostConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *HostConnection) FQDN() string {
	if p.Conf == nil {
		return ""
	}
	return p.Conf.Host
}
