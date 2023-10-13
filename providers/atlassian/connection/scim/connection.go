// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scim

import (
	"errors"
	"os"

	"github.com/ctreminiom/go-atlassian/admin"
	"github.com/rs/zerolog/log"
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
	host   string
}

func NewConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*ScimConnection, error) {
	token := conf.Options["token"]
	if token == "" {
		token = os.Getenv("ATLASSIAN_SCIM_TOKEN")
	}
	if token == "" {
		return nil, errors.New("you need to provide atlassian admin token via ATLASSIAN_SCIM_TOKEN env")
	}

	client, err := admin.New(nil)
	if err != nil {
		log.Fatal().Err(err)
	}

	client.Auth.SetBearerToken(token)
	client.Auth.SetUserAgent("curl/7.54.0")

	conn := &ScimConnection{
		Conf:   conf,
		id:     id,
		asset:  asset,
		client: client,
		host:   "admin.atlassian.com",
	}

	return conn, nil
}

func (c *ScimConnection) Name() string {
	return "jira"
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

func (c *ScimConnection) Host() string {
	return c.host
}
