// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/azauth"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v11/providers/azure/connection/shared"
)

const (
	OptionTenantID         = "tenant-id"
	OptionClientID         = "client-id"
	OptionDataReport       = "mondoo-ms365-datareport"
	OptionSubscriptionID   = "subscription-id"
	OptionPlatformOverride = "platform-override"
)

type AzureConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset
	token azcore.TokenCredential
	// note: in the future, we might make this optional if we have a tenant-level asset.
	subscriptionId string
	clientOptions  policy.ClientOptions
}

func NewAzureConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AzureConnection, error) {
	tenantId := conf.Options[OptionTenantID]
	clientId := conf.Options[OptionClientID]
	subId := conf.Options[OptionSubscriptionID]

	var cred *vault.Credential
	if len(conf.Credentials) != 0 {
		cred = conf.Credentials[0]
	}

	token, err := azauth.GetTokenFromCredential(cred, tenantId, clientId)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch credentials for microsoft provider")
	}
	return &AzureConnection{
		Connection:     plugin.NewConnection(id, asset),
		Conf:           conf,
		asset:          asset,
		token:          token,
		subscriptionId: subId,
	}, nil
}

func (h *AzureConnection) Name() string {
	return "azure"
}

func (p *AzureConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *AzureConnection) SubId() string {
	return p.subscriptionId
}

func (p *AzureConnection) Token() azcore.TokenCredential {
	return p.token
}

func (p *AzureConnection) PlatformId() string {
	return "//platformid.api.mondoo.app/runtime/azure/subscriptions/" + p.subscriptionId
}

func (p *AzureConnection) ClientOptions() policy.ClientOptions {
	return p.clientOptions
}

func (p *AzureConnection) Config() *inventory.Config {
	return p.Conf
}

func (p *AzureConnection) Type() shared.ConnectionType {
	return "azure"
}
