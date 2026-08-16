# ADR 040: Cross-version MQL — schema resolution, migration lenses, and down-adaptation

## Status

Proposed

## Context

Compiled MQL — and, less obviously, MQL *source* — is silently bound to one
exact schema version. Compilation resolves names into concrete types against a
schema and freezes them into bytecode and checksums; execution assumes the
runtime schema is byte-for-byte identical. When it isn't, everything from a
single field to a whole policy shatters. Because the server routinely ships
content (raw or compiled) down to clients and is usually **newer** than they are,
version skew is the steady state, not the exception.

The problem is **general schema evolution across skewed nodes, in both
directions**, not the narrow case of renaming a resource. A rename of a
structurally-identical resource is the *easy* 10%; the real difficulty is
**structural change** (a scalar becoming a typed reference, a list becoming a
single value, a field changing meaning) and the **new→old direction** (content
produced on a newer schema meeting an older node that has never heard of the new
names, types, or language features). Any solution that only handles renames, or
only the old→new direction, does not address the problem.

We treat the schema as an ambient global constant. It is **data**, and it must
travel with — or be resolvable alongside — the content, with a reconciliation
step at load time. This is the same problem Avro (writer schema vs reader schema
+ a resolution algorithm), Kafka's schema registry, and Cambria (bidirectional
lenses for "distributed data without a shared schema") exist to solve. We have
none of that machinery today.

### Two version axes, and we track neither

Reconciliation needs to know two independent things about a piece of content, and
we currently record neither reliably:

- **Which schema it was compiled against** (the *writer schema*, per provider).
  Today the bundle records no schema identity at all — the schema is only ambient
  at compile time.
- **The minimum MQL engine version required to execute it** (the *language/VM*
  axis — new opcodes, builtins, type kinds like ADR 031's `asset<root>`). This is
  distinct from the schema axis: an old VM can fail on new bytecode regardless of
  schema.

What we have is close but wrong for the job:

- `CodeBundle.Version` is set to `mql.APIVersion()` (`mqlc/mqlc.go:2267`) — a
  *writer* stamp (which engine compiled it), **not a minimum** and **not a schema
  identity**.
- `CodeBundle.min_mondoo_version` (proto field 22) is **never set by the engine**.
- Per resource/field, `min_provider_version` is populated
  (`providers-sdk/v1/mqlr/lrcore/versions.go:78-115`) but **read by nothing at
  runtime**, and it only covers the *provider-schema* axis.
- The per-field `min_mondoo_version` — the old *engine-version* signal — is
  **deprecated** (`resources.proto:79,96`, "We now use min_provider_version"). So
  when we moved to provider versions we **lost the min-MQL-version axis
  entirely**.

## Mechanics (grounded in the current code)

**A type is its name, frozen into content.** `types.Type` is a string whose first
byte is a tag; a resource type is tag byte `byteResource` (27) followed by the
name verbatim (`types/types.go:182-193`). `Resource("claude.code.repo")` is the
bytes `"\x1bclaude.code.repo"`. Human name and type identity are the same bytes,
so a rename is simultaneously an identity and a type change — and both get baked
in: the compiler copies the schema's `Field.Type` verbatim into field chunks
(`mqlc/mqlc.go:1089`), stamps the resource name into `Chunk.Id` and the resource
chunk type (`mqlc/mqlc.go:1144,1156`), and **folds the type string into the
checksum** (`llx/chunk.go:113`), from which the bundle identity `CodeV2.Id`
derives. Changing a field's type changes the content identity of every query that
touches it, even if the source is unchanged — which forks cnspec scoring, since
scoring keys on those checksums.

**The compiler has no version awareness.** A missing resource/field is a hard
`ErrIdentifierNotFound` (`mqlc/mqlc.go:25-35`) — never "needs a newer provider,"
never a fallback. Resolution is "is this key in the merged aggregate schema right
now" (`providers/extensible_schema.go`; `resources.Schema.Add` at
`providers-sdk/v1/resources/schema.go:21-123`).

**Source already rides along with bytecode, and there are two execution paths.**
`CodeBundle` carries both the raw `source` (field 3) and the compiled `CodeV2`
(field 6). `exec.Exec`/`mqlx.Env.Compile` recompile source against the *local*
schema (`exec/exec.go:35-76`); `exec.ExecuteCode` runs a pre-compiled bundle
**as-is with no recompilation** (`exec/exec.go:78-106`). The server-ships-bytecode
path lands in `ExecuteCode`. The presence of `source` in every bundle is the
escape hatch this ADR relies on.

**Skew fails lazily and inconsistently at runtime.** No pre-flight check. A
missing field degrades to `types.Unset` + a warning (`llx/builtin.go:850-861` —
its comment already names this exact skew scenario), a missing resource errors
(`providers/runtime.go:1040-1046`, `llx/builtin.go:825-843`), and a typeless
primitive coerces to null (`llx/data_conversions.go:817-838`). Combined with
three-valued logic (`null && null == true`), a degraded bundle can make an
assertion **pass** while verifying nothing.

## Decision

Make schema a first-class, versioned input to execution, reconcile writer against
reader at load time, and put the burden of bridging versions on the side that
owns the newer definitions. Five parts: four at runtime, one at build time.

### 1. Content carries its provenance; execution resolves writer against reader

Every compiled bundle records **both** version axes:

- a **schema fingerprint/version per referenced provider** (the writer schema
  identity), and
- a **minimum MQL engine version** required to execute it (revive the dead axis;
  populate the currently-inert `CodeBundle.min_mondoo_version`, or a dedicated
  field, and compute it from the language features actually used).

Providers keep a **schema registry / version history** (or a bounded
compatibility window) so any node can obtain a *writer* schema it did not compile.
Execution becomes `reconcile(writerSchema, readerSchema) → adapted plan` instead
of assuming `writer == reader`. When a node holds `source` and detects an epoch
mismatch it may recompile locally rather than run stale chunks — treating shipped
bytecode as a cache, not a contract. Bundle identity is defined as a **canonical
key resolved through the migration chain** (part 2), not the raw baked type
strings, so the same logical query keeps one identity across a migration and
scoring survives.

### 2. Migrations are composable, directional lenses — not aliases

For each version step that changes shape, the provider ships a directional
transform between adjacent schema versions, written as **arbitrary Go code**, not
a restricted mapping DSL. Two reasons: a lens is minute next to the provider it
lives in, so the cost of full generality is negligible; and authors need the
freedom to do whatever a real structural change demands (call an API, resolve an
id to a resource, restructure a value) rather than being boxed in by a mapping
language. We start with a thin lens interface and **grow a shared utility
library** (id→resource resolvers, common field hoists, list/scalar adapters) as
recurring patterns emerge. Renames are the degenerate identity lens; the
load-bearing ones are structural:

- `repos: []string → []git.repo` via `git.repo(url: $)` — a real constructor
  adapter, not a name swap.
- `vpcId string → vpc() aws.vpc` — resolve the id to a typed resource.
- hoist/flatten (`linuxParameters.maxSwap → linuxMaxSwap`), split, merge.

Lenses **compose along the chain** (`v3→v4→…→v7`) so an arbitrary A→B is bridged
by traversing intermediate steps, and where a lens is invertible it also runs
backward (needed for part 3). A lens adapts the *plan/AST* (rewrite chunks into
the reader's vocabulary) and/or the *data* (coerce values at the boundary),
whichever the change requires.

### 3. The newer side adapts *down* for the older side

The direction that felt unsolvable — new content on an old node (cases where the
server is newer) — is solvable because the burden belongs to the **producer**,
which is newer and owns the lenses. Before shipping to a client on an older
version, the server **down-migrates content into that client's schema + engine
vocabulary**: the client receives content expressed entirely in names, types, and
language features it already knows, and compiles/runs it natively — it never has
to learn the new world. This is the real fix for the common skew; it lives in the
resolver (upstream cnspec), but the engine must *produce* the version metadata
(part 1) and *own* the lenses (part 2) that make it possible.

### 4. Partial execution is the floor — never silent, never falsely green

When a change is genuinely unbridgeable in a given direction (a new capability
with no backward projection), the producer **omits that datapoint for that
client** and reports `requires provider ≥ V` / `requires mql ≥ V`. Anything that
still slips through resolves to a typed null **carrying that reason**, and
assertions over an unavailable field **fail, not pass** (set explicit error/false
state on the stub to defuse the `null && null` footgun). Skew degrades per-field —
loud, localized, attributable — instead of shattering the whole query or, worse,
passing while verifying nothing.

### 5. Breaking changes are detected at build time and force a migration

None of the above helps if an author changes a type and simply forgets to ship a
lens. So detection is not advisory — it is a **build-time gate**, the enforcement
counterpart to the runtime machinery. Codegen already holds both schemas: the
previously-committed `*.resources.json` and the one freshly generated from the
edited `.lr`. It **diffs them** and classifies every delta:

- **Additive** (new resource, new field, a widened optional) — allowed; bump the
  schema fingerprint; no lens required.
- **Breaking** (a field's `Field.Type` changed, a resource/field removed or
  renamed, a semantics marker flipped) — **hard-fail the build** unless a
  migration lens (part 2) covering that exact transition exists, or the author
  explicitly annotates the change as intentionally unbridgeable (so part 4 can
  degrade it cleanly rather than guess).

This is the discipline `buf` enforces for protobuf: you cannot merge a breaking
schema change without acknowledging it. It supersedes today's purely prohibitive
rule ("never change a shipped field's type") — the type *can* change now, but the
build will not let you change it **without a lens**. The gate also drives part 1's
provenance: a detected breaking change forces the schema fingerprint (and, when
language features are involved, the min-MQL-version) to advance, so provenance can
never silently drift out of sync with the schema it describes.

## Phased plan

1. **Provenance, no wire break yet.** Stamp bundles with the per-provider schema
   fingerprint and a computed min-MQL-engine version (part 1, producer side).
   Wire `min_provider_version` into compile-time diagnostics so opaque
   `ErrIdentifierNotFound` becomes "field X requires provider ≥ V." Add the
   codegen schema-diff (part 5) as a **warning** so drift becomes visible
   immediately. Pure metadata + tooling; ships independently.
2. **Resolution + lens runtime.** Schema registry / version history; the
   `reconcile(writer, reader)` step; the lens interface and composition engine.
   Redefine bundle identity as the migration-chain-canonical key (part 1/2).
   **Escalate the schema-diff gate to hard-fail** now that lenses exist to satisfy
   it. Land one structural migration end-to-end as the proof (e.g. `repos:
   []string → []git.repo`, exercised in both directions).
3. **Down-adaptation in the resolver (upstream).** Server consumes client
   `ProviderVersions` + bundle provenance and down-migrates or withholds
   content per target (part 3).
4. **Degradation policy hardening.** Consistent typed-null-with-reason;
   assertion-safe stubs; per-field partial results (part 4).

## Consequences

- Structural type change and the new→old direction — the actual problem — become
  tractable, not just renames and not just old→new.
- `min_provider_version` (already shipped, currently dead) and a revived
  min-MQL-version axis become the backbone of diagnostics and content selection,
  for near-zero new wire data.
- Bundle identity decouples from baked type strings, so migrations stop forking
  cnspec scoring.
- Real new cost and surface: a schema registry / version history per provider,
  an authored-and-maintained lens per structural change, and down-adaptation
  logic in the resolver. Lenses are code that must be tested in both directions;
  an unbridgeable change must be *declared* so part 4 can degrade cleanly rather
  than guess.
- Lenses accumulate; they need a deprecation horizon (warn in vN, drop in vN+2,
  mirroring ADR 031's namespace evolution) or the chain grows without bound.
- Breakage can no longer ship silently: the codegen gate (part 5) turns an
  un-migrated type change into a build failure in the author's PR, not a runtime
  surprise on a customer's older client.

## Alternatives considered

- **Server compiles per-client-version content (negotiation, no lenses).** Solves
  compatibility but requires the server to retain and compile against every
  historical schema and helps neither offline nor peer-to-peer. Subsumed as the
  narrow "withhold/down-level" fallback within part 3, not a general strategy.
- **Always recompile from source, never execute foreign bytecode.** Collapses the
  bytecode axis, but without migration-chain-canonical identity (part 1) different
  clients compile the same source to different identities under drift, forking
  scoring; and it still can't compile *source* that names resources the local
  schema lacks. Retained as the recompile-on-mismatch option inside part 1, not
  the whole answer.
- **Runtime graceful degradation alone.** The missing-field→null path already
  exists (`llx/builtin.go:850`). Insufficient as a solution — silent nulls plus
  three-valued logic pass assertions that verified nothing — so it is demoted to
  the part 4 floor, hardened to fail attributably.
- **Rename-only aliasing (the previous draft of this ADR).** Handles a
  structurally-identical resource in the old→new direction only. It is exactly the
  degenerate identity lens in part 2 and does not touch structural change or the
  new→old direction; rejected as a headline, absorbed as a special case.

## Relationship to late binding ([ADR 041](041-late-binding-resource-types.md))

This ADR reconciles versions while still *baking* concrete types into content.
The root cause — that a type is its name, frozen in at compile time — can be
attacked directly by **late binding**: reference fields by a stable contract and
resolve concrete types against the local schema at load time. That would make the
common evolutions (rename, reorg, additive) non-breaking *by construction*,
shrinking how often a part-2 lens is even needed.

Late binding is **additive to this ADR, not a replacement**: structural/semantic
changes and new→old capability gaps still need lenses (part 2), down-adaptation
(part 3), and the degradation floor (part 4) regardless. Because 1-4 stands on its
own and does not require the invasive v14 wire/VM change late binding entails, and
because this ADR's identity model (migration-chain-canonical, part 1) is
deliberately written to be satisfied by a late-bound world, late binding is
proposed as a **separate decision (ADR 041)** on its own timeline rather than
folded here. Folding it in would hold this shippable compatibility layer hostage
to the hardest, most invasive change. (If we decide the lens burden is only
tolerable *after* late binding shrinks it, the two collapse into one — but 1-4's
content is unchanged either way.)

## Open questions

- **Fingerprint granularity:** per-provider version, a hash of the merged schema,
  or per-resource? Per-provider is cheapest and matches `min_provider_version`.
- **How the min-MQL-version is computed** from a bundle — a static table of
  language features → introducing engine version, or a coarser high-water mark?
- **Where lenses are authored** when a migration crosses provider boundaries
  (`claude.code.repo` in os → a shared `git.repo`): old owner, new owner, or core?
  Ties into ADR 031's cross-provider schema availability and the `others`
  duplication field slated for removal.
- **Invertibility:** invertible-by-construction lenses, or manually paired
  up/down lenses with a build-enforced round-trip test? (Expressiveness itself is
  settled — lenses are arbitrary Go code, with a shared utility library grown over
  time.)
- **Diff classification edge cases:** is widening a field optional/nullable
  additive or breaking? Does reordering an enum's documented values count? The
  part-5 gate needs a precise, conservative rulebook.

## Follow-ups

- Prototype `claude.code.repo → git.repo` (structurally identical) and a genuinely
  structural change (e.g. an id→typed-ref) as the two reference migrations once
  phase 2 lands; capture both in `CLAUDE.md`.
- Compiler test: identity stability across a migration (same source + lens ⇒
  identical migration-chain-canonical id).
- Coordinate rollout with [ADR 041](041-late-binding-resource-types.md) (late
  binding depends on this ADR's version axis); cross-link ADR 031 (typed roots /
  cross-provider schema) and ADR 030.
