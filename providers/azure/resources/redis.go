// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v4"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionCacheService) id() (string, error) {
	return "azure.subscription.cache/" + a.SubscriptionId.Data, nil
}

type mqlAzureSubscriptionCacheServiceRedisInstanceInternal struct {
	// cacheEncryptionKeyURI stores the customer-managed key URI for encryption.
	// Note: armredis/v3 does not expose encryption fields on Properties.
	// This will be populated when the SDK adds support for CMK encryption configuration.
	cacheEncryptionKeyURI           string
	cachePrivateEndpointConnections []*armredis.PrivateEndpointConnection
	cacheUserAssignedIdentityIds    []string
	cacheSystemData                 any
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstance) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstance) userAssignedIdentities() ([]any, error) {
	return resolveUserAssignedIdentities(a.MqlRuntime, a.cacheUserAssignedIdentityIds)
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstance) subnet() (*mqlAzureSubscriptionNetworkServiceSubnet, error) {
	if a.SubnetId.Data == "" {
		a.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.subnet",
		map[string]*llx.RawData{"id": llx.StringData(a.SubnetId.Data)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceSubnet), nil
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
	id, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("id must be a non-nil string value")
	}
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, nil, err
	}
	cacheName, err := resourceID.Component("Redis")
	if err != nil {
		return nil, nil, err
	}

	clientFactory, err := armredis.NewClientFactory(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, err
	}
	resp, err := clientFactory.NewClient().Get(context.Background(), resourceID.ResourceGroup, cacheName, nil)
	if err != nil {
		return nil, nil, err
	}

	rawData, err := createRedisInstanceRawData(runtime, &resp.ResourceInfo)
	if err != nil {
		return nil, nil, err
	}
	res, err := CreateResource(runtime, "azure.subscription.cacheService.redisInstance", rawData)
	if err != nil {
		return nil, nil, err
	}
	mqlRedis := res.(*mqlAzureSubscriptionCacheServiceRedisInstance)
	if resp.Properties != nil {
		mqlRedis.cachePrivateEndpointConnections = resp.Properties.PrivateEndpointConnections
	}
	if resp.Identity != nil {
		mqlRedis.cacheUserAssignedIdentityIds = sortedUserAssignedIdentityIDs(resp.Identity.UserAssignedIdentities)
	}
	sysData, err := convert.JsonToDict(resp.ResourceInfo.SystemData)
	if err != nil {
		return nil, nil, err
	}
	mqlRedis.cacheSystemData = sysData
	return args, mqlRedis, nil
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
			mqlRedis := cacheData.(*mqlAzureSubscriptionCacheServiceRedisInstance)
			if cache.Properties != nil {
				mqlRedis.cachePrivateEndpointConnections = cache.Properties.PrivateEndpointConnections
			}
			if cache.Identity != nil {
				mqlRedis.cacheUserAssignedIdentityIds = sortedUserAssignedIdentityIDs(cache.Identity.UserAssignedIdentities)
			}
			sysData, err := convert.JsonToDict(cache.SystemData)
			if err != nil {
				return nil, err
			}
			mqlRedis.cacheSystemData = sysData
			caches = append(caches, mqlRedis)
		}
	}

	return caches, nil
}

func createRedisInstanceRawData(runtime *plugin.Runtime, cache *armredis.ResourceInfo) (map[string]*llx.RawData, error) {
	// Properties is a nullable pointer that the field reads below dereference
	// throughout; normalize to an empty struct so a cache returned without
	// properties doesn't panic here (the callers reach this before their own
	// nil guards).
	if cache.Properties == nil {
		cache.Properties = &armredis.Properties{}
	}
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

	var principalId *string
	if cache.Identity != nil {
		principalId = cache.Identity.PrincipalID
	}

	zones := []any{}
	for _, z := range cache.Zones {
		if z != nil {
			zones = append(zones, *z)
		}
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
		"minimumTlsVersion":   llx.StringDataPtr(minimumTlsVersion),
		"redisConfiguration":  llx.DictData(redisConfiguration),
		"shardCount":          llx.IntDataPtr(cache.Properties.ShardCount),
		"staticIp":            llx.StringDataPtr(cache.Properties.StaticIP),
		"subnetId":            llx.StringDataPtr(cache.Properties.SubnetID),
		"zones":               llx.ArrayData(zones, types.String),
		"identity":            llx.DictData(identity),
		"principalId":         llx.StringDataPtr(principalId),
	}, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstance) privateEndpointConnections() ([]any, error) {
	res := []any{}
	for _, pec := range a.cachePrivateEndpointConnections {
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
		groupIds := []any{}
		if pec.Properties != nil {
			for _, g := range pec.Properties.GroupIDs {
				if g != nil {
					groupIds = append(groupIds, *g)
				}
			}
		}
		pecResource, err := CreateResource(a.MqlRuntime, "azure.subscription.cacheService.redisInstance.privateEndpointConnection",
			map[string]*llx.RawData{
				"id":                llx.StringDataPtr(pec.ID),
				"name":              llx.StringDataPtr(pec.Name),
				"type":              llx.StringDataPtr(pec.Type),
				"privateEndpointId": llx.StringDataPtr(privateEndpointId),
				"status":            llx.StringDataPtr(status),
				"description":       llx.StringDataPtr(description),
				"provisioningState": llx.StringDataPtr(pecProvisioningState),
				"groupIds":          llx.ArrayData(groupIds, types.String),
			})
		if err != nil {
			return nil, err
		}
		sysData, err := convert.JsonToDict(pec.SystemData)
		if err != nil {
			return nil, err
		}
		pecResource.(*mqlAzureSubscriptionCacheServiceRedisInstancePrivateEndpointConnection).cacheSystemData = sysData
		res = append(res, pecResource)
	}
	return res, nil
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
			sysData, err := convert.JsonToDict(rule.SystemData)
			if err != nil {
				return nil, err
			}
			mqlRule.(*mqlAzureSubscriptionCacheServiceRedisInstanceFirewallRule).cacheSystemData = sysData
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
			sysData, err := convert.JsonToDict(schedule.SystemData)
			if err != nil {
				return nil, err
			}
			mqlSchedule.(*mqlAzureSubscriptionCacheServiceRedisInstancePatchSchedule).cacheSystemData = sysData
			res = append(res, mqlSchedule)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstance) encryptionKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.cacheEncryptionKeyURI == "" {
		a.EncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.cacheEncryptionKeyURI)
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstanceFirewallRule) id() (string, error) {
	return a.Id.Data, nil
}

type mqlAzureSubscriptionCacheServiceRedisInstanceFirewallRuleInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstanceFirewallRule) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstancePatchSchedule) id() (string, error) {
	return a.Id.Data, nil
}

type mqlAzureSubscriptionCacheServiceRedisInstancePatchScheduleInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstancePatchSchedule) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstancePrivateEndpointConnection) id() (string, error) {
	return a.Id.Data, nil
}

type mqlAzureSubscriptionCacheServiceRedisInstancePrivateEndpointConnectionInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstancePrivateEndpointConnection) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCacheServiceRedisInstancePrivateEndpointConnection) privateEndpoint() (*mqlAzureSubscriptionNetworkServicePrivateEndpoint, error) {
	peId := a.PrivateEndpointId.Data
	if peId == "" {
		a.PrivateEndpoint.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.privateEndpoint", map[string]*llx.RawData{
		"id": llx.StringData(peId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServicePrivateEndpoint), nil
}
