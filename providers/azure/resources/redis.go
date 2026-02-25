// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionCacheService) id() (string, error) {
	return "azure.subscription.cache/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstance) id() (string, error) {
	return a.Id.Data, nil
}

func initAzureSubscriptionCacheService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func initAzureSubscriptionCacheServiceRedisInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure cache redis instance")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.cacheService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	cacheSvc := res.(*mqlAzureSubscriptionCacheService)
	redisList := cacheSvc.GetRedis()
	if redisList.Error != nil {
		return nil, nil, redisList.Error
	}
	id, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("id must be a non-nil string value")
	}
	for _, entry := range redisList.Data {
		instance := entry.(*mqlAzureSubscriptionCacheServiceRedisInstance)
		if instance.Id.Data == id {
			return args, instance, nil
		}
	}

	return nil, nil, errors.New("azure cache redis instance does not exist")
}

func (a *mqlAzureSubscriptionCacheService) redis() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	subscriptionID := a.SubscriptionId.Data

	clientFactory, err := armredis.NewClientFactory(subscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
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

			rawData, err := createRedisInstanceRawData(a.MqlRuntime, cache)
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

func createRedisInstanceRawData(runtime *plugin.Runtime, cache *armredis.ResourceInfo) (map[string]*llx.RawData, error) {
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
	// minimumTlsVersion is an enum with possible values: "1.0", "1.1", "1.2"
	var minimumTlsVersion *string
	if cache.Properties.MinimumTLSVersion != nil {
		val := string(*cache.Properties.MinimumTLSVersion)
		minimumTlsVersion = &val
	}

	redisConfiguration, err := convert.JsonToDict(cache.Properties.RedisConfiguration)
	if err != nil {
		return nil, err
	}

	identity, err := convert.JsonToDict(cache.Identity)
	if err != nil {
		return nil, err
	}

	zones := []any{}
	for _, z := range cache.Zones {
		if z != nil {
			zones = append(zones, *z)
		}
	}

	// Map private endpoint connections as typed sub-resources
	privateEndpointConns := []any{}
	if runtime != nil {
		for _, pec := range cache.Properties.PrivateEndpointConnections {
			if pec == nil {
				continue
			}
			var privateEndpointId *string
			if pec.Properties != nil && pec.Properties.PrivateEndpoint != nil {
				privateEndpointId = pec.Properties.PrivateEndpoint.ID
			}
			var status *string
			if pec.Properties != nil && pec.Properties.PrivateLinkServiceConnectionState != nil && pec.Properties.PrivateLinkServiceConnectionState.Status != nil {
				val := string(*pec.Properties.PrivateLinkServiceConnectionState.Status)
				status = &val
			}
			var description *string
			if pec.Properties != nil && pec.Properties.PrivateLinkServiceConnectionState != nil {
				description = pec.Properties.PrivateLinkServiceConnectionState.Description
			}
			var pecProvisioningState *string
			if pec.Properties != nil && pec.Properties.ProvisioningState != nil {
				val := string(*pec.Properties.ProvisioningState)
				pecProvisioningState = &val
			}
			pecResource, err := CreateResource(runtime, "azure.subscription.cacheService.redisInstance.privateEndpointConnection",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(pec.ID),
					"name":              llx.StringDataPtr(pec.Name),
					"type":              llx.StringDataPtr(pec.Type),
					"privateEndpointId": llx.StringDataPtr(privateEndpointId),
					"status":            llx.StringDataPtr(status),
					"description":       llx.StringDataPtr(description),
					"provisioningState": llx.StringDataPtr(pecProvisioningState),
				})
			if err != nil {
				return nil, err
			}
			privateEndpointConns = append(privateEndpointConns, pecResource)
		}
	}

	return map[string]*llx.RawData{
		"id":                         llx.StringDataPtr(cache.ID),
		"name":                       llx.StringDataPtr(cache.Name),
		"type":                       llx.StringDataPtr(cache.Type),
		"location":                   llx.StringDataPtr(cache.Location),
		"properties":                 llx.DictData(properties),
		"hostName":                   llx.StringDataPtr(cache.Properties.HostName),
		"enableNonSslPort":           llx.BoolDataPtr(cache.Properties.EnableNonSSLPort),
		"publicNetworkAccess":        llx.StringDataPtr(publicNetworkAccess),
		"port":                       llx.IntDataPtr(cache.Properties.Port),
		"sslPort":                    llx.IntDataPtr(cache.Properties.SSLPort),
		"provisioningState":          llx.StringDataPtr(provisioningState),
		"redisVersion":               llx.StringDataPtr(cache.Properties.RedisVersion),
		"replicasPerMaster":          llx.IntDataPtr(cache.Properties.ReplicasPerMaster),
		"replicasPerPrimary":         llx.IntDataPtr(cache.Properties.ReplicasPerPrimary),
		"sku":                        llx.DictData(sku),
		"tags":                       llx.MapData(convert.PtrMapStrToInterface(cache.Tags), types.String),
		"minimumTlsVersion":          llx.StringDataPtr(minimumTlsVersion),
		"redisConfiguration":         llx.DictData(redisConfiguration),
		"shardCount":                 llx.IntDataPtr(cache.Properties.ShardCount),
		"staticIp":                   llx.StringDataPtr(cache.Properties.StaticIP),
		"subnetId":                   llx.StringDataPtr(cache.Properties.SubnetID),
		"zones":                      llx.ArrayData(zones, types.String),
		"identity":                   llx.DictData(identity),
		"privateEndpointConnections": llx.ArrayData(privateEndpointConns, types.Resource("azure.subscription.cacheService.redisInstance.privateEndpointConnection")),
	}, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstance) firewallRules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	cacheName, err := resourceID.Component("redis")
	if err != nil {
		return nil, err
	}

	firewallClient, err := armredis.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := firewallClient.NewListPager(resourceID.ResourceGroup, cacheName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return []any{}, nil
			}
			return nil, err
		}
		for _, rule := range page.Value {
			if rule == nil {
				continue
			}
			var startIP, endIP *string
			if rule.Properties != nil {
				startIP = rule.Properties.StartIP
				endIP = rule.Properties.EndIP
			}
			args := map[string]*llx.RawData{
				"id":             llx.StringDataPtr(rule.ID),
				"name":           llx.StringDataPtr(rule.Name),
				"type":           llx.StringDataPtr(rule.Type),
				"startIpAddress": llx.StringDataPtr(startIP),
				"endIpAddress":   llx.StringDataPtr(endIP),
			}
			mqlRule, err := CreateResource(a.MqlRuntime, "azure.subscription.cacheService.redisInstance.firewallRule", args)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRule)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstance) patchSchedules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	cacheName, err := resourceID.Component("redis")
	if err != nil {
		return nil, err
	}

	patchClient, err := armredis.NewPatchSchedulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := patchClient.NewListByRedisResourcePager(resourceID.ResourceGroup, cacheName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return []any{}, nil
			}
			return nil, err
		}
		for _, schedule := range page.Value {
			if schedule == nil {
				continue
			}

			entries := []any{}
			if schedule.Properties != nil {
				for _, entry := range schedule.Properties.ScheduleEntries {
					if entry == nil {
						continue
					}
					entryDict, err := convert.JsonToDict(entry)
					if err != nil {
						return nil, err
					}
					entries = append(entries, entryDict)
				}
			}

			mqlSchedule, err := CreateResource(a.MqlRuntime, "azure.subscription.cacheService.redisInstance.patchSchedule",
				map[string]*llx.RawData{
					"id":       llx.StringDataPtr(schedule.ID),
					"name":     llx.StringDataPtr(schedule.Name),
					"location": llx.StringDataPtr(schedule.Location),
					"entries":  llx.ArrayData(entries, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSchedule)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstanceFirewallRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstancePatchSchedule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstancePrivateEndpointConnection) id() (string, error) {
	return a.Id.Data, nil
}
