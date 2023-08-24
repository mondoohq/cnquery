// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
)

type GoogleWorkspaceConnection struct {
	id    uint32
	Conf  *inventory.Config
	asset *inventory.Asset
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

	return conn, nil
}

func (c *GoogleWorkspaceConnection) Name() string {
	return "google-workspace"
}

func (c *GoogleWorkspaceConnection) ID() uint32 {
	return c.id
}

func (c *GoogleWorkspaceConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *GoogleWorkspaceConnection) CustomerID() string {
	return c.customerId
}
