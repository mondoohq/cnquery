// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"os"

	"github.com/digitalocean/godo"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"golang.org/x/oauth2"
)

type DigitaloceanConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	client *godo.Client
}

func NewDigitaloceanConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*DigitaloceanConnection, error) {
	conn := &DigitaloceanConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if len(conf.Credentials) > 0 {
		for _, cred := range conf.Credentials {
			if cred.Type == vault.CredentialType_password {
				token = string(cred.Secret)
			}
		}
	}
	if token == "" {
		return nil, errors.New("a valid DigitalOcean token is required (set DIGITALOCEAN_TOKEN or use --token)")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	conn.client = godo.NewClient(oauthClient)

	return conn, nil
}

func (c *DigitaloceanConnection) Name() string {
	return "digitalocean"
}

func (c *DigitaloceanConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *DigitaloceanConnection) Client() *godo.Client {
	return c.client
}
