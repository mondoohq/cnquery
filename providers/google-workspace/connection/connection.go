// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
)

type GoogleWorkspaceConnection struct {
	id       uint32
	parentId *uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
	// Add custom connection fields here
	serviceAccountSubject string
	customerId            string
	cred                  *vault.Credential
}

func NewGoogleWorkspaceConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*GoogleWorkspaceConnection, error) {
	conn := &GoogleWorkspaceConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}
	if len(asset.Connections) > 0 && asset.Connections[0].ParentConnectionId > 0 {
		conn.parentId = &asset.Connections[0].ParentConnectionId
	}

	if len(conf.Credentials) != 0 {
		conn.cred = conf.Credentials[0]
	}

	if conn.cred == nil {
		return nil, errors.New("google workspace provider requires a service account")
	}

	conn.customerId = conf.Options["customer-id"]
	conn.serviceAccountSubject = conf.Options["impersonated-user-email"]

	// check if we have access to the workspace
	_, err := conn.GetWorkspaceCustomer(conn.customerId)
	if err != nil {
		log.Error().Err(err).Msgf("could not access to Google Workspace %s", conn.customerId)
		return nil, err
	}

	return conn, nil
}

func (c *GoogleWorkspaceConnection) Name() string {
	return "google-workspace"
}

func (c *GoogleWorkspaceConnection) ID() uint32 {
	return c.id
}

func (c *GoogleWorkspaceConnection) ParentID() *uint32 {
	return c.parentId
}

func (c *GoogleWorkspaceConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *GoogleWorkspaceConnection) CustomerID() string {
	return c.customerId
}
