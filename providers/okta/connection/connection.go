// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

type OktaConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	// custom connection fields
	organization string
	client       *okta.Client
	token        string
}

func NewOktaConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*OktaConnection, error) {
	conn := &OktaConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// initialize connection
	if conf.Type != "okta" {
		return nil, errors.New("provider type does not match") // TODO: switch to plugin.ErrProviderTypeDoesNotMatch
	}

	if conf.Options == nil || conf.Options["organization"] == "" {
		return nil, errors.New("okta provider requires an organization id. please set option `organization`")
	}

	org := conf.Options["organization"]

	var token string
	if len(conf.Credentials) > 0 {
		log.Debug().Int("credentials", len(conf.Credentials)).Msg("credentials")
		for i := range conf.Credentials {
			cred := conf.Credentials[i]
			if cred.Type == vault.CredentialType_password {
				token = string(cred.Secret)
			} else {
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for Okta provider")
			}
		}
	}
	if token == "" {
		return nil, errors.New("a valid Okta token is required, pass --token '<yourtoken>' or set OKTA_API_TOKEN environment variable")
	}

	_, client, err := okta.NewClient(
		context.Background(),
		okta.WithOrgUrl("https://"+org),
		okta.WithToken(token),
	)
	if err != nil {
		return nil, err
	}

	conn.organization = org
	conn.client = client
	conn.token = token

	return conn, nil
}

func (c *OktaConnection) Name() string {
	return "okta"
}

func (c *OktaConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *OktaConnection) OrganizationID() string {
	return c.organization
}

func (c *OktaConnection) Client() *okta.Client {
	return c.client
}

func (c *OktaConnection) Token() string {
	return c.token
}

func (c *OktaConnection) Identifier() (string, error) {
	settings, _, err := c.client.OrgSetting.GetOrgSettings(context.Background())
	if err != nil {
		return "", errors.Join(errors.New("failed to get Okta org ID"), err)
	}

	return settings.Id, nil
}
