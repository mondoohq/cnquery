# Align Azure `DiscoveryFilters` with the AWS pattern (behavior-neutral)

Date: 2026-07-17
Status: Approved

## Goal

Re-express the Azure provider's existing subscription include/exclude discovery
filter through the **exact** structural pattern the AWS provider uses
(`connection/filters.go` → `DiscoveryFilters` struct → `conn.Filters` field →
`DiscoveryFiltersFromOpts(conf.Discover.GetFilter())`), and add a `--filters`
KeyValue flag as an equivalent transport alongside the existing dedicated
`--subscription*` flags.

This is a sideways refactor: existing-flag behavior is unchanged (backwards
compatible), and the `--filters` path is purely additive. The point is to
establish the identical seam AWS uses so the follow-up feature work
(tag / location / resource-group filtering + subscription-tag propagation) drops
into the same place with no further plumbing.

## Non-goals

- No new filtering *capability* beyond what exists today (subscriptions only).
  The `DiscoveryFilters` struct starts minimal — just `Subscriptions`.
- No tag/location/resource-group filters, no subscription-tag propagation. Those
  are the follow-up feature, out of scope here.
- No `.lr` schema change, therefore no `.lr.versions` or `permissions.json`
  change.

## Reference implementations

- `providers/aws/connection/filters.go` — `DiscoveryFilters`,
  `DiscoveryFiltersFromOpts`.
- `providers/aws/connection/connection.go:103` —
  `c.Filters = DiscoveryFiltersFromOpts(conf.Discover.GetFilter())`.
- `providers/aws/provider/provider.go:95` — `parseFlagsToFiltersOpts`
  (KeyValue `.Map` → allowlisted opts map → `Discovery.Filter`).
- `providers/aws/config/config.go:154` — the `--filters` KeyValue flag.
- `providers-sdk/v1/util/filteropts/filteropts.go` — shared `ParseCsvSliceOpt`
  / `ParseBoolOpt`.

## Data flow

Two flag styles converge on one filter map, then flow through the AWS-identical
path:

```
--subscription <id>                     ┐
--subscriptions <csv>                    ├─► Discovery.Filter ─► DiscoveryFiltersFromOpts() ─► conn.Filters.Subscriptions ─► GetSubscriptions
--subscriptions-exclude <csv>            ┘        ▲
--filters subscriptions=<csv>            ─────────┘   (allowlisted keys: subscriptions, subscriptions-exclude)
--filters subscriptions-exclude=<csv>
```

### Before vs. after

```
BEFORE:  --subscriptions ─► Options["subscriptions"] ─► Discover() builds SubscriptionsFilter inline ─► GetSubscriptions
AFTER:   --subscriptions ─► Discovery.Filter["subscriptions"] ─► DiscoveryFiltersFromOpts() ─► conn.Filters.Subscriptions ─► GetSubscriptions
```

### Precedence

When both styles set the same key, `--filters` is the base and a dedicated flag
**overrides** it when present. Among the dedicated flags, plural
`--subscriptions` overrides singular `--subscription` (today's precedence,
preserved). Rationale: existing `--subscriptions` scripts get identical results
regardless of any `--filters` value — the strongest backwards-compat guarantee.

## File-by-file changes

### 1. `providers/azure/config/config.go`

Add the `--filters` flag (mirrors `aws/config.go:154`):

```go
{
    Long:    "filters",
    Type:    plugin.FlagType_KeyValue,
    Default: "",
    Desc:    "Filter options, e.g., --filters subscriptions=<id1>,<id2> --filters subscriptions-exclude=<id3>",
}
```

### 2. `providers/azure/connection/filters.go` (NEW)

Mirrors `aws/connection/filters.go`. Holds the `DiscoveryFilters` struct and the
`SubscriptionsFilter` type (moved here from `client_subscriptions.go` to match
AWS, where all filter types live in `filters.go`):

```go
type DiscoveryFilters struct {
    Subscriptions SubscriptionsFilter
}

type SubscriptionsFilter struct {
    Exclude []string
    Include []string
}

func DiscoveryFiltersFromOpts(opts map[string]string) DiscoveryFilters {
    return DiscoveryFilters{
        Subscriptions: SubscriptionsFilter{
            Include: filteropts.ParseCsvSliceOpt(opts, "subscriptions"),
            Exclude: filteropts.ParseCsvSliceOpt(opts, "subscriptions-exclude"),
        },
    }
}
```

### 3. `providers/azure/connection/client_subscriptions.go`

Delete the moved `SubscriptionsFilter` type definition. `skipSub` and
`GetSubscriptions` stay put (same package, no import churn) and continue to
reference the type.

### 4. `providers/azure/connection/connection.go`

- Add `Filters DiscoveryFilters` to the `AzureConnection` struct.
- In `NewAzureConnection`:
  `c.Filters = DiscoveryFiltersFromOpts(conf.Discover.GetFilter())`.

Nil-safe: `GetFilter()` on a nil `Discover` (the compute-subcommand case)
returns a nil map, and `ParseCsvSliceOpt` handles a nil map — no guard needed.

### 5. `providers/azure/provider/provider.go`

- New `parseFlagsToFiltersOpts(flags)` mirroring AWS: read the `--filters`
  KeyValue `.Map`, allowlist `subscriptions` / `subscriptions-exclude`, then
  overlay the dedicated `--subscription` / `--subscriptions` /
  `--subscriptions-exclude` flags (dedicated wins; plural over singular).
- Assign the merged map to `Discovery.Filter` (through `parseDiscover`).
- Remove the old `opts["subscriptions"]` / `opts["subscriptions-exclude"]`
  writes.

### 6. `providers/azure/resources/discovery.go`

`Discover` reads `conn.Filters.Subscriptions` instead of rebuilding the filter
from `rootConf.Options` (current lines 228–236 collapse to one line).

## Behavior-neutrality argument (existing flags)

- Dedicated flags are unchanged and still win → existing usage is byte-for-byte
  identical.
- `Options["subscriptions"]` / `["subscriptions-exclude"]` are read **only** at
  `discovery.go:228-229` (verified by grep across `providers/azure/`), which is
  updated in the same change — no orphaned reader.
- `ParseCsvSliceOpt` returns a non-nil empty slice where the old code produced
  `nil`; `skipSub` uses `len(filter.Include) > 0` / `len(filter.Exclude) > 0`
  (length checks, not nil checks), so the two are indistinguishable.
- Child asset configs already strip `Discover` (`WithoutDiscovery()`) and never
  re-run subscription filtering, so moving the values off `Options` has no
  downstream effect.

## Additive path (`--filters`)

`--filters subscriptions=...` / `--filters subscriptions-exclude=...` become
valid input, flowing through the same `Discovery.Filter` → `conn.Filters` path.
Adding the flag regenerates `dist/azure.json` on build. Provider version bump is
left to the release step, not this change.

## Testing

- New `providers/azure/connection/filters_test.go`: `DiscoveryFiltersFromOpts`
  covering include-only, exclude-only, both, CSV multi-value, and empty/absent
  keys (→ empty slices).
- `ParseCLI` tests: dedicated flags → `Discovery.Filter`; `--filters` →
  `Discovery.Filter`; both-set precedence (dedicated wins); singular-vs-plural
  precedence.
- Run existing Azure connection tests to confirm no regression.
