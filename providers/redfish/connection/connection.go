// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"fmt"

	"github.com/stmcginnis/gofish"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const DefaultPort = 443

type RedfishConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	client *gofish.APIClient
	vendor Vendor
	id     string
}

func NewRedfishConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*RedfishConnection, error) {
	conn := &RedfishConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	if conf == nil || conf.Type != "redfish" {
		return nil, errors.New("provider type does not match")
	}
	if conf.Host == "" {
		return nil, errors.New("missing host for redfish provider")
	}

	cred, err := vault.GetPassword(conf.Credentials)
	if err != nil {
		return nil, errors.New("missing password for redfish provider")
	}

	port := conf.Port
	if port == 0 {
		port = DefaultPort
	}

	insecure := conf.Options != nil && conf.Options["insecure"] == "true"

	client, err := gofish.Connect(gofish.ClientConfig{
		Endpoint: fmt.Sprintf("https://%s:%d", conf.Host, port),
		Username: cred.User,
		Password: string(cred.Secret),
		Insecure: insecure,
	})
	if err != nil {
		return nil, err
	}

	conn.client = client
	conn.vendor = detectVendorFromService(client)
	return conn, nil
}

func (c *RedfishConnection) Name() string {
	return "redfish"
}

func (c *RedfishConnection) Asset() *inventory.Asset {
	return c.asset
}

// Client returns the connected gofish API client.
func (c *RedfishConnection) Client() *gofish.APIClient {
	return c.client
}

// Vendor returns the detected hardware vendor.
func (c *RedfishConnection) Vendor() Vendor {
	return c.vendor
}

// Close logs out the Redfish session.
func (c *RedfishConnection) Close() {
	if c.client != nil {
		c.client.Logout()
	}
}

// Identifier derives a stable platform ID from the first system or manager
// UUID, falling back to the host when neither is available.
func (c *RedfishConnection) Identifier() (string, error) {
	if c.id != "" {
		return c.id, nil
	}

	uid := ""
	if c.client != nil && c.client.Service != nil {
		if systems, err := c.client.Service.Systems(); err == nil {
			for _, s := range systems {
				if s.UUID != "" {
					uid = s.UUID
					break
				}
			}
		}
		if uid == "" {
			if managers, err := c.client.Service.Managers(); err == nil {
				for _, m := range managers {
					if m.UUID != "" {
						uid = m.UUID
						break
					}
				}
			}
		}
	}
	if uid == "" {
		uid = c.Conf.Host
	}

	c.id = "//platformid.api.mondoo.app/runtime/redfish/uuid/" + uid
	return c.id, nil
}
