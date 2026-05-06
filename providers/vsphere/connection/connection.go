// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"sync"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"

	"github.com/rs/zerolog/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/rest"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
)

type VsphereConnection struct {
	plugin.Connection
	Conf               *inventory.Config
	asset              *inventory.Asset
	client             *govmomi.Client
	selectedPlatformID string

	restMu     sync.Mutex
	restClient *rest.Client
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
		return nil, plugin.ErrUnsupportedProvider
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

// RestClient returns a logged-in vAPI REST client for this connection,
// creating it on the first call. The same client is reused for every
// vAPI request from this connection (tag lookups, etc.), so a single
// `mql run` doesn't pay the ~800ms vAPI login cost more than once per
// connection.
func (c *VsphereConnection) RestClient(ctx context.Context) (*rest.Client, error) {
	c.restMu.Lock()
	defer c.restMu.Unlock()
	if c.restClient != nil {
		return c.restClient, nil
	}
	creds, err := vault.GetPassword(c.Conf.Credentials)
	if err != nil {
		return nil, err
	}
	rc := rest.NewClient(c.client.Client)
	if err := rc.Login(ctx, url.UserPassword(creds.User, string(creds.Secret))); err != nil {
		return nil, err
	}
	c.restClient = rc
	return rc, nil
}

func (c *VsphereConnection) Close() {
	c.restMu.Lock()
	rc := c.restClient
	c.restClient = nil
	c.restMu.Unlock()
	if rc != nil {
		if err := rc.Logout(context.Background()); err != nil {
			log.Error().Err(err).Msg("failed to logout from vSphere REST session")
		}
	}
	if c.client != nil {
		if err := c.client.Logout(context.Background()); err != nil {
			log.Error().Err(err).Msg("failed to logout from vSphere connection")
		}
	}
}
