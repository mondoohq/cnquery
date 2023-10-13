// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"

	v2 "github.com/ctreminiom/go-atlassian/jira/v2"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers/atlassian/connection/shared"
)

const (
	Jira shared.ConnectionType = "jira"
)

type JiraConnection struct {
	id     uint32
	Conf   *inventory.Config
	asset  *inventory.Asset
	client *v2.Client
}

func NewAtlassianConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*JiraConnection, error) {
	host := conf.Options["host"]
	if host == "" {
		host = os.Getenv("ATLASSIAN_HOST")
	}
	if host == "" {
		return nil, errors.New("you need to provide atlassian host via ATLASSIAN_HOST env")
	}

	user := conf.Options["user"]
	if user == "" {
		user = os.Getenv("ATLASSIAN_USER")
	}
	if user == "" {
		return nil, errors.New("you need to provide atlassian user via ATLASSIAN_USER env")
	}

	userToken := conf.Options["usertoken"]
	if userToken == "" {
		userToken = os.Getenv("ATLASSIAN_USER_TOKEN")
	}
	if userToken == "" {
		return nil, errors.New("you need to provide atlassian user token via ATLASSIAN_USER_TOKEN env")
	}

	client, err := v2.New(nil, host)
	if err != nil {
		log.Fatal().Err(err)
	}

	client.Auth.SetBasicAuth(user, userToken)
	client.Auth.SetUserAgent("curl/7.54.0")

	conn := &JiraConnection{
		Conf:   conf,
		id:     id,
		asset:  asset,
		client: client,
	}

	return conn, nil
}

func (c *JiraConnection) Name() string {
	return "jira"
}

func (c *JiraConnection) ID() uint32 {
	return c.id
}

func (c *JiraConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *JiraConnection) Client() *v2.Client {
	return c.client
}

func (p *JiraConnection) Type() shared.ConnectionType {
	return Jira
}
