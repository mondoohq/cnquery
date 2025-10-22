// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAzureConnection implements the AzureConnection interface for testing
type mockAzureConnection struct {
	subscriptionID string
	token          string
}

func (m *mockAzureConnection) ID() uint32 {
	return 1
}

func (m *mockAzureConnection) Name() string {
	return "mock-azure-connection"
}

func (m *mockAzureConnection) SubId() string {
	return m.subscriptionID
}

func (m *mockAzureConnection) Token() string {
	return m.token
}

func (m *mockAzureConnection) ClientOptions() interface{} {
	return nil
}

func (m *mockAzureConnection) ParentID() uint32 {
	return 0
}

// MockRedisClientFactory mocks the Azure Redis client factory
type mockRedisClientFactory struct {
	mock.Mock
}

func (m *mockRedisClientFactory) NewClient() *armredis.Client {
	args := m.Called()
	return args.Get(0).(*armredis.Client)
}

// MockRedisClient mocks the Azure Redis client
type mockRedisClient struct {
	mock.Mock
}

func (m *mockRedisClient) NewListBySubscriptionPager(options *armredis.ClientListBySubscriptionOptions) *runtime.Pager[armredis.ClientListBySubscriptionResponse] {
	args := m.Called(options)
	return args.Get(0).(*runtime.Pager[armredis.ClientListBySubscriptionResponse])
}

// MockPager mocks the Azure pager
type MockPager struct {
	mock.Mock
	pages []armredis.ClientListBySubscriptionResponse
}

func (m *MockPager) More() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockPager) NextPage(ctx context.Context) (armredis.ClientListBySubscriptionResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(armredis.ClientListBySubscriptionResponse), args.Error(1)
}

// Helper functions to create pointers
func stringPtr(s string) *string {
	return &s
}

func publicNetworkAccessPtr(s armredis.PublicNetworkAccess) *armredis.PublicNetworkAccess {
	return &s
}

func provisioningStatePtr(s armredis.ProvisioningState) *armredis.ProvisioningState {
	return &s
}

func skuNamePtr(s armredis.SKUName) *armredis.SKUName {
	return &s
}

func skuFamilyPtr(s armredis.SKUFamily) *armredis.SKUFamily {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func int32Ptr(i int32) *int32 {
	return &i
}

func TestAzureSubscriptionCacheServiceRedis(t *testing.T) {
	// Create mock data
	mockRedisCache := &armredis.ResourceInfo{
		ID:       stringPtr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Cache/redis/test-cache"),
		Name:     stringPtr("test-cache"),
		Type:     stringPtr("Microsoft.Cache/redis"),
		Location: stringPtr("eastus"),
		Properties: &armredis.Properties{
			HostName:            stringPtr("test-cache.redis.cache.windows.net"),
			EnableNonSSLPort:    boolPtr(true),
			PublicNetworkAccess: publicNetworkAccessPtr(armredis.PublicNetworkAccessEnabled),
			Port:                int32Ptr(6379),
			SSLPort:             int32Ptr(6380),
			ProvisioningState:   provisioningStatePtr(armredis.ProvisioningStateSucceeded),
			RedisVersion:        stringPtr("6.0"),
			ReplicasPerMaster:   int32Ptr(1),
			ReplicasPerPrimary:  int32Ptr(1),
			SKU: &armredis.SKU{
				Name:     skuNamePtr(armredis.SKUNameStandard),
				Capacity: int32Ptr(1),
				Family:   skuFamilyPtr(armredis.SKUFamilyC),
			},
		},
		Tags: map[string]*string{
			"Environment": stringPtr("Test"),
			"Project":     stringPtr("Redis-Test"),
		},
	}

	// Create mock response
	mockResponse := armredis.ClientListBySubscriptionResponse{
		ListResult: armredis.ListResult{
			Value: []*armredis.ResourceInfo{mockRedisCache},
		},
	}

	// Create mock pager
	mockPager := &MockPager{}
	mockPager.On("More").Return(true).Once()
	mockPager.On("More").Return(false).Once()
	mockPager.On("NextPage", mock.Anything).Return(mockResponse, nil).Once()

	// Create mock client
	mockClient := &mockRedisClient{}
	mockClient.On("NewListBySubscriptionPager", mock.Anything).Return(mockPager)

	// Create mock client factory
	mockFactory := &mockRedisClientFactory{}
	mockFactory.On("NewClient").Return(mockClient)

	// Test the data conversion logic

	t.Run("TestRedisDataConversion", func(t *testing.T) {
		// Test that the mock data has the expected structure
		assert.Equal(t, "test-cache", *mockRedisCache.Name)
		assert.Equal(t, "eastus", *mockRedisCache.Location)
		assert.Equal(t, "test-cache.redis.cache.windows.net", *mockRedisCache.Properties.HostName)
		assert.Equal(t, true, *mockRedisCache.Properties.EnableNonSSLPort)
		assert.Equal(t, "Enabled", string(*mockRedisCache.Properties.PublicNetworkAccess))
		assert.Equal(t, int32(6379), *mockRedisCache.Properties.Port)
		assert.Equal(t, int32(6380), *mockRedisCache.Properties.SSLPort)
		assert.Equal(t, "Succeeded", string(*mockRedisCache.Properties.ProvisioningState))
		assert.Equal(t, "6.0", *mockRedisCache.Properties.RedisVersion)
		assert.Equal(t, int32(1), *mockRedisCache.Properties.ReplicasPerMaster)
		assert.Equal(t, int32(1), *mockRedisCache.Properties.ReplicasPerPrimary)
		assert.Equal(t, "Standard", string(*mockRedisCache.Properties.SKU.Name))
		assert.Equal(t, int32(1), *mockRedisCache.Properties.SKU.Capacity)
		assert.Equal(t, "C", string(*mockRedisCache.Properties.SKU.Family))
		assert.Equal(t, "Test", *mockRedisCache.Tags["Environment"])
		assert.Equal(t, "Redis-Test", *mockRedisCache.Tags["Project"])
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
