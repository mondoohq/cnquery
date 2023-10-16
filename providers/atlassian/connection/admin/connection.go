// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package admin

import (
	"errors"
	"os"

	"github.com/ctreminiom/go-atlassian/admin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/shared"
)

const (
	Admin shared.ConnectionType = "admin"
)

type AdminConnection struct {
	id     uint32
	Conf   *inventory.Config
	asset  *inventory.Asset
	client *admin.Client
	host   string
}

func NewConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AdminConnection, error) {
	adminToken := conf.Options["admintoken"]
	if adminToken == "" {
		adminToken = os.Getenv("ATLASSIAN_ADMIN_TOKEN")
	}
	if adminToken == "" {
		return nil, errors.New("you need to provide atlassian admin token via ATLASSIAN_ADMIN_TOKEN env")
	}

	client, err := admin.New(nil)
	if err != nil {
		return nil, err
	}

	client.Auth.SetBearerToken(adminToken)
	client.Auth.SetUserAgent("curl/7.54.0")

	conn := &AdminConnection{
		Conf:   conf,
		id:     id,
		asset:  asset,
		client: client,
		host:   "admin.atlassian.com",
	}

	return conn, nil
}

func (c *AdminConnection) Name() string {
	return "atlassian"
}

func (c *AdminConnection) ID() uint32 {
	return c.id
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

func (c *AdminConnection) Host() string {
	return c.host
}

func (c *AdminConnection) ConnectionType() string {
	return "admin"
}

func (c *AdminConnection) Config() *inventory.Config {
	return c.Conf
}
