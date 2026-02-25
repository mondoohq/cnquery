// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	armbatch "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v3"
	"github.com/stretchr/testify/assert"
)

func TestBatchAccountDataConversion(t *testing.T) {
	aadMode := armbatch.AuthenticationModeAAD
	sharedKeyMode := armbatch.AuthenticationModeSharedKey

	mockAccount := &armbatch.Account{
		ID:       ptr("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/test-account"),
		Name:     ptr("test-account"),
		Type:     ptr("Microsoft.Batch/batchAccounts"),
		Location: ptr("eastus"),
		Identity: &armbatch.AccountIdentity{
			Type:        ptr(armbatch.ResourceIdentityTypeSystemAssigned),
			PrincipalID: ptr("principal-123"),
			TenantID:    ptr("tenant-456"),
		},
		Properties: &armbatch.AccountProperties{
			AccountEndpoint:                       ptr("test-account.eastus.batch.azure.com"),
			ProvisioningState:                     ptr(armbatch.ProvisioningStateSucceeded),
			PoolAllocationMode:                    ptr(armbatch.PoolAllocationModeBatchService),
			PublicNetworkAccess:                   ptr(armbatch.PublicNetworkAccessTypeEnabled),
			NodeManagementEndpoint:                ptr("https://test-account.eastus.service.batch.azure.com"),
			ActiveJobAndJobScheduleQuota:          ptr(int32(300)),
			DedicatedCoreQuota:                    ptr(int32(500)),
			DedicatedCoreQuotaPerVMFamilyEnforced: ptr(true),
			LowPriorityCoreQuota:                  ptr(int32(100)),
			PoolQuota:                             ptr(int32(64)),
			AllowedAuthenticationModes: []*armbatch.AuthenticationMode{
				&aadMode,
				&sharedKeyMode,
			},
		},
		Tags: map[string]*string{
			"Environment": ptr("Test"),
			"Team":        ptr("Platform"),
		},
	}

	t.Run("FullDataConversion", func(t *testing.T) {
		rawData, err := createBatchAccountRawData(mockAccount)
		assert.NoError(t, err)
		assert.NotNil(t, rawData)

		assert.Equal(t, "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/test-account", rawData["id"].Value)
		assert.Equal(t, "test-account", rawData["name"].Value)
		assert.Equal(t, "Microsoft.Batch/batchAccounts", rawData["type"].Value)
		assert.Equal(t, "eastus", rawData["location"].Value)

		// String fields
		assert.Equal(t, "test-account.eastus.batch.azure.com", rawData["accountEndpoint"].Value)
		assert.Equal(t, "https://test-account.eastus.service.batch.azure.com", rawData["nodeManagementEndpoint"].Value)

		// Enum fields converted to strings
		assert.Equal(t, "Succeeded", rawData["provisioningState"].Value)
		assert.Equal(t, "BatchService", rawData["poolAllocationMode"].Value)
		assert.Equal(t, "Enabled", rawData["publicNetworkAccess"].Value)

		// Quota fields
		assert.Equal(t, int64(300), rawData["activeJobAndJobScheduleQuota"].Value)
		assert.Equal(t, int64(500), rawData["dedicatedCoreQuota"].Value)
		assert.Equal(t, int64(100), rawData["lowPriorityCoreQuota"].Value)
		assert.Equal(t, int64(64), rawData["poolQuota"].Value)
		assert.Equal(t, true, rawData["dedicatedCoreQuotaPerVmFamilyEnforced"].Value)

		// Array of enums
		authModes := rawData["allowedAuthenticationModes"].Value.([]any)
		assert.Len(t, authModes, 2)
		assert.Equal(t, "AAD", authModes[0])
		assert.Equal(t, "SharedKey", authModes[1])

		// Dict fields
		assert.NotNil(t, rawData["properties"].Value)
		assert.NotNil(t, rawData["identity"].Value)
		assert.NotNil(t, rawData["tags"].Value)
	})

	t.Run("EnumConversions", func(t *testing.T) {
		provisioningState := string(*mockAccount.Properties.ProvisioningState)
		assert.Equal(t, "Succeeded", provisioningState)

		poolAllocationMode := string(*mockAccount.Properties.PoolAllocationMode)
		assert.Equal(t, "BatchService", poolAllocationMode)

		publicNetworkAccess := string(*mockAccount.Properties.PublicNetworkAccess)
		assert.Equal(t, "Enabled", publicNetworkAccess)
	})

	t.Run("NilOptionalFields", func(t *testing.T) {
		minimalAccount := &armbatch.Account{
			ID:       ptr("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/minimal"),
			Name:     ptr("minimal"),
			Type:     ptr("Microsoft.Batch/batchAccounts"),
			Location: ptr("westus"),
			Properties: &armbatch.AccountProperties{
				ProvisioningState: ptr(armbatch.ProvisioningStateSucceeded),
			},
		}

		rawData, err := createBatchAccountRawData(minimalAccount)
		assert.NoError(t, err)
		assert.NotNil(t, rawData)

		// Nil optional fields
		assert.Nil(t, rawData["accountEndpoint"].Value)
		assert.Nil(t, rawData["poolAllocationMode"].Value)
		assert.Nil(t, rawData["publicNetworkAccess"].Value)
		assert.Nil(t, rawData["nodeManagementEndpoint"].Value)
		assert.Nil(t, rawData["activeJobAndJobScheduleQuota"].Value)
		assert.Nil(t, rawData["dedicatedCoreQuota"].Value)
		assert.Nil(t, rawData["lowPriorityCoreQuota"].Value)
		assert.Nil(t, rawData["poolQuota"].Value)
		assert.Nil(t, rawData["dedicatedCoreQuotaPerVmFamilyEnforced"].Value)
		assert.Nil(t, rawData["allowedAuthenticationModes"].Value)
		assert.Nil(t, rawData["autoStorage"].Value)
		assert.Nil(t, rawData["encryption"].Value)
		assert.Nil(t, rawData["keyVaultReference"].Value)
		assert.Nil(t, rawData["networkProfile"].Value)
		assert.Nil(t, rawData["privateEndpointConnections"].Value)
		assert.Nil(t, rawData["dedicatedCoreQuotaPerVmFamily"].Value)

		// Nil identity
		assert.Nil(t, rawData["identity"].Value)

		// Present fields still work
		assert.Equal(t, "Succeeded", rawData["provisioningState"].Value)
	})

	t.Run("NilProperties", func(t *testing.T) {
		noPropsAccount := &armbatch.Account{
			ID:       ptr("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/no-props"),
			Name:     ptr("no-props"),
			Type:     ptr("Microsoft.Batch/batchAccounts"),
			Location: ptr("westus"),
		}

		rawData, err := createBatchAccountRawData(noPropsAccount)
		assert.NoError(t, err)
		assert.NotNil(t, rawData)

		assert.Equal(t, "no-props", rawData["name"].Value)
		assert.Nil(t, rawData["properties"].Value)
		assert.Nil(t, rawData["provisioningState"].Value)
	})
}

func TestBatchPoolDataConversion(t *testing.T) {
	mockPool := &armbatch.Pool{
		ID:   ptr("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/test-account/pools/test-pool"),
		Name: ptr("test-pool"),
		Type: ptr("Microsoft.Batch/batchAccounts/pools"),
		Etag: ptr("W/\"0x8D12345678\""),
		Properties: &armbatch.PoolProperties{
			VMSize:            ptr("Standard_D2s_v3"),
			ProvisioningState: ptr(armbatch.PoolProvisioningStateSucceeded),
			DeploymentConfiguration: &armbatch.DeploymentConfiguration{
				VirtualMachineConfiguration: &armbatch.VirtualMachineConfiguration{
					NodeAgentSKUID: ptr("batch.node.ubuntu 22.04"),
					ImageReference: &armbatch.ImageReference{
						Publisher: ptr("canonical"),
						Offer:     ptr("0001-com-ubuntu-server-jammy"),
						SKU:       ptr("22_04-lts"),
						Version:   ptr("latest"),
					},
				},
			},
		},
	}

	t.Run("FullDataConversion", func(t *testing.T) {
		rawData, err := createBatchPoolRawData(mockPool)
		assert.NoError(t, err)
		assert.NotNil(t, rawData)

		assert.Equal(t, "/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/test-account/pools/test-pool", rawData["id"].Value)
		assert.Equal(t, "test-pool", rawData["name"].Value)
		assert.Equal(t, "Microsoft.Batch/batchAccounts/pools", rawData["type"].Value)
		assert.Equal(t, "W/\"0x8D12345678\"", rawData["etag"].Value)
		assert.Equal(t, "Standard_D2s_v3", rawData["vmSize"].Value)
		assert.Equal(t, "Succeeded", rawData["provisioningState"].Value)

		// Dict fields
		assert.NotNil(t, rawData["properties"].Value)
		assert.NotNil(t, rawData["deploymentConfiguration"].Value)
		assert.NotNil(t, rawData["virtualMachineConfiguration"].Value)
	})

	t.Run("NilOptionalFields", func(t *testing.T) {
		minimalPool := &armbatch.Pool{
			ID:   ptr("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/test-account/pools/minimal"),
			Name: ptr("minimal"),
			Type: ptr("Microsoft.Batch/batchAccounts/pools"),
		}

		rawData, err := createBatchPoolRawData(minimalPool)
		assert.NoError(t, err)
		assert.NotNil(t, rawData)

		assert.Equal(t, "minimal", rawData["name"].Value)
		assert.Nil(t, rawData["etag"].Value)
		assert.Nil(t, rawData["properties"].Value)
		assert.Nil(t, rawData["identity"].Value)
		assert.Nil(t, rawData["deploymentConfiguration"].Value)
		assert.Nil(t, rawData["virtualMachineConfiguration"].Value)
		assert.Nil(t, rawData["vmSize"].Value)
		assert.Nil(t, rawData["provisioningState"].Value)
	})
}
