// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package connection

import (
	"errors"
	"os"

	"github.com/cloudflare/cloudflare-go"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

const (
	OPTION_API_TOKEN = "api-token"
)

type CloudflareConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	Cf *cloudflare.API
}

func NewCloudflareConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*CloudflareConnection, error) {
	conn := &CloudflareConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// initialize your connection here
	token := conf.Options[OPTION_API_TOKEN]
	if token == "" {
		token = os.Getenv("CLOUDFLARE_TOKEN")
		if token == "" {
			return nil, errors.New("a valid Cloudflare authentication is required, pass --token '<yourtoken>', set CLOUDFLARE_TOKEN environment variable")
		}
	}

	api, err := cloudflare.NewWithAPIToken(token)
	if err != nil {
		return nil, err
	}
	conn.Cf = api

	return conn, nil
}

func (c *CloudflareConnection) Name() string {
	return "cloudflare"
}

func (c *CloudflareConnection) Asset() *inventory.Asset {
	return c.asset
}
