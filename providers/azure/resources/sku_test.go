// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	armautomation "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/automation/armautomation"
	armbatch "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
	compute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v8"
	armredis "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The point of the per-member options is that a resource publishes only what
// its ARM SKU struct models. Publishing the union instead would report an
// empty skuTier on every Redis cache and every Automation account, which reads
// as "the tier is blank" rather than as "this service has no tier".
func TestSkuFieldsPublishOnlyTheMembersPassed(t *testing.T) {
	t.Run("a SKU with no tier never publishes skuTier", func(t *testing.T) {
		sku := armredis.SKU{
			Name:     ptr(armredis.SKUNamePremium),
			Family:   ptr(armredis.SKUFamilyP),
			Capacity: ptr(int32(3)),
		}
		args := skuFields(skuName(sku.Name), skuFamily(sku.Family), skuCapacity(sku.Capacity))

		assert.NotContains(t, args, "skuTier")
		assert.NotContains(t, args, "skuSize")
		require.Contains(t, args, "skuName")
		assert.Equal(t, "Premium", args["skuName"].Value)
		assert.Equal(t, "P", args["skuFamily"].Value)
		assert.Equal(t, int64(3), args["skuCapacity"].Value)
	})

	t.Run("a name-only SKU publishes name alone", func(t *testing.T) {
		name := armautomation.SKUNameEnumBasic
		args := skuFields(skuName(&name))

		require.Len(t, args, 1)
		assert.Equal(t, "Basic", args["skuName"].Value)
	})

	// armcompute types Capacity as *int64 where nearly every other service
	// types it *int32; both have to land on the same int64 field.
	t.Run("capacity is published from either width", func(t *testing.T) {
		wide := skuFields(skuCapacity(ptr(int64(12))))
		narrow := skuFields(skuCapacity(ptr(int32(12))))

		assert.Equal(t, int64(12), wide["skuCapacity"].Value)
		assert.Equal(t, int64(12), narrow["skuCapacity"].Value)
	})

	// An absent member is still declared, so the field resolves to null rather
	// than failing with "no type information".
	t.Run("an absent member is set and null, not missing", func(t *testing.T) {
		var sku compute.SKU
		args := skuFields(skuName(sku.Name), skuTier(sku.Tier), skuCapacity(sku.Capacity))

		require.Contains(t, args, "skuName")
		assert.Nil(t, args["skuName"].Value)
		assert.Nil(t, args["skuTier"].Value)
		assert.Nil(t, args["skuCapacity"].Value)
	})
}

func TestIdentityFieldsPublishOnlyTheMembersPassed(t *testing.T) {
	t.Run("the shared shape publishes all three scalars", func(t *testing.T) {
		identity := compute.VirtualMachineIdentity{
			Type:        ptr(compute.ResourceIdentityTypeSystemAssignedUserAssigned),
			PrincipalID: ptr("principal-1"),
			TenantID:    ptr("tenant-1"),
			UserAssignedIdentities: map[string]*compute.UserAssignedIdentitiesValue{
				"/subscriptions/s/b": {},
				"/subscriptions/s/a": {},
			},
		}
		got := identityFields(
			identityType(identity.Type),
			identityPrincipalId(identity.PrincipalID),
			identityTenantId(identity.TenantID),
		)

		assert.Equal(t, "SystemAssigned, UserAssigned", got["identityType"].Value)
		assert.Equal(t, "principal-1", got["principalId"].Value)
		assert.Equal(t, "tenant-1", got["tenantId"].Value)
		// Sorted, so the attached identities do not reorder between scans.
		assert.Equal(t,
			[]string{"/subscriptions/s/a", "/subscriptions/s/b"},
			sortedUserAssignedIdentityIDs(identity.UserAssignedIdentities))
	})

	// armbatch.PoolIdentity has no PrincipalID or TenantID. Publishing them
	// anyway would report every Batch pool as having a system-assigned
	// identity with a blank principal, which is not what Azure said.
	t.Run("a batch pool identity publishes the type alone", func(t *testing.T) {
		identity := armbatch.PoolIdentity{
			Type: ptr(armbatch.PoolIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armbatch.UserAssignedIdentities{
				"/subscriptions/s/a": {},
			},
		}
		got := identityFields(identityType(identity.Type))

		require.Len(t, got, 1)
		assert.Equal(t, "UserAssigned", got["identityType"].Value)
		assert.NotContains(t, got, "principalId")
		assert.NotContains(t, got, "tenantId")
	})

	t.Run("no attached identities is nil, not an empty entry", func(t *testing.T) {
		assert.Nil(t, sortedUserAssignedIdentityIDs(map[string]*compute.UserAssignedIdentitiesValue{}))
		assert.Nil(t, sortedUserAssignedIdentityIDs[*compute.UserAssignedIdentitiesValue](nil))
	})
}
