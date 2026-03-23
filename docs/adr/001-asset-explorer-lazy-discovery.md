# ADR 001: Lazy, Caller-Driven Asset Discovery via AssetExplorer

## Status

Accepted

## Context

mql's `DiscoverAssets()` function eagerly discovers the **entire** asset tree recursively, using a pool of 10 worker goroutines. Every discovered asset gets a live connection (runtime) immediately, and all runtimes are returned to the caller in one batch. This works fine for batch scanning (the `run` command), but breaks down in two scenarios:

- **Large inventories (50k+ assets):** Memory grows linearly because every discovered asset holds an open gRPC connection. A Kubernetes cluster with thousands of pods, or an AWS account with thousands of instances, can exhaust memory before the user even picks which asset to query.
- **Interactive shell:** The shell command only needs **one** asset's connection at a time, but the eager approach forces it to discover and connect to everything upfront — wasting resources and adding startup latency.

We needed a discovery mechanism that lets the caller control **when** connections are created and **which** assets to explore, without changing the existing batch discovery path.

## Decision

Introduce `AssetExplorer` — a stateful, lazy discovery struct that discovers one level at a time and lets the caller drive traversal. It coexists with the existing `DiscoverAssets()` function; only the `shell` command is migrated to use it.

### Asset Lifecycle State Machine

```
AssetDiscovered ──Connect()──→ AssetConnected ──CloseAsset()──→ AssetClosed
                                                                 (terminal)
```

- **Discovered:** Known from a parent's inventory, no connection or runtime allocated.
- **Connected:** Runtime created, provider connection live, immediate children discovered.
- **Closed:** Connection disposed, runtime released. Terminal — closed assets cannot be reconnected.

There are no transient states (no "connecting" or "closing"). State transitions are atomic under a global mutex, which simplifies reasoning in UI code.

### Core API

| Method | Purpose |
|--------|---------|
| `NewAssetExplorer(ctx, cfg)` | Connects to root asset(s), discovers first-level children |
| `Connect(asset)` | Connects to a discovered asset, discovers its children, runs dedup |
| `CloseAsset(asset)` | Disposes a single connection (terminal) |
| `Shutdown()` | Closes all connected assets |
| `Discovered()` / `Connected()` / `Errors()` | Thread-safe snapshots of current state |

### Deduplication

Same subset/superset platform-ID logic as v1:

1. **New asset is a subset of an existing connected asset:** New asset is evicted (closed immediately), `Connect()` returns `ErrDuplicateAsset`.
2. **Existing connected assets are subsets of the new asset:** Existing subsets are evicted (closed), new asset is kept.
3. **Disjoint platform IDs:** Both coexist.

Two dedup layers exist by design:
- **`discoverChildren()` (fast, simple):** Skips children whose platform IDs already appear in `allAssets`. Cheap but may miss subset relationships.
- **`Connect()` (full):** Runs complete subset/superset analysis after the runtime is created and platform IDs are known.

### Staged Discovery

All connections created through the shell path set the `OptionStagedDiscovery` flag. This tells providers they may split discovery into phases (e.g., K8s discovers the cluster first, then namespaces on demand). The flag is transparent to `AssetExplorer` — providers opt in independently.

### Shell Command Integration

The `shell` command drives an interactive traversal loop:

1. Root asset is connected; its children are presented to the user.
2. User picks an asset from the list.
3. If the selected asset has children, a menu appears:
   - **"» Connect to \<asset\>"** — launch shell on the current asset.
   - **Child assets** — navigate deeper.
4. This repeats until the user picks a leaf or chooses "Connect here".
5. Shell launches with the selected asset's runtime; `Shutdown()` cleans up everything else on exit.

`TrackedAsset` implements the `SelectableItem` interface (`Display() string`) for direct integration with the `components.Select()` interactive prompt.

### Coexistence with v1

| Aspect | DiscoverAssets (v1) | AssetExplorer (v2) |
|--------|---------------------|---------------------|
| Discovery mode | Eager, recursive, entire tree | Lazy, one level, caller-driven |
| Parallelism | 10-goroutine worker pool | None; caller controls sequencing |
| Runtime ownership | Returned in batch; caller owns cleanup | Held in explorer; explicit `CloseAsset()` / `Shutdown()` |
| Deduplication | `DiscoveredAssets.Add()` | `dedup()` + `discoverChildren()` |
| Use case | Batch scanning (`run`, `plugin.go`) | Interactive shell |

Both share the same underlying utilities: `createRuntimeForAsset()`, `prepareAsset()`, coordinator connection dedup, and `slicesx.IsSubsetOf()` for platform-ID matching.

## Alternatives Considered

### Modify DiscoverAssets() to support lazy mode

Adding a "lazy" flag to the existing function was considered, but rejected because:
- The recursive worker-pool architecture is deeply coupled to eager semantics.
- Retrofitting caller-driven control would require breaking the return type (`DiscoveredAssets`) and every caller.
- A separate struct with its own lifecycle is simpler to reason about and test independently.

### Connection pooling / LRU eviction

Instead of caller-driven connect/close, we could have kept eager discovery but evicted idle connections via an LRU cache. Rejected because:
- Still discovers the full tree upfront (API rate limits, metadata overhead).
- LRU eviction makes connection availability unpredictable for the shell's "pick and query" UX.
- Adds complexity without solving the core problem of unnecessary discovery.

### Stream-based discovery (channels / iterators)

A channel-based approach where discovered assets are streamed to the caller was considered. Rejected because:
- The shell needs to present **all siblings at once** for selection, not one-at-a-time.
- Channel semantics complicate backpressure and cancellation for an interactive UI.
- The state machine approach is more explicit and testable.

## Consequences

### Positive

- **Constant memory for shell:** Only the assets the user navigates to consume connections. A 50k-asset inventory uses the same memory as a 5-asset one until the user drills in.
- **Faster shell startup:** Only root + first-level children are connected, not the entire tree.
- **No v1 breakage:** `DiscoverAssets()` is untouched; all existing callers (`run`, plugins, scanning) continue to work identically.
- **Testable state machine:** 11 unit tests cover state transitions, dedup, closed-asset rejection, parent chains, and error cases — without needing cloud credentials.
- **Foundation for future callers:** Any new command that needs incremental discovery (e.g., a TUI explorer, programmatic API) can use `AssetExplorer` directly.

### Negative

- **Caller responsibility for cleanup:** Unlike v1 which returns everything in one shot, v2 requires explicit `CloseAsset()` or `Shutdown()`. Forgetting to call these leaks connections.
- **No concurrent `Connect()` safety:** The global mutex serializes all state mutations. Multiple goroutines calling `Connect()` concurrently is not supported (acceptable for the single-threaded shell UI, but would need per-asset locks for parallel callers).
- **Two discovery paths to maintain:** v1 and v2 share utilities but have separate entry points. Changes to discovery semantics (e.g., new dedup rules) must be applied in both places.
- **Closed assets are terminal:** An asset that was closed cannot be reconnected — the caller must create a new explorer. This is by design (prevents stale state) but limits flexibility for long-running sessions.
