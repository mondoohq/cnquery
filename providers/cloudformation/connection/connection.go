// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"os"

	"github.com/aws-cloudformation/rain/cft"
	"github.com/aws-cloudformation/rain/cft/parse"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

var _ plugin.Connection = (*CloudformationConnection)(nil)

type CloudformationConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	// Add custom connection fields here
	path        string
	cftTemplate cft.Template
}

func NewCloudformationConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*CloudformationConnection, error) {
	conn := &CloudformationConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}
	// initialize your connection here
	cc := asset.Connections[0]
	path := cc.Options["path"]
	conn.path = path

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cftTemplate, err := parse.Reader(f)
	if err != nil {
		return nil, err
	}
	conn.cftTemplate = cftTemplate

	return conn, nil
}

func (c *CloudformationConnection) Name() string {
	return "cloudformation"
}

func (c *CloudformationConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *CloudformationConnection) CftTemplate() cft.Template {
	return c.cftTemplate
}
