// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
)

type HostConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
}

func NewHostConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) *HostConnection {
	return &HostConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}
}

func (h *HostConnection) Name() string {
	return "host"
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
