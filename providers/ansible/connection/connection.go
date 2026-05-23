// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"fmt"
	"io"
	"os"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/ansible/play"
)

var _ plugin.Connection = (*AnsibleConnection)(nil)

type AnsibleConnection struct {
	plugin.Connection
	Conf     *inventory.Config
	asset    *inventory.Asset
	path     string
	playbook play.Playbook
}

func NewAnsibleConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AnsibleConnection, error) {
	if asset == nil || len(asset.Connections) == 0 {
		return nil, errors.New("ansible: no connection options for asset")
	}
	cc := asset.Connections[0]
	path := ""
	if cc.Options != nil {
		path = cc.Options["path"]
	}
	if path == "" {
		return nil, errors.New("ansible: no playbook path provided (set the `path` option)")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ansible: cannot open playbook %q: %w", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("ansible: cannot read playbook %q: %w", path, err)
	}

	playbook, err := play.DecodePlaybook(data)
	if err != nil {
		return nil, fmt.Errorf("ansible: cannot decode playbook %q: %w", path, err)
	}

	return &AnsibleConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		path:       path,
		playbook:   playbook,
	}, nil
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
