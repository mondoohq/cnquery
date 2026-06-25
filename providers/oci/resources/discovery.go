// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"slices"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/utils/stringx"
)

// Discovery target constants. `auto` covers the tenancy plus every listed
// fine-grained API resource; `all` is currently identical to `auto`. Users
// may also pass any individual target directly (e.g. `--discover identity-users`).
const (
	DiscoveryAuto    = "auto"
	DiscoveryAll     = "all"
	DiscoveryTenancy = "tenancy"

	DiscoverySecurityLists         = "network-securitylists"
	DiscoveryUsers                 = "identity-users"
	DiscoveryPolicies              = "identity-policies"
	DiscoveryBuckets               = "objectstorage-buckets"
	DiscoveryAPIGatewayDeployments = "apigateway-deployments"
	DiscoveryLoadBalancers         = "loadbalancer-loadbalancers"
	DiscoveryRedisClusters         = "redis-clusters"
	DiscoveryVaultSecrets          = "vault-secrets"
	DiscoveryOkeClusters           = "oke-clusters"
)

// AllAPIResources lists every fine-grained per-resource discovery target.
// Keep sorted alphabetically by target string for diff stability.
var AllAPIResources = []string{
	DiscoveryAPIGatewayDeployments,
	DiscoveryBuckets,
	DiscoveryLoadBalancers,
	DiscoveryOkeClusters,
	DiscoveryPolicies,
	DiscoveryRedisClusters,
	DiscoverySecurityLists,
	DiscoveryUsers,
	DiscoveryVaultSecrets,
}

// Auto expands to the tenancy plus all API resources. The order puts tenancy
// first so it's visible in `cnspec scan` output before any sub-assets.
var Auto = append(
	[]string{DiscoveryTenancy},
	AllAPIResources...,
)

// All mirrors Auto today but is kept as a separate slice so `all` vs `auto`
// can diverge if we later add heavier-weight targets that shouldn't run by
// default.
var All = slices.Clone(Auto)

// Discover is the provider's discovery entry point. It iterates the configured
// discovery targets and emits one inventory.Asset per fine-grained resource so
// per-resource cnspec security checks can run against them.
func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.OciConnection)
	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	targets := getDiscoveryTargets(conn.Conf)
	for _, target := range targets {
		list, err := discover(runtime, conn, target)
		if err != nil {
			log.Error().Err(err).Str("target", target).Msg("error during OCI discovery")
			continue
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	}

	return in, nil
}

// getDiscoveryTargets resolves aliases (`auto`, `all`) to concrete target
// strings and deduplicates the result.
func getDiscoveryTargets(config *inventory.Config) []string {
	targets := config.GetDiscover().GetTargets()

	if stringx.Contains(targets, DiscoveryAll) {
		return All
	}

	res := []string{}
	for _, target := range targets {
		switch target {
		case DiscoveryAuto:
			res = append(res, Auto...)
		default:
			res = append(res, target)
		}
	}
	return stringx.DedupStringArray(res)
}

func discover(runtime *plugin.Runtime, conn *connection.OciConnection, target string) ([]*inventory.Asset, error) {
	tenantID := conn.TenantID()
	assetList := []*inventory.Asset{}

	switch target {
	case DiscoveryTenancy:
		// The tenancy asset already exists in the request (the user connected
		// to it directly); no work needed here.
	case DiscoverySecurityLists:
		res, err := NewResource(runtime, "oci.network", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		network := res.(*mqlOciNetwork)
		secLists := network.GetSecurityLists()
		if secLists.Error != nil {
			return nil, secLists.Error
		}
		for i := range secLists.Data {
			sl := secLists.Data[i].(*mqlOciNetworkSecurityList)
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: sl.CompartmentID.Data,
				// cacheRegion was populated when the security list was
				// enumerated; empty only when the enumeration didn't hit a
				// region (defensive fallback).
				region:     fallbackRegion(sl.cacheRegion),
				id:         sl.Id.Data,
				service:    "network",
				objectType: "securitylist",
			}, sl.Name.Data, tagsToLabels(sl.FreeformTags.Data), conn))
		}
	case DiscoveryBuckets:
		res, err := NewResource(runtime, "oci.objectStorage", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		os := res.(*mqlOciObjectStorage)
		buckets := os.GetBuckets()
		if buckets.Error != nil {
			return nil, buckets.Error
		}
		for i := range buckets.Data {
			b := buckets.Data[i].(*mqlOciObjectStorageBucket)
			// Region is exposed as a typed oci.region resource on the bucket;
			// pull its id (region key) for the platform id.
			regionKey := ""
			region := b.GetRegion()
			if region.Error == nil && region.Data != nil {
				regionKey = region.Data.Id.Data
			}
			// Tags on bucket are lazy-loaded (require an extra GetBucket call);
			// at discovery time surface empty labels so we don't pay N API
			// round-trips per bucket just to populate discovery labels.
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: b.CompartmentID.Data,
				region:      fallbackRegion(regionKey),
				// Buckets aren't globally unique by name alone — namespace
				// qualifies them — so use namespace/name as the platform id
				// suffix to match the existing __id.
				id:         b.Namespace.Data + "/" + b.Name.Data,
				service:    "objectstorage",
				objectType: "bucket",
			}, b.Name.Data, map[string]string{}, conn))
		}
	case DiscoveryUsers:
		res, err := NewResource(runtime, "oci.identity", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		identity := res.(*mqlOciIdentity)
		users := identity.GetUsers()
		if users.Error != nil {
			return nil, users.Error
		}
		for i := range users.Data {
			u := users.Data[i].(*mqlOciIdentityUser)
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: u.CompartmentID.Data,
				// OCI IAM is global (single realm per tenancy); mark users as
				// such so the platform id stays stable regardless of which
				// region the scan connects to.
				region:     "global",
				id:         u.Id.Data,
				service:    "identity",
				objectType: "user",
			}, u.Name.Data, tagsToLabels(u.FreeformTags.Data), conn))
		}
	case DiscoveryPolicies:
		res, err := NewResource(runtime, "oci.identity", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		identity := res.(*mqlOciIdentity)
		policies := identity.GetPolicies()
		if policies.Error != nil {
			return nil, policies.Error
		}
		for i := range policies.Data {
			p := policies.Data[i].(*mqlOciIdentityPolicy)
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: p.CompartmentID.Data,
				region:      "global",
				id:          p.Id.Data,
				service:     "identity",
				objectType:  "policy",
			}, p.Name.Data, tagsToLabels(p.FreeformTags.Data), conn))
		}
	case DiscoveryAPIGatewayDeployments:
		res, err := NewResource(runtime, "oci.apigateway", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		apigw := res.(*mqlOciApigateway)
		deps := apigw.GetDeployments()
		if deps.Error != nil {
			return nil, deps.Error
		}
		for i := range deps.Data {
			d := deps.Data[i].(*mqlOciApigatewayDeployment)
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: d.CompartmentID.Data,
				region:      fallbackRegion(d.region),
				id:          d.Id.Data,
				service:     "apigateway",
				objectType:  "deployment",
			}, d.Name.Data, tagsToLabels(d.FreeformTags.Data), conn))
		}
	case DiscoveryLoadBalancers:
		res, err := NewResource(runtime, "oci.loadBalancer", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		lbSvc := res.(*mqlOciLoadBalancer)
		lbs := lbSvc.GetLoadBalancers()
		if lbs.Error != nil {
			return nil, lbs.Error
		}
		for i := range lbs.Data {
			lb := lbs.Data[i].(*mqlOciLoadBalancerLoadBalancer)
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: lb.CompartmentID.Data,
				region:      fallbackRegion(lb.cacheRegion),
				id:          lb.Id.Data,
				service:     "loadbalancer",
				objectType:  "loadBalancer",
			}, lb.Name.Data, tagsToLabels(lb.FreeformTags.Data), conn))
		}
	case DiscoveryRedisClusters:
		res, err := NewResource(runtime, "oci.redis", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		redis := res.(*mqlOciRedis)
		clusters := redis.GetClusters()
		if clusters.Error != nil {
			return nil, clusters.Error
		}
		for i := range clusters.Data {
			c := clusters.Data[i].(*mqlOciRedisCluster)
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: c.CompartmentID.Data,
				region:      fallbackRegion(c.cacheRegion),
				id:          c.Id.Data,
				service:     "redis",
				objectType:  "cluster",
			}, c.Name.Data, tagsToLabels(c.FreeformTags.Data), conn))
		}
	case DiscoveryVaultSecrets:
		res, err := NewResource(runtime, "oci.vault", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		v := res.(*mqlOciVault)
		secrets := v.GetSecrets()
		if secrets.Error != nil {
			return nil, secrets.Error
		}
		for i := range secrets.Data {
			s := secrets.Data[i].(*mqlOciVaultSecret)
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: s.CompartmentID.Data,
				region:      fallbackRegion(s.cacheRegion),
				id:          s.Id.Data,
				service:     "vault",
				objectType:  "secret",
			}, s.Name.Data, tagsToLabels(s.FreeformTags.Data), conn))
		}
	case DiscoveryOkeClusters:
		res, err := NewResource(runtime, "oci.oke", map[string]*llx.RawData{})
		if err != nil {
			return nil, err
		}
		oke := res.(*mqlOciOke)
		clusters := oke.GetClusters()
		if clusters.Error != nil {
			return nil, clusters.Error
		}
		for i := range clusters.Data {
			c := clusters.Data[i].(*mqlOciOkeCluster)
			appendIfNotNil(&assetList, ociObjectToAsset(ociObject{
				tenantID:    tenantID,
				compartment: c.CompartmentID.Data,
				region:      fallbackRegion(c.region),
				id:          c.Id.Data,
				service:     "oke",
				objectType:  "cluster",
			}, c.Name.Data, tagsToLabels(c.FreeformTags.Data), conn))
		}
	default:
		log.Warn().Str("target", target).Msg("oci discovery: unknown target; skipping")
	}
	return assetList, nil
}

// appendIfNotNil guards the common ociObjectToAsset call pattern against
// returning a nil asset (unknown platform name). Nil entries in the
// inventory list cause downstream panics when cnspec iterates.
func appendIfNotNil(list *[]*inventory.Asset, a *inventory.Asset) {
	if a == nil {
		return
	}
	*list = append(*list, a)
}

// ociObject is the fine-grained handle used to construct a discovery asset.
type ociObject struct {
	tenantID    string
	compartment string
	region      string
	id          string // OCID or composite id (e.g. namespace/name for buckets)
	service     string
	objectType  string
}

// mondooOciObjectID builds the canonical platform id for a fine-grained OCI
// resource. Format mirrors AWS/GCP:
//
//	//platformid.api.mondoo.app/runtime/oci/<service>/v1/tenancies/<tenant>/regions/<region>/<objectType>/<id>
func mondooOciObjectID(obj ociObject) string {
	return "//platformid.api.mondoo.app/runtime/oci/" + obj.service +
		"/v1/tenancies/" + obj.tenantID +
		"/regions/" + obj.region +
		"/" + obj.objectType + "/" + obj.id
}

// getPlatformName maps (service, objectType) to the platform name used by
// cnspec policy filters. Returning "" for an unknown pair makes the caller
// skip the asset rather than emit a broken one.
func getPlatformName(obj ociObject) string {
	switch obj.service {
	case "network":
		if obj.objectType == "securitylist" {
			return "oci-network-securitylist"
		}
	case "identity":
		switch obj.objectType {
		case "user":
			return "oci-identity-user"
		case "policy":
			return "oci-identity-policy"
		}
	case "objectstorage":
		if obj.objectType == "bucket" {
			return "oci-objectstorage-bucket"
		}
	case "apigateway":
		if obj.objectType == "deployment" {
			return "oci-apigateway-deployment"
		}
	case "loadbalancer":
		if obj.objectType == "loadBalancer" {
			return "oci-loadbalancer"
		}
	case "redis":
		if obj.objectType == "cluster" {
			return "oci-redis-cluster"
		}
	case "vault":
		if obj.objectType == "secret" {
			return "oci-vault-secret"
		}
	case "oke":
		if obj.objectType == "cluster" {
			return "oci-oke-cluster"
		}
	}
	return ""
}

// ociObjectToAsset wraps an ociObject into an inventory.Asset suitable for
// returning from Discover(). Returns nil if the object can't be mapped to a
// known platform (discovery then skips it rather than emitting a broken asset).
func ociObjectToAsset(obj ociObject, name string, labels map[string]string, conn *connection.OciConnection) *inventory.Asset {
	platformName := getPlatformName(obj)
	if platformName == "" {
		log.Warn().Str("service", obj.service).Str("objectType", obj.objectType).
			Msg("oci discovery: unknown service/objectType pair; skipping asset")
		return nil
	}
	if name == "" {
		name = obj.id
	}
	platformID := mondooOciObjectID(obj)
	// Clone to avoid mutating the parent connection's config under concurrent
	// discovery, and strip the discovery options so the sub-asset doesn't
	// recursively trigger another pass.
	clonedConfig := conn.Conf.Clone(
		inventory.WithoutDiscovery(),
		inventory.WithParentConnectionId(conn.Conf.Id),
	)
	clonedConfig.PlatformId = platformID
	platform := &inventory.Platform{}
	PlatformByName(platformName).Apply(platform)
	return &inventory.Asset{
		PlatformIds: []string{platformID},
		Name:        name,
		Platform:    platform,
		Labels:      labels,
		Connections: []*inventory.Config{clonedConfig},
	}
}

// tagsToLabels converts an MQL freeformTags map (map[string]interface{}) to
// the plain string map the asset schema expects. Non-string values are
// skipped — OCI freeform tags are declared as strings, but the MQL layer
// types them as `any` because they flow through the dict path.
func tagsToLabels(in map[string]interface{}) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// fallbackRegion returns "unknown" when a resource didn't expose a region at
// enumeration time. Using a literal string here (rather than "") keeps the
// platform id valid while making the gap obvious in asset listings.
func fallbackRegion(r string) string {
	if r == "" {
		return "unknown"
	}
	return r
}

// ociArgString reads a string-valued arg, returning "" if the key is missing,
// nil-valued, or not a string. Used by init functions that need to tolerate
// sparsely-populated args maps.
func ociArgString(args map[string]*llx.RawData, key string) string {
	if args == nil {
		return ""
	}
	raw, ok := args[key]
	if !ok || raw == nil || raw.Value == nil {
		return ""
	}
	s, ok := raw.Value.(string)
	if !ok {
		return ""
	}
	return s
}

// parsedOciPlatformID captures the components extracted from a canonical OCI
// asset platform id. It mirrors the structure produced by mondooOciObjectID.
type parsedOciPlatformID struct {
	tenantID   string
	region     string
	service    string
	objectType string
	// id is the last path segment. For most resources this is the OCID; for
	// object-storage buckets it's the composite "<namespace>/<name>" we emit
	// from discovery, so callers that need one or the other must split further.
	id string
}

// parseOciObjectPlatformID extracts the fine-grained components from a
// discovered asset's platform id. Returns (nil, false) when the string is not
// a recognized per-resource OCI platform id (e.g. the parent tenancy platform,
// or an unrelated provider).
//
// Expected format (see mondooOciObjectID):
//
//	//platformid.api.mondoo.app/runtime/oci/<service>/v1/tenancies/<tenant>/regions/<region>/<objectType>/<id>
//
// The final `<id>` segment may itself contain "/" (bucket composite id).
func parseOciObjectPlatformID(platformID string) (*parsedOciPlatformID, bool) {
	const prefix = "//platformid.api.mondoo.app/runtime/oci/"
	if !strings.HasPrefix(platformID, prefix) {
		return nil, false
	}
	rest := platformID[len(prefix):]
	// After the prefix we expect:
	//   <service>/v1/tenancies/<tenant>/regions/<region>/<objectType>/<id...>
	// Splitting on "/" with a cap of 8 leaves everything after the 7th slash
	// (i.e. the id) in the final element untouched — important for buckets
	// whose id is "<namespace>/<name>".
	parts := strings.SplitN(rest, "/", 8)
	if len(parts) < 8 {
		return nil, false
	}
	if parts[1] != "v1" || parts[2] != "tenancies" || parts[4] != "regions" {
		return nil, false
	}
	return &parsedOciPlatformID{
		service:    parts[0],
		tenantID:   parts[3],
		region:     parts[5],
		objectType: parts[6],
		id:         parts[7],
	}, true
}
