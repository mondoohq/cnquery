// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"

	"github.com/vmware/govmomi"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

type VsphereConnection struct {
	plugin.Connection
	Conf               *inventory.Config
	asset              *inventory.Asset
	client             *govmomi.Client
	selectedPlatformID string
}

func vSphereConnectionURL(hostname string, port int32, user string, password string) (*url.URL, error) {
	host := hostname
	if port > 0 {
		host = hostname + ":" + strconv.Itoa(int(port))
	}

	u, err := url.Parse("https://" + host + "/sdk")
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(user, password)
	return u, nil
}

func NewVsphereConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*VsphereConnection, error) {
	conn := &VsphereConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	// initialize vSphere connection
	if conf.Type != "vsphere" {
		return nil, errors.New("connection type is not supported for vSphere connection: " + conf.Type)
	}

	// search for password secret
	c, err := vault.GetPassword(conf.Credentials)
	if err != nil {
		return nil, errors.New("missing password for vSphere transport")
	}

	// derive vsphere connection url from Provider Config
	vsphereUrl, err := vSphereConnectionURL(conf.Host, conf.Port, c.User, string(c.Secret))
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, vsphereUrl, true)
	if err != nil {
		return nil, err
	}
	conn.client = client
	conn.selectedPlatformID = conf.PlatformId

	return conn, nil
}

func (c *VsphereConnection) Name() string {
	return "vsphere"
}

func (c *VsphereConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *VsphereConnection) Client() *govmomi.Client {
	return c.client
}
