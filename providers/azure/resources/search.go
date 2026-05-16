// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/search/armsearch"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscriptionSearchService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionSearchService) id() (string, error) {
	return "azure.subscription.searchService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionSearchServiceService) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSearchService) services() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	ctx := context.Background()
	client, err := armsearch.NewServicesClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	// NewListBySubscriptionPager takes two optional params: *SearchManagementRequestOptions
	// and *ServicesClientListBySubscriptionOptions. Both are left as default (nil).
	pager := client.NewListBySubscriptionPager(nil, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list search services due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, svc := range page.Value {
			if svc == nil {
				continue
			}
			mqlSvc, err := searchServiceToMql(a.MqlRuntime, svc)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSvc)
		}
	}
	return res, nil
}

func searchServiceToMql(runtime *plugin.Runtime, svc *armsearch.Service) (*mqlAzureSubscriptionSearchServiceService, error) {
	sku, err := convert.JsonToDict(svc.SKU)
	if err != nil {
		return nil, err
	}
	identity, err := convert.JsonToDict(svc.Identity)
	if err != nil {
		return nil, err
	}

	var publicNetworkAccess, cmkEnforcement, cmkComplianceStatus string
	var hostingMode, endpoint, provisioningState, status string
	var disableLocalAuth bool
	var replicaCount, partitionCount int64
	networkRuleSet := llx.NilData

	if p := svc.Properties; p != nil {
		if p.PublicNetworkAccess != nil {
			publicNetworkAccess = string(*p.PublicNetworkAccess)
		}
		if p.DisableLocalAuth != nil {
			disableLocalAuth = *p.DisableLocalAuth
		}
		if cmk := p.EncryptionWithCmk; cmk != nil {
			if cmk.Enforcement != nil {
				cmkEnforcement = string(*cmk.Enforcement)
			}
			if cmk.EncryptionComplianceStatus != nil {
				cmkComplianceStatus = string(*cmk.EncryptionComplianceStatus)
			}
		}
		if p.ReplicaCount != nil {
			replicaCount = int64(*p.ReplicaCount)
		}
		if p.PartitionCount != nil {
			partitionCount = int64(*p.PartitionCount)
		}
		if p.HostingMode != nil {
			hostingMode = string(*p.HostingMode)
		}
		if p.Endpoint != nil {
			endpoint = *p.Endpoint
		}
		if p.ProvisioningState != nil {
			provisioningState = string(*p.ProvisioningState)
		}
		if p.Status != nil {
			status = string(*p.Status)
		}
		if p.NetworkRuleSet != nil {
			d, err := convert.JsonToDict(p.NetworkRuleSet)
			if err != nil {
				return nil, err
			}
			networkRuleSet = llx.DictData(d)
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.searchService.service", map[string]*llx.RawData{
		"id":                            llx.StringDataPtr(svc.ID),
		"name":                          llx.StringDataPtr(svc.Name),
		"location":                      llx.StringDataPtr(svc.Location),
		"tags":                          llx.MapData(convert.PtrMapStrToInterface(svc.Tags), types.String),
		"sku":                           llx.DictData(sku),
		"identity":                      llx.DictData(identity),
		"publicNetworkAccess":           llx.StringData(publicNetworkAccess),
		"disableLocalAuth":              llx.BoolData(disableLocalAuth),
		"cmkEnforcement":                llx.StringData(cmkEnforcement),
		"cmkEncryptionComplianceStatus": llx.StringData(cmkComplianceStatus),
		"replicaCount":                  llx.IntData(replicaCount),
		"partitionCount":                llx.IntData(partitionCount),
		"hostingMode":                   llx.StringData(hostingMode),
		"endpoint":                      llx.StringData(endpoint),
		"provisioningState":             llx.StringData(provisioningState),
		"status":                        llx.StringData(status),
		"networkRuleSet":                networkRuleSet,
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSearchServiceService), nil
}
