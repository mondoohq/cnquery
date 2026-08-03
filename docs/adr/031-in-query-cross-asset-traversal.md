# ADR 031: Typed asset roots and cross-asset resolution

## Status

Proposed

## Context

[ADR 030](030-asset-tree-anchored-relationships.md) added a native `asset` value
(returned by `running` on `*.mcpServer` resources) and opt-in `mcp-servers`
discovery, and explicitly deferred *live cross-asset traversal*:

```
claude.code.mcpServers.where(name == "dummy").running.tools
```

Investigating that traversal surfaced a bigger realization: cross-asset
resolution should not be a special case bolted onto MQL — it should be the **core
model**. Rather than a flat global resource namespace per runtime, **every asset
exposes a typed root resource**, an `asset` value is **typed to that root**, and
resolving into another asset is just chaining off its root. `running.tools` is the
first concrete application of that foundation.

### The direction (north star)

- **Every asset has a root resource type.** After connecting to (or otherwise
  resolving) an asset, its resources chain off a typed root instead of the flat
  global namespace we have today. The small set of truly global resources (the
  `core` provider) stays global; everything else hangs off a root.
- **`asset` values are typed to a root.** `running` returns `asset<mcp>` — an
  asset value parameterized by the `mcp` root resource type — so `.tools`,
  `.prompts`, … type-check and chain against that schema.
- **Composition is recursive.** A resource in one asset's tree can itself be an
  `asset<root>` that is the typed root of the *next* runtime. Federation nests
  arbitrarily: `A … .running.<…>.running.<…>`.
- **Resolution is backend-agnostic — recording can jump in.** Resolving
  `asset<root>.field` does NOT require a live local connection. The recording
  layer (especially **upstream** recording) can serve the data for an asset we
  can't or won't connect to locally. Live connect is one backend; recorded data
  (local or upstream) is another. The **ADR-030 anchor is the identity** the
  recording layer keys on.

### Why this is the right foundation (and where it's hard)

It generalizes cleanly: the type system already parameterizes types by inner type
(`Resource(name)`, `Array(child)`, `Map(k,v)`), so `Asset(root)` fits. It turns
"connect to B and query it" into "resolve a typed value against its backing,"
which unifies live and recorded resolution. And it gives chained cross-provider
queries a real static type, instead of a dynamic escape hatch.

The hard parts, called out honestly:

- **Compiler must resolve fields of a root that lives in another provider's
  schema** — cross-provider schema availability at compile time.
- **Namespace migration** (global → root-chaining) is large and must be
  evolutionary; existing `aws.ec2.instances`-style queries cannot break.
- **Recording keying depends on upstream registration** — the target asset must be
  resolvable from the anchor (its platform-ids/MRN must exist upstream). Ties
  directly to ADR 030's correlation.
- **Cycles / depth** — recursive federation can create deep or cyclic connect
  chains; needs guards.
- **Providers must declare their asset's root** — a new schema concept.

## Mechanics (grounded in the current code)

**The connect-and-query primitives exist** (discovery uses them today):
`Coordinator.RuntimeFor(asset, DefaultRuntime())` (`providers/coordinator.go:141`) +
`Runtime.Connect(&plugin.ConnectReq{Asset, Features})` (`providers/runtime.go:349`),
provider selected by connection `Type` via `EnsureProvider(ConnType)`
(`runtime.go:313`); canonical recipe in `discovery/discovery.go:52`
`createRuntimeForAsset`. Compile+run against a runtime: `mqlc.Compile(q, nil,
mqlc.NewConfig(runtime.Schema(), features))` (`mqlc/mqlc.go:2331`) +
`exec.ExecuteCode(runtime, bundle, …)` (`exec/exec.go:78`). Runtimes dedupe per
asset by Mrn/PlatformIds (`coordinator.go:169`); the caller closes B's runtime
(`Runtime.Close()` `runtime.go:94`, as `discovery/asset_explorer.go:250` does).

**Recording already holds multiple assets keyed by identity.** `EnsureAsset`
keys by Mrn/PlatformIds (`providers-sdk/v1/recording/recording.go:337`) and
`GetAssetData(assetMrn)` exists (`recording.go:547`); `providers.Runtime` already
resolves fields **recording-first** before hitting the provider
(`runtime.go:606`). So "serve asset B's fields from (upstream) recording, keyed by
B's identity" is a natural extension of an existing path — and needs no
coordinator.

**The wall — llx cannot originate a connect.** A builtin reaches only
`e.ctx.runtime`, typed `llx.Runtime` (`llx/runtime.go:11`), which has no
coordinator and no `Connect`; `llx` does not import `providers`. So federation
must be driven from **`providers.Runtime`** (host-side, reaches coordinator +
`exec` + recording), not from llx.

**Cross-provider routing today is same-asset only.** `lookupFieldProvider`
(`runtime.go:1066`) + the StoreData bridge (`runtime.go:591`) route a field to its
owning provider, but every sibling connect derives from `crossProviderAsset()` — a
clone of **A's** connection (`runtime.go:264`). Cross-*asset* resolution does not
exist; this ADR adds it.

**No `mcp`/`ai` provider in this repo** — `Type:"mcp"` is served by the enterprise
`ai` provider, so live `.tools` needs it installed; in-repo automated tests can't
depend on it (verification is interactive). Recording-backed resolution can be
tested in-repo.

## Decision

Adopt the typed-root model (Option B from the exploration; the earlier "widen
`llx.Runtime` with a dynamic `RunOnAsset`" Option A is dropped as the target — it
would entrench an untyped escape hatch we'd have to unwind).

1. **Typed asset values — root referenced by name (forward reference).** Extend the
   type system so `types.Asset` is parameterized by a root resource *type name*:
   `Asset("mcp")`, mirroring `Resource(name)`. Crucially, the declaring provider
   references the root **by name only** — `os` declares `running asset<mcp>` without
   needing the `mcp` schema at build/codegen time. The name is a forward reference;
   we know the next resource is "supposed to be" an `mcp` even though `mcp` is not
   defined in the `os` schema. This is a **declared cross-provider type reference**
   (legal under the v14/v15 rules below). The value carries: the **root type name**
   (for typing/chaining), the **ADR-030 anchor + identifiers** (for correlation and
   for resolving the upstream MRN), and optionally a **connection Config** (for live
   connect). Anchor stays the correlation key; the root name is a type parameter,
   not data; secrets are never in the value.
2. **Compile-time behavior — always know the name, degrade on the schema.** Because
   the root name is declared at the field, the compiler always knows the chain's
   root type. If the referenced schema is **loaded** (e.g. `ai` installed) →
   `.tools`/`.prompts` type-check statically. If **absent** → **degrade and warn**
   (root known, member checking deferred to runtime, where the real schema resolves
   it). No dynamic-root problem: chaining always has a statically-declared root name.
   (This is the in-tree case. A CLI call to an *uninstalled connector*
   — `mql run some-connect -c …` — is the separate, pre-existing "provider not
   installed" failure, out of scope here.)
3. **The connection declares its root (runtime authority).** The root is grounded in
   the *connection*, not the provider/platform: different connection types emit
   different roots, and the connection declares which root it exposes (the `ai`
   provider's `mcp` connection → root `mcp`). The field's static `asset<mcp>` is the
   compile-time name; the connection-declared root is the runtime authority that
   selects the schema to resolve against.
4. **Cross-asset resolution in `providers.Runtime`.** When resolving a field chain
   under an `asset<root>` value, the runtime resolves against the target asset's
   **best available backing**, in order:
   a. **recording** — local or **upstream**, keyed by the value's identity. For an
      upstream recording, the value must supply **identifiers to resolve the target
      resource's MRN** (derived from the ADR-030 anchor + host context); local
      recordings key on the same identity. This is the un-connectable /
      no-coordinator path, and often the primary one.
   b. **live connect** (`RuntimeFor` + `Connect` + resolve + close) when no
      recording is available and a Config is present.
   This generalizes the existing recording-first, then-provider resolution across
   assets.
5. **Namespace evolution (v14 → v15).** This ADR is part of the v14 line.
   - **v14:** collect and **warn** when a resource is used from the global namespace
     in a way that isn't expected — `time`/`regex` (core) are fine; calling `aws`
     from inside the `os` provider is not. Undeclared cross-provider calls are
     deprecated (warn only, still resolve).
   - **v15:** undeclared cross-provider calls **no longer resolve**; the model is
     root-chaining with only `core` global.
   - **Stays legal (both versions):** providers **extending** each other's resources
     with new fields, and **intentional, declared** cross-provider calls (a resource
     deliberately referencing another provider's resource — including
     `running asset<mcp>`). This ADR is what makes it *explicit* when a resource in a
     tree points at another provider.

## Phased plan

1. **Type system + typed `running`.** `types.Asset("mcp")` parameterization (root by
   name, forward reference); helpers (`AssetData`, conversions, builtins) carry the
   root name; `running` typed `asset<mcp>`. Compiler type-checks `.tools` against the
   `mcp` schema when loaded, and **degrades + warns** (root known, members deferred
   to runtime) when it isn't — no build-time dependency on the `ai` schema.
2. **Recording-backed cross-asset resolution.** `providers.Runtime` resolves
   `asset<root>.field` from recording (local + upstream) keyed by the anchor/
   identity. Testable in-repo with a recording fixture; no coordinator, no live
   connect. This is the first working slice and exercises the upstream-jump-in goal.
3. **Live-connect backend.** Add the `RuntimeFor`+`Connect`+resolve+close path for
   when there's no recording; sub-runtime lifecycle/cleanup; per-item error
   isolation; timeouts. Interactive verification: `…running.tools` against the
   installed `ai` provider + the dummy server.
4. **Per-asset roots + namespace migration.** Introduce the root-resource schema
   concept broadly; migrate providers; long horizon.
5. **Secrets for live connect** (out of the earlier phases): vault/credential
   provisioning keyed to the asset (no `--env` inside a traversal).
6. **Guards:** recursion depth / cycle detection; connection pooling for
   `map(running.tools)`.

## Consequences

- A foundational change: touches the type system, compiler, `providers.Runtime`
  resolution, the recording layer, and provider schemas. Must land evolutionarily.
- Recording becomes a first-class resolution backend for *other* assets, not just a
  replay of the current one — aligned with upstream serving data we can't fetch
  locally.
- Runtime dependency: live `.tools` needs the `ai` provider; recording-backed
  resolution does not.
- Reconciles with ADR 030: the anchor is the identity for both correlation and
  recording lookup; the typed value adds a root type but keeps the value
  secret-free.

## Alternatives considered

- **Dynamic/untyped traversal (widen `llx.Runtime` with `RunOnAsset`).** Smaller
  first slice but entrenches an untyped escape hatch and no chaining/type-checking;
  rejected as the target in favor of the typed-root foundation. May still be used as
  a throwaway spike to de-risk the live-connect plumbing.
- **Platform-side correlation only (no in-query traversal).** Already works via
  ADR 030 (discover MCP assets, scan them, correlate by anchor). Remains the way to
  get the data without any of this; `running.tools` is the in-query ergonomic on
  top.

## Resolved (from review)

- **Root declaration → the connection declares its root** (runtime authority),
  while the **field declares the root by name** (forward reference; the declaring
  provider needs only the name, not the referenced schema).
- **Compile-time absence → degrade + warn** for in-tree chaining (root name always
  known; member checking deferred to runtime when the schema is absent). A CLI call
  to an uninstalled connector is the separate, pre-existing "provider not installed"
  failure.
- **Static vs. dynamic root → moot.** The field statically declares the root *name*,
  so chaining always has a statically-known root; no dynamic-root chaining needed.
- **Namespace migration → v14 warns, v15 enforces.** Extensions and declared
  cross-calls stay legal; undeclared global-namespace cross-calls are deprecated in
  v14 and stop resolving in v15.
- **Recording backend → local or upstream.** Upstream requires identifiers on the
  value to resolve the target resource's MRN; local keys on the same identity.
- **Secrets → explicit, never implicit** (consistent with the `--env` decision).

## Follow-ups

- **Explicit secret source for live-connect traversal.** There is no `--env` inside
  a traversal, so explicit secrets must come from somewhere the non-interactive path
  can read. Candidate: the existing vault/`conf.Credentials` mechanism, keyed to the
  target asset's identity, resolved at connect — source + association model to be
  designed when the live-connect backend (phase 3) is built. This does **not** block
  the earlier phases: recording/upstream resolution needs no secrets, and no-secret
  live servers work without it.
