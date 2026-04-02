# ADR 002: Staged Discovery for Hierarchical Providers

## Status

Accepted

## Context

Cloud providers organize resources in hierarchies: Kubernetes has clusters → namespaces → workloads, GCP has organizations → projects → services → resources, AWS has organizations → accounts → regions → resources. When mql discovers assets in these environments, the naive approach is to enumerate **everything** in a single pass — all namespaces, all pods, all deployments, cluster-wide.

This creates three problems:

- **MQL resource cache bloat:** Each provider runtime maintains an in-memory MQL resource cache. In single-pass discovery, every discovered asset shares the root connection's runtime, so **all** resource objects (every pod, every deployment, every instance) accumulate in a single cache that is never released until the entire scan completes. For a K8s cluster with 50 namespaces and 200 pods per namespace, that's 10,000+ MQL resource objects pinned in one runtime's cache — even though we only need one namespace's worth at a time. Breaking the parent-child relationship into separate connections with their own runtimes ensures each scope has its own cache that is released when that scope is closed.
- **Connection memory:** A single discovery pass creates and holds open gRPC connections (runtimes) for all assets at once. For large clusters with hundreds of namespaces and thousands of workloads, this exhausts memory before scanning even begins.
- **API pressure:** A single discovery pass issues API calls for every resource type across every scope simultaneously, hitting rate limits and creating unnecessary load on the control plane.

The root cause is that single-pass discovery attaches everything to a single root runtime. All child assets, their API responses, and their MQL resource objects accumulate in that root's cache and connections, with no way to release any of it incrementally.

With `AssetExplorer` (see [ADR 001](001-asset-explorer-lazy-discovery.md)) driving caller-controlled, one-level-at-a-time traversal, providers can now split discovery into stages that align with their natural hierarchy. Each stage creates **separate runtimes with their own caches** — the caller connects to one level, scans it, closes it (releasing its cache and connections), then moves to the next.

## Decision

Introduce **staged discovery** — an opt-in provider pattern where discovery is split into multiple phases that match the provider's resource hierarchy. The `OptionStagedDiscovery` flag (`staged-discovery`) is set on inventory connection options by callers (shell, run, cnspec scan). Providers check for this flag and route to stage-specific discovery logic. When the flag is absent, the legacy single-pass discovery runs unchanged.

### Backward Compatibility

Staged discovery is **additive** — providers must continue supporting the legacy single-pass flow for older clients that don't set `OptionStagedDiscovery`. Providers are distributed independently of mql/cnspec, and older client versions (which don't set the flag) will continue connecting to updated providers. If a provider only implemented the staged path, those older clients would get no assets.

The `OptionStagedDiscovery` flag acts as a capability negotiation: when present, the provider uses the new staged path; when absent, it falls back to the existing single-pass discovery. Both paths must return the same set of assets — they differ only in how discovery is chunked and how memory is managed.

**Legacy single-pass support will be dropped in the next major release (v14).** At that point, all supported clients will set `OptionStagedDiscovery`, and providers can remove the legacy code path.

### How It Works

Each stage returns a set of assets whose connection configs are pre-configured to trigger the next stage when `AssetExplorer` connects to them. The caller doesn't need to know about stages — it just sees discovered children and connects to them as usual.

### Reference Implementation: Kubernetes

The K8s provider splits discovery into two stages:

**Stage 1 — Cluster scope** (`discoverClusterStage`):
- Triggered when `OptionStagedDiscovery` is set and `OPTION_NAMESPACE` is empty
- Discovers the cluster asset itself (platform ID from kube-system namespace UID)
- Discovers all namespaces as separate assets
- Each namespace asset's connection config is cloned with `OPTION_NAMESPACE` set to that namespace's name — this is what triggers Stage 2 when the namespace is connected
- **Critical:** Namespace assets do NOT use `WithParentConnectionId`. Each namespace gets its own independent runtime and MQL resource cache. This is the key to memory isolation — when a namespace is closed, its entire cache (all pods, deployments, etc. discovered under it) is released.
- Returns: 1 cluster asset + N namespace assets
- Does NOT load any workloads (pods, deployments, etc.)

**Stage 2 — Namespace scope** (`discoverNamespaceStage`):
- Triggered when `OptionStagedDiscovery` is set and `OPTION_NAMESPACE` is non-empty
- Reads the namespace name from the connection config
- Discovers workloads scoped to that single namespace (pods, deployments, jobs, services, ingresses, etc.)
- Workload assets use `WithParentConnectionId` to share the **namespace** connection's API cache (not the cluster root's cache). This means workloads within a namespace share API clients efficiently, but their cache is scoped to the namespace runtime and released when the namespace is closed.
- Returns: workload assets for that one namespace

**The traversal flow:**
```
AssetExplorer connects to K8s cluster
  → Stage 1: returns [cluster, ns-default, ns-kube-system, ns-prod, ...]

AssetExplorer connects to ns-default
  → Stage 2: returns [pod-a, pod-b, deployment-x, ...]
  → Scan pod-a, pod-b, deployment-x → close ns-default

AssetExplorer connects to ns-prod
  → Stage 2: returns [pod-c, pod-d, ...]
  → Scan pod-c, pod-d → close ns-prod
```

At no point are ns-default's workloads and ns-prod's workloads in memory simultaneously.

**Gating namespace-scoped resources at cluster scope:**

When scanning the cluster asset (Stage 1), namespace-scoped resource methods (e.g., `k8s.pods`, `k8s.deployments`) should return empty lists rather than loading everything cluster-wide. This is gated behind a connection scope check:
- If `OptionStagedDiscovery` is set AND `OPTION_NAMESPACE` is empty → cluster scope → return `[]` for namespaced resources
- If `OPTION_NAMESPACE` is set → namespace scope → return resources filtered to that namespace
- If `OptionStagedDiscovery` is absent → legacy path → load everything (backward compatible)

### Traversal-Only Assets (Discovery Target Filtering)

Staged discovery introduces a second concern: **not every intermediate asset should be scanned**. When a user specifies discovery targets like `--discover pods`, they want only pods as scannable assets. Namespaces are still needed for traversal (connecting to a namespace triggers Stage 2 which discovers pods), but namespaces themselves should not appear in scan results.

This is solved by **stripping platform IDs** from intermediate assets that don't match the requested discovery targets. `AssetExplorer` and the scanner already skip assets without platform IDs (they log a warning, close the asset, and continue). By emitting these assets without platform IDs, they serve purely as traversal nodes — `AssetExplorer` connects to them (triggering the next discovery stage and populating their children), but the scanner never adds them to the progress bar or sends them for scanning.

**Provider side** — the provider already knows the discovery targets from `invConfig.Discover.Targets`. When emitting intermediate assets, check whether that level is a target and strip platform IDs if not:

```go
// In discoverClusterStage, when emitting namespace assets:
nsIsScannable := stringx.ContainsAnyOf(invConfig.Discover.Targets,
    DiscoveryNamespaces, DiscoveryAuto, DiscoveryAll)

for _, ns := range nss {
    nsConfig := invConfig.Clone()
    nsConfig.Options[shared.OPTION_NAMESPACE] = ns.Name

    // Namespaces that aren't a discovery target get their platform IDs
    // stripped. AssetExplorer still connects to them (triggering stage 2)
    // but the scanner skips them because they have no platform IDs.
    if !nsIsScannable {
        ns.PlatformIds = nil
    }

    ns.Connections = []*inventory.Config{nsConfig}
    in.Spec.Assets = append(in.Spec.Assets, ns)
}
```

**No caller-side changes needed.** The existing "no platform IDs → skip" logic in `AssetExplorer` and the scanner handles everything:
- Assets without platform IDs are not added to the progress bar
- Assets without platform IDs are not sent for scanning
- Assets without platform IDs are still connected (to discover children), then closed

**How this generalizes:**

| Command | No platform IDs (traversal only) | With platform IDs (scannable) |
|---|---|---|
| `k8s --discover pods` | namespaces | pods |
| `k8s --discover namespaces` | (none) | cluster + namespaces |
| `k8s --discover all` | (none) | cluster + namespaces + all workloads |
| `gcp --discover compute-instances` | org, projects, service groups | compute instances |
| `aws --discover ec2-instances` | accounts, regions | EC2 instances |

**Mixed targets** (`--discover pods,namespaces`): namespaces are both scannable AND traversal nodes. The provider keeps their platform IDs intact. They get scanned and their children get discovered. No special handling needed.

### Applying to Other Providers

The pattern generalizes to any provider with a hierarchical resource model. The key insight: **each level of the hierarchy becomes a discovery stage, and the connection config for child assets encodes which stage to run next.** Crucially, each stage boundary creates a new runtime with its own MQL resource cache — when that scope is closed, all cached resources under it are released.

**GCP (proposed):**
```
Stage 1 — Organization scope:
  → Discovers organization asset + project assets
  → Each project gets its own runtime (no WithParentConnectionId)
  → Each project asset's config includes project-id

Stage 2 — Project scope:
  → Discovers project asset + service grouping assets (Compute, GKE, Cloud SQL, ...)
  → Each service gets its own runtime — closing it releases all resources
    discovered under that service from memory
  → Each service asset's config includes project-id + service filter

Stage 3 — Service scope (optional, for very large projects):
  → Discovers individual resources within that service
  → Resources use WithParentConnectionId to share the service runtime's
    API client cache (not the project's or org's)
```

Traversal: `org → project-A → compute → [instances...] → close compute (releases all instance data) → gke → [clusters...] → close gke → close project-A (releases project data) → project-B → ...`

**AWS (proposed):**
```
Stage 1 — Organization/account scope:
  → Discovers account asset + regional grouping assets
  → Each region gets its own runtime (no WithParentConnectionId)
  → Each region asset's config includes region filter

Stage 2 — Region scope:
  → Discovers resources in that region across services
  → Resources use WithParentConnectionId to share the region runtime's
    API clients — when the region is closed, all its resource data is freed
```

### Provider Implementation Guide

To add staged discovery to a provider:

1. **Check the flag** in your `Discover()` function:
   ```go
   if _, ok := invConfig.Options[plugin.OptionStagedDiscovery]; ok {
       // Route to stage-specific logic based on connection config
   }
   ```

2. **Return child assets with pre-configured connections** that encode the next stage's scope:
   ```go
   childConfig := invConfig.Clone()
   childConfig.Options["my-provider-scope"] = scopeValue
   childAsset.Connections = []*inventory.Config{childConfig}
   ```

3. **Break the cache chain at stage boundaries.** Child assets that represent a new scope (e.g., namespaces, projects, regions) must NOT use `WithParentConnectionId` — they need their own independent runtime so their MQL resource cache is released when the scope is closed. Only leaf assets within the same scope (e.g., pods within a namespace) should use `WithParentConnectionId` to share their parent scope's API cache. If all children share the root's connection, their resources accumulate in the root's cache and are never released until the entire scan completes — defeating the purpose of staged discovery.

4. **Gate resource methods** so higher scopes don't load lower-scope data:
   - Organization scope: only org-level resources
   - Project/account scope: only project/account-level resources
   - Service/region scope: all resources within that scope

5. **Keep the legacy path** — when `OptionStagedDiscovery` is absent, run single-pass discovery unchanged. Older clients that don't set the flag must continue to work exactly as they do today. Both paths must produce the same final set of assets. The legacy path will be removed in v14.

## Alternatives Considered

### Pagination within single-pass discovery

Instead of splitting into stages, paginate API responses and process resources in streaming fashion. Rejected because:
- Doesn't solve the hierarchical scoping problem — you still need all API clients open simultaneously.
- Pagination is orthogonal to staged discovery; providers should paginate within each stage regardless.
- Doesn't enable the "scan one branch, close it, move to next" memory model.

### Provider-side batching

Have providers internally batch resources and yield them in chunks. Rejected because:
- Moves complexity into every provider rather than leveraging the shared `AssetExplorer` traversal.
- The caller (scanner/shell) still holds connections to all batches simultaneously.
- Doesn't compose with `AssetExplorer`'s cleanup model (`CloseAsset` after scanning).

### Automatic hierarchy inference

Have `AssetExplorer` automatically infer hierarchy from platform IDs or asset metadata. Rejected because:
- Hierarchies are provider-specific and not always derivable from IDs alone.
- Providers know their optimal discovery boundaries (e.g., K8s namespace is the right boundary for cache sharing).
- Explicit stages give providers control over connection caching (`WithParentConnectionId`) and API call patterns.

## Consequences

### Positive

- **Bounded memory per branch:** Each scope boundary creates a separate runtime with its own MQL resource cache. When a scope is closed (`CloseAsset`), its entire cache — all MQL resource objects, API responses, and connection state — is released. Only one branch of the hierarchy is in memory at a time. A 1000-namespace cluster uses the same peak memory as a 5-namespace cluster.
- **No root cache accumulation:** In single-pass discovery, all resources attach to the root runtime's cache and are never released until the scan completes. Staged discovery breaks this by giving each scope its own cache — pods in namespace A are cached in namespace A's runtime, not the cluster root's. When namespace A is closed, those pods are gone from memory.
- **Reduced API pressure:** Each stage only queries the APIs needed for its scope. No cluster-wide enumeration of every resource type.
- **Discovery target filtering with zero caller changes:** Providers strip platform IDs from intermediate assets that don't match discovery targets. The existing "no platform IDs → skip" logic in `AssetExplorer` and the scanner handles the rest — no new flags, fields, or methods needed on the caller side.
- **Composable with AssetExplorer:** Callers don't need to understand stages — they just connect discovered children as usual. The staging is entirely provider-internal.
- **Backward compatible:** The `OptionStagedDiscovery` flag is opt-in. Providers without staged discovery and callers that don't set the flag continue working unchanged.
- **Cache sharing within scope:** `WithParentConnectionId` lets leaf assets within a scope (e.g., pods within a namespace) share that scope's API client cache, avoiding redundant API calls — while keeping the cache isolated from other scopes.

### Negative

- **Per-provider implementation effort:** Each provider must implement its own staging logic — there's no generic framework that auto-stages. The pattern is documented but the work is manual.
- **Discovery latency for full enumeration:** When a caller does want all assets (e.g., `connectAll` in the run command), staged discovery adds round-trips compared to single-pass. Each stage requires a connect → discover cycle. For interactive use this is a feature (incremental reveal); for batch scanning it's a minor overhead.
- **Connection config as state machine input:** The child asset's connection config encodes which stage to run next (e.g., `OPTION_NAMESPACE` triggers Stage 2). This is implicit — there's no formal stage declaration, just convention. Providers must document their stage transitions.
- **Legacy path maintenance until v14:** Providers must maintain both staged and legacy discovery paths through the v13 lifecycle. Older clients that don't set `OptionStagedDiscovery` must continue to work unchanged. This doubles the discovery code surface area per provider until the legacy path is dropped in v14.
