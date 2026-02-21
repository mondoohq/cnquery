// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v3"
	"github.com/stretchr/testify/assert"
)

// Helper function to create pointers using generics
func ptr[T any](v T) *T {
	return &v
}

func TestAzureSubscriptionCacheServiceRedis(t *testing.T) {
	// Create mock data
	mockRedisCache := &armredis.ResourceInfo{
		ID:       ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Cache/redis/test-cache"),
		Name:     ptr("test-cache"),
		Type:     ptr("Microsoft.Cache/redis"),
		Location: ptr("eastus"),
		Zones:    []*string{ptr("1"), ptr("2")},
		Identity: &armredis.ManagedServiceIdentity{
			Type:        ptr(armredis.ManagedServiceIdentityTypeSystemAssigned),
			PrincipalID: ptr("principal-123"),
			TenantID:    ptr("tenant-456"),
		},
		Properties: &armredis.Properties{
			HostName:            ptr("test-cache.redis.cache.windows.net"),
			EnableNonSSLPort:    ptr(true),
			PublicNetworkAccess: ptr(armredis.PublicNetworkAccessEnabled),
			Port:                ptr(int32(6379)),
			SSLPort:             ptr(int32(6380)),
			ProvisioningState:   ptr(armredis.ProvisioningStateSucceeded),
			RedisVersion:        ptr("6.0"),
			ReplicasPerMaster:   ptr(int32(1)),
			ReplicasPerPrimary:  ptr(int32(1)),
			MinimumTLSVersion:   ptr(armredis.TLSVersionOne2),
			ShardCount:          ptr(int32(3)),
			StaticIP:            ptr("10.0.0.5"),
			SubnetID:            ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
			RedisConfiguration: &armredis.CommonPropertiesRedisConfiguration{
				MaxmemoryPolicy:  ptr("allkeys-lru"),
				RdbBackupEnabled: ptr("true"),
			},
			SKU: &armredis.SKU{
				Name:     ptr(armredis.SKUNameStandard),
				Capacity: ptr(int32(1)),
				Family:   ptr(armredis.SKUFamilyC),
			},
		},
		Tags: map[string]*string{
			"Environment": ptr("Test"),
			"Project":     ptr("Redis-Test"),
		},
	}

	t.Run("TestRedisDataConversion", func(t *testing.T) {
		// Test the actual conversion function from redis.go
		rawData, err := createRedisInstanceRawData(nil, mockRedisCache)
		assert.NoError(t, err)
		assert.NotNil(t, rawData)

		// Verify the converted data matches our mock input
		assert.Equal(t, "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Cache/redis/test-cache", rawData["id"].Value)
		assert.Equal(t, "test-cache", rawData["name"].Value)
		assert.Equal(t, "Microsoft.Cache/redis", rawData["type"].Value)
		assert.Equal(t, "eastus", rawData["location"].Value)
		assert.Equal(t, "test-cache.redis.cache.windows.net", rawData["hostName"].Value)
		assert.Equal(t, true, rawData["enableNonSslPort"].Value)
		assert.Equal(t, "Enabled", rawData["publicNetworkAccess"].Value)
		assert.Equal(t, int64(6379), rawData["port"].Value)
		assert.Equal(t, int64(6380), rawData["sslPort"].Value)
		assert.Equal(t, "Succeeded", rawData["provisioningState"].Value)
		assert.Equal(t, "6.0", rawData["redisVersion"].Value)
		assert.Equal(t, int64(1), rawData["replicasPerMaster"].Value)
		assert.Equal(t, int64(1), rawData["replicasPerPrimary"].Value)

		// Verify properties dict is not nil
		assert.NotNil(t, rawData["properties"].Value)

		// Verify SKU dict is not nil
		assert.NotNil(t, rawData["sku"].Value)

		// Verify tags are converted properly
		assert.NotNil(t, rawData["tags"].Value)

		// Verify new fields
		assert.Equal(t, "1.2", rawData["minimumTlsVersion"].Value)
		assert.Equal(t, int64(3), rawData["shardCount"].Value)
		assert.Equal(t, "10.0.0.5", rawData["staticIp"].Value)
		assert.Equal(t, "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", rawData["subnetId"].Value)

		// Verify zones
		zones := rawData["zones"].Value.([]any)
		assert.Len(t, zones, 2)
		assert.Equal(t, "1", zones[0])
		assert.Equal(t, "2", zones[1])

		// Verify redisConfiguration dict is not nil
		assert.NotNil(t, rawData["redisConfiguration"].Value)

		// Verify identity dict is not nil
		assert.NotNil(t, rawData["identity"].Value)

		// Verify privateEndpointConnections is empty array (no PECs in mock data)
		pecs := rawData["privateEndpointConnections"].Value.([]any)
		assert.Len(t, pecs, 0)
	})

	t.Run("TestEnumConversions", func(t *testing.T) {
		// Test enum to string conversions
		publicNetworkAccess := string(*mockRedisCache.Properties.PublicNetworkAccess)
		assert.Equal(t, "Enabled", publicNetworkAccess)

		provisioningState := string(*mockRedisCache.Properties.ProvisioningState)
		assert.Equal(t, "Succeeded", provisioningState)

		skuName := string(*mockRedisCache.Properties.SKU.Name)
		assert.Equal(t, "Standard", skuName)

		skuFamily := string(*mockRedisCache.Properties.SKU.Family)
		assert.Equal(t, "C", skuFamily)

		minimumTlsVersion := string(*mockRedisCache.Properties.MinimumTLSVersion)
		assert.Equal(t, "1.2", minimumTlsVersion)
	})

	t.Run("TestNilOptionalFields", func(t *testing.T) {
		// Test with minimal mock data to ensure nil fields are handled gracefully
		minimalCache := &armredis.ResourceInfo{
			ID:       ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Cache/redis/minimal-cache"),
			Name:     ptr("minimal-cache"),
			Type:     ptr("Microsoft.Cache/redis"),
			Location: ptr("westus"),
			Properties: &armredis.Properties{
				SKU: &armredis.SKU{
					Name:     ptr(armredis.SKUNameBasic),
					Capacity: ptr(int32(0)),
					Family:   ptr(armredis.SKUFamilyC),
				},
			},
		}

		rawData, err := createRedisInstanceRawData(nil, minimalCache)
		assert.NoError(t, err)
		assert.NotNil(t, rawData)

		// Nil enum fields should result in nil string values
		assert.Nil(t, rawData["minimumTlsVersion"].Value)
		assert.Nil(t, rawData["publicNetworkAccess"].Value)
		assert.Nil(t, rawData["provisioningState"].Value)

		// Nil optional fields
		assert.Nil(t, rawData["staticIp"].Value)
		assert.Nil(t, rawData["subnetId"].Value)
		assert.Nil(t, rawData["shardCount"].Value)
		assert.Nil(t, rawData["hostName"].Value)

		// Empty zones
		zones := rawData["zones"].Value.([]any)
		assert.Len(t, zones, 0)

		// Nil identity
		assert.Nil(t, rawData["identity"].Value)

		// Empty PECs
		pecs := rawData["privateEndpointConnections"].Value.([]any)
		assert.Len(t, pecs, 0)
	})
}
