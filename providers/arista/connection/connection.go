// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"

	"github.com/aristanetworks/goeapi"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
)

type AristaConnection struct {
	id       uint32
	parentId *uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
	// custom connection fields
	node *goeapi.Node
}

func NewAristaConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AristaConnection, error) {
	conn := &AristaConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}
	if len(asset.Connections) > 0 && asset.Connections[0].ParentConnectionId > 0 {
		conn.parentId = &asset.Connections[0].ParentConnectionId
	}

	// initialize connection
	port := conf.Port
	if port == 0 {
		port = goeapi.UseDefaultPortNum
	}

	if len(conf.Credentials) == 0 {
		return nil, errors.New("missing password for arista connection")
	}

	// search for password secret
	c, err := vault.GetPassword(conf.Credentials)
	if err != nil {
		return nil, errors.New("missing password for arista connection")
	}

	// NOTE: we explicitly do not support http, since there is no real reason to support http
	// the goeapi is always running in insecure mode since it does not verify the server
	// setup which allows potential man-in-the-middle attacks, consider opening a PR
	// https://github.com/aristanetworks/goeapi/blob/7944bcedaf212bb60e5f9baaf471469f49113f47/eapilib.go#L527
	node, err := goeapi.Connect("https", conf.Host, c.User, string(c.Secret), int(port))
	if err != nil {
		return nil, err
	}

	conn.node = node
	return conn, nil
}

func (c *AristaConnection) Name() string {
	return "arista"
}

func (c *AristaConnection) ID() uint32 {
	return c.id
}

func (c *AristaConnection) ParentID() *uint32 {
	return c.parentId
}

func (c *AristaConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *AristaConnection) Client() *goeapi.Node {
	return c.node
}

func (c *AristaConnection) GetVersion() (ShowVersion, error) {
	return GetVersion(c.node)
}
