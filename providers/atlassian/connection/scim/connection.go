// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scim

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ctreminiom/go-atlassian/admin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/atlassian/connection/shared"
)

const (
	Scim shared.ConnectionType = "scim"
)

type ScimConnection struct {
	plugin.Connection
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
		return nil, errors.New("you must provide an Atlassian SCIM token via the ATLASSIAN_SCIM_TOKEN env or via the --scim-token flag")
	}

	if conf.Options["directory-id"] == "" {
		return nil, errors.New("you must provide a directory ID for SCIM")
	}

	client, err := admin.New(nil)
	if err != nil {
		return nil, err
	}

	client.Auth.SetBearerToken(token)
	client.Auth.SetUserAgent("curl/7.54.0")

	_, response, _ := client.SCIM.Schema.User(context.Background(), conf.Options["directory-id"])
	if response != nil {
		if response.StatusCode == 401 {
			return nil, errors.New("failed to authenticate")
		}
	}

	name := fmt.Sprintf("Directory %s", conf.Options["directory-id"])

	return &ScimConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		client:     client,
		name:       name,
	}, nil
}

func (c *ScimConnection) Name() string {
	return c.name
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
