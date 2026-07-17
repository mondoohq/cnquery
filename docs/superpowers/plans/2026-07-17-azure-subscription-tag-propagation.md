# Azure Subscription-Tag Propagation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an opt-in `--filters propagate-subscription-tags=true` to the Azure provider that merges each subscription's tags onto the assets discovered under it.

**Architecture:** Extend the existing `connection.DiscoveryFilters` seam (built by commit `544831aa8`) with `PropagateSubscriptionTags bool` + a `SubscriptionTags map[string]string` injected override. During `Discover()`, per subscription, resolve tags (override, else the Subscriptions `Get` API) and merge them onto that subscription's assets — keyed by the `azure.mondoo.com/subscription` label each asset already carries — with fill-missing / asset-wins semantics and raw tag keys. Mirrors GCP's per-project `propagateProjectLabelsToAssets`.

**Tech Stack:** Go, `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2`, `stretchr/testify`, the shared `providers-sdk/v1/util/filteropts` helpers.

## Global Constraints

- **Copyright header** on every new/modified source file, exactly:
  `// Copyright Mondoo, Inc. 2024, 2026` then `// SPDX-License-Identifier: BUSL-1.1`. (Modified files already have it — do not duplicate.)
- **Go module:** all `go`/`test` commands run from `/Users/vsirakov/Development/mql/providers/azure` (the Azure provider is its own module).
- **No `.lr` schema, no code generation, no `.lr.versions` changes** — this is discovery/connection Go code only.
- **Label semantics are fixed:** raw tag keys (no prefix), fill-missing merge, the asset's own label wins on collision. Do not add prefixing or a configurable collision strategy.
- **Discovery must never fail** because subscription tags could not be read: a per-subscription fetch error is logged (`log.Warn()`) and skipped.
- **Feature is off by default** (`PropagateSubscriptionTags` defaults to `false`).
- **Run `gofmt -w`** on every changed `.go` file before each commit.
- Provider version bump (patch, `13.31.0` → next) and any `azure.permissions.json` regeneration are handled in Task 6, not per-task.

**Spec:** `docs/superpowers/specs/2026-07-17-azure-subscription-tag-propagation-design.md`

---

### Task 1: Parse the new filter keys into `DiscoveryFilters`

**Files:**
- Modify: `providers/azure/connection/filters.go`
- Test: `providers/azure/connection/filters_test.go`

**Interfaces:**
- Consumes: nothing (foundation task).
- Produces:
  - `connection.DiscoveryFilters` gains fields `PropagateSubscriptionTags bool` and `SubscriptionTags map[string]string`.
  - `DiscoveryFiltersFromOpts(opts map[string]string) DiscoveryFilters` populates them from the keys `propagate-subscription-tags` (bool) and the `subscription-tag:` prefix family (map).

- [ ] **Step 1: Write the failing tests**

Add these subtests to the existing `TestDiscoveryFiltersFromOpts` function in `providers/azure/connection/filters_test.go` (append inside the function body, after the last existing `t.Run`):

```go
	t.Run("propagate-subscription-tags parses to true", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"propagate-subscription-tags": "true",
		})
		assert.True(t, f.PropagateSubscriptionTags)
	})

	t.Run("propagate-subscription-tags defaults to false", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(nil)
		assert.False(t, f.PropagateSubscriptionTags)
		assert.Empty(t, f.SubscriptionTags)
	})

	t.Run("subscription-tag: entries parse into the override map", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"subscription-tag:env":  "prod",
			"subscription-tag:team": "payments",
		})
		assert.Equal(t, map[string]string{"env": "prod", "team": "payments"}, f.SubscriptionTags)
	})

	t.Run("subscription-tag: skips empty values", func(t *testing.T) {
		f := DiscoveryFiltersFromOpts(map[string]string{
			"subscription-tag:env": "",
		})
		assert.Empty(t, f.SubscriptionTags)
	})
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./connection/ -run TestDiscoveryFiltersFromOpts -v`
Expected: FAIL to compile — `f.PropagateSubscriptionTags` and `f.SubscriptionTags` are undefined fields.

- [ ] **Step 3: Add the struct fields, parser, and helper**

In `providers/azure/connection/filters.go`, add `"strings"` to the import block:

```go
import (
	"slices"
	"strings"

	"go.mondoo.com/mql/v13/providers-sdk/v1/util/filteropts"
)
```

Add the two fields to `DiscoveryFilters`:

```go
type DiscoveryFilters struct {
	Subscriptions SubscriptionsFilter
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

Update `DiscoveryFiltersFromOpts` to populate them:

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

Add the `parseMapOpt` helper at the end of the file (mirrors `providers/aws/connection/filters.go:248`):

```go
// parseMapOpt collects all opts whose key starts with keyPrefix into a map,
// trimming the prefix from each key. Empty keys or values are skipped. Returns a
// non-nil empty map when nothing matches.
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

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./connection/ -run TestDiscoveryFiltersFromOpts -v`
Expected: PASS (all subtests, including the pre-existing ones).

- [ ] **Step 5: gofmt and commit**

```bash
cd /Users/vsirakov/Development/mql/providers/azure
gofmt -w connection/filters.go connection/filters_test.go
git add connection/filters.go connection/filters_test.go
git commit -m "✨ parse subscription-tag propagation filters in azure DiscoveryFilters"
```

---

### Task 2: Pass the new keys through the `--filters` CLI allowlist

**Files:**
- Modify: `providers/azure/provider/provider.go` (`parseFlagsToFiltersOpts`, imports)
- Modify: `providers/azure/config/config.go` (`--filters` flag `Desc`)
- Test: `providers/azure/provider/provider_test.go` (extend `TestParseCLIFiltersFlag`)

**Interfaces:**
- Consumes: nothing from earlier tasks (operates on the raw `--filters` map, before `DiscoveryFiltersFromOpts` runs at connection time).
- Produces: `propagate-subscription-tags` and `subscription-tag:<k>` keys survive from `--filters` into `inventory.Discovery.Filter`.

- [ ] **Step 1: Write the failing tests**

Add these subtests to the existing `TestParseCLIFiltersFlag` function in `providers/azure/provider/provider_test.go` (append inside the function body, after the last existing `t.Run`):

```go
	t.Run("propagate-subscription-tags passes through --filters", func(t *testing.T) {
		res, err := s.ParseCLI(&plugin.ParseCLIReq{
			Flags: map[string]*llx.Primitive{
				"filters": {Map: map[string]*llx.Primitive{
					"propagate-subscription-tags": llx.StringPrimitive("true"),
				}},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "true", connFilter(t, res)["propagate-subscription-tags"])
	})

	t.Run("subscription-tag: entries pass through --filters", func(t *testing.T) {
		res, err := s.ParseCLI(&plugin.ParseCLIReq{
			Flags: map[string]*llx.Primitive{
				"filters": {Map: map[string]*llx.Primitive{
					"subscription-tag:env":  llx.StringPrimitive("prod"),
					"subscription-tag:team": llx.StringPrimitive("payments"),
				}},
			},
		})
		require.NoError(t, err)
		f := connFilter(t, res)
		assert.Equal(t, "prod", f["subscription-tag:env"])
		assert.Equal(t, "payments", f["subscription-tag:team"])
	})
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./provider/ -run TestParseCLIFiltersFlag -v`
Expected: FAIL — the new keys are dropped by the allowlist, so the assertions on `connFilter(...)` values fail (empty string, not `"true"`/`"prod"`).

- [ ] **Step 3: Extend the allowlist and update the flag description**

In `providers/azure/provider/provider.go`, add `"strings"` to the import block (it is not currently imported):

```go
import (
	"context"
	"errors"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	// ... existing imports unchanged ...
)
```

Replace the `--filters` allowlist loop in `parseFlagsToFiltersOpts` (currently matching only `subscriptions`/`subscriptions-exclude`) with:

```go
	// base: the --filters key/value flag (allowlisted keys only)
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

Leave the dedicated-flag overlay below it (the `--subscription` / `--subscriptions` / `--subscriptions-exclude` block) unchanged.

In `providers/azure/config/config.go`, update the `--filters` flag `Desc` (currently at config.go:123) to document the new keys:

```go
					Desc:    "Filter options, e.g., --filters subscriptions=<id1>,<id2> --filters subscriptions-exclude=<id3> --filters propagate-subscription-tags=true --filters subscription-tag:<key>=<value>",
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./provider/ -run TestParseCLIFiltersFlag -v`
Expected: PASS (all subtests, including the pre-existing ones — the `"unknown filters keys are ignored"` case still passes because `regions` matches no allowlist branch).

- [ ] **Step 5: gofmt and commit**

```bash
cd /Users/vsirakov/Development/mql/providers/azure
gofmt -w provider/provider.go config/config.go provider/provider_test.go
git add provider/provider.go config/config.go provider/provider_test.go
git commit -m "✨ allow subscription-tag propagation keys through azure --filters"
```

---

### Task 3: Add the connection-layer `GetSubscriptionTags`

**Files:**
- Modify: `providers/azure/connection/client_subscriptions.go` (add method + `convert` import)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: method `(*subscriptionsClient).GetSubscriptionTags(subscriptionID string) (map[string]string, error)` — used by Task 5's `applySubscriptionTags`.

**Note on testing:** this is a thin wrapper over the ARM Subscriptions `Get` endpoint (it constructs a live `subscriptions.NewClient` and makes a network call), exactly like the existing `GetSubscriptions` in the same file, which has no unit test. There is no mock/recording harness for it in this package, so it is not unit-tested here; it is exercised by the interactive verification in Task 6. Correctness is guaranteed by a compile check in this task and by Task 5 (whose test uses the injected override, never reaching this method).

- [ ] **Step 1: Add the method and import**

In `providers/azure/connection/client_subscriptions.go`, add the `convert` import:

```go
import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)
```

Add the method at the end of the file (after `GetSubscriptions`):

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

- [ ] **Step 2: Verify it compiles**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go build ./connection/...`
Expected: no output (success).

- [ ] **Step 3: gofmt and commit**

```bash
cd /Users/vsirakov/Development/mql/providers/azure
gofmt -w connection/client_subscriptions.go
git add connection/client_subscriptions.go
git commit -m "✨ add GetSubscriptionTags to azure subscriptions client"
```

---

### Task 4: Add the pure merge helpers

**Files:**
- Modify: `providers/azure/resources/discovery.go` (add two functions)
- Test: `providers/azure/resources/discovery_test.go`

**Interfaces:**
- Consumes: the existing `SubscriptionLabel` constant (`"azure.mondoo.com/subscription"`, defined in `discovery.go`).
- Produces:
  - `propagateSubscriptionTagsToAssets(assets []*inventory.Asset, subscriptionTags map[string]string)` — merges tags into each asset, fill-missing, asset-wins, nil-safe.
  - `assetsForSubscription(assets []*inventory.Asset, subID string) []*inventory.Asset` — returns the assets whose `SubscriptionLabel` equals `subID`.

- [ ] **Step 1: Write the failing tests**

Add to `providers/azure/resources/discovery_test.go` (the file already imports `testing`, `require`, and `inventory` — no new imports needed for this task):

```go
func TestPropagateSubscriptionTagsToAssets(t *testing.T) {
	t.Run("fills missing keys", func(t *testing.T) {
		assets := []*inventory.Asset{{Labels: map[string]string{"a": "1"}}}
		propagateSubscriptionTagsToAssets(assets, map[string]string{"b": "2"})
		require.Equal(t, map[string]string{"a": "1", "b": "2"}, assets[0].Labels)
	})

	t.Run("asset label wins on collision", func(t *testing.T) {
		assets := []*inventory.Asset{{Labels: map[string]string{"env": "dev"}}}
		propagateSubscriptionTagsToAssets(assets, map[string]string{"env": "prod"})
		require.Equal(t, "dev", assets[0].Labels["env"])
	})

	t.Run("nil asset labels are initialized", func(t *testing.T) {
		assets := []*inventory.Asset{{}}
		propagateSubscriptionTagsToAssets(assets, map[string]string{"b": "2"})
		require.Equal(t, map[string]string{"b": "2"}, assets[0].Labels)
	})

	t.Run("empty tags is a no-op", func(t *testing.T) {
		assets := []*inventory.Asset{{Labels: map[string]string{"a": "1"}}}
		propagateSubscriptionTagsToAssets(assets, nil)
		require.Equal(t, map[string]string{"a": "1"}, assets[0].Labels)
	})

	t.Run("nil asset in slice is skipped", func(t *testing.T) {
		assets := []*inventory.Asset{nil, {Labels: map[string]string{}}}
		require.NotPanics(t, func() {
			propagateSubscriptionTagsToAssets(assets, map[string]string{"b": "2"})
		})
		require.Equal(t, map[string]string{"b": "2"}, assets[1].Labels)
	})
}

func TestAssetsForSubscription(t *testing.T) {
	assets := []*inventory.Asset{
		{Name: "a", Labels: map[string]string{SubscriptionLabel: "sub-1"}},
		{Name: "b", Labels: map[string]string{SubscriptionLabel: "sub-2"}},
		{Name: "c", Labels: map[string]string{SubscriptionLabel: "sub-1"}},
		nil,
		{Name: "d", Labels: nil},
	}
	got := assetsForSubscription(assets, "sub-1")
	require.Len(t, got, 2)
	require.Equal(t, "a", got[0].Name)
	require.Equal(t, "c", got[1].Name)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./resources/ -run 'TestPropagateSubscriptionTagsToAssets|TestAssetsForSubscription' -v`
Expected: FAIL to compile — `propagateSubscriptionTagsToAssets` and `assetsForSubscription` are undefined.

- [ ] **Step 3: Implement the helpers**

Add to `providers/azure/resources/discovery.go` (near the other discovery helpers, e.g. just after `subToAsset`):

```go
// propagateSubscriptionTagsToAssets merges subscriptionTags into every asset in
// the slice. An asset's own labels take precedence, so subscription tags only
// fill in keys the asset doesn't already define. Mirrors GCP's
// propagateProjectLabelsToAssets.
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

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./resources/ -run 'TestPropagateSubscriptionTagsToAssets|TestAssetsForSubscription' -v`
Expected: PASS.

- [ ] **Step 5: gofmt and commit**

```bash
cd /Users/vsirakov/Development/mql/providers/azure
gofmt -w resources/discovery.go resources/discovery_test.go
git add resources/discovery.go resources/discovery_test.go
git commit -m "✨ add azure subscription-tag merge helpers"
```

---

### Task 5: Wire propagation into discovery

**Files:**
- Modify: `providers/azure/resources/discovery.go` (`subToAsset`, add `applySubscriptionTags`, call in `Discover`)
- Test: `providers/azure/resources/discovery_test.go`

**Interfaces:**
- Consumes:
  - `propagateSubscriptionTagsToAssets`, `assetsForSubscription` (Task 4).
  - `connection.DiscoveryFilters.PropagateSubscriptionTags` / `.SubscriptionTags` (Task 1).
  - `(*connection.subscriptionsClient).GetSubscriptionTags` via `connection.NewSubscriptionsClient(...)` (Task 3).
  - Existing: `subWithConfig` struct (`{ sub subscriptions.Subscription; conf *inventory.Config }`), `SubscriptionLabel` constant, `conn.Token()`, `conn.ClientOptions()`.
- Produces: `applySubscriptionTags(conn *connection.AzureConnection, subs []subWithConfig, assets []*inventory.Asset)`; the subscription asset from `subToAsset` now carries `SubscriptionLabel`.

- [ ] **Step 1: Write the failing tests**

Add to the import block of `providers/azure/resources/discovery_test.go` (it currently imports `errors`, `slices`, `testing`, `require`, `inventory`, `plugin`):

```go
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2"
	"go.mondoo.com/mql/v13/providers/azure/connection"
```

Add these tests:

```go
func TestSubToAsset_SetsSubscriptionLabel(t *testing.T) {
	asset := subToAsset(subWithConfig{
		sub: subscriptions.Subscription{
			SubscriptionID: to.Ptr("sub-1"),
			DisplayName:    to.Ptr("My Sub"),
			TenantID:       to.Ptr("tenant-1"),
		},
		conf: &inventory.Config{},
	})
	require.Equal(t, "sub-1", asset.Labels[SubscriptionLabel])
}

func TestApplySubscriptionTags_Override(t *testing.T) {
	conn := &connection.AzureConnection{
		Filters: connection.DiscoveryFilters{
			PropagateSubscriptionTags: true,
			SubscriptionTags:          map[string]string{"env": "prod"},
		},
	}
	subs := []subWithConfig{
		{sub: subscriptions.Subscription{SubscriptionID: to.Ptr("sub-1")}},
		{sub: subscriptions.Subscription{SubscriptionID: to.Ptr("sub-2")}},
	}
	assets := []*inventory.Asset{
		{Name: "vm1", Labels: map[string]string{SubscriptionLabel: "sub-1"}},
		{Name: "vm2", Labels: map[string]string{SubscriptionLabel: "sub-2", "env": "dev"}},
		{Name: "orphan", Labels: map[string]string{SubscriptionLabel: "sub-3"}},
	}

	applySubscriptionTags(conn, subs, assets)

	require.Equal(t, "prod", assets[0].Labels["env"]) // filled from the override
	require.Equal(t, "dev", assets[1].Labels["env"])  // asset value wins on collision
	require.NotContains(t, assets[2].Labels, "env")   // sub-3 not in subs list — untouched
}
```

> Note: `TestApplySubscriptionTags_Override` sets the injected override, so `applySubscriptionTags` never constructs a live client (the `len(tags) == 0` branch is not taken). No network access, no auth.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./resources/ -run 'TestSubToAsset_SetsSubscriptionLabel|TestApplySubscriptionTags_Override' -v`
Expected: FAIL — `TestSubToAsset_SetsSubscriptionLabel` fails on the label assertion (subscription asset has no `Labels`, so the map read returns `""`), and `applySubscriptionTags` is undefined (compile error).

- [ ] **Step 3: Set the subscription label, add `applySubscriptionTags`, and call it from `Discover`**

In `providers/azure/resources/discovery.go`, in `subToAsset`, add the `Labels` field to the returned asset:

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

Add `applySubscriptionTags` (next to the Task 4 helpers):

```go
// applySubscriptionTags merges each subscription's tags into the assets
// discovered within it. Tags come from the injected override when provided,
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

		propagateSubscriptionTagsToAssets(assetsForSubscription(assets, subID), tags)
	}
}
```

In `Discover`, add the gated call after the generic assets are appended and before the final `log.Debug(...)`/`return` (i.e. right after `assets = append(assets, genericAssets...)`):

```go
	if conn.Filters.PropagateSubscriptionTags {
		applySubscriptionTags(conn, subsWithConfigs, assets)
	}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./resources/ -run 'TestSubToAsset_SetsSubscriptionLabel|TestApplySubscriptionTags_Override' -v`
Expected: PASS.

- [ ] **Step 5: Run the full resources + connection + provider test packages**

Run: `cd /Users/vsirakov/Development/mql/providers/azure && go test ./resources/ ./connection/ ./provider/`
Expected: PASS (`ok` for each package) — no regressions in the existing discovery/label tests.

- [ ] **Step 6: gofmt and commit**

```bash
cd /Users/vsirakov/Development/mql/providers/azure
gofmt -w resources/discovery.go resources/discovery_test.go
git add resources/discovery.go resources/discovery_test.go
git commit -m "✨ propagate azure subscription tags onto discovered assets"
```

---

### Task 6: Build, verify against live Azure, and finalize

**Files:**
- Possibly regenerated: `providers/azure/resources/azure.permissions.json`

**Interfaces:**
- Consumes: everything from Tasks 1–5.
- Produces: an installed provider binary and a confirmed end-to-end behavior; committed `azure.permissions.json` if the build changed it.

- [ ] **Step 1: Build and install the provider**

```bash
cd /Users/vsirakov/Development/mql
make providers/build/azure && make providers/install/azure
```
Expected: builds and installs without error.

- [ ] **Step 2: Check whether the permissions manifest changed**

Run: `cd /Users/vsirakov/Development/mql && git status --porcelain providers/azure/resources/azure.permissions.json`
Expected: no output (no change — the `subscriptions.Get` permission is already used by the `azure.subscription` resource). If it *did* change, that is a legitimate part of this PR: `git add` and `git commit -m "✨ update azure permissions for subscription-tag propagation"`.

- [ ] **Step 3: Verify the feature is OFF by default**

Requires a working Azure auth context (Azure CLI logged in). Run:

```bash
mql shell azure --discover all
```

In the shell, pick any resource asset and confirm subscription tags are NOT present:
```
asset { name labels }
```
Expected: labels contain `azure.mondoo.com/subscription`, `mondoo.com/location`, resource-group and the resource's own tags — but NOT the subscription-level tags.

- [ ] **Step 4: Verify the feature is ON with the flag**

```bash
mql shell azure --discover all --filters propagate-subscription-tags=true
```

In the shell:
```
asset { name labels }
```
Expected: every discovered asset's `labels` now additionally contains the subscription's tags as raw keys. Confirm two behaviors against a subscription that has at least one tag set (set one in the Azure portal / `az tag create` on the subscription scope first if needed):
1. A subscription-level tag key that the resource does NOT define appears on the asset with the subscription's value.
2. A tag key defined on BOTH the subscription and the resource keeps the RESOURCE's value (asset-wins).

Also confirm the injected-override path shape:
```bash
mql shell azure --discover all --filters propagate-subscription-tags=true --filters subscription-tag:injected=yes
> asset { name labels }
```
Expected: every asset carries `injected: yes` (the override replaces the API fetch for all subscriptions).

- [ ] **Step 5: Confirm `go.mod` is clean and the tree matches generated code**

```bash
cd /Users/vsirakov/Development/mql/providers/azure && go mod tidy && git diff --exit-code go.mod go.sum
```
Expected: no diff.

- [ ] **Step 6: Final state**

All tasks committed on branch `azure-subscription-tag-propagation`. The provider version bump (`13.31.0` → next patch) is handled separately at release time via the `provider-release` skill / `version` tooling — do not hand-edit `config/config.go`'s `Version` in this PR unless the team's convention is to bump in-feature.

---

## Notes for the executor

- If any existing test in `./resources/`, `./connection/`, or `./provider/` breaks, STOP and investigate — the change is intended to be additive. The one intentional behavior change is that `subToAsset` now sets `Labels` (previously `nil`); if a test asserted the subscription asset had nil labels, update it to expect the `SubscriptionLabel`.
- Azure tag names cannot contain `/`, so raw subscription-tag keys can never collide with the `mondoo.com/*` / `azure.mondoo.com/*` informational labels — no special-casing needed.
- `to.Ptr` comes from `github.com/Azure/azure-sdk-for-go/sdk/azcore/to`, already an indirect dependency of the provider (used across the Azure SDK). If `go` reports it as not directly required after adding the test import, run `go mod tidy` (Step 5 of Task 6 covers this).
```
