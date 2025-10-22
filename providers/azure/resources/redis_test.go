// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v2"
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
		rawData, err := createRedisInstanceRawData(mockRedisCache)
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
	})
}
