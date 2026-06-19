# AWS Provider: Cross-Resource Edges for Security Policy Authoring

**Date:** 2026-06-18
**Status:** Design — pending review
**Provider:** `providers/aws` (cnquery/cnspec)

## Problem

AWS security checks today operate on **one resource type in isolation**. A real
example from `cnspec-enterprise-policies/queries/aws-storage.mql.yaml`:

```
aws.ec2.securityGroups.where(ipPermissions.any(fromPort <= 2049 && toPort >= 2049))
  .all(ipPermissions.where(...).all(ipRanges.none(_ == "0.0.0.0/0")))
```

This asks *"does a security group allow 0.0.0.0/0 on port 2049?"* It **cannot**
ask the question that actually matters and avoids false positives: *"is there an
**instance** genuinely exposed — in a public subnet, with an attached SG opening a
sensitive port?"* That requires traversing instance → ENI → subnet → security
group, and the **reverse** edges needed for it (SG → instances, subnet →
instances) do not exist in the schema.

The result is high-false-positive checks: a security group flagged for allowing
`0.0.0.0/0` even when it is attached to nothing, or attached only to instances in
a private subnet.

## Goal

Make relationship traversal cheap and reliable for policy authors by filling in
the **missing reverse and convenience edges** in the AWS resource schema. This is
"Approach A" from brainstorming, scoped to two mechanisms:

- **1.1 Backreference fields** — the missing reverse edges.
- **1.2 Missing forward edges** — convenience hops authors expect.

**Out of scope for this spec** (deferred to later passes):

- Universal `aws.resource(arn)` resolver (Approach A, mechanism 1.3).
- Computed exposure/reachability fields (Approach B).
- Provenance edges (CloudFormation stack, CloudTrail creator).

## Design

### The linking model

A backreference is resolved by **scanning the corresponding cached top-level
collection and filtering by membership** — not by issuing per-resource API calls.

This is the crux of the design's efficiency. Mechanics confirmed in the codebase:

- Top-level lists (`aws.ec2.instances()`, `aws.ec2.networkInterfaces()`) are
  materialized once and cached in `runtime.Resources` (see the cache lookup
  pattern in `resources/aws_kms.go:1208`).
- Forward edges call `NewResource(runtime, ResourceX, {arn})`, which **dedups
  against the `__id`-keyed cache** — linking to an already-materialized resource
  is free.
- Security policies already enumerate these top-level lists, so a backref scan
  costs **zero extra API calls** when the list is loaded, and **one cached
  list-fetch** when it is not. No N+1.

**Authoritative attachment source:** Security groups and subnets attach to
**Elastic Network Interfaces (ENIs)**, not directly to instances. `aws.ec2.networkinterface`
already exposes `instance()`, `subnet()`, and `securityGroups()`. Therefore SG→
and subnet→ backrefs are computed by scanning `aws.ec2.networkInterfaces` and
filtering, then projecting to instances where needed. This is more correct than
scanning instances directly (it captures ENIs not attached to instances — e.g.
load balancers, RDS, Lambda ENIs).

### Edge inventory

**Backreference fields (1.1):**

| Resource | New field | Type | Resolution |
|---|---|---|---|
| `aws.ec2.securitygroup` | `networkInterfaces()` | `[]aws.ec2.networkinterface` | scan `aws.ec2.networkInterfaces`, keep where `securityGroups` contains self |
| `aws.ec2.securitygroup` | `instances()` | `[]aws.ec2.instance` | project `networkInterfaces().instance`, dedup non-null |
| `aws.vpc.subnet` | `networkInterfaces()` | `[]aws.ec2.networkinterface` | scan `aws.ec2.networkInterfaces`, keep where `subnet.id == self.id` |
| `aws.vpc.subnet` | `instances()` | `[]aws.ec2.instance` | project `networkInterfaces().instance`, dedup non-null |
| `aws.iam.role` | `usedByInstances()` | `[]aws.ec2.instance` | scan `aws.ec2.instances`, keep where `iamInstanceProfile.iamRoles` contains self |
| `aws.kms.key` | `encryptedVolumes()` | `[]aws.ec2.volume` | scan `aws.ec2.volumes`, keep where `volume.kmsKey().arn == self.arn` (EBS-scoped start) |

**Forward convenience edges (1.2):**

| Resource | New field | Type | Resolution |
|---|---|---|---|
| `aws.ec2.instance` | `subnet()` | `aws.vpc.subnet` | the primary network interface's `subnet()` |

**Deliberately excluded (already exist):** `aws.ec2.networkinterface.instance`,
`aws.ec2.networkinterface.subnet`, `aws.ec2.instance.networkInterfaces`,
`aws.ec2.instance.securityGroups`, `aws.ec2.instance.vpc`.

### Implementation per edge

1. Add the field to `resources/aws.lr` with a doc comment matching the schema's
   existing style (one-line `//` description above the field).
2. Run `make lr` (or the provider's codegen target) to regenerate accessors.
3. Implement the resolver in the relevant `resources/aws_*.go` file:
   - Obtain the cached sibling list via `NewResource`/the top-level list resolver.
   - Filter by membership; build deduped target handles via `NewResource(..., {arn})`.
   - Set `*.State = plugin.StateIsSet | plugin.StateIsNull` and return `nil` on the
     empty path, matching the existing convention (see `aws_ec2.go:1730`).
4. Each field is a pure function of already-fetched data where possible; only fall
   back to a list-fetch when the sibling collection is not cached.

### Edge cases & correctness

- **Cross-region:** ENIs, subnets, and SGs are regional. Backref scans must filter
  to the same region as the source resource to avoid cross-region false matches.
- **ENIs without instances:** SG→networkInterfaces returns all ENIs; SG→instances
  returns only those with a non-null `instance()`. Document this distinction so
  authors pick the right edge ("is this SG used at all" vs "which instances").
- **IAM instance profile vs role:** an instance references an instance *profile*,
  which contains roles. `role.usedByInstances` must resolve through
  `instance.iamInstanceProfile.iamRoles()` (plural) and match self by ARN.
- **KMS scope:** `encryptedVolumes` is intentionally EBS-only in this pass.
  Expanding to RDS/S3/etc. is future work; the field name is specific so it does
  not overpromise.

## Testing

Each new field gets a unit test in the existing per-service `*_test.go` file,
asserting the backref against mocked describe data:

- SG attached to two ENIs (one with instance, one without) → `networkInterfaces`
  returns 2, `instances` returns 1.
- Subnet with mixed ENIs → correct instance projection.
- Role attached to an instance via instance profile → `usedByInstances` returns it.
- KMS key as a volume's `kmsKeyId` → `encryptedVolumes` returns it.
- Cross-region isolation: a same-named/overlapping resource in another region is
  excluded.

## Validation

A follow-up (not in this spec) rewrites at least one existing high-false-positive
policy check to use the new edges (e.g. the open-SG check → only flag SGs whose
`networkInterfaces` is non-empty), demonstrating the false-positive reduction.

## Rollout

- Additive only — no existing field changes, no breaking changes to policies.
- New fields are lazily resolved, so no cost unless a policy references them.
- Ship behind normal provider release; document the new edges in the AWS provider
  reference docs.
