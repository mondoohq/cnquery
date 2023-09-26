// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/slack-go/slack"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
)

type SlackConnection struct {
	id       uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
	client   *slack.Client
	teamInfo *slack.TeamInfo
}

func NewMockConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) *SlackConnection {
	return &SlackConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}
}

func NewSlackConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*SlackConnection, error) {
	sc := &SlackConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}

	// if a secret was provided, it always overrides the env variable since it has precedence
	token := os.Getenv("SLACK_TOKEN")
	if len(conf.Credentials) > 0 {
		for i := range conf.Credentials {
			cred := conf.Credentials[i]
			if cred.Type == vault.CredentialType_password {
				token = string(cred.Secret)
			} else {
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for Slack provider")
			}
		}
	}

	if token == "" {
		return nil, errors.New("a valid Slack token is required, pass --token '<yourtoken>' or set SLACK_TOKEN environment variable")
	}

	client := slack.New(token)
	teamInfo, err := client.GetTeamInfo()
	if err != nil {
		return nil, err
	}

	sc.client = client
	sc.teamInfo = teamInfo
	sc.asset.Name = teamInfo.Name
	return sc, nil
}

func (s *SlackConnection) Name() string {
	return "slack"
}

func (s *SlackConnection) ID() uint32 {
	return s.id
}

func (s *SlackConnection) Asset() *inventory.Asset {
	return s.asset
}

func (s *SlackConnection) Client() *slack.Client {
	return s.client
}

func (p *SlackConnection) TeamID() string {
	return p.teamInfo.ID
}

func (p *SlackConnection) TeamName() string {
	return p.teamInfo.Name
}
