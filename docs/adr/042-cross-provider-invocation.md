# ADR 042: Cross-Provider Invocation

## Status

Proposed

## Context

Providers in mql are independent Go modules that run as separate gRPC
subprocesses (except `core`, which is built in). Despite that isolation, a
resource implemented in one provider routinely needs data from a resource
owned by another provider. The `os` provider builds the `network` provider's
`socket` and `tls` resources to model open ports; a dozen language-package
resources build the `core` provider's `cpe`; `aws` and `k8s` build `core`'s
`certificates`. This is real, load-bearing, in-production behavior.

### Relationship to ADR 030 / ADR 031

[ADR 030](030-asset-tree-anchored-relationships.md) and
[ADR 031](031-in-query-cross-asset-traversal.md) address a different axis:
**cross-*asset*** resolution, where asset A's resource tree points at a
*separate* asset B (`…mcpServers.first.running.tools`), resolved through a
typed `asset<root>` value backed by recording or a fresh connect. This ADR
addresses **cross-*provider*, same-asset** invocation: P2 answering for the
asset P1 is already connected to. No new asset is created; the two axes are
orthogonal in both mechanism and code surface (031 touches `types/`, `mqlc/`,
`llx/`, and recording; this ADR touches the provider manifest, the coordinator
gate, and plugin lifecycle).

They meet at one point, and this ADR owns it. ADR 031 originally carried a
v14/v15 rule deprecating "undeclared cross-provider calls," but supplied only
one declaration form — the typed field forward-reference (`running
asset<mcp>`), which covers cross-asset references only. That left every
same-asset `CreateSharedResource` call — 35 sites across `os`, `k8s`, `aws`,
and `vsphere` — inside the scope of an enforcement rule with no declaration
form available to satisfy it. **This ADR is therefore the authority on
cross-provider call legality**: it defines the declaration form for same-asset
calls, and carries the v14/v15 enforcement timeline for both forms (see
[Enforcement timeline](#enforcement-timeline-v14--v15)). ADR 031 retains the
namespace half of its original rule (global namespace → root chaining) and
defers legality here.

### How it works today

A provider calls, on its SDK-side runtime, one of:

- `Runtime.CreateSharedResource(name, args)` — instantiate a resource by name.
- `Runtime.GetSharedData(name, id, field)` — read a field by name.

Neither call names a provider or a connection. The request round-trips over
gRPC back to the coordinator process
(`providers-sdk/v1/plugin/runtime.go:150`), lands in
`providerCallbacks.GetData` (`providers/runtime.go:779`), and the coordinator
`Runtime` resolves it:

1. `lookupResource` asks the aggregated schema which provider owns the name:
   `coordinator.Schema().Lookup(name)` (`providers/runtime.go:997`,
   `providers-sdk/v1/resources/schema.go:125`). Every `ResourceInfo` carries a
   `Provider` string, plus `Others` for names defined by more than one
   provider.
2. If the owning provider is not already connected, the coordinator lazily
   starts it and calls `Connect` with a **clone of the primary provider's
   asset** (`crossProviderAsset()`, `providers/runtime.go:264`).
3. It dispatches `GetData` to that provider's connection.

### The two things we want to remove

**1. The hardcoded whitelist.** Before step 2, the coordinator gates the
lookup against a literal slice baked into core
(`providers/runtime.go:940`):

```go
crossProviderList := []string{
    "go.mondoo.com/mql/providers/core",
    "go.mondoo.com/mql/providers/network",
    "go.mondoo.com/mql/providers/os",
    "go.mondoo.com/mql/providers/ms365",
    ...
}
if info.Provider != providerConn && !stringx.Contains(crossProviderList, info.Provider) {
    return ..., errors.New("incorrect provider for asset, not adding " + info.Provider)
}
```

This is exactly backwards for a distributed provider model. A provider is a
self-contained, independently-versioned, independently-shipped artifact — but
whether it may participate in cross-calling is decided by a constant in a
*different* module (core). Adding a cross-callable provider means editing core
and re-releasing it. The list has already accreted three eras of legacy module
paths (`mql/...`, `cnquery/...`, `cnquery/v9/...`) with `FIXME: DEPRECATED`
markers. It does not scale, it is not discoverable, and it couples every
provider's capability to a core release.

**2. The half-built parallel mechanism.** `Provider.CrossProviderTypes`
(`providers-sdk/v1/plugin/start.go:23`) lets a provider declare asset
connection types it is willing to serve resources for. Its own doc comment
calls it "only a hotfix" and sketches the intended end state ("each provider
creating an asset object when it tries to call out"). Only `networkdiscovery`
uses it (`providers/defaults.go:1060`). We have two overlapping,
half-specified mechanisms and one hardcoded gate doing related jobs.

### The three shapes of cross-calling

Untangling the mechanism starts with naming what actually happens between a
primary provider **P1** (owns the asset and the current connection) and a
secondary provider **P2**. There are three distinct interactions, and the
current code smears them together onto one asset-cloning path:

- **Mode 1 — Enrichment (P1 pulls from P2, same asset).** P1 explicitly asks
  P2 to fill in data for the *existing* asset. The asset does not change; P2
  receives the asset context plus the resource arguments P1 supplies, and
  answers. `os` building `network.socket` for the host it is already connected
  to is this mode. This is what `CreateSharedResource` does today.

- **Mode 2 — Asset creation (P1 spawns a new asset via P2).** P1 explicitly
  asks P2 to *connect to a new asset*. P2 opens its own connection, and from
  there behaves as a primary provider for that new asset. `os` finding a
  container image reference on disk and handing it to the `os`/container
  provider to connect as a *separate* asset is this mode. There is no clean
  API for this today — `crossProviderAsset()` reuses the same asset, which is
  wrong here. **Mode 2 is out of scope for this ADR and is owned by
  [ADR 031](031-in-query-cross-asset-traversal.md)** — it is a new-asset
  connect, which is exactly 031's cross-asset machinery
  (`RuntimeFor`+`Connect`+resolve+close) reached imperatively from Go instead
  of through a typed `asset<root>` value in a query. Specifying it separately
  here would fork that path and, critically, bypass 031's recording-first
  resolution, making Mode-2 assets invisible to replay. It is named here only
  to complete the taxonomy.

- **Mode 3 — Attachment (P2 attaches itself to P1, implicit).** The inverse of
  Mode 1: P2 declares that it contributes resources/fields to assets primarily
  owned by P1, and gets pulled in *implicitly* when a user queries one of those
  fields. P1 never wrote a `CreateSharedResource` call. P2 pulls the asset and
  resource context from P1. This is the schema-extension / `networkdiscovery →
  network` shape, and the direction `CrossProviderTypes` was groping toward.

The whitelist and `CrossProviderTypes` fail because they try to answer "may
these two providers cross-call?" with one boolean, when the real questions are
"in which mode?" and "declared by whom?".

Modes 1 and 3 are this ADR's subject. Mode 2 belongs to ADR 031.

## Decision

Replace the centralized whitelist and the ad-hoc `CrossProviderTypes` with a
**distributed capability declaration**: each provider declares, in its own
manifest and schema, what cross-provider interactions it participates in and in
which mode. The coordinator's job shrinks from *deciding policy* to *reading
declarations and brokering the call*. No provider's cross-calling capability
lives in core anymore.

### Declaration model

Extend the provider `Provider` manifest (`providers-sdk/v1/plugin/start.go`,
serialized into each `dist/<provider>.json`) with a single `CrossProvider`
block that subsumes and replaces `CrossProviderTypes`:

```go
type CrossProvider struct {
    // Offers declares that this provider is willing to answer cross-provider
    // resource requests (Mode 1 / Mode 3). Absent an entry, the provider is
    // never auto-started to satisfy another provider's lookup.
    Offers []CrossProviderOffer
}

type CrossProviderOffer struct {
    // Mode is "enrich" (Mode 1: reuse the caller's asset) or "attach"
    // (Mode 3: this provider extends the caller's asset via schema
    // extension). Mode 2 is not declared here; it is an explicit caller API
    // (see below).
    Mode string

    // Resources this offer covers. Empty means "any resource this provider
    // owns in the schema" (the common case for core).
    Resources []string `json:",omitempty"`

    // ForConnectionTypes restricts the offer to callers whose primary asset
    // has one of these connection types. Empty means "any caller".
    ForConnectionTypes []string `json:",omitempty"`
}
```

The coordinator builds its allow-set at schema-aggregation time from these
declarations, keyed by resource name and caller connection type — the same
place `Schema.Lookup` and `Dependencies` are already assembled
(`providers-sdk/v1/resources/schema.go`). The runtime gate at
`providers/runtime.go:975` becomes:

```go
// was: stringx.Contains(hardcodedCrossProviderList, info.Provider)
if info.Provider != providerConn && !r.schema.AllowsCrossCall(info.Provider, resource, callerConnType) {
    return ..., errors.New("provider " + info.Provider + " does not offer '" + resource + "' for cross-provider use")
}
```

Because the declaration ships *inside each provider's manifest*, a new
cross-callable provider requires zero changes to core. Installing the provider
installs its cross-call contract.

### Mode 1 — Enrichment (formalize the existing path)

Keep `CreateSharedResource` / `GetSharedData` as the caller API. The only
behavioral change: the gate consults declarations instead of the whitelist.
The asset handed to P2 stays the `crossProviderAsset()` clone (same asset,
`ParentConnectionId` cleared, per the reasoning already documented at
`providers/runtime.go:259`).

Providers that are cross-called today declare, in their manifest:

- `core`: `{Mode: "enrich"}` with empty `Resources`/`ForConnectionTypes` (it
  serves `cpe`, `certificates`, etc. to everyone).
- `network`: `{Mode: "enrich", ForConnectionTypes: ["ssh","local",...]}` for
  `socket`/`tls`.

This is a pure refactor of behavior we already ship — the migration target for
everything currently in the whitelist.

### Mode 2 — Asset creation (deferred to ADR 031)

Not specified here. An imperative Go-side entry point for "connect to a new
asset through P2" should be a thin wrapper over ADR 031's cross-asset
resolution path, so that recording-first resolution, cycle/depth guards, and
child-runtime lifecycle are shared rather than reimplemented. Mode 2 needs no
`Offers` declaration in any case: the gate exists to stop a provider from being
silently co-opted onto *someone else's* asset, and creating a genuinely new
asset is not that.

### Mode 3 — Attachment (declared extension, implicit call)

Mode 3 is Mode 1's contract plus a *schema extension* so the call is implicit.
P2 declares `{Mode: "attach", ForConnectionTypes: [<P1 types>]}` and defines
`extend` resources/fields in its `.lr` (the existing `IsExtension` mechanism,
`providers-sdk/v1/resources/schema.go:38`). When a user queries an extended
field:

1. `lookupFieldProvider` already finds the field is owned by P2, not P1.
2. The gate sees P2 declared an `attach` offer for P1's connection type and
   allows it.
3. P2 connects against the `crossProviderAsset()` clone and answers — pulling
   P1's asset and resource context exactly as Mode 1 does, but without P1
   having written any call site.

This turns `CrossProviderTypes` (a flat list of asset types) into a typed,
per-mode declaration, and reuses the extension machinery we already have rather
than a parallel one. `networkdiscovery`'s existing entry becomes an `attach`
offer.

### What the coordinator keeps doing

Brokering stays centralized — and should. The coordinator is the only process
that sees all providers; it owns lazy start, connection reuse, the asset clone,
and dispatch. The change is narrow: it stops being the *authority on policy*
(the hardcoded list) and becomes a *reader of declarations*. Providers never
talk to each other directly; every cross-call still flows P2 → coordinator →
P1's-owned target, so recording/replay, upstream config, and feature flags
continue to work unchanged.

### Enforcement timeline (v14 → v15)

Moved here from ADR 031 point 5, because the rule is enforced at the
coordinator gate (`providers/runtime.go:1020`) — this ADR's surface — and
because it is unenforceable without the declaration form defined above.

- **v14:** an undeclared cross-provider call still resolves, but warns. The
  gate accepts `schema.AllowsCrossCall(...)` **or** the legacy whitelist, and
  logs when only the legacy path matched. This surfaces every missing
  declaration without breaking a single user.
- **v15:** undeclared cross-provider calls no longer resolve. The whitelist and
  `CrossProviderTypes` are deleted.
- **Stays legal in both:** providers **extending** each other's resources with
  new fields (the `IsExtension` mechanism), and any call covered by a
  declaration — an `Offers` entry (same-asset, this ADR) or a typed field
  forward-reference such as `running asset<mcp>` (cross-asset, ADR 031).

Both declaration forms are covered by one enforcement date. The v15 cutover
must not land until every currently-cross-called provider carries an
equivalent declaration — see the migration plan below, which gates the
whitelist deletion on a full release with zero legacy-only log hits.

ADR 031 keeps the **namespace** half of its original point 5: the global
namespace narrows to `core` and everything else chains off a typed root. That
migration is 031's to sequence and is independent of this gate.

## Consequences

**Positive**

- A provider's cross-call contract ships with the provider. Adding or removing
  a cross-callable provider no longer touches core or requires a core release.
- Three genuinely different interactions get three names, instead of one
  asset-cloning path plus a boolean gate. Two of them (enrich, attach) are
  specified here; the third (asset creation) is routed to ADR 031's existing
  cross-asset path rather than given a parallel implementation.
- ADR 031's v15 rule becomes enforceable. It currently deprecates a class of
  call for which no declaration form exists, which would strand 35 shipped
  call sites.
- The whitelist's accumulated legacy-module-path cruft
  (`providers/runtime.go:951-972`) is deleted, not extended again.
- Declarations are introspectable — `mql providers list` can show cross-call
  offers, and CI can validate that every `CreateSharedResource` target has a
  matching `Offers` declaration somewhere in the installed set.

**Negative / risks**

- Manifest schema change: every provider's `dist/<provider>.json` regenerates.
  This is a wide diff (mechanical, via `make providers`).
- Migration must be careful: the whitelist and `CrossProviderTypes` must keep
  working until every currently-cross-called provider carries an equivalent
  `Offers` declaration, or live cross-calls (os→network, *→core) break.
- A missing or wrong declaration turns a previously-working cross-call into a
  runtime error. Mitigate with a CI check that scans for
  `CreateSharedResource`/`GetSharedData` targets lacking a declaration, and by
  keeping the old whitelist as a logged fallback for one release.
- "Any provider can declare itself cross-callable" is more permissive than a
  curated list. That is the intent (distributed ownership), but it means a
  buggy provider can offer a resource it answers poorly. The gate protects the
  *asset* (you cannot be pulled onto an asset you did not opt into), not
  resource quality — quality stays the owning provider's responsibility, same
  as today.

## Migration plan

1. Add the `CrossProvider`/`Offers` manifest fields (additive, alongside the
   existing `CrossProviderTypes`).
2. Populate `Offers` for the current cross-called set: `core` (enrich, all),
   `network`/`os`/`ms365`/`azure`/`ai`/`ipinfo`/`yara` (enrich, scoped),
   `networkdiscovery` (attach). Regenerate `dist/*.json`.
3. Change the runtime gate to `schema.AllowsCrossCall(...)` **OR** the legacy
   whitelist (union), logging when only the legacy path matched — surfaces any
   missing declaration without breaking users.
4. Add the CI check (every shared-resource target has a declaration).
5. After one release with zero "legacy-only" log hits, delete the hardcoded
   whitelist and `CrossProviderTypes`. This is the v15 cutover; it must be
   sequenced with ADR 031's typed-field declarations so both forms are
   enforced together.

## Alternatives considered

- **Keep the whitelist, just move it to a config file.** Still centralized;
  still a release-coupled edit; does not distinguish the three modes. Rejected.
- **Let providers cross-call with no gate at all.** Removes the coupling but
  also removes the protection that a provider cannot be silently attached to an
  asset it never opted into. The `attach` vs `enrich` distinction exists
  precisely to keep that opt-in. Rejected.
- **Direct provider-to-provider gRPC (skip the coordinator).** Breaks
  recording/replay, upstream config propagation, and connection reuse, and
  would require every provider to discover every other provider's socket.
  Rejected; brokering through the coordinator is the right call.
