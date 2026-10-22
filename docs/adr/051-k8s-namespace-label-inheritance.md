# ADR 051: Kubernetes namespace label inheritance

**Status:** Accepted
**Date:** 2026-10-22

Design and phased plan:
`docs/superpowers/specs/2026-10-20-k8s-namespace-label-inheritance-design.md`.

---

## Context

Customers label namespaces with ownership facts, for example
`my.company.com/team: aaabbbccc`, and want every workload under such a
namespace to carry the same label. The immediate consumer is org-level asset
routing on the platform (server ADR 020): an agent reporting at organization
scope has each uploaded asset matched against routing rules on the asset's own
incoming labels, first match wins, and the asset is placed in the matching
team's space. A workload can only be routed by its team if the team label is
on the workload asset itself.

Today the k8s provider gives a workload asset its own object labels plus the
`k8s.mondoo.com/{namespace,name,uid,kind,apiVersion,resource-version,cluster-id,owner-*}`
keys (`addMondooAssetLabels` in `providers/k8s/resources/discovery.go`). The
namespace's labels end up only on the namespace asset. Nothing between
discovery and upload fills the gap: the client-side discovery walk copies
labels from the *root* asset (the cluster) to its children, and only keys
matching `*mondoo.com/*` (`AddMondooLabels` in the SDK), so a customer key
never crosses from a namespace to a workload.

Two platform behaviours shape the design:

- The asset store replaces an asset's labels on every sync and merges its
  annotations (`replaceLabels` / `mergeAnnotations` in the server's asset
  store). A label is therefore scanner-reported state with a clean lifecycle;
  an annotation is sticky and never clears.
- Under staged discovery, which cnspec always enables, the provider emits no
  related-asset links. Every cluster, namespace, workload and image is uploaded
  as an independent top-level asset and routed on its own labels.

## Decision

### The option

A k8s connection option `inherit-namespace-labels` (constant
`OPTION_INHERIT_NAMESPACE_LABELS`), also exposed as the CLI flag
`--inherit-namespace-labels`, holds a comma-separated list of label keys. Globs
are accepted with the same matcher the `images` and `namespaces` filters use,
so `my.company.com/*` inherits a whole prefix and `*` inherits every label of
the namespace. An invalid glob fails discovery with an error naming the option,
like the existing filters. Unset or empty, discovery output is byte-identical
to before.

### Labels, not annotations

Inherited values are labels. The user asked for labels, the k8s object's own
labels are already asset labels, and replace-on-sync is the right lifecycle:
relabel or unlabel the namespace, and the next scan updates every workload.
Every path that scans a workload is this provider fed from the same
configuration, so no other scanner strips the key. Annotations, the choice the
platform made for the AWS account id (server ADR 271), exist for values that
must survive scans by paths that do not know them, and they never clear.

### The namespace only fills gaps

For each listed key, a workload that already carries the key keeps its own
value; the namespace supplies the key only when the workload lacks it. Nothing
is ever removed. This is the Kubernetes reading (the more specific object
overrides the more general one) and it never destroys a label a workload set
deliberately. Accepted consequence: a pod author can set a listed key on the
pod and route it elsewhere. The alternative, where the namespace is the
authority and keys it lacks are stripped from workloads, was rejected as a
visible, destructive change to existing labels.

### Which assets

Every namespaced workload asset discovered under the namespace: pods, jobs,
cronjobs, services, statefulsets, deployments, replicasets, daemonsets,
ingresses. Not the namespace asset (it already carries its own labels), not
the cluster, not admission-review assets (no namespace object in hand, and a
deprecated path), not cluster-scoped objects.

Only the namespace's **own** labels are inherited. The namespace asset also
carries the provider's `k8s.mondoo.com/*` keys; those are never copied, which
is why the legacy discovery path reads the raw namespace labels rather than the
namespace asset.

### Where it runs

Once per namespace, from the two places that already hold the namespace
object: the staged namespace stage (`discoverNamespaceStage`, which fetches the
namespace it is about to discover) and the legacy per-namespace loop in
`discoverLegacy`, which gets each namespace's own labels from
`discoverNamespaces`. No Kubernetes API call is added per asset. A manifest
scan whose objects reference a namespace the manifest never declares sees the
synthesized bare namespace and inherits nothing, without error.

### Container images (planned, Phase 3)

An image is one asset per digest, shared by every namespace, cluster and
registry that runs or scans it, and routing places an asset in exactly one
space, so a single-valued team label on an image is a heuristic. Images are
therefore excluded from this change and get their own rule in a follow-up:

- Per inherited key, over the in-scope namespaces that run the image and carry
  the key: all agree → the image gets the value; they disagree → the key stays
  unset and the image gets the marker label
  `k8s.mondoo.com/namespace-label-conflict` naming the conflicting keys, so a
  routing rule can send shared images to a shared space ahead of the catch-all.
- The stage-1 cluster stage builds the set of ambiguous images once, from pod
  status, and hands it to every namespace stage.
- The rule is judged within one cluster scan. Under org routing that is sound,
  because the value selects the space and disagreeing clusters never share an
  asset. In a single space fed by several clusters the shared image flips
  between teams on each scan; accepted and documented as a known limit.

## Consequences

- A scanner path that does not pass the option strips the inherited label on
  its next sync, because labels are replaced per sync. The operator must pass
  the option to its scheduled scan, its external-cluster scan, its
  container-image scan and its resource watcher alike.
- Routing does not move an asset. Relabeling a namespace creates the workload
  anew in the new team's space and leaves the old asset to go stale in the
  previous space; that is routing's existing behaviour.
- A workload's own label for a listed key is authoritative; the platform
  cannot tell an inherited value from an own one, which is the intended
  reading.
- `discoverNamespaces` now returns each namespace's own labels alongside the
  assets; the cluster stage ignores them.
- Console and server parity for the setting (so it can be given on first
  connection and managed from the app) is a follow-up; until then the feature
  is configured through the operator CR or `cnspec scan k8s`.

---

## Verification

```
cd providers/k8s
go test ./resources -run 'TestInherit|TestDiscover' -count=1
go test ./provider -run TestParseCLI -count=1

cnspec scan k8s --discover pods --inherit-namespace-labels 'my.company.com/team'
```

Against a cluster whose namespace `team-a` carries `my.company.com/team`: every
pod discovered in `team-a` reports the label, a pod that sets the key itself
keeps its own value, and pods in namespaces without the key are unchanged.
