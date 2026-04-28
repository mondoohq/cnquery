// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1
package connection

import (
	"errors"
	"os"

	v0cloudflare "github.com/cloudflare/cloudflare-go"
	cloudflare "github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/option"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

const (
	OPTION_API_TOKEN = "api-token"
)

type CloudflareConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	// Cf is the cloudflare-go v6 client. New resource code should use this.
	Cf *cloudflare.Client
	// LegacyCf is the cloudflare-go v0 client kept alongside Cf during the
	// staged v0→v6 migration tracked in #7385. Resource files that haven't
	// been migrated yet use this; once a file is migrated, its usages move
	// to Cf. The field will be dropped once the last migration PR lands.
	LegacyCf *v0cloudflare.API
}

func NewCloudflareConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*CloudflareConnection, error) {
	conn := &CloudflareConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	token := conf.Options[OPTION_API_TOKEN]
	if token == "" {
		token = os.Getenv("CLOUDFLARE_TOKEN")
		if token == "" {
			return nil, errors.New("a valid Cloudflare authentication is required, pass --token '<yourtoken>', set CLOUDFLARE_TOKEN environment variable")
		}
	}

	conn.Cf = cloudflare.NewClient(option.WithAPIToken(token))

	legacy, err := v0cloudflare.NewWithAPIToken(token)
	if err != nil {
		return nil, err
	}
	conn.LegacyCf = legacy

	return conn, nil
}

func (c *CloudflareConnection) Name() string {
	return "cloudflare"
}

func (c *CloudflareConnection) Asset() *inventory.Asset {
	return c.asset
}
