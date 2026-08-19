# Rancher Provider

Query the multi-cluster governance layer of a [Rancher
Manager](https://ranchermanager.docs.rancher.com/) install.

## What this provider is for

Rancher sits a level above a Kubernetes cluster. The `k8s` provider audits one
cluster's workloads; this one audits the fleet policy that governs all of them:

- which clusters exist, how each was provisioned, and which is the local cluster
- who can administer the fleet (global roles and their bindings)
- who can administer a cluster or a project (role templates and their bindings),
  and which templates grant wildcard or escalating permissions
- which pod security admission templates exist and which clusters apply them
- which cluster templates exist and whether new clusters must be built from one
- which authentication providers are enabled, and whether local accounts remain
  a way in alongside them
- which API tokens are in circulation, when they were last used, and which never
  expire
- which registry credentials are distributed, and how far

**Workloads are out of scope.** Point the `k8s` provider at a downstream cluster
for pods, deployments, and per-cluster RBAC.

## Authentication

A Rancher API key is a pair: a public access key that names the token object,
and a secret half. Pass the whole thing, or the halves separately.

```bash
export RANCHER_URL=https://rancher.example.com
export RANCHER_TOKEN=token-abcde:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
mql shell rancher
```

```bash
mql shell rancher \
  --url https://rancher.example.com \
  --access-key token-abcde \
  --secret-key xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

| flag | environment variable | meaning |
|---|---|---|
| `--url` | `RANCHER_URL` | Rancher Manager endpoint |
| `--token` | `RANCHER_TOKEN` | whole API token, `access-key:secret-key` |
| `--access-key` | `RANCHER_ACCESS_KEY` | public half of the key |
| `--secret-key` | `RANCHER_SECRET_KEY` | secret half of the key |
| `--ca-cert` | `RANCHER_CACERT` | certificate authority to trust, PEM path or inline PEM |
| `--tls-skip-verify` | | skip certificate verification, lab servers only |

Rancher is very commonly published under a private certificate authority.
Trust it rather than turning verification off:

```bash
mql shell rancher --ca-cert /etc/ssl/certs/rancher-ca.pem
```

## Permissions

A **restricted admin**, or a custom global role granting `get` and `list` on the
`management.cattle.io` resources, is enough. A standard user can connect, but
sees only the clusters and projects they are a member of and only their own
tokens, so a fleet-wide audit run under one will silently under-report. The
`rancher.tokens` doc comment says this too, because a short token list is a
statement about the caller as much as about the server.

## Examples

```coffee
# clusters that never got a pod security admission template
rancher.clusters.where(podSecurityAdmissionTemplate == null) { name provider }

# who can administer the fleet
rancher.globalRoles.where(grantsFullAdmin)
  { name bindings { subjectKind subjectName } }

# custom role templates that can escalate permissions
rancher.roleTemplates.where(builtin == false && grantsPrivilegeEscalation)
  { name context rules }

# API keys that never expire and have never been used
rancher.tokens.where(neverExpires && lastUsedAt == null) { id userId created }

# local accounts still enabled next to single sign-on
rancher.localAuthEnabled && rancher.externalAuthEnabled

# pod security templates nobody applies
rancher.podSecurityAdmissionConfigurationTemplates.where(clusters.length == 0) { name }

# registry credentials pushed into every namespace of a project
rancher.registryCredentials.where(scope == "project") { name registries project.name }
```

## Version differences that change the answers

Rancher's management API is not stable across minor releases, and three of the
things this provider reports moved:

| what | where it stands |
|---|---|
| **cluster templates** | removed in Rancher **2.12** with RKE1. `rancher.clusterTemplates` is empty on 2.12 and newer, and `clusterTemplateEnforcement` is **null** rather than `false`, because there is no control to be off. |
| **`telemetry-opt` setting** | removed after Rancher **2.9**. `rancher.telemetryOptIn` is null on newer servers. |
| **`roleTemplate.administrative`** | Rancher no longer acts on the field. It is kept as `@maturity("deprecated")`; use `grantsFullAdmin` and `grantsPrivilegeEscalation`, which are computed from the rules. |

## Secrets

No field in this provider carries credential material:

- the value of an API token is not decoded anywhere (`tokenRecord` has no field
  for it);
- a user's password is not decoded;
- a registry credential's `password`, `auth` and `email` are not decoded, only
  the registry host and user name;
- a cluster template revision's `clusterConfig` is not modeled at all, because
  it carries the whole cluster specification.

`resources/decode_test.go` proves this by re-marshaling what was decoded from a
fixture that deliberately contains those fields, and by sweeping every wire
record type for a credential-shaped JSON tag.

## Status

This provider has **not been run against a live Rancher server**. It has been
run end to end through the `mql` runtime against a mock server replaying the
fixtures, which proves the plumbing and the null handling but says nothing about
whether the values match a real Rancher. See
[TESTING-TODO.md](TESTING-TODO.md) for what still needs verifying and how.
