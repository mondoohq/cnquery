---
name: provider-api-call-dedup
description: >-
  Use when a provider scan is slow, times out, or trips rate limits (429,
  throttling, Retry-After), when the same request URL appears many times in a
  debug log, when an asset's fields look like they cost one API call per
  scanned asset, or when auditing any provider under providers/ for redundant
  per-asset API traffic. Applies to aws, azure, gcp, k8s, okta and any other
  provider, since the resource cache and init dispatch are shared machinery.
---

# Deduplicating provider API calls

## Overview

Two pieces of shared mql machinery decide how many times a provider asks the
same question. Both fail quietly: the scan still returns correct data, it just
costs N times more calls than it needs, and the redundancy is what provokes
throttling.

**Mechanism 1 — the resource cache is per connection.** `providers.Coordinator`
gives every discovered asset its own `Runtime`, and MQL resource values live in
`runtime.Resources`. Unless a provider says otherwise, nothing is shared: data
that belongs to a subscription / project / account is re-fetched once per asset
beneath it. See *The parent-child cache model* below.

**Mechanism 2 — `init` runs before the cache is read.** In the generated
`<provider>.lr.go`, `NewResource` calls `f.Init(runtime, args)` and only *then*
checks `runtime.Resources.Get(id)`. An init that performs I/O therefore spends
its call once per asset, and on a cache hit its freshly built resource is thrown
away in favour of the cached one.

It has to be this way in general: the cache key is the resolved resource's
`MqlID()`, which for a resource whose identity is computed during init is not
knowable beforehand. The opening exists only for callers that already supply
the id — see *cache-first* in Phase 4.

## The parent-child cache model

Worth understanding on its own, not just for init migration: it governs every
cross-asset reuse decision a provider makes.

A child asset opts in by carrying `ParentConnectionId` on its inventory config,
set at discovery time with `inventory.WithParentConnectionId(parentID)`. At
connect time `providers-sdk/v1/plugin/service.go` resolves the parent and does:

```go
runtime.Resources = parentRuntime.Resources   // the same map, not a copy
```

That one assignment has consequences well past list/get:

| Property | Consequence |
|---|---|
| It shares the **map object**, not a snapshot | Anything any asset resolves becomes visible to every sibling, in both directions, for the parent's lifetime |
| Reuse is per **resolved field**, not per call | A `sync.Once` on an Internal struct, a memoizer hung on a root resource, a lazily fetched sub-object — all of it is shared once the instance is |
| Identity is `resourceName + "\x00" + __id` | Sharing only pays when the `__id` is stable across assets. An `__id` that embeds the asset, or is empty, silently defeats it |
| Writes are shared too | Internal-struct caches must be safe for concurrent use once assets can be scanned in parallel |
| Lifetime is the **parent's** | Children look the parent up at connect time; if it is already closed they fall back to their own cache and `service.go` logs `parent connection not found`. Memory frees only when the parent closes |
| It is same-provider only | `runtime.go`'s `crossProviderAsset` clears the parent id before handing an asset to another provider |

**Choosing the scope is a correctness decision, not a performance knob.** Share
at the level the data belongs to. Share *above* it and resources that read their
scope from the connection answer for the wrong scope — silently, because a
wrong-but-present answer never errors. Share *below* it and nothing is reused.

**Three ways to share, picked by what the data is keyed on:**

| Pattern | Use when | Example |
|---|---|---|
| Parent connection id | Data belongs to a scope that has its own connection | aws, gcp, k8s, github discovery |
| Memoizer on a root MQL resource | Data is not attached to any one resource, but is scope-wide | github `mqlGithubInternal.memoize` for `getUser`/`getOrg` |
| Process-wide cache keyed by identity | Data outlives connections entirely and is keyed on credentials, not scope | azure `credentialCache`; `plugin.Service.Memoizer` |

The second only works if the first is in place — a memoizer on a root resource
is per resource cache, so without parenting it is per asset. Reach for the third
when there is no connection to hang the scope on.

### What already caches, and what it misses

Before concluding something is re-fetched, know which of the three existing
layers should have caught it. Two of them are easy to mistake for full coverage:

| Layer | Where | Covers |
|---|---|---|
| `GetData` cache check | `plugin/service.go` | The client asking for a field on a resource id already in the cache. No init runs. |
| `GetOrCompute` | `plugin/runtime.go` | A field memoized on the **referring instance**, so `nicA.subnet` computes once however often it is read |
| `NewResource` cache check | generated `<provider>.lr.go` | A resource whose init already ran — but only **after** that init has run and fetched |

What none of them covers: **a new referring instance resolving a reference for
the first time.** Thirty NICs on four subnets means thirty separate `Subnet`
fields, thirty `NewResource` calls, thirty inits, thirty fetches — for four
subnets. `GetOrCompute` does not help because each NIC has its own field;
`GetData` does not help because the provider is minting the reference rather
than being asked for a field on a known id.

## When to use

- A scan times out, or raising parallelism makes it worse rather than better
- Debug logs show the same request URL tens of times
- Adding a provider's resources to a scan multiplies API traffic
- Before shipping a new provider, as a design review of its inits and discovery

**Not for:** a provider that is genuinely slow because the API is slow, or one
that fetches too much per call. Measure first — this skill only removes
*repeats*.

## Phase 1 — Measure before touching anything

Everything below is worthless without a before/after number. Providers log
their calls at debug (azure: `apiTracePolicy` in `connection/connection.go`); if
yours does not, add an equivalent per-call log line first.

```bash
mql run <provider> <scope-flags> --discover <target> -c "<query>" --log-level debug > before.log 2>&1

# total vs distinct, grouped by URL shape
grep "api call" before.log | sed -E 's/.*url=([^ ]+).*/\1/' \
  | sed -E 's#/resourceGroups/[^/]+/#/RG/#; s#/[^/]+$#/{name}#' \
  | sort | uniq -c | sort -rn
```

A/B against the released provider by moving your build aside and letting mql
reinstall the published one:

```bash
P=~/.config/mondoo/providers/<provider>
mv $P /tmp/mine && mql run ... > released.log 2>&1; rm -rf $P && mv /tmp/mine $P
```

**Four measurement traps that will fool you.** Each of these produced a wrong
conclusion in the work this skill came from:

| Trap | Guard |
|---|---|
| The binary under test is not yours — mql auto-updates providers, and A/B swaps leave the published build installed | Put a unique debug line in your build and `grep -c` it in every log before reading any number |
| Counting output lines instead of resolved values — a field that errors still prints a line | Count `field: "` (resolved) against `field: no data` separately |
| Trace logs usually strip the query string, so paginated list calls collapse onto one URL and read as duplication | Confirm a suspected duplicate is constant as asset count grows; if it is, it is pagination |
| **The probe never reaches the code you changed**, so before and after match and the fix reads as useless | A/B against a build with the change *neutered*. If neutered and real produce **identical** counts, suspect the probe, not the fix — then confirm the path by reading the accessor |

That last one is the expensive mistake. A typed reference may be resolved by an
accessor that fetches inline rather than through `NewResource`, in which case it
never enters the init and no init-level change can move its numbers. Verify the
path reaches your code before concluding anything from a flat A/B.

## Phase 2 — Classify every init

`classify-inits.awk` sorts a provider's init functions into three buckets:

```bash
awk -f .claude/skills/provider-api-call-dedup/classify-inits.awk providers/<name>/resources/*.go | sort
```

| Bucket | Meaning |
|---|---|
| `ARGS-ONLY` | Fills args, no I/O. Fine. |
| `FROM-LIST` | Resolves out of a parent list. The target state. |
| `API-CALL` | Fetches one resource itself. **Candidate.** |

Then intersect `API-CALL` with the provider's **discovery targets** — that
intersection is where the cost is, because a discovery target resolves its own
resource once per asset:

```bash
grep -n "Discovery[A-Z][A-Za-z]* *=" providers/<name>/resources/discovery.go
grep -l "getAssetIdentifier\|getAssetName" providers/<name>/resources/*.go
```

An `API-CALL` init that is *not* a discovery target is reached through typed
accessors instead — one call per **reference** rather than per asset. Its cost
is set by fan-in, so count the referring call sites before sizing it:

```bash
rg -c "NewResource\([^)]*\"<resource.mql.name>\"" providers/<name>/resources/*.go | grep -v _test
```

### The classifier's blind spot: accessors that never reach an init

**`classify-inits.awk` only reads `init` functions, so an accessor that builds a
client and fetches inline is invisible to it.** Such an accessor never calls
`NewResource`, never enters an init, and therefore cannot be fixed by anything
in Phase 4 — while an audit that only ran the classifier will read as complete.

Enumerate them properly rather than grepping for client construction in a
window — that over-counts badly. And **do not pin the receiver name**: providers
use `a`, `g`, `k`, `o`, `r`, `m`, `c`, `p`, `i`. Pinning one silently reports
zero for every provider that picked another letter.

Real counts, receiver-agnostic:

| provider | INLINE | BOTH | VIA-NEW |
|---|---|---|---|
| aws | 41 | 6 | 550 |
| azure | 38 | 0 | 136 |
| github | 16 | 1 | 5 |
| ms365 | 14 | 0 | 11 |
| gcp | 10 | 0 | 71 |
| okta | 1 | 0 | 4 |
| k8s | **0** | 0 | 23 |

k8s at zero is the shape to aim for. Everything else has some.

```bash
awk -f .claude/skills/provider-api-call-dedup/classify-accessors.awk \
    providers/<name>/resources/*.go | cut -f1 | sort | uniq -c
```

**Then split them by fan-in, because most are not worth touching.** An inline
accessor falls into one of two kinds:

| Kind | Shape | Worth fixing |
|---|---|---|
| **Owned sub-object** | A config child belonging to exactly one parent — a database's audit policy, a storage account's management policy, a site's auth settings | **No.** One fetch per parent is the floor, and `GetOrCompute` already memoizes it on that parent. Nothing repeats |
| **Shared resource** | A subnet, a public IP, a firewall policy, a disk — something several resources point at | **Yes.** Fan-in means repeats, and nothing dedupes them |

In azure that split is roughly 30 owned sub-objects against ~12 shared targets,
so the backlog is small and specific, not half the provider. Measure fan-in per
target before proposing anything:

```bash
rg -c "\"<target.mql.name>\"" providers/<name>/resources/*.go | grep -v _test | wc -l
```

## Phase 3 — Scope the resource cache

```bash
rg -n "WithParentConnectionId" providers/<name> -g '*.go' | grep -v _test
```

No hits means every asset is paying for its own copy of everything. Fix that
first: it is the larger win and it makes Phase 4 measurable. Hits mean the
provider shares *something* — read them and write down **at which level**, then
run the gate below anyway, because sharing at the wrong level is the more
dangerous state of the two.

**Check the scope is safe.** Find every place a resource reads its scope from
the *connection* rather than from its own id or args:

```bash
# whatever the provider calls its scope: SubId, ProjectID, AccountId, ResourceID
rg -c "conn\.(SubId|ProjectID|AccountId|ResourceID)\(\)" providers/<name>/resources -g '*.go'
```

Every hit is a place where a cache shared across two scopes returns **the wrong
scope's answer instead of failing**. The sharp test is per init: does it honour
`args` for the scope, and only fall back to the connection when they are absent?
An init that reads the connection unconditionally will answer for whichever
scope connected first.

```bash
# inits that fall back to the connection without first honouring args
rg -l "conn\.(SubId|ProjectID|AccountId|ResourceID)\(\)" providers/<name>/resources -g '*.go'
```

`initGcpProject` (`providers/gcp/resources/project.go`) is the worked fix: it
resolves the project the *caller* asked for and documents why in place. Also
look for a singleton memo on a root resource (azure pins the subscription on
`mqlAzure.sub`), which pins the first scope connected for the whole run.

If the provider discovers many scopes from one root connection, there is no
per-scope connection to parent onto yet — that is what staged discovery builds.
**REQUIRED SUB-SKILL:** use `staged-discovery` for that half.

## Phase 4 — Migrate inits

Three shapes, picked by what the init is resolving. They compose in this order:
**cache first, then the list, then fetch.**

| Shape | Use for | Miss behaviour |
|---|---|---|
| **cache-first** | Anything where the caller supplies the id | Fall through to the next shape |
| **from-list** | An **asset** the scan was handed | **Error** — the asset should be there |
| **list-first-with-fallback** | A **typed reference** | Fall through to the existing fetch |

### Cache-first

`NewResource` cannot check the cache before the init in general, because the key
is the resolved `MqlID()`. But when the caller already supplied the id — which
reference resolution always does — the key is known up front, so the init can
look it up before spending a call:

```go
if cached := cachedResource(runtime, ResourceFooBar, id); cached != nil {
    return args, cached, nil
}
```

Cheap, no pre-flight needed, and the only option when there is no list to
resolve from. The lookup is exact, because that is how the runtime keys the
cache; a casing mismatch simply falls through to the fetch, as today.

### From-list and list-first

Target shape, mirroring k8s's `initNamespacedResource`
(`providers/k8s/resources/common.go`): resolve the parent service, walk its
already-fetched list, return the instance that is there. With more than a couple
of resources, write one shared generic helper rather than copying the body.

**Asset inits and reference inits need different miss behaviour.** For an asset,
a miss means something is wrong — error. For a typed reference, a miss is
routine: it may point at another scope, at something deleted, or at something
the caller cannot read. Existing inits usually say so in a comment; converting
them to a hard error turns graceful degradation into a failed query. Give
references a lookup that returns nil and let the caller keep its own fetch.

A reference lookup should also **skip the list when the id names a different
scope** than the connection — the list only covers this one, so consulting it
wastes the walk and risks matching a same-named resource elsewhere.

**Run three pre-flight checks per resource. Skipping the second or third ships a
regression.**

1. **Does a sibling list accessor exist** returning the same resource type?
2. **Do the list and the init build the resource the same way?** If they share a
   builder (`fooToMql`), migrating is a pure win. If the init sets fields or
   primes caches the list does not, the list is a *reduced projection* and
   migrating **adds** a call per resource. Fix the list first — often a
   different SDK pager returns full objects (azure Key Vault: `List` returns a
   bare `TrackedResource`, `ListBySubscription` returns the full `Vault`).
3. **Do both ids come from the same API?** An asset's id usually comes from a
   generic inventory endpoint while the list comes from a service endpoint, and
   clouds do not agree with themselves on casing between the two. Match with
   `strings.EqualFold`, never `==`.

Two rules for the body:

- For an **asset** init, a miss is an **error**, never `return args, nil, nil` —
  falling through has the runtime build a blank resource whose fields are unset
  rather than null, which surfaces client-side as an untyped null with nothing
  pointing at the cause.
- Keep the `len(args) > N` fast path, so callers that already supplied
  everything do not trigger a list walk.

**State the trade in the PR.** Resolving one asset in isolation now pulls the
whole scope's list instead of one record. That is right during a scan, where the
list is wanted anyway, and wrong for a one-off shell query against a single
asset.

## Gotchas

| Symptom | Cause |
|---|---|
| Child assets re-fetch everything despite a parent id | Parent runtime already closed, or the parent id belongs to another provider. `service.go` logs `parent connection not found` — grep for it |
| A whole scope's assets report another scope's data | Cache shared above the scope the resources read off the connection |
| Traversal ends at the parent; no leaf assets appear | Child config was cloned `WithoutDiscovery()`; the next stage needs its targets |
| Every leaf re-runs discovery for the whole scope | Leaf config kept its discovery targets; leaves must be cloned `WithoutDiscovery()` |
| Init-resolved asset reports "not found" for a resource that exists | Case-sensitive id comparison |
| Migrating an init *increased* calls | The list is a reduced projection; pre-flight check 2 |
| A reference resolves fine but the fix changes nothing | The accessor fetches inline and never enters an init — see the classifier's blind spot |
| A converted reference now errors where it used to return a bare reference | A reference miss must fall through, not error; only asset inits error |
| Field count looks right but every value is empty | Counting lines, not resolved values |

## Done criteria

- Before/after call counts from the same query, both logs provenance-checked,
  and the probe confirmed to reach the changed code
- Per-resource calls equal the number of **distinct resources**, not assets
- Accessors that fetch inline are counted and reported as out of scope, so the
  audit does not overstate its coverage
- The discovered asset set is byte-identical to the base branch
- At least one migrated resource cross-checked against the cloud CLI, not just
  against itself
- Resources with no test infrastructure are named as unverified in the PR
- Unit tests for the shared helper: resolve, miss-is-an-error, empty list,
  fast path, casing-only difference matches, different path still does not
