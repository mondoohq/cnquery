// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "go.mondoo.com/mql/llx"

// skuOption sets one flattened SKU member on a resource argument map.
type skuOption func(map[string]*llx.RawData)

// addSkuFields writes the sku* arguments a resource publishes, taken from the
// members its ARM SKU struct actually models.
//
// ARM models a different subset of {Name, Tier, Size, Family, Capacity} per
// service: armredis.SKU and armautomation.SKU carry no Tier, and
// armsearch.SKU, armcdn.SKU and armappconfiguration.SKU carry only Name. A
// member the caller does not pass stays out of the map, so a resource whose
// SKU has no tier never publishes an empty skuTier. That keeps the schema a
// faithful description of what the API returns rather than a union of every
// service's SKU shape.
func addSkuFields(args map[string]*llx.RawData, opts ...skuOption) {
	for _, opt := range opts {
		opt(args)
	}
}

// skuFields is addSkuFields against a fresh map, for callers that merge the
// result themselves and for tests.
func skuFields(opts ...skuOption) map[string]*llx.RawData {
	args := make(map[string]*llx.RawData, len(opts))
	addSkuFields(args, opts...)
	return args
}

// skuName publishes SKU.Name. The type parameter accepts the named string
// types ARM uses for SKU name enums as well as a plain *string.
func skuName[T ~string](v *T) skuOption {
	return func(args map[string]*llx.RawData) {
		args["skuName"] = llx.StringDataPtr(stringEnumPtr(v))
	}
}

// skuTier publishes SKU.Tier.
func skuTier[T ~string](v *T) skuOption {
	return func(args map[string]*llx.RawData) {
		args["skuTier"] = llx.StringDataPtr(stringEnumPtr(v))
	}
}

// skuSize publishes SKU.Size.
func skuSize[T ~string](v *T) skuOption {
	return func(args map[string]*llx.RawData) {
		args["skuSize"] = llx.StringDataPtr(stringEnumPtr(v))
	}
}

// skuFamily publishes SKU.Family.
func skuFamily[T ~string](v *T) skuOption {
	return func(args map[string]*llx.RawData) {
		args["skuFamily"] = llx.StringDataPtr(stringEnumPtr(v))
	}
}

// skuCapacity publishes SKU.Capacity. ARM types it *int64 on armcompute.SKU
// and armiothub.SKUInfo and *int32 nearly everywhere else.
func skuCapacity[T int32 | int64 | int](v *T) skuOption {
	return func(args map[string]*llx.RawData) {
		args["skuCapacity"] = llx.IntDataPtr(v)
	}
}

// identityOption sets one flattened managed-identity member on a resource
// argument map.
type identityOption func(map[string]*llx.RawData)

// addIdentityFields writes the flattened managed-identity arguments a resource
// publishes.
//
// The ARM managed-identity block is {Type, PrincipalID, TenantID,
// UserAssignedIdentities} across armcompute, armcontainerservice,
// armcontainerinstance, armappservice, armredis, armsearch, armmachinelearning,
// armbatch and armresources. It is not universal: armbatch.PoolIdentity carries
// only Type and UserAssignedIdentities, so a pool passes identityType alone and
// never publishes a principalId that Azure does not report. The user-assigned
// identities themselves stay a lazy accessor backed by
// cacheUserAssignedIdentityIds, because resolving them eagerly would run one
// managedIdentity init per attached identity per resource in the scan.
func addIdentityFields(args map[string]*llx.RawData, opts ...identityOption) {
	for _, opt := range opts {
		opt(args)
	}
}

// identityFields is addIdentityFields against a fresh map, for tests.
func identityFields(opts ...identityOption) map[string]*llx.RawData {
	args := make(map[string]*llx.RawData, len(opts))
	addIdentityFields(args, opts...)
	return args
}

// identityType publishes Identity.Type.
func identityType[T ~string](v *T) identityOption {
	return func(args map[string]*llx.RawData) {
		args["identityType"] = llx.StringDataPtr(stringEnumPtr(v))
	}
}

// identityPrincipalId publishes Identity.PrincipalID, the object ID of the
// system-assigned identity.
func identityPrincipalId(v *string) identityOption {
	return func(args map[string]*llx.RawData) {
		args["principalId"] = llx.StringDataPtr(v)
	}
}

// identityTenantId publishes Identity.TenantID.
func identityTenantId(v *string) identityOption {
	return func(args map[string]*llx.RawData) {
		args["tenantId"] = llx.StringDataPtr(v)
	}
}

// addIdentity publishes the flattened managed-identity fields for the ARM
// shape shared across services and returns the ARM resource IDs of the
// attached user-assigned identities in stable order, for the caller to store
// on its Internal struct.
func addIdentity[E ~string, V any](args map[string]*llx.RawData, typ *E, principalID, tenantID *string, userAssigned map[string]V) []string {
	addIdentityFields(args,
		identityType(typ),
		identityPrincipalId(principalID),
		identityTenantId(tenantID),
	)
	return sortedUserAssignedIdentityIDs(userAssigned)
}
