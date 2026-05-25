// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"os"
	"strings"

	together "github.com/togethercomputer/together-go"
	"github.com/togethercomputer/together-go/option"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	OptionToken   = "token"
	OptionBaseURL = "base-url"
	OptionProject = "project"
)

type TogetherConnection struct {
	plugin.Connection
	Conf    *inventory.Config
	asset   *inventory.Asset
	client  together.Client
	token   string
	project string
}

func NewTogetherConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*TogetherConnection, error) {
	token := tokenFromConfig(conf)
	if token == "" {
		return nil, fmt.Errorf("together: API token is required (use --token or set TOGETHER_API_KEY)")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(token),
	}

	if conf.Options != nil {
		if raw := strings.TrimSpace(conf.Options[OptionBaseURL]); raw != "" {
			opts = append(opts, option.WithBaseURL(raw))
		}
	}

	client := together.NewClient(opts...)

	var project string
	if conf.Options != nil {
		project = strings.TrimSpace(conf.Options[OptionProject])
	}

	conn := &TogetherConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		client:     client,
		token:      token,
		project:    project,
	}

	return conn, nil
}

func (c *TogetherConnection) Name() string {
	return "together"
}

func (c *TogetherConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *TogetherConnection) Client() *together.Client {
	return &c.client
}

func (c *TogetherConnection) Token() string {
	return c.token
}

func (c *TogetherConnection) Project() string {
	return c.project
}

func tokenFromConfig(conf *inventory.Config) string {
	token := os.Getenv("TOGETHER_API_KEY")
	if conf.Options != nil && conf.Options[OptionToken] != "" {
		token = conf.Options[OptionToken]
	}
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password {
			token = string(cred.Secret)
		}
	}
	return strings.TrimSpace(token)
}
