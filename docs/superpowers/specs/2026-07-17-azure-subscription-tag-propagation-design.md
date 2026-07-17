# Propagate Azure subscription tags to discovered assets

## Goal

During Azure discovery, every asset discovered under a subscription should inherit
that subscription's tags as labels. The asset's own labels take precedence on key
collisions. Because Azure discovery can span multiple subscriptions in a single
run, propagation is **per-subscription**: each subscription's tags land only on the
assets discovered within that subscription (and on the subscription asset itself).

This mirrors the GCP provider's `propagate-project-labels` feature (per-project
label propagation) and the AWS provider's `propagate-account-tags` feature (the
"idea"), adapted to Azure's multi-subscription discovery model.

## Motivation

Today, `providers/azure/resources/discovery.go` discovers child assets (VMs,
storage accounts, key vaults, AKS clusters, etc.) and populates each asset's
`Labels` from the resource's own Azure tags (via `addInformationalLabels`).
Subscription-level tags are not propagated, so downstream consumers can't filter
or group discovered assets by subscription-scoped metadata (e.g. `CostCenter`,
`Owner`, `Environment` set at the subscription level).

The seam for this feature was deliberately built by commit `544831aa8`
("🧹 Align discovery filters in Azure provider"), which reshaped Azure's
subscription filtering into the same `connection.DiscoveryFilters` / `conn.Filters`
structure the AWS provider uses. Its commit message explicitly states it "sets up
the seam for follow-up ... subscription-tag propagation." This spec fills that seam.

## Design decisions (resolved in brainstorming)

- **Label key format: raw tag keys, no prefix.** A subscription tag `env=prod`
  becomes label `env: prod`, exactly like AWS/GCP. This is collision-safe against
  our own informational labels (`azure.mondoo.com/subscription`, `mondoo.com/location`,
  `azure.mondoo.com/resourcegroup`, etc.) because Azure disallows `/` in tag names,
  so a subscription tag key can never equal one of those.
- **Tag source: API fetch with an injected override.** Tags are fetched per
  subscription via the Azure Subscriptions `Get` API (the same call
  `resources/subscription.go` already uses for the `azure.subscription.tags` MQL
  field — no new permission). An injected `SubscriptionTags` override map, when
  provided, is used instead of the API call (mirroring AWS's `AccountTags`).
- **Structure follows GCP.** A GCP-shaped `propagateSubscriptionTagsToAssets(assets,
  tags)` helper is invoked once per subscription, exactly like GCP calls
  `propagateProjectLabelsToAssets` once per project.

## Collision rule

The asset's own label wins. Subscription tags only fill gaps. Rationale:
per-resource tags are more specific and should not be silently overwritten by
subscription-wide defaults. Identical to AWS/GCP.

## Scope

Tags are applied to:

- Every discovered resource asset under a subscription (VM, storage account,
  key vault, AKS cluster, etc.), keyed by the `azure.mondoo.com/subscription`
  label each asset already carries.
- The subscription asset itself (so it carries its own tags as labels), the
  analog of GCP's `40d08f33e` project-asset fix.

The feature is **opt-in** and **off by default**. When
`PropagateSubscriptionTags` is false, no subscription tags are fetched or merged.

## Design

All changes are Go-only. There are **no `.lr` schema edits, no code generation,
and no `.lr.versions` changes** — this touches discovery and connection code only.

### 1. Filter surface — `providers/azure/connection/filters.go`

Extend the existing `DiscoveryFilters` struct:

```go
type DiscoveryFilters struct {
	Subscriptions             SubscriptionsFilter
	// PropagateSubscriptionTags merges each subscription's tags into every asset
	// discovered under that subscription (an asset's own labels win on collision).
	// Off by default. Mirrors the GCP provider's PropagateProjectLabels and the
	// AWS provider's PropagateAccountTags.
	PropagateSubscriptionTags bool
	// SubscriptionTags is an optional injected override. When non-empty it is used
	// instead of fetching each subscription's tags from the API, and applies to
	// every discovered subscription. Mirrors the AWS provider's AccountTags.
	SubscriptionTags map[string]string
}
```

Parse them in `DiscoveryFiltersFromOpts`:

```go
func DiscoveryFiltersFromOpts(opts map[string]string) DiscoveryFilters {
	return DiscoveryFilters{
		Subscriptions: SubscriptionsFilter{
			Include: filteropts.ParseCsvSliceOpt(opts, "subscriptions"),
			Exclude: filteropts.ParseCsvSliceOpt(opts, "subscriptions-exclude"),
		},
		PropagateSubscriptionTags: filteropts.ParseBoolOpt(opts, "propagate-subscription-tags", false),
		SubscriptionTags:          parseMapOpt(opts, "subscription-tag:"),
	}
}
```

Add a local `parseMapOpt` helper mirroring
`providers/aws/connection/filters.go:248` (a small copy; promoting it to the
shared `filteropts` package is a possible future cleanup but out of scope to keep
this change contained and AWS untouched):

```go
func parseMapOpt(opts map[string]string, keyPrefix string) map[string]string {
	res := map[string]string{}
	for k, v := range opts {
		if k == "" || v == "" {
			continue
		}
		if !strings.HasPrefix(k, keyPrefix) {
			continue
		}
		res[strings.TrimPrefix(k, keyPrefix)] = v
	}
	return res
}
```

### 2. CLI plumbing — `providers/azure/provider/provider.go`

Extend the `--filters` allowlist in `parseFlagsToFiltersOpts` to pass through the
two new keys (exact match for the boolean, prefix match for the override family):

```go
if x, ok := flags["filters"]; ok && len(x.Map) != 0 {
	for k, v := range x.Map {
		switch {
		case k == "subscriptions" || k == "subscriptions-exclude" || k == "propagate-subscription-tags":
			o[k] = string(v.Value)
		case strings.HasPrefix(k, "subscription-tag:"):
			o[k] = string(v.Value)
		}
	}
}
```

The dedicated-flag overlay for `--subscription*` is unchanged. Both CLI-supplied
values and programmatically-injected values (e.g. from an upstream integration
writing to `inventory.Discovery.Filter`) flow through the same map, so the
override works either way.

Update the `--filters` flag `Desc` in `providers/azure/config/config.go` to
document the new keys, e.g.:

```
Filter discovered resources, e.g., --filters subscriptions=<id1>,<id2>
--filters subscriptions-exclude=<id3> --filters propagate-subscription-tags=true
--filters subscription-tag:env=prod
```

### 3. Subscription asset carries its subscription label — `subToAsset`

`subToAsset` (`providers/azure/resources/discovery.go:617`) currently sets no
`Labels` at all. Give the subscription asset the `SubscriptionLabel` so it is
uniformly keyable by the propagation pass (and so it correctly advertises which
subscription it represents):

```go
return &inventory.Asset{
	Id:          platformId,
	Platform:    platform,
	Name:        fmt.Sprintf("Azure subscription %s", *sub.DisplayName),
	Connections: []*inventory.Config{copyConf},
	PlatformIds: []string{platformId},
	Labels:      map[string]string{SubscriptionLabel: *sub.SubscriptionID},
}
```

This is a small, always-on correctness improvement (informational label only; the
subscription's *tags* are still merged only when the feature is enabled).

### 4. Core algorithm — `providers/azure/resources/discovery.go`

At the end of `Discover()`, after all assets are appended and before the return:

```go
if conn.Filters.PropagateSubscriptionTags {
	applySubscriptionTags(conn, subsWithConfigs, assets)
}
```

`applySubscriptionTags` iterates subscriptions and calls the GCP-shaped helper
once per subscription with that subscription's assets:

```go
// applySubscriptionTags merges each subscription's tags into the assets
// discovered within it. Tags come from an injected override when provided,
// otherwise from the Azure Subscriptions Get API. A per-subscription fetch
// failure is logged and skipped so discovery never fails.
func applySubscriptionTags(conn *connection.AzureConnection, subs []subWithConfig, assets []*inventory.Asset) {
	for _, s := range subs {
		if s.sub.SubscriptionID == nil {
			continue
		}
		subID := *s.sub.SubscriptionID

		tags := conn.Filters.SubscriptionTags
		if len(tags) == 0 {
			fetched, err := connection.NewSubscriptionsClient(conn.Token(), conn.ClientOptions()).
				GetSubscriptionTags(subID)
			if err != nil {
				log.Warn().Err(err).Str("subscription", subID).
					Msg("azure.discovery> failed to fetch subscription tags for propagation")
				continue
			}
			tags = fetched
		}
		if len(tags) == 0 {
			continue
		}

		subAssets := assetsForSubscription(assets, subID)
		propagateSubscriptionTagsToAssets(subAssets, tags)
	}
}

// assetsForSubscription returns the assets whose SubscriptionLabel matches subID.
func assetsForSubscription(assets []*inventory.Asset, subID string) []*inventory.Asset {
	res := []*inventory.Asset{}
	for _, a := range assets {
		if a != nil && a.Labels[SubscriptionLabel] == subID {
			res = append(res, a)
		}
	}
	return res
}
```

The subscription `Get` call lives in the connection layer alongside the existing
`GetSubscriptions`, keeping the ARM SDK usage out of the resources package
(consistent with how discovery already obtains subscriptions). Add to
`providers/azure/connection/client_subscriptions.go`:

```go
// GetSubscriptionTags reads a single subscription's tags via the Subscriptions
// Get API — the same endpoint resources/subscription.go uses for the
// azure.subscription.tags field, so it needs no additional permission.
func (client *subscriptionsClient) GetSubscriptionTags(subscriptionID string) (map[string]string, error) {
	subscriptionsC, err := subscriptions.NewClient(client.token, &arm.ClientOptions{
		ClientOptions: client.clientOptions,
	})
	if err != nil {
		return nil, err
	}
	resp, err := subscriptionsC.Get(context.Background(), subscriptionID, &subscriptions.ClientGetOptions{})
	if err != nil {
		return nil, err
	}
	return convert.PtrMapStrToStr(resp.Tags), nil
}
```

The merge helper is a direct analog of GCP's `propagateProjectLabelsToAssets`
(`providers/gcp/resources/discovery.go:1780`) — fill-missing, asset-own-wins,
nil-safe — and is independently unit-testable:

```go
// propagateSubscriptionTagsToAssets merges subscriptionTags into every asset in
// the slice. An asset's own labels take precedence, so subscription tags only
// fill in keys the asset doesn't already define.
func propagateSubscriptionTagsToAssets(assets []*inventory.Asset, subscriptionTags map[string]string) {
	if len(subscriptionTags) == 0 {
		return
	}
	for _, a := range assets {
		if a == nil {
			continue
		}
		if a.Labels == nil {
			a.Labels = map[string]string{}
		}
		for k, v := range subscriptionTags {
			if _, exists := a.Labels[k]; !exists {
				a.Labels[k] = v
			}
		}
	}
}
```

## Multi-subscription semantics

AWS has a single primary account; Azure discovery can span many subscriptions. So
propagation is per-subscription — each subscription's tags land only on the assets
whose `SubscriptionLabel` matches that subscription. This is closer to GCP's
per-project model than to AWS's single-account model, which is why the structure
follows GCP.

The injected `SubscriptionTags` override, when set, applies to **all** discovered
subscriptions (explicit-injection semantics matching AWS's `AccountTags`). Per-sub
override maps are a possible future extension and are out of scope here.

## Edge cases

- **Feature off (default)** — no tags fetched, no merge; only the always-on
  `SubscriptionLabel` on the subscription asset (from §3) changes.
- **Empty or nil subscription tags** — merge is a no-op; existing labels unchanged.
- **Asset with nil `Labels`** — initialized before merge.
- **`Get` fetch error** (e.g. transient API error, missing read permission) —
  warning logged for that subscription; discovery continues; other subscriptions
  unaffected.
- **Subscription with no `SubscriptionID`** — skipped defensively.
- **Key collision** — asset's own label wins per the collision rule.
- **Collision with informational labels** — impossible: Azure tag names cannot
  contain `/`, so a raw tag key can never equal `azure.mondoo.com/subscription`,
  `mondoo.com/location`, or `azure.mondoo.com/resourcegroup`.
- **Override + multiple subscriptions** — the override applies to every
  subscription (documented, intentional).

## Testing

Extend existing test files (no new packages):

1. `providers/azure/connection/filters_test.go` — `DiscoveryFiltersFromOpts`
   parses `propagate-subscription-tags=true` into `PropagateSubscriptionTags`, and
   `subscription-tag:env=prod` into `SubscriptionTags{"env":"prod"}`; absent keys
   yield the zero values; `parseMapOpt` trims the prefix correctly.
2. `providers/azure/resources/discovery_test.go` — `propagateSubscriptionTagsToAssets`
   (mirroring GCP's `TestPropagateProjectLabelsToAssets`): fill-missing; collision
   preserves the asset's value; nil input labels handled; empty tags is a no-op.
3. `providers/azure/resources/discovery_test.go` — `assetsForSubscription` /
   `applySubscriptionTags` with an in-memory `[]*inventory.Asset` spanning two
   subscriptions and an injected `SubscriptionTags` override (so no live client is
   needed): asserts each subscription's assets receive only their own tags, the
   subscription asset receives its own tags, and cross-subscription bleed does not
   occur. The live `Get` fetch is isolated in the connection layer's
   `GetSubscriptionTags` and is not unit-tested.

## Interactive verification

After `make providers/build/azure && make providers/install/azure`:

```
# baseline (feature off): subscription tags should NOT appear on child assets
mql shell azure --discover all
> asset { name labels }

# feature on: subscription tags appear as raw-key labels, resource tags win
mql shell azure --discover all --filters propagate-subscription-tags=true
> asset { name labels }
```

Confirm a known subscription-level tag shows up on VMs/storage/etc. and that a tag
key shared between the subscription and a resource keeps the resource's value.

## Files changed

- `providers/azure/connection/filters.go` — struct fields, parsing, `parseMapOpt`
- `providers/azure/connection/filters_test.go` — parsing tests
- `providers/azure/connection/client_subscriptions.go` — `GetSubscriptionTags` method
- `providers/azure/provider/provider.go` — `--filters` allowlist
- `providers/azure/config/config.go` — `--filters` flag `Desc`
- `providers/azure/resources/discovery.go` — `subToAsset` label, `applySubscriptionTags`,
  `assetsForSubscription`, `propagateSubscriptionTagsToAssets`, and the gated call
  in `Discover()`
- `providers/azure/resources/discovery_test.go` — propagation/merge tests
- `providers/azure/resources/azure.permissions.json` — regenerated by the provider
  build (expected no-op; the `subscriptions.Get` call already exists)
- Provider version bump (patch, `13.31.0` → next) handled via the release tooling

## Non-goals

- Per-subscription override maps. The single injected override applies to all subs.
- Configurable collision strategy. The rule is fixed: asset wins.
- Namespacing subscription tags with a prefix. Rejected in brainstorming — the
  collision rule plus Azure's `/`-free tag names make namespacing unnecessary.
- Propagating tenant-, management-group-, or resource-group-level tags. Only
  subscription tags are in scope.
- Inventory-supplied (root-asset) label propagation (the direction of GCP's
  unmerged `ccd975c92`). Could reuse the same helper later; out of scope here.
