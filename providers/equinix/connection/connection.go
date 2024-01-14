// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"os"

	"github.com/packethost/packngo"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
)

type ResourceType int

const (
	Unknown ResourceType = iota
	Project
	Organization
)

type EquinixConnection struct {
	id    uint32
	Conf  *inventory.Config
	asset *inventory.Asset
	// custom connection fields
	client       *packngo.Client
	resourceType ResourceType
	resourceID   string
	project      *packngo.Project
	org          *packngo.Organization
}

func NewEquinixConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*EquinixConnection, error) {
	conn := &EquinixConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}

	// if a secret was provided, it always overrides the env variable since it has precedence
	token := os.Getenv("PACKET_AUTH_TOKEN")
	if len(conf.Credentials) > 0 {
		for i := range conf.Credentials {
			cred := conf.Credentials[i]
			if cred.Type == vault.CredentialType_password {
				token = string(cred.Secret)
			} else {
				log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for Equinix provider")
			}
		}
	}
	if token == "" {
		return nil, errors.New("a valid Equinix token is required, pass --token '<yourtoken>' or set PACKET_AUTH_TOKEN environment variable")
	}

	// initialize your connection
	if conf.Type != "equinix" {
		return nil, plugin.ErrProviderTypeDoesNotMatch
	}

	c, err := packngo.NewClient(packngo.WithAuth("packngo lib", token))
	if err != nil {
		return nil, err
	}

	if conf.Options["project-id"] != "" {
		projectId := conf.Options["project-id"]
		// NOTE: we cannot check the project itself because it throws a 404
		// https://github.com/packethost/packngo/issues/245
		//project, _, err := c.Projects.Get(projectId, nil)
		//if err != nil {
		//	return nil, errors.Wrap(err, "could not find the requested equinix project: "+projectId)
		//}

		ps, _, err := c.Projects.List(nil)
		if err != nil {
			return nil, errors.Join(errors.New("cannot retrieve equinix projects"), err)
		}

		var project *packngo.Project
		for _, p := range ps {
			if p.ID == projectId {
				project = &p
			}
		}
		if project == nil {
			return nil, errors.Join(errors.New("could not find the requested equinix project: "+projectId), err)
		}
		conn.resourceID = projectId
		conn.resourceType = Project
		conn.project = project
	} else if conf.Options["org-id"] != "" {
		orgId := conf.Options["org-id"]
		org, _, err := c.Organizations.Get(orgId, nil)
		if err != nil {
			return nil, errors.Join(errors.New("could not find the requested equinix organization: "+orgId), err)
		}
		conn.resourceID = orgId
		conn.resourceType = Organization
		conn.org = org
	} else {
		return nil, errors.New("equinix provider requires an project id or organization id")
	}

	conn.client = c
	return conn, nil
}

func (c *EquinixConnection) Name() string {
	return "equinix"
}

func (c *EquinixConnection) ID() uint32 {
	return c.id
}

func (c *EquinixConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *EquinixConnection) Client() *packngo.Client {
	return c.client
}

func (c *EquinixConnection) Project() *packngo.Project {
	return c.project
}

func (c *EquinixConnection) Organization() *packngo.Organization {
	if c.resourceType == Organization {
		return c.org
	} else if c.resourceType == Project {
		if c.org != nil {
			return c.org
		}

		client := c.Client()
		// NOTE: if we are going to support multiple projects, we need to change this logic
		project := c.Project()

		// we need to list the organization to circumvent the get issue
		// if we request the project and try to access the org, it only returns the url
		// its similar to https://github.com/packethost/packngo/issues/245
		var org *packngo.Organization
		orgs, _, err := client.Organizations.List(nil)
		if err != nil {
			return nil
		}

		for i := range orgs {
			o := orgs[i]
			if o.URL == project.Organization.URL {
				org = &o
				break
			}
		}
		c.org = org
		return org
	}
	return nil
}

func (c *EquinixConnection) Identifier() (string, error) {
	switch c.resourceType {
	case Project:
		return "//platformid.api.mondoo.app/runtime/equinix/projects/" + c.resourceID, nil
	case Organization:
		return "//platformid.api.mondoo.app/runtime/equinix/organizations/" + c.resourceID, nil
	default:
		return "", errors.New("unknown resource type")
	}
}
