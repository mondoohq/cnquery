// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package admin

import (
	"context"
	"errors"
	"os"

	"github.com/ctreminiom/go-atlassian/admin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/atlassian/connection/shared"
)

const (
	Admin shared.ConnectionType = "admin"
)

type AdminConnection struct {
	id       uint32
	parentId *uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
	client   *admin.Client
	name     string
}

func NewConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AdminConnection, error) {
	adminToken := conf.Options["admin-token"]
	if adminToken == "" {
		adminToken = os.Getenv("ATLASSIAN_ADMIN_TOKEN")
	}
	if adminToken == "" {
		return nil, errors.New("you must provide an Atlassian admin token via the ATLASSIAN_ADMIN_TOKEN env or via the --admin-token flag")
	}

	client, err := admin.New(nil)
	if err != nil {
		return nil, err
	}

	client.Auth.SetBearerToken(adminToken)
	client.Auth.SetUserAgent("curl/7.54.0")

	_, response, _ := client.Organization.Gets(context.Background(), "")
	if response != nil {
		if response.StatusCode == 401 {
			return nil, errors.New("Failed to authenticate")
		}
	}

	conn := &AdminConnection{
		Conf:   conf,
		id:     id,
		asset:  asset,
		client: client,
		name:   "admin.atlassian.com",
	}
	if len(asset.Connections) > 0 && asset.Connections[0].ParentConnectionId > 0 {
		conn.parentId = &asset.Connections[0].ParentConnectionId
	}

	return conn, nil
}

func (c *AdminConnection) Name() string {
	return c.name
}

func (c *AdminConnection) ID() uint32 {
	return c.id
}

func (c *AdminConnection) ParentID() *uint32 {
	return c.parentId
}

func (c *AdminConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *AdminConnection) Client() *admin.Client {
	return c.client
}

func (c *AdminConnection) Type() shared.ConnectionType {
	return Admin
}

func (c *AdminConnection) ConnectionType() string {
	return "admin"
}

func (c *AdminConnection) Config() *inventory.Config {
	return c.Conf
}
