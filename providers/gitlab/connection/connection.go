// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"net/url"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
)

type GitLabConnection struct {
	id          uint32
	Conf        *inventory.Config
	asset       *inventory.Asset
	group       *gitlab.Group
	project     *gitlab.Project
	projectID   string // only used for initial setup, use project.ID afterwards!
	groupPath   string
	projectPath string
	client      *gitlab.Client
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

	conn := &GitLabConnection{
		Conf:        conf,
		id:          id,
		asset:       asset,
		groupPath:   conf.Options["group"],
		projectPath: conf.Options["project"],
		projectID:   conf.Options["project-id"],
		client:      client,
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

func (c *GitLabConnection) Group() (*gitlab.Group, error) {
	if c.group != nil {
		return c.group, nil
	}
	if c.groupPath == "" {
		return nil, errors.New("cannot look up gitlab group, no group path defined")
	}

	var err error
	c.group, _, err = c.Client().Groups.GetGroup(c.groupPath, nil)
	return c.group, err
}

func (c *GitLabConnection) IsGroup() bool {
	return c.groupPath != ""
}

func (c *GitLabConnection) IsProject() bool {
	return c.projectPath != "" || c.projectID != ""
}

func (c *GitLabConnection) GID() (interface{}, error) {
	if c.groupPath == "" {
		return nil, errors.New("cannot look up gitlab group, no group path defined")
	}
	return url.QueryEscape(c.groupPath), nil
}

func (c *GitLabConnection) PID() (interface{}, error) {
	if c.projectID != "" {
		return c.projectID, nil
	}

	if c.groupPath == "" {
		return nil, errors.New("cannot look up gitlab group, no group path defined")
	}
	if c.projectPath == "" {
		return nil, errors.New("cannot look up gitlab project, no project path defined")
	}
	return url.QueryEscape(c.groupPath) + "/" + url.QueryEscape(c.projectPath), nil
}

func (c *GitLabConnection) Project() (*gitlab.Project, error) {
	if c.project != nil {
		return c.project, nil
	}

	pid, err := c.PID()
	if err != nil {
		return nil, err
	}
	c.project, _, err = c.Client().Projects.GetProject(pid, nil)
	return c.project, err
}
