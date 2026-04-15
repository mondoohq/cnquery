# Propagate primary AWS account tags to discovered assets

## Goal

During AWS discovery, every asset discovered from the primary AWS account should inherit the account's tags as labels. The asset's own labels take precedence on key collisions. Sibling account assets discovered via the organization target are excluded.

## Motivation

Today, `providers/aws/resources/discovery.go` discovers child assets (EC2 instances, S3 buckets, RDS clusters, etc.) and populates each asset's `Labels` from the resource's own tags. Account-level tags (e.g. `CostCenter`, `Owner`, `Environment` set at the AWS account level) are not propagated, so downstream consumers can't filter or group discovered assets by account-scoped metadata.

## Collision rule

The asset's own label wins. Account tags only fill gaps. Rationale: per-resource tags are more specific and should not be silently overwritten by account-wide defaults.

## Scope

Tags are applied to:

- Every discovered resource asset (EC2 instance, S3 bucket, VPC, etc.)
- The primary account asset itself (so it carries its own tags as labels)

Tags are NOT applied to:

- Sibling account assets emitted by `DiscoveryOrg` (these represent other AWS accounts with their own identity)

## Design

### Fetch account tags once

In `Discover()` (`providers/aws/resources/discovery.go`), after `awsAccount` is created, fetch tags once:

```go
accountTags := map[string]string{}
if t := awsAccount.GetTags(); t != nil && t.Error == nil {
    accountTags = mapStringInterfaceToStringString(t.Data)
}
```

On error (e.g. missing `organizations:ListTagsForResource` permission), log a warning and proceed with an empty map. Discovery must not fail because tags could not be read.

### Merge helper

Add to `discovery_conversion.go`:

```go
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

### Sibling-account detection

Sibling org account assets are identifiable by platform name `"aws-account"` (set by `GetPlatformForObject("", accountId)` inside `accountAsset`). A sibling is any asset whose platform name is `"aws-account"` AND whose account id (extractable from the platform id or a dedicated field) differs from the primary account id.

Implement as a small predicate in `discovery_conversion.go`:

```go
func isSiblingOrgAccountAsset(a *inventory.Asset, primaryAccountId string) bool {
    if a == nil || a.Platform == nil || a.Platform.Name != "aws-account" {
        return false
    }
    // platform id form: //platformid.api.mondoo.app/runtime/aws/accounts/<accountId>
    for _, pid := range a.PlatformIds {
        if strings.HasSuffix(pid, "/accounts/"+primaryAccountId) {
            return false
        }
    }
    return true
}
```

### Apply after the discovery loop

At the end of `Discover()`, after all targets have been processed:

```go
primaryAccountId := trimAwsAccountIdToJustId(awsAccount.Id.Data)
for _, a := range in.Spec.Assets {
    if isSiblingOrgAccountAsset(a, primaryAccountId) {
        continue
    }
    a.Labels = mergeAccountTagsIntoLabels(a.Labels, accountTags)
}
```

### Initialize Labels on the primary account asset

`accountAsset()` currently leaves `Labels` unset. Initialize it to an empty map so the primary account asset participates in the merge pass uniformly (no special-case code path).

## Edge cases

- **Empty or nil account tags** — merge is a no-op; existing labels unchanged.
- **Asset with nil `Labels`** — initialized before merge.
- **`GetTags()` error** — warning logged; discovery continues with empty account tags.
- **Sibling org accounts** — skipped via platform-name + account-id check.
- **`DiscoveryOrg` branch** — currently only emits account assets, so no resource assets from other accounts get accidentally tagged by the primary's tags.
- **Key collision** — asset's own label wins per the collision rule.

## Testing

Add unit tests to `providers/aws/resources/discovery_test.go`:

1. `mergeAccountTagsIntoLabels` — collision preserves asset value; gaps are filled; nil input labels are handled; empty account tags is a no-op.
2. `isSiblingOrgAccountAsset` — primary account asset returns false; sibling account asset returns true; non-account assets return false; nil-platform assets return false.
3. An end-to-end style test (if feasible without a live connection) that constructs a fake `[]*inventory.Asset` slice containing a primary account asset, a sibling org account asset, and a resource asset, runs the merge pass, and asserts the resulting labels on each.

## Non-goals

- Propagating tags from sibling accounts to their own child resources. Current `DiscoveryOrg` does not emit child resources of other accounts, so this is moot.
- Configurable collision strategy. The rule is fixed: asset wins.
- Namespacing account tags with a prefix. Rejected in brainstorming — the chosen collision rule makes namespacing unnecessary.
