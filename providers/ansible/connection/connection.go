// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/ansible/play"
	"io"
	"os"
)

var _ plugin.Connection = (*AnsibleConnection)(nil)

type AnsibleConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	// Add custom connection fields here
	path     string
	playbook play.Playbook
}

func NewAnsibleConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AnsibleConnection, error) {
	conn := &AnsibleConnection{
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

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	playbook, err := play.DecodePlaybook(data)
	if err != nil {
		return nil, err
	}
	conn.playbook = playbook

	return conn, nil
}

func (c *AnsibleConnection) Name() string {
	return "ansible"
}

func (c *AnsibleConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *AnsibleConnection) Playbook() play.Playbook {
	return c.playbook
}
