// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scim

import (
	"errors"
	"fmt"
	"os"

	"github.com/ctreminiom/go-atlassian/admin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/shared"
)

const (
	Scim shared.ConnectionType = "scim"
)

type ScimConnection struct {
	id     uint32
	Conf   *inventory.Config
	asset  *inventory.Asset
	client *admin.Client
	name   string
}

func NewConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*ScimConnection, error) {
	token := conf.Options["scim-token"]
	if token == "" {
		token = os.Getenv("ATLASSIAN_SCIM_TOKEN")
	}
	if token == "" {
		return nil, errors.New("you need to provide atlassian scim token via ATLASSIAN_SCIM_TOKEN env or via --admin-token flag")
	}

	if conf.Options["directory-id"] == "" {
		return nil, errors.New("you need to provide a directory id for scim")
	}

	client, err := admin.New(nil)
	if err != nil {
		return nil, err
	}

	client.Auth.SetBearerToken(token)
	client.Auth.SetUserAgent("curl/7.54.0")

	name := fmt.Sprintf("Directory %s", conf.Options["directory-id"])
	conn := &ScimConnection{
		Conf:   conf,
		id:     id,
		asset:  asset,
		client: client,
		name:   name,
	}

	return conn, nil
}

func (c *ScimConnection) Name() string {
	return c.name
}

func (c *ScimConnection) ID() uint32 {
	return c.id
}

func (c *ScimConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *ScimConnection) Client() *admin.Client {
	return c.client
}

func (p *ScimConnection) Type() shared.ConnectionType {
	return Scim
}

func (c *ScimConnection) Directory() string {
	return c.Conf.Options["directory-id"]
}

func (c *ScimConnection) ConnectionType() string {
	return "scim"
}

func (c *ScimConnection) Config() *inventory.Config {
	return c.Conf
}
