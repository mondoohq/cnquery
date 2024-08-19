// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jira

import (
	"context"
	"errors"
	"net/url"
	"os"

	v2 "github.com/ctreminiom/go-atlassian/jira/v2"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/atlassian/connection/shared"
)

const (
	Jira shared.ConnectionType = "jira"
)

type JiraConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	client *v2.Client
	name   string
}

func isValidHostValue(val string) bool {
	u, err := url.Parse(val)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func NewConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*JiraConnection, error) {
	host := conf.Options["host"]
	if host == "" {
		host = os.Getenv("ATLASSIAN_HOST")
	}
	if host == "" || !isValidHostValue(host) {
		return nil, errors.New("you must provide an Atlassian host e.g. via ATLASSIAN_HOST env or via the --host flag. Host must be prefixed with https://")
	}

	user := conf.Options["user"]
	if user == "" {
		user = os.Getenv("ATLASSIAN_USER")
	}
	if user == "" {
		return nil, errors.New("you must provide an Atlassian user e.g. via ATLASSIAN_USER env or via the --user flag")
	}

	token := conf.Options["user-token"]
	if token == "" {
		token = os.Getenv("ATLASSIAN_USER_TOKEN")
	}
	if token == "" {
		return nil, errors.New("you must provide an Atlassian user token e.g. via ATLASSIAN_USER_TOKEN env or via the --user-token flag")
	}

	client, err := v2.New(nil, host)
	if err != nil {
		return nil, err
	}

	client.Auth.SetBasicAuth(user, token)
	client.Auth.SetUserAgent("curl/7.54.0")

	expand := []string{""}
	_, response, _ := client.MySelf.Details(context.Background(), expand)
	if response != nil {
		if response.StatusCode == 401 {
			return nil, errors.New("failed to authenticate")
		}
	}

	return &JiraConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		client:     client,
		name:       host,
	}, nil
}

func (c *JiraConnection) Name() string {
	return c.name
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

func (c *JiraConnection) ConnectionType() string {
	return "jira"
}

func (c *JiraConnection) Config() *inventory.Config {
	return c.Conf
}
