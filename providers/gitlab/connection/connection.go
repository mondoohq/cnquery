// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

type GitLabConnection struct {
	plugin.Connection
	Conf        *inventory.Config
	asset       *inventory.Asset
	group       *gitlab.Group
	project     *gitlab.Project
	projectID   string // only used for initial setup, use project.ID afterwards!
	groupName   string
	groupID     string
	projectName string
	client      *gitlab.Client
	url         string
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
		return nil, errors.New("you must provide GitLab token e.g. via GITLAB_TOKEN env")
	}

	var opts gitlab.ClientOptionFunc
	url := conf.Options["url"]
	if url != "" {
		opts = gitlab.WithBaseURL(url)
	}

	client, err := gitlab.NewClient(token, opts)
	if err != nil {
		return nil, err
	}

	return &GitLabConnection{
		Connection:  plugin.NewConnection(id, asset),
		Conf:        conf,
		asset:       asset,
		groupName:   conf.Options["group"],
		groupID:     conf.Options["group-id"],
		projectName: conf.Options["project"],
		projectID:   conf.Options["project-id"],
		url:         conf.Options["url"],
		client:      client,
	}, nil
}

func (c *GitLabConnection) Name() string {
	return "gitlab"
}

func (c *GitLabConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *GitLabConnection) GroupName() string {
	return c.groupName
}

func (c *GitLabConnection) GroupID() int {
	i, err := strconv.Atoi(c.groupID)
	if err == nil {
		return i
	}
	return 0
}

func (c *GitLabConnection) Client() *gitlab.Client {
	return c.client
}

func (c *GitLabConnection) Group() (*gitlab.Group, error) {
	if c.group != nil {
		return c.group, nil
	}
	if c.groupName == "" && c.groupID == "" {
		return nil, errors.New("cannot look up gitlab group, no group name defined")
	}
	gid := c.groupID
	if gid == "" {
		gid = c.groupName
	}
	log.Debug().Str("id", gid).Msgf("finding group")

	if c.groupID == "" {
		// if group name has a slash, we know its a subgroup
		if names := strings.Split(c.groupName, "/"); len(names) > 1 {
			return c.findSubgroup(names[0], names[1])
		}
	}

	var err error
	c.group, _, err = c.Client().Groups.GetGroup(gid, nil)
	return c.group, err
}

func (c *GitLabConnection) findSubgroup(parentId string, name string) (*gitlab.Group, error) {
	log.Debug().Msgf("find subgroup for %s %s", parentId, name)
	groups, err := DiscoverSubAndDescendantGroupsForGroup(c, parentId)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if name == groups[i].Name {
			return groups[i], nil
		}
	}
	return nil, errors.New("not found")
}

func (c *GitLabConnection) IsGroup() bool {
	return c.groupName != ""
}

func (c *GitLabConnection) IsProject() bool {
	return c.projectName != "" || c.projectID != ""
}

func (c *GitLabConnection) GID() (interface{}, error) {
	if c.groupName == "" {
		return nil, errors.New("cannot look up gitlab group, no group path defined")
	}
	return url.QueryEscape(c.groupName), nil
}

func (c *GitLabConnection) PID() (interface{}, error) {
	if c.projectID != "" {
		return c.projectID, nil
	}

	if c.groupName == "" {
		return nil, errors.New("cannot look up gitlab group, no group path defined")
	}
	if c.projectName == "" {
		return nil, errors.New("cannot look up gitlab project, no project path defined")
	}
	return url.QueryEscape(c.groupName) + "/" + url.QueryEscape(c.projectName), nil
}

func (c *GitLabConnection) Project() (*gitlab.Project, error) {
	if c.project != nil {
		return c.project, nil
	}

	pid, err := c.PID()
	if err != nil {
		return nil, err
	}
	log.Debug().Interface("id", pid).Msgf("finding project")

	c.project, _, err = c.Client().Projects.GetProject(pid, nil)
	return c.project, err
}

func DiscoverSubAndDescendantGroupsForGroup(conn *GitLabConnection, rootGroup string) ([]*gitlab.Group, error) {
	var list []*gitlab.Group
	// discover subgroups
	subgroups, err := groupSubgroups(conn, rootGroup)
	if err != nil {
		log.Debug().Err(err).Msgf("cannot discover subgroups for %v", rootGroup)
	} else {
		list = append(list, subgroups...)
	}
	// discover descendant groups
	descgroups, err := groupDescendantGroups(conn, rootGroup)
	if err != nil {
		log.Debug().Err(err).Msgf("cannot discover descendant groups for %v", rootGroup)
	} else {
		list = append(list, descgroups...)
	}
	return list, nil
}

func groupDescendantGroups(conn *GitLabConnection, gid interface{}) ([]*gitlab.Group, error) {
	log.Debug().Msgf("calling list descendant groups with %v", gid)
	perPage := 50
	page := 1
	total := 50
	groups := []*gitlab.Group{}
	for page*perPage <= total {
		grps, resp, err := conn.Client().Groups.ListDescendantGroups(gid, &gitlab.ListDescendantGroupsOptions{ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage}})
		if err != nil {
			return nil, err
		}
		groups = append(groups, grps...)
		total = resp.TotalItems
		page += 1
	}

	return groups, nil
}

func groupSubgroups(conn *GitLabConnection, gid interface{}) ([]*gitlab.Group, error) {
	log.Debug().Msgf("calling list subgroups with %v", gid)
	perPage := 50
	page := 1
	total := 50
	groups := []*gitlab.Group{}
	for page*perPage <= total {
		grps, resp, err := conn.Client().Groups.ListSubGroups(gid, &gitlab.ListSubGroupsOptions{ListOptions: gitlab.ListOptions{Page: page, PerPage: perPage}})
		if err != nil {
			return nil, err
		}
		groups = append(groups, grps...)
		total = resp.TotalItems
		page += 1
	}

	return groups, nil
}
