// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
)

type GitLabConnection struct {
	id        uint32
	Conf      *inventory.Config
	asset     *inventory.Asset
	GroupPath string
	client    *gitlab.Client
}

func NewGitLabConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*GitLabConnection, error) {
	// check if the token was provided by the option. This way is deprecated since it does not pass the token as secret
	token := conf.Options["token"]

	// if no token was provided, lets read the env variable
	if token == "" {
		token = os.Getenv("GITLAB_TOKEN")
	}

	// if a secret was provided, it always overrides the env variable since it has precedence
	if len(conf.Credentials) > 0 {
		for i := range conf.Credentials {
			cred := conf.Credentials[i]
			if cred.Type == vault.CredentialType_password {
				token = string(cred.Secret)
			} else {
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for GitHub provider")
			}
		}
	}

	if token == "" {
		return nil, errors.New("you need to provide GitLab token e.g. via GITLAB_TOKEN env")
	}

	client, err := gitlab.NewClient(token)
	if err != nil {
		return nil, err
	}

	if conf.Options["group"] == "" {
		return nil, errors.New("you need to provide a group for gitlab")
	}

	conn := &GitLabConnection{
		Conf:      conf,
		id:        id,
		asset:     asset,
		GroupPath: conf.Options["group"],
		client:    client,
	}

	return conn, nil
}

func (c *GitLabConnection) Name() string {
	return "gitlab"
}

func (c *GitLabConnection) ID() uint32 {
	return c.id
}

func (c *GitLabConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *GitLabConnection) Client() *gitlab.Client {
	return c.client
}
