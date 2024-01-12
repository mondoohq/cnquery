// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
)

type OpcuaConnection struct {
	id       uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
	client   *opcua.Client
	endpoint string
}

func NewOpcuaConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*OpcuaConnection, error) {
	conn := &OpcuaConnection{
		Conf:  conf,
		id:    id,
		asset: asset,
	}

	// initialize connection
	if conf.Type != "opcua" {
		return nil, plugin.ErrProviderTypeDoesNotMatch
	}

	if conf.Options == nil || conf.Options["endpoint"] == "" {
		return nil, errors.New("opcua provider requires an endpoint. please set option `endpoint`")
	}

	endpoint := conf.Options["endpoint"]

	policy := "None" // None, Basic128Rsa15, Basic256, Basic256Sha256. Default: auto"
	mode := "None"   //  None, Sign, SignAndEncrypt. Default: auto
	// certFile := "created/server_cert.der"
	// keyFile := "created/server_key.der"

	ctx := context.Background()

	endpoints, err := opcua.GetEndpoints(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	ep := opcua.SelectEndpoint(endpoints, policy, ua.MessageSecurityModeFromString(mode))
	if ep == nil {
		return nil, errors.New("failed to find suitable endpoint")
	}

	opts := []opcua.Option{
		opcua.SecurityPolicy(policy),
		opcua.SecurityModeString(mode),
		// opcua.CertificateFile(certFile),
		// opcua.PrivateKeyFile(keyFile),
		opcua.AuthAnonymous(),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
	}

	c, err := opcua.NewClient(endpoint, opts...)
	if err != nil {
		return nil, err
	}
	if err := c.Connect(ctx); err != nil {
		return nil, err
	}

	conn.client = c
	conn.endpoint = endpoint

	return conn, nil
}

func (c *OpcuaConnection) Name() string {
	return "opcua"
}

func (c *OpcuaConnection) ID() uint32 {
	return c.id
}

func (c *OpcuaConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *OpcuaConnection) Client() *opcua.Client {
	return c.client
}

func (c *OpcuaConnection) Endpoint() string {
	return c.endpoint
}
