// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"fmt"

	clustermgmtapi "github.com/nutanix/ntnx-api-golang-clients/clustermgmt-go-client/v4/api"
	clustermgmtclient "github.com/nutanix/ntnx-api-golang-clients/clustermgmt-go-client/v4/client"
	vmmapi "github.com/nutanix/ntnx-api-golang-clients/vmm-go-client/v4/api"
	vmmclient "github.com/nutanix/ntnx-api-golang-clients/vmm-go-client/v4/client"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const defaultPort = 9440

type NutanixConnection struct {
	plugin.Connection
	Conf     *inventory.Config
	asset    *inventory.Asset
	endpoint string
	port     int
	// SDK clients, one per API namespace
	cmgClient *clustermgmtclient.ApiClient
	vmmClient *vmmclient.ApiClient
}

func NewNutanixConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*NutanixConnection, error) {
	endpoint := conf.Host
	if endpoint == "" {
		endpoint = conf.Options["endpoint"]
	}
	if endpoint == "" {
		return nil, errors.New("missing Prism Central endpoint, use --endpoint")
	}

	port := defaultPort
	if p, ok := conf.Options["port"]; ok && p != "" {
		if _, err := fmt.Sscanf(p, "%d", &port); err != nil {
			return nil, fmt.Errorf("invalid port %q: %w", p, err)
		}
	}

	user := conf.Options["user"]
	apiKey := conf.Options["api-key"]

	var password string
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password {
			if cred.User != "" {
				user = cred.User
			}
			password = string(cred.Secret)
		}
	}

	if apiKey == "" && (user == "" || password == "") {
		return nil, errors.New("missing credentials: provide --user with --password/--ask-pass, or --api-key")
	}

	conn := &NutanixConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		endpoint:   endpoint,
		port:       port,
	}

	cmgClient := clustermgmtclient.NewApiClient()
	cmgClient.Host = endpoint
	cmgClient.Port = port
	cmgClient.SetVerifySSL(!conf.Insecure)
	if apiKey != "" {
		if err := cmgClient.SetApiKey(apiKey); err != nil {
			return nil, err
		}
	} else {
		cmgClient.Username = user
		cmgClient.Password = password
	}
	conn.cmgClient = cmgClient

	vmmClient := vmmclient.NewApiClient()
	vmmClient.Host = endpoint
	vmmClient.Port = port
	vmmClient.SetVerifySSL(!conf.Insecure)
	if apiKey != "" {
		if err := vmmClient.SetApiKey(apiKey); err != nil {
			return nil, err
		}
	} else {
		vmmClient.Username = user
		vmmClient.Password = password
	}
	conn.vmmClient = vmmClient

	return conn, nil
}

func (c *NutanixConnection) Name() string {
	return "nutanix"
}

func (c *NutanixConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *NutanixConnection) Endpoint() string {
	return c.endpoint
}

// ClustersApi returns the cluster-management API for clusters and hosts.
func (c *NutanixConnection) ClustersApi() *clustermgmtapi.ClustersApi {
	return clustermgmtapi.NewClustersApi(c.cmgClient)
}

// VmApi returns the VM-management API for virtual machines.
func (c *NutanixConnection) VmApi() *vmmapi.VmApi {
	return vmmapi.NewVmApi(c.vmmClient)
}
