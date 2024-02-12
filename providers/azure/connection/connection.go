// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/azure/connection/auth"
	"go.mondoo.com/cnquery/v10/providers/azure/connection/shared"
)

const (
	OptionTenantID         = "tenant-id"
	OptionClientID         = "client-id"
	OptionDataReport       = "mondoo-ms365-datareport"
	OptionSubscriptionID   = "subscription-id"
	OptionPlatformOverride = "platform-override"
)

type AzureConnection struct {
	id       uint32
	parentId *uint32
	Conf     *inventory.Config
	asset    *inventory.Asset
	token    azcore.TokenCredential
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

	token, err := auth.GetTokenCredential(cred, tenantId, clientId)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch credentials for microsoft provider")
	}
	conn := &AzureConnection{
		Conf:           conf,
		id:             id,
		asset:          asset,
		token:          token,
		subscriptionId: subId,
	}
	if len(asset.Connections) > 0 && asset.Connections[0].ParentConnectionId > 0 {
		conn.parentId = &asset.Connections[0].ParentConnectionId
	}
	return conn, nil
}

func (h *AzureConnection) Name() string {
	return "azure"
}

func (h *AzureConnection) ID() uint32 {
	return h.id
}

func (p *AzureConnection) ParentID() *uint32 {
	return p.parentId
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
