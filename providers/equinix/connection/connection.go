// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"github.com/packethost/packngo"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
	"os"
)

type EquinixConnection struct {
	id    uint32
	Conf  *inventory.Config
	asset *inventory.Asset
	// custom connection fields
	client    *packngo.Client
	projectId string
	project   *packngo.Project
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

	projectId := conf.Options["projectID"]

	if conf.Options == nil || len(projectId) == 0 {
		return nil, errors.New("equinix provider requires an project id")
	}

	c, err := packngo.NewClient(packngo.WithAuth("packngo lib", token))
	if err != nil {
		return nil, err
	}

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

	conn.client = c
	conn.projectId = projectId
	conn.project = project

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

func (c *EquinixConnection) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/equinix/projects/" + c.projectId, nil
}
