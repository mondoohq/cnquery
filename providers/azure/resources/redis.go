// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v2"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/azure/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func initAzureSubscriptionCacheService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, fmt.Errorf("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionCacheService) redis() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	subscriptionID := a.SubscriptionId.Data

	clientFactory, err := armredis.NewClientFactory(subscriptionID, token, nil)
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewClient()
	cachePager := client.NewListBySubscriptionPager(nil)
	var caches []any

	for cachePager.More() {
		page, err := cachePager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, cache := range page.Value {
			if cache == nil {
				continue
			}

			rawData, err := createRedisInstanceRawData(cache)
			if err != nil {
				return nil, err
			}

			cacheData, err := CreateResource(
				a.MqlRuntime,
				"azure.subscription.cacheService.redisInstance",
				rawData,
			)
			if err != nil {
				return nil, err
			}
			caches = append(caches, cacheData)
		}
	}

	return caches, nil
}

func createRedisInstanceRawData(cache *armredis.ResourceInfo) (map[string]*llx.RawData, error) {
	properties, err := convert.JsonToDict(cache)
	if err != nil {
		return nil, err
	}

	sku, err := convert.JsonToDict(cache.Properties.SKU)
	if err != nil {
		return nil, err
	}
	// publicNetworkAccess is an enum with possible values: Enabled, Disabled
	var publicNetworkAccess *string
	if cache.Properties.PublicNetworkAccess != nil {
		val := string(*cache.Properties.PublicNetworkAccess)
		publicNetworkAccess = &val
	}
	// provisioningState is an enum with possible values: Creating, Deleting, Failed, Succeeded, Updating
	var provisioningState *string
	if cache.Properties.ProvisioningState != nil {
		val := string(*cache.Properties.ProvisioningState)
		provisioningState = &val
	}

	return map[string]*llx.RawData{
		"id":                  llx.StringDataPtr(cache.ID),
		"name":                llx.StringDataPtr(cache.Name),
		"type":                llx.StringDataPtr(cache.Type),
		"location":            llx.StringDataPtr(cache.Location),
		"properties":          llx.DictData(properties),
		"hostName":            llx.StringDataPtr(cache.Properties.HostName),
		"enableNonSslPort":    llx.BoolDataPtr(cache.Properties.EnableNonSSLPort),
		"publicNetworkAccess": llx.StringDataPtr(publicNetworkAccess),
		"port":                llx.IntDataPtr(cache.Properties.Port),
		"sslPort":             llx.IntDataPtr(cache.Properties.SSLPort),
		"provisioningState":   llx.StringDataPtr(provisioningState),
		"redisVersion":        llx.StringDataPtr(cache.Properties.RedisVersion),
		"replicasPerMaster":   llx.IntDataPtr(cache.Properties.ReplicasPerMaster),
		"replicasPerPrimary":  llx.IntDataPtr(cache.Properties.ReplicasPerPrimary),
		"sku":                 llx.DictData(sku),
		"tags":                llx.MapData(convert.PtrMapStrToInterface(cache.Tags), types.String),
	}, nil
}
