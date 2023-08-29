// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/pkg/errors"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
)

type microsoftAssetType int32

const (
	OptionTenantID         = "tenant-id"
	OptionClientID         = "client-id"
	OptionDataReport       = "mondoo-ms365-datareport"
	OptionSubscriptionID   = "subscription-id"
	OptionPlatformOverride = "platform-override"
)

const (
	ms365 microsoftAssetType = 0
	azure microsoftAssetType = 1
)

type AzureConnection struct {
	id    uint32
	Conf  *inventory.Config
	asset *inventory.Asset
	token azcore.TokenCredential
	// note: in the future, we might make this optional if we have a tenant-level asset.
	subscriptionId          string
	subscriptionDisplayName string
	tenantId                string
	clientId                string
}

func NewAzureConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*AzureConnection, error) {
	var assetType microsoftAssetType
	if conf.Type == "azure" {
		assetType = azure
	} else if conf.Type == "ms365" {
		assetType = ms365
	}
	tenantId := conf.Options[OptionTenantID]
	clientId := conf.Options[OptionClientID]
	// we need credentials for ms365. for azure these are optional, we fallback to the AZ cli (if installed)
	if assetType == ms365 && (len(conf.Credentials) != 1 || conf.Credentials[0] == nil) {
		return nil, errors.New("microsoft provider requires a credentials file, pass path via --certificate-path option")
	}

	var cred *vault.Credential
	if len(conf.Credentials) != 0 {
		cred = conf.Credentials[0]
	}

	if assetType == ms365 && len(tenantId) == 0 {
		return nil, errors.New("ms365 backend requires a tenant-id")
	}
	token, err := getTokenCredential(cred, tenantId, clientId)
	if err != nil {
		return nil, errors.Wrap(err, "cannot fetch credentials for microsoft provider")
	}
	subsClient := NewSubscriptionsClient(token)
	subs, err := subsClient.GetSubscriptions(SubscriptionsFilter{})
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, errors.New("cannot find an azure subscription")
	}

	// TODO: discover other subs too once we can do > 1 assets
	sub := subs[0]

	subDisplayName := *sub.SubscriptionID
	if sub.DisplayName != nil {
		subDisplayName = *sub.DisplayName
	}
	return &AzureConnection{
		Conf:                    conf,
		id:                      id,
		asset:                   asset,
		token:                   token,
		subscriptionId:          *sub.SubscriptionID,
		tenantId:                *sub.TenantID,
		subscriptionDisplayName: subDisplayName,
		clientId:                clientId,
	}, nil
}

func (h *AzureConnection) Name() string {
	return "azure"
}

func (h *AzureConnection) ID() uint32 {
	return h.id
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

func (p *AzureConnection) SubName() string {
	return p.subscriptionDisplayName
}
