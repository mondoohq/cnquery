# AWS Account Tag Propagation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Copy tags from the primary AWS account asset onto all discovered AWS resource assets during discovery, with per-asset labels winning on key collisions and sibling org account assets excluded.

**Architecture:** Post-process the asset list at the end of `Discover()` in `providers/aws/resources/discovery.go`. Fetch primary account tags once via `awsAccount.GetTags()`, then iterate the final asset list and merge tags into each asset's `Labels` using a helper in `discovery_conversion.go`. A predicate distinguishes sibling org account assets (which are skipped) from the primary account asset (which is merged).

**Tech Stack:** Go, mql provider SDK, AWS SDK v2, testify.

**Spec:** `docs/superpowers/specs/2026-04-15-aws-account-tags-propagation-design.md`

---

## File Structure

- **Modify** `providers/aws/resources/discovery.go` — fetch tags once in `Discover()` and apply merge pass after discovery loop.
- **Modify** `providers/aws/resources/discovery_conversion.go` — add `mergeAccountTagsIntoLabels` and `isSiblingOrgAccountAsset` helpers; initialize `Labels` on the primary account asset in `accountAsset()`.
- **Modify** `providers/aws/resources/discovery_test.go` — add unit tests for the two helpers and an end-to-end merge pass test.

No new files. All changes live next to existing discovery code.

---

### Task 1: Add `mergeAccountTagsIntoLabels` helper with tests

**Files:**
- Modify: `providers/aws/resources/discovery_conversion.go` (append new helper near the end, above `instanceInfo` struct around line 275)
- Modify: `providers/aws/resources/discovery_test.go` (append tests at end of file)

- [ ] **Step 1: Write the failing test**

Append to `providers/aws/resources/discovery_test.go`:

```go
func TestMergeAccountTagsIntoLabels(t *testing.T) {
	t.Run("asset label wins on collision", func(t *testing.T) {
		labels := map[string]string{"Environment": "prod"}
		accountTags := map[string]string{"Environment": "staging", "Owner": "team-a"}

		got := mergeAccountTagsIntoLabels(labels, accountTags)

		require.Equal(t, "prod", got["Environment"])
		require.Equal(t, "team-a", got["Owner"])
	})

	t.Run("nil labels initialized", func(t *testing.T) {
		accountTags := map[string]string{"Owner": "team-a"}

		got := mergeAccountTagsIntoLabels(nil, accountTags)

		require.Equal(t, map[string]string{"Owner": "team-a"}, got)
	})

	t.Run("empty account tags is a no-op", func(t *testing.T) {
		labels := map[string]string{"Environment": "prod"}

		got := mergeAccountTagsIntoLabels(labels, map[string]string{})

		require.Equal(t, map[string]string{"Environment": "prod"}, got)
	})

	t.Run("nil account tags is a no-op", func(t *testing.T) {
		labels := map[string]string{"Environment": "prod"}

		got := mergeAccountTagsIntoLabels(labels, nil)

		require.Equal(t, map[string]string{"Environment": "prod"}, got)
	})

	t.Run("fills gaps without touching existing keys", func(t *testing.T) {
		labels := map[string]string{"Name": "web-01"}
		accountTags := map[string]string{"Owner": "team-a", "CostCenter": "cc-42"}

		got := mergeAccountTagsIntoLabels(labels, accountTags)

		require.Equal(t, "web-01", got["Name"])
		require.Equal(t, "team-a", got["Owner"])
		require.Equal(t, "cc-42", got["CostCenter"])
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./providers/aws/resources/ -run TestMergeAccountTagsIntoLabels -v`
Expected: FAIL — compilation error "undefined: mergeAccountTagsIntoLabels".

- [ ] **Step 3: Implement the helper**

Append to `providers/aws/resources/discovery_conversion.go` (just above the `instanceInfo` struct near line 275):

```go
// mergeAccountTagsIntoLabels merges account-level tags into an asset's labels.
// The asset's existing labels always win on key collisions — account tags only
// fill gaps. A nil labels map is initialized. An empty or nil accountTags map
// is a no-op.
func mergeAccountTagsIntoLabels(labels, accountTags map[string]string) map[string]string {
	if len(accountTags) == 0 {
		return labels
	}
	if labels == nil {
		labels = map[string]string{}
	}
	for k, v := range accountTags {
		if _, exists := labels[k]; !exists {
			labels[k] = v
		}
	}
	return labels
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./providers/aws/resources/ -run TestMergeAccountTagsIntoLabels -v`
Expected: PASS — all five subtests pass.

- [ ] **Step 5: Commit**

```bash
git add providers/aws/resources/discovery_conversion.go providers/aws/resources/discovery_test.go
git commit -m "$(cat <<'EOF'
🌟 add mergeAccountTagsIntoLabels helper for AWS discovery

Asset labels win on key collisions; account tags fill gaps only.
EOF
)"
```

---

### Task 2: Add `isSiblingOrgAccountAsset` predicate with tests

**Files:**
- Modify: `providers/aws/resources/discovery_conversion.go`
- Modify: `providers/aws/resources/discovery_test.go`

- [ ] **Step 1: Write the failing test**

Append to `providers/aws/resources/discovery_test.go`:

```go
func TestIsSiblingOrgAccountAsset(t *testing.T) {
	primaryAccountId := "111111111111"

	t.Run("primary account asset returns false", func(t *testing.T) {
		a := &inventory.Asset{
			Platform:    &inventory.Platform{Name: "aws-account"},
			PlatformIds: []string{"//platformid.api.mondoo.app/runtime/aws/accounts/111111111111"},
		}
		require.False(t, isSiblingOrgAccountAsset(a, primaryAccountId))
	})

	t.Run("sibling account asset returns true", func(t *testing.T) {
		a := &inventory.Asset{
			Platform:    &inventory.Platform{Name: "aws-account"},
			PlatformIds: []string{"//platformid.api.mondoo.app/runtime/aws/accounts/222222222222"},
		}
		require.True(t, isSiblingOrgAccountAsset(a, primaryAccountId))
	})

	t.Run("non-account asset returns false", func(t *testing.T) {
		a := &inventory.Asset{
			Platform:    &inventory.Platform{Name: "aws-ec2-instance"},
			PlatformIds: []string{"//platformid.api.mondoo.app/runtime/aws/accounts/222222222222/regions/us-east-1/instances/i-abc"},
		}
		require.False(t, isSiblingOrgAccountAsset(a, primaryAccountId))
	})

	t.Run("nil asset returns false", func(t *testing.T) {
		require.False(t, isSiblingOrgAccountAsset(nil, primaryAccountId))
	})

	t.Run("asset with nil platform returns false", func(t *testing.T) {
		a := &inventory.Asset{
			PlatformIds: []string{"//platformid.api.mondoo.app/runtime/aws/accounts/222222222222"},
		}
		require.False(t, isSiblingOrgAccountAsset(a, primaryAccountId))
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./providers/aws/resources/ -run TestIsSiblingOrgAccountAsset -v`
Expected: FAIL — compilation error "undefined: isSiblingOrgAccountAsset".

- [ ] **Step 3: Implement the predicate**

Append to `providers/aws/resources/discovery_conversion.go` right after `mergeAccountTagsIntoLabels`:

```go
// isSiblingOrgAccountAsset returns true when the asset represents an AWS
// account other than the primary account (i.e. a sibling discovered via the
// organization target). These assets must not inherit the primary account's
// tags.
func isSiblingOrgAccountAsset(a *inventory.Asset, primaryAccountId string) bool {
	if a == nil || a.Platform == nil || a.Platform.Name != "aws-account" {
		return false
	}
	suffix := "/accounts/" + primaryAccountId
	for _, pid := range a.PlatformIds {
		if strings.HasSuffix(pid, suffix) {
			return false
		}
	}
	return true
}
```

Verify `strings` is already imported in `discovery_conversion.go`. It is (used by `trimAwsAccountIdToJustId`).

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./providers/aws/resources/ -run TestIsSiblingOrgAccountAsset -v`
Expected: PASS — all five subtests pass.

- [ ] **Step 5: Commit**

```bash
git add providers/aws/resources/discovery_conversion.go providers/aws/resources/discovery_test.go
git commit -m "$(cat <<'EOF'
🌟 add isSiblingOrgAccountAsset predicate for AWS discovery

Distinguishes the primary account asset from sibling accounts
discovered via the organization target.
EOF
)"
```

---

### Task 3: Initialize Labels on the primary account asset

**Files:**
- Modify: `providers/aws/resources/discovery_conversion.go:245-251` (the return in `accountAsset`)

- [ ] **Step 1: Read the current `accountAsset` return block**

The current code at `providers/aws/resources/discovery_conversion.go:245-251`:

```go
	return &inventory.Asset{
		PlatformIds: []string{id, accountArn},
		Name:        name,
		Platform:    connection.GetPlatformForObject("", accountId),
		Connections: []*inventory.Config{clonedConfig},
		Options:     conn.ConnectionOptions(),
	}
```

- [ ] **Step 2: Add `Labels` field initialized to an empty map**

Replace the return block in `accountAsset` with:

```go
	return &inventory.Asset{
		PlatformIds: []string{id, accountArn},
		Name:        name,
		Platform:    connection.GetPlatformForObject("", accountId),
		Labels:      map[string]string{},
		Connections: []*inventory.Config{clonedConfig},
		Options:     conn.ConnectionOptions(),
	}
```

- [ ] **Step 3: Verify build still passes**

Run: `go build ./providers/aws/resources/...`
Expected: no output, exit 0.

- [ ] **Step 4: Run existing discovery tests**

Run: `go test ./providers/aws/resources/ -run TestDiscovery -v`
Expected: all existing tests still pass.

- [ ] **Step 5: Commit**

```bash
git add providers/aws/resources/discovery_conversion.go
git commit -m "$(cat <<'EOF'
🧹 initialize Labels map on AWS account asset

Prepares the primary account asset to participate uniformly in the
tag-propagation merge pass.
EOF
)"
```

---

### Task 4: Wire merge pass into `Discover()` with end-to-end test

**Files:**
- Modify: `providers/aws/resources/discovery.go:112-135` (the `Discover` function)
- Modify: `providers/aws/resources/discovery_test.go` (add end-to-end merge pass test)

- [ ] **Step 1: Write the failing test**

Append to `providers/aws/resources/discovery_test.go`:

```go
func TestApplyAccountTagsToAssets(t *testing.T) {
	primaryAccountId := "111111111111"
	accountTags := map[string]string{
		"Owner":       "team-a",
		"CostCenter":  "cc-42",
		"Environment": "staging",
	}

	primaryAccount := &inventory.Asset{
		Name:        "primary",
		Platform:    &inventory.Platform{Name: "aws-account"},
		PlatformIds: []string{"//platformid.api.mondoo.app/runtime/aws/accounts/111111111111"},
		Labels:      map[string]string{},
	}
	siblingAccount := &inventory.Asset{
		Name:        "sibling",
		Platform:    &inventory.Platform{Name: "aws-account"},
		PlatformIds: []string{"//platformid.api.mondoo.app/runtime/aws/accounts/222222222222"},
		Labels:      map[string]string{"OtherAccountTag": "x"},
	}
	ec2Asset := &inventory.Asset{
		Name:        "web-01",
		Platform:    &inventory.Platform{Name: "aws-ec2-instance"},
		PlatformIds: []string{"//platformid.api.mondoo.app/runtime/aws/accounts/111111111111/regions/us-east-1/instances/i-abc"},
		Labels:      map[string]string{"Environment": "prod", "Name": "web-01"},
	}
	assets := []*inventory.Asset{primaryAccount, siblingAccount, ec2Asset}

	applyAccountTagsToAssets(assets, accountTags, primaryAccountId)

	// Primary account asset receives all account tags.
	require.Equal(t, "team-a", primaryAccount.Labels["Owner"])
	require.Equal(t, "cc-42", primaryAccount.Labels["CostCenter"])
	require.Equal(t, "staging", primaryAccount.Labels["Environment"])

	// Sibling account is untouched.
	require.Equal(t, map[string]string{"OtherAccountTag": "x"}, siblingAccount.Labels)

	// EC2 asset: collision on "Environment" keeps the asset's value; other tags fill gaps.
	require.Equal(t, "prod", ec2Asset.Labels["Environment"])
	require.Equal(t, "web-01", ec2Asset.Labels["Name"])
	require.Equal(t, "team-a", ec2Asset.Labels["Owner"])
	require.Equal(t, "cc-42", ec2Asset.Labels["CostCenter"])
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./providers/aws/resources/ -run TestApplyAccountTagsToAssets -v`
Expected: FAIL — compilation error "undefined: applyAccountTagsToAssets".

- [ ] **Step 3: Add the `applyAccountTagsToAssets` helper**

Append to `providers/aws/resources/discovery_conversion.go` right after `isSiblingOrgAccountAsset`:

```go
// applyAccountTagsToAssets merges the primary account's tags into every asset
// in the list, except sibling org account assets which are skipped. The merge
// follows mergeAccountTagsIntoLabels semantics: asset labels win on collision.
func applyAccountTagsToAssets(assets []*inventory.Asset, accountTags map[string]string, primaryAccountId string) {
	if len(accountTags) == 0 {
		return
	}
	for _, a := range assets {
		if a == nil {
			continue
		}
		if isSiblingOrgAccountAsset(a, primaryAccountId) {
			continue
		}
		a.Labels = mergeAccountTagsIntoLabels(a.Labels, accountTags)
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./providers/aws/resources/ -run TestApplyAccountTagsToAssets -v`
Expected: PASS.

- [ ] **Step 5: Wire the merge pass into `Discover()`**

The current `Discover` function in `providers/aws/resources/discovery.go:112-135`:

```go
func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.AwsConnection)
	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	res, err := NewResource(runtime, ResourceAwsAccount, map[string]*llx.RawData{"id": llx.StringData("aws.account/" + conn.AccountId())})
	if err != nil {
		return nil, err
	}

	awsAccount := res.(*mqlAwsAccount)

	targets := getDiscoveryTargets(conn.Conf)
	for _, target := range targets {
		list, err := discover(runtime, awsAccount, target, conn.Filters)
		if err != nil {
			log.Error().Err(err).Msg("error during discovery")
			continue
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	}
	return in, nil
}
```

Replace it with:

```go
func Discover(runtime *plugin.Runtime) (*inventory.Inventory, error) {
	conn := runtime.Connection.(*connection.AwsConnection)
	in := &inventory.Inventory{Spec: &inventory.InventorySpec{
		Assets: []*inventory.Asset{},
	}}

	res, err := NewResource(runtime, ResourceAwsAccount, map[string]*llx.RawData{"id": llx.StringData("aws.account/" + conn.AccountId())})
	if err != nil {
		return nil, err
	}

	awsAccount := res.(*mqlAwsAccount)

	accountTags := fetchPrimaryAccountTags(awsAccount)
	primaryAccountId := trimAwsAccountIdToJustId(awsAccount.Id.Data)

	targets := getDiscoveryTargets(conn.Conf)
	for _, target := range targets {
		list, err := discover(runtime, awsAccount, target, conn.Filters)
		if err != nil {
			log.Error().Err(err).Msg("error during discovery")
			continue
		}
		in.Spec.Assets = append(in.Spec.Assets, list...)
	}

	applyAccountTagsToAssets(in.Spec.Assets, accountTags, primaryAccountId)

	return in, nil
}

// fetchPrimaryAccountTags returns the primary AWS account's tags as a plain
// string map. Any error reading tags is logged and an empty map is returned so
// discovery can proceed.
func fetchPrimaryAccountTags(awsAccount *mqlAwsAccount) map[string]string {
	t := awsAccount.GetTags()
	if t == nil {
		return map[string]string{}
	}
	if t.Error != nil {
		log.Warn().Err(t.Error).Msg("failed to read AWS account tags; proceeding without account-level tag propagation")
		return map[string]string{}
	}
	if t.Data == nil {
		return map[string]string{}
	}
	return mapStringInterfaceToStringString(t.Data)
}
```

- [ ] **Step 6: Build the provider**

Run: `go build ./providers/aws/...`
Expected: no output, exit 0.

- [ ] **Step 7: Run all discovery tests**

Run: `go test ./providers/aws/resources/ -run 'TestDiscovery|TestMergeAccountTagsIntoLabels|TestIsSiblingOrgAccountAsset|TestApplyAccountTagsToAssets|TestAllResolvedResources' -v`
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add providers/aws/resources/discovery.go providers/aws/resources/discovery_conversion.go providers/aws/resources/discovery_test.go
git commit -m "$(cat <<'EOF'
🌟 propagate AWS account tags to discovered assets

Fetch the primary account's tags once in Discover() and merge them
into every discovered asset's Labels. Per-asset labels win on key
collisions, and sibling org account assets are skipped.
EOF
)"
```

---

### Task 5: Build and install the provider; manual verification

**Files:** none (verification only)

- [ ] **Step 1: Format any changed Go files**

Run: `gofmt -w providers/aws/resources/discovery.go providers/aws/resources/discovery_conversion.go providers/aws/resources/discovery_test.go`
Expected: no output.

- [ ] **Step 2: Run linting on the resources package**

Run: `go vet ./providers/aws/resources/...`
Expected: no output.

- [ ] **Step 3: Build and install the provider**

Run: `make providers/build/aws && make providers/install/aws`
Expected: provider installs to `~/.config/mondoo/providers/aws`.

- [ ] **Step 4: Manual verification via mql shell**

Run: `mql run aws --discover accounts,ec2-instances -c "asset { name labels }"`

Expected: the primary account asset's tags appear as labels on both the account asset itself and the EC2 instance assets. If an EC2 instance already has a label matching an account tag key, the instance's value is preserved.

If you don't have a tagged AWS account available, verify by adding a temporary tag via `aws organizations tag-resource --resource-id <account-id> --tags Key=TestTag,Value=propagated` before running the command.

- [ ] **Step 5: Run the full resources test package**

Run: `go test ./providers/aws/resources/...`
Expected: all pass.

---

## Self-Review Notes

**Spec coverage:**
- Fetch tags once in Discover — Task 4 step 5.
- Merge helper with collision rule — Task 1.
- Sibling detection predicate — Task 2.
- Apply-to-all pass — Task 4 (helper + wiring).
- Initialize Labels on primary account asset — Task 3.
- Error handling for `GetTags()` failure — Task 4 `fetchPrimaryAccountTags`.
- Unit tests for both helpers and end-to-end merge pass — Tasks 1, 2, 4.

**No placeholders.** Every code step shows the exact code to write.

**Type consistency.** `mergeAccountTagsIntoLabels`, `isSiblingOrgAccountAsset`, `applyAccountTagsToAssets`, and `fetchPrimaryAccountTags` are named consistently across tasks.

**Non-goals (from spec):** tag propagation from sibling accounts, configurable collision strategy, namespacing — none of these are in the plan, as expected.
