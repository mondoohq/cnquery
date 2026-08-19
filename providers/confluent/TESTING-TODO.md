# Confluent Cloud provider: live verification, not yet done

**Status: no field in this provider has ever run against a live Confluent Cloud
organization.** The schema was built from Confluent's own OpenAPI specifications
(`ccloud-sdk-go-v2`: `org/v2`, `cmk/v2`, `iam/v2`, `apikeys/v2`, `mds/v2`,
`srcm/v3`, `networking/v1`, `byok/v1`, `kafkarest/v3`) and the unit tests decode
payloads shaped like the documented responses. A fixture built from the
documentation reproduces the documentation, so the suite being green says
nothing about whether these fields read correctly from a real organization.

This document is the handoff. Work top to bottom. Everything is unchecked.

**The record types now carry Confluent's own generated models for the nested
blocks** (see §8). That narrows what a live run has to check: the field names and
nesting of the adopted blocks are the vendor's rather than ours, so a live run no
longer has to prove them by hand. **It does not move the verification bar.** No
field in this provider has still ever been read from a real organization, and a
vendor-maintained struct tag says nothing about whether the endpoint is reachable
with the credentials we send, whether the value means what we think, or whether
the listing is complete. Everything below stays required.

---

## 1. Credentials and fixtures needed

### Credentials

| Variable | What it is | Needed for |
|---|---|---|
| `CONFLUENT_CLOUD_API_KEY` | Cloud API key ID | everything |
| `CONFLUENT_CLOUD_API_SECRET` | Cloud API key secret | everything |
| `CONFLUENT_KAFKA_API_KEY` | cluster-scoped Kafka API key ID | `topics`, `acls` |
| `CONFLUENT_KAFKA_API_SECRET` | cluster-scoped Kafka API key secret | `topics`, `acls` |
| `CONFLUENT_KAFKA_API_KEY_LKC_<ID>` | per-cluster override | multi-cluster runs |
| `CONFLUENT_KAFKA_API_SECRET_LKC_<ID>` | per-cluster override | multi-cluster runs |

Create them:

```shell
confluent api-key create --resource cloud
confluent api-key create --resource lkc-<CLUSTER_ID>
```

The Cloud API key must hold `OrganizationAdmin`, or the verification of
`roleBindings`, `users`, `apiKeys` and `encryptionKeys` will hit 403s that are
not provider bugs.

### Fixtures that must exist in the organization

- [ ] At least **two environments**, so `confluent.environments` proves it
      paginates and `environment.kafkaClusters` proves it partitions rather
      than returning every cluster.
- [ ] One **Basic or Standard Kafka cluster with a public endpoint**.
- [ ] One **Dedicated or Enterprise cluster reached over private link or a
      private network interface** (this is the expensive one; see §3, it is
      what makes `isPublic` meaningful).
- [ ] One **topic with a wildcard ACL**: `resource_type=TOPIC`,
      `pattern_type=LITERAL`, `resource_name=*`, `permission=ALLOW`.
- [ ] One **topic with a scoped ACL**: `pattern_type=LITERAL`,
      `resource_name=<that topic>`.
- [ ] One **PREFIXED ACL**, to exercise the prefix branch of
      `topic.acls` and `grantsWildcardResource`.
- [ ] One **DENY ACL**, to confirm `permission` is reported and that
      `grantsWildcardResource` is orthogonal to it.
- [ ] One **topic no ACL reaches**, so `topics.where(acls.length == 0)`
      returns something rather than nothing on a healthy fixture.
- [ ] At least one **service account** with a **cluster-scoped API key**.
- [ ] At least one **Cloud API key** (the one you authenticate with counts).
- [ ] One **API key created more than a day ago** and one **created during the
      test run**, for `ageDays`.
- [ ] At least one **role binding at the organization scope** and one **at the
      cluster scope**.
- [ ] A **Schema Registry cluster** in one environment (creating any cluster in
      an environment usually provisions one).
- [ ] Optional but valuable: a **BYOK key** attached to a Dedicated cluster, and
      a **dedicated network**.

Create the ACL fixtures:

```shell
# wildcard grant
confluent kafka acl create --cluster lkc-<ID> --allow --service-account sa-<ID> \
  --operations read --topic '*'
# scoped grant
confluent kafka acl create --cluster lkc-<ID> --allow --service-account sa-<ID> \
  --operations read --topic payments
# prefixed grant
confluent kafka acl create --cluster lkc-<ID> --allow --service-account sa-<ID> \
  --operations write --topic pay --prefix
# deny
confluent kafka acl create --cluster lkc-<ID> --deny --service-account sa-<ID> \
  --operations delete --topic payments
```

---

## 2. Build under test

Do **not** install over the release provider. Build from this branch and point
`PROVIDERS_PATH` at a scratch directory:

```shell
make providers/build/confluent
mkdir -p /tmp/pd/confluent && cp providers/confluent/dist/confluent* /tmp/pd/confluent/
PROVIDERS_PATH=/tmp/pd mql shell confluent
```

`make providers/install/confluent` **copies from `dist/`, it does not build**. A
failed build leaves the previous binary in place and install ships it happily,
so always build first and check the `dist/` mtime before trusting a result that
contradicts the code.

Every query below assumes the `PROVIDERS_PATH=/tmp/pd` prefix.

---

## 3. Two-state requirement

A field that reads the same before and after you change the setting cannot tell
a resource bug from a fixture that never moved. Drive each of these to both
states and read it in both directions. **Do not tick the single-state row for
any of these without the paired reading.**

| Field | State A | State B |
|---|---|---|
| `kafkaCluster.isPublic` | public cluster reports `true` | private-link cluster reports `false` |
| `kafkaCluster.hasPrivateEndpoint` | public-only cluster reports `false` | private cluster reports `true` |
| `kafkaCluster.connectionTypes` | `["PUBLIC"]` | contains `PRIVATE_LINK` or `PRIVATE_NETWORK_INTERFACE` |
| `kafkaCluster.network` | Basic cluster reports `null` | Dedicated cluster in a network reports the network |
| `kafkaCluster.customerManagedEncryption` | Confluent-managed cluster reports `false` | BYOK cluster reports `true` |
| `kafkaCluster.deletionProtection` | off reports `false` | switch it on, re-read, reports `true` |
| `kafkaCluster.cku` / `maxEcku` | Dedicated: `cku` set, `maxEcku` null | Basic: `cku` null, `maxEcku` set or null |
| `acl.grantsWildcardResource` | wildcard ACL reports `true` | scoped ACL reports `false` |
| `acl.grantsAnyPrincipal` | ACL on `User:*` reports `true` | ACL on `User:sa-...` reports `false` |
| `acl.grantsAllOperations` | ACL with `ALL` reports `true` | ACL with `READ` reports `false` |
| `acl.permission` | ALLOW entry | DENY entry |
| `acl.patternType` | LITERAL entry | PREFIXED entry |
| `topic.acls` | topic covered by the wildcard and the scoped ACL returns both | topic covered by nothing returns `[]` |
| `topic.replicationFactor` | 3 on a normal topic | different on an internal topic if one exists |
| `apiKey.ageDays` | key created during the run reports `0` | key created a year ago reports `>= 365` |
| `apiKey.isCloudKey` | Cloud API key reports `true` | cluster-scoped key reports `false` |
| `apiKey.ownerKind` | service-account key reports `ServiceAccount` | user-owned key reports `User` |
| `roleBinding.isOrganizationScoped` | organization binding reports `true` | cluster binding reports `false` |
| `roleBinding.scopeKind` | `organization` | `cloud-cluster` |
| `user.authType` | SSO account reports `AUTH_TYPE_SSO` | local account reports `AUTH_TYPE_LOCAL` |
| `schemaRegistryCluster.isPublic` | registry with a public endpoint reports `true` | private-networking-only registry reports `false` |
| `encryptionKey.state` | unused key reports `AVAILABLE` | key attached to a cluster reports `IN_USE` |
| `environment.streamGovernancePackage` | `ESSENTIALS` environment | `ADVANCED` environment |

---

## 4. Per-field checklist

Every field, the query that reads it, and what to expect. Tick only what you
actually read.

### `confluent` (root)

```
mql> confluent { organizationId organizationName jitEnabled }
```

- [ ] `organizationId` - the organization UUID; matches the `organization=` segment of any `resourceName`
- [ ] `organizationName` - the display name shown in the Confluent Cloud console
- [ ] `jitEnabled` - `true`/`false` on an SSO organization, **null** where the API does not report it (must be null, not false)
- [ ] `environments` - one entry per environment, count matches `confluent environment list`
- [ ] `kafkaClusters` - every cluster across every environment, count matches the sum of `confluent kafka cluster list` per environment
- [ ] `schemaRegistryClusters` - one per environment that has one
- [ ] `networks` - matches `confluent network list` per environment; empty is legitimate
- [ ] `serviceAccounts` - matches `confluent iam service-account list`
- [ ] `users` - matches `confluent iam user list`
- [ ] `apiKeys` - matches `confluent api-key list`
- [ ] `roleBindings` - matches `confluent iam rbac role-binding list --current-user` plus everyone else's
- [ ] `encryptionKeys` - matches `confluent byok list`; empty is legitimate
- [ ] `auditLog` - see the audit log section, this is the highest-risk field

### `confluent.environment`

```
mql> confluent.environments { id displayName resourceName streamGovernancePackage createdAt updatedAt }
```

- [ ] `id` - `env-...`
- [ ] `displayName` - matches the console
- [ ] `resourceName` - `crn://confluent.cloud/organization=<uuid>/environment=env-...`
- [ ] `streamGovernancePackage` - `ESSENTIALS` or `ADVANCED`, **null** when the environment has no package
- [ ] `createdAt` - a real date, not 1 January year 1
- [ ] `updatedAt` - a real date or null
- [ ] `kafkaClusters` - only this environment's clusters (verify with two environments)
- [ ] `schemaRegistryClusters` - only this environment's registry
- [ ] `networks` - only this environment's networks

### `confluent.kafkaCluster`

```
mql> confluent.kafkaClusters {
       id displayName resourceName cloud region availability clusterType
       cku maxEcku zones deletionProtection phase createdAt updatedAt
       bootstrapEndpoint restEndpoint endpoints connectionTypes
       isPublic hasPrivateEndpoint customerManagedEncryption
     }
```

- [ ] `id` - `lkc-...`
- [ ] `displayName`
- [ ] `resourceName` - CRN ending in `cloud-cluster=lkc-...`
- [ ] `cloud` - `AWS`, `GCP` or `AZURE`
- [ ] `region` - matches the console
- [ ] `availability` - `SINGLE_ZONE`, `MULTI_ZONE`, `LOW` or `HIGH`
- [ ] `clusterType` - `Basic`, `Standard`, `Dedicated`, `Enterprise` or `Freight` (**not** the enum name, the `spec.config.kind` value)
- [ ] `cku` - integer on Dedicated, **null** elsewhere (see §3)
- [ ] `maxEcku` - integer on elastic types, **null** on Dedicated
- [ ] `zones` - three entries on a multi-zone Dedicated cluster, empty otherwise
- [ ] `deletionProtection` - see §3
- [ ] `phase` - `PROVISIONED`
- [ ] `createdAt`, `updatedAt` - real dates
- [ ] `bootstrapEndpoint` - the value `confluent kafka cluster describe` shows as the endpoint
- [ ] `restEndpoint` - the value `confluent kafka cluster describe` shows as the REST endpoint; **this is the URL the topic and ACL listings are read from, so if it is wrong those two fail**
- [ ] `endpoints` - a list of dicts with `accessPointId`, `connectionType`, `bootstrapEndpoint`, `httpEndpoint`
- [ ] `connectionTypes` - see §3
- [ ] `isPublic` - see §3
- [ ] `hasPrivateEndpoint` - see §3
- [ ] `customerManagedEncryption` - see §3
- [ ] `environment` - resolves to the owning environment, non-null
- [ ] `network` - see §3
- [ ] `encryptionKey` - the BYOK key on a BYOK cluster, null elsewhere
- [ ] `topics` - see the topic section
- [ ] `acls` - see the ACL section
- [ ] `apiKeys` - only the keys scoped to this cluster, not the Cloud API keys
- [ ] `roleBindings` - only the bindings whose scope names this cluster

### `confluent.topic`

```
mql> confluent.kafkaClusters.first.topics { name isInternal partitionsCount replicationFactor configs }
```

- [ ] `name`
- [ ] `isInternal` - `true` on `__consumer_offsets` style topics if the API returns them, `false` on yours
- [ ] `partitionsCount` - matches `confluent kafka topic describe`
- [ ] `replicationFactor` - `3` on Confluent Cloud
- [ ] `configs` - a map with `min.insync.replicas`, `cleanup.policy`, `retention.ms`; confirm one value against `confluent kafka topic describe`
- [ ] `configs` on a topic with a sensitive setting - the key is present with an empty value, not missing
- [ ] `cluster` - resolves back to the owning cluster
- [ ] `acls` - see §3 (both states)

### `confluent.acl`

```
mql> confluent.kafkaClusters.first.acls {
       principal host resourceType resourceName patternType operation permission
       grantsWildcardResource grantsAnyPrincipal grantsAllOperations
     }
```

- [ ] `principal` - `User:sa-...` or `UserV2:sa-...`; **record which form this organization actually returns** (see §6)
- [ ] `host` - `*`
- [ ] `resourceType` - `TOPIC`, `GROUP`, `CLUSTER`, `TRANSACTIONAL_ID`
- [ ] `resourceName` - the topic or group name, or `*`
- [ ] `patternType` - `LITERAL` and `PREFIXED` (both states, §3)
- [ ] `operation` - `READ`, `WRITE`, `DESCRIBE`, `ALL`
- [ ] `permission` - `ALLOW` and `DENY` (both states, §3)
- [ ] `grantsWildcardResource` - see §3
- [ ] `grantsAnyPrincipal` - see §3
- [ ] `grantsAllOperations` - see §3
- [ ] `cluster` - resolves back to the owning cluster
- [ ] `serviceAccount` - resolves to the named service account; **null** when the principal is a user or a wildcard
- [ ] **Count check:** `confluent.kafkaClusters.first.acls.length` equals `confluent kafka acl list --cluster lkc-<ID> -o json | jq length`. A mismatch by an exact multiple points at the pagination guard; a mismatch by a few points at the `__id` tuple collapsing two entries into one.
- [ ] **Identity check:** create two ACLs that differ only in `operation` (READ and WRITE, same principal, same topic). Both must appear. If only one does, the `__id` is short a dimension.

### `confluent.serviceAccount`

```
mql> confluent.serviceAccounts { id displayName description resourceName createdAt updatedAt }
```

- [ ] `id` - `sa-...`
- [ ] `displayName`, `description`
- [ ] `resourceName` - `crn://confluent.cloud/service-account=sa-...`
- [ ] `createdAt`, `updatedAt`
- [ ] `apiKeys` - only the keys this account owns
- [ ] `roleBindings` - only the bindings held by this account

### `confluent.user`

```
mql> confluent.users { id email fullName authType resourceName createdAt updatedAt }
```

- [ ] `id` - `u-...`
- [ ] `email`, `fullName`
- [ ] `authType` - see §3
- [ ] `resourceName`
- [ ] `createdAt`, `updatedAt`
- [ ] `apiKeys`, `roleBindings` - scoped to this user

### `confluent.apiKey`

```
mql> confluent.apiKeys { id displayName description resourceName createdAt updatedAt ageDays ownerKind isCloudKey resourceKind }
```

- [ ] `id` - the key ID, which is the public half a client presents
- [ ] `displayName`, `description`
- [ ] `resourceName`
- [ ] `createdAt`, `updatedAt`
- [ ] `ageDays` - see §3; must be **null**, never 0, on a key with no creation time
- [ ] `ownerKind` - see §3
- [ ] `isCloudKey` - see §3
- [ ] `resourceKind` - `cmk.v2.Cluster` on a Kafka key, `srcm.v3.Cluster` on a registry key, empty on a Cloud key
- [ ] `serviceAccount` / `user` - exactly one is non-null per key
- [ ] `cluster` - non-null on a Kafka key, null on a Cloud key
- [ ] `schemaRegistryCluster` - non-null on a registry key, null otherwise
- [ ] **Secret sweep, mandatory:** see §5

### `confluent.roleBinding`

```
mql> confluent.roleBindings { id principal roleName crnPattern scopeKind isOrganizationScoped }
```

- [ ] `id` - `rb-...`
- [ ] `principal` - `User:u-...` or `User:sa-...`
- [ ] `roleName` - `OrganizationAdmin`, `CloudClusterAdmin`, `DeveloperRead`, ...
- [ ] `crnPattern` - full CRN
- [ ] `scopeKind` - see §3
- [ ] `isOrganizationScoped` - see §3
- [ ] `serviceAccount` / `user` - resolves the principal; both null on a `Group:` binding
- [ ] `environment` - non-null on an environment or cluster binding, null on an organization binding
- [ ] `cluster` - non-null on a cluster binding
- [ ] **Coverage check:** compare the count against `confluent iam rbac role-binding list --current-user` plus a listing for each other principal. The listing is rooted at the organization CRN as a *partial* match, and if Confluent treats it as an exact match instead the list will only hold the organization-wide bindings. This is a named risk in §6.

### `confluent.schemaRegistryCluster`

```
mql> confluent.schemaRegistryClusters {
       id displayName resourceName streamGovernancePackage cloud region
       httpEndpoint catalogHttpEndpoint privateHttpEndpoint privateRegionalEndpoints
       isPublic phase createdAt updatedAt
     }
```

- [ ] `id` - `lsrc-...`
- [ ] `displayName`
- [ ] `resourceName`
- [ ] `streamGovernancePackage` - `ESSENTIALS` or `ADVANCED`
- [ ] `cloud`, `region`
- [ ] `httpEndpoint` - matches `confluent schema-registry cluster describe`
- [ ] `catalogHttpEndpoint` - **may be empty; confirm the field name against a real response** (§6)
- [ ] `privateHttpEndpoint` - empty on a public registry
- [ ] `privateRegionalEndpoints` - empty map on a public registry
- [ ] `isPublic` - see §3
- [ ] `phase` - `PROVISIONED`
- [ ] `createdAt`, `updatedAt`
- [ ] `environment` - resolves to the owning environment

### `confluent.network`

```
mql> confluent.networks {
       id displayName resourceName cloud region cidr zones
       connectionTypes activeConnectionTypes supportedConnectionTypes
       dnsDomain dnsResolution phase createdAt
     }
```

- [ ] `id` - `n-...`
- [ ] `displayName`, `resourceName`
- [ ] `cloud`, `region`
- [ ] `cidr` - `/16` on a peering or transit gateway network, empty on private link
- [ ] `zones` - three entries
- [ ] `connectionTypes` - `PEERING`, `TRANSITGATEWAY`, `PRIVATELINK`
- [ ] `activeConnectionTypes` - subset of the above
- [ ] `supportedConnectionTypes`
- [ ] `dnsDomain` - populated on a private link network
- [ ] `dnsResolution` - `PRIVATE` or `CHASED_PRIVATE`, empty when the network has no DNS config
- [ ] `phase` - `READY`
- [ ] `createdAt`
- [ ] `environment` - resolves to the owning environment
- [ ] `kafkaClusters` - the clusters placed in this network, `[]` on an unused network

### `confluent.encryptionKey`

```
mql> confluent.encryptionKeys {
       id displayName resourceName provider state keyReference
       keyVaultId tenantId applicationId securityGroup roles
       validationPhase validationMessage validationSince createdAt
     }
```

- [ ] `id` - `cck-...`
- [ ] `displayName`, `resourceName`
- [ ] `provider` - `AWS`, `Azure` or `GCP` (note the mixed case Confluent uses)
- [ ] `state` - see §3
- [ ] `keyReference` - the KMS ARN on AWS, the key object URL on Azure, the resource path on GCP
- [ ] `keyVaultId`, `tenantId`, `applicationId` - populated on Azure only
- [ ] `securityGroup` - populated on GCP only
- [ ] `roles` - IAM role ARNs on AWS only
- [ ] `validationPhase` - `VALID` on a working key
- [ ] `validationMessage`, `validationSince`
- [ ] `createdAt`
- [ ] `kafkaClusters` - the clusters protected by the key, `[]` on an unattached key

### `confluent.auditLogConfig`

```
mql> confluent.auditLog { enabled topicName cluster { id } environment { id } serviceAccount { id displayName } }
```

This field is **null-when-unknown by construction**. `enabled` is `true` or
`false` only when the account endpoint returned an audit log block; a payload
without one leaves it null, and a failed read (401, 403, 404, 5xx, or an
unparseable body) errors rather than answering. Confirm both halves of that
below, because null is only trustworthy if it is reached for the right reason.

- [ ] `enabled` - `true` on an organization with audit logging, `false` only where the endpoint affirmatively reports no writing service account
- [ ] `enabled` - **null**, never `false`, on an organization whose payload carries no audit log block
- [ ] `topicName` - `confluent-audit-log-events`; null whenever `enabled` is null
- [ ] `cluster` - the audit log cluster, or **null** if the API key cannot list it
- [ ] `environment` - the audit log environment, or null
- [ ] `serviceAccount` - the writing account, or null
- [ ] **Cross-check against `confluent audit-log describe`.** Every one of the four values must match. If the query errors instead, read §6.1 first, this is the most likely thing in the provider to be wrong.
- [ ] **On an organization you know is audited, `enabled` must be `true`, not null.** A null there means the payload shape is not what this provider decodes, and §6.1 has to be resolved before the field is usable.
- [ ] **On an organization you know is not audited, `enabled` must be `false`, not null.** A null there means the endpoint does not report the disabled state the way the CLI implies.

---

## 5. The four checks

### 5.1 The absent case must FAIL, not pass vacuously

A check written against a field that is null must not be satisfied by the null.
Run each of these and confirm it **errors or reports false** rather than
quietly passing:

- [ ] Against an organization with **no networks**:
      `confluent.networks.all(dnsResolution == "PRIVATE")` -
      an empty list makes `all` vacuously true, which is correct MQL, but
      `confluent.networks.length > 0` must be the guard in any real policy.
      Confirm `confluent.networks` is `[]` and not an error.
- [ ] Against a cluster with **no Kafka API key configured**:
      `confluent.kafkaClusters.first.acls` must **error** with a message naming
      the cluster and `CONFLUENT_KAFKA_API_KEY_LKC_...`. It must **not** return
      `[]`. An empty ACL list is the exact answer a "no wildcard grants" policy
      is looking for, so this is the single most important negative check here.
- [ ] Same for `confluent.kafkaClusters.first.topics`.
- [ ] Same for `confluent.kafkaClusters.first.topics.first.configs`.
- [ ] With a **Cloud API key that lacks RBAC read**: `confluent.roleBindings`
      must error with a 403, not return `[]`.
- [ ] With a **revoked or wrong secret**: every listing must error. Confirm no
      field silently degrades to null.
- [ ] Kill network access mid-run (or point `--base-url` at a closed port):
      confirm the failure surfaces as an error and is not classified as
      "absent". The unit tests pin this at the classifier level; confirm it end
      to end.

### 5.2 Null over invented defaults

- [ ] `jitEnabled` is **null**, not `false`, on an organization that does not
      report it.
- [ ] `cku` is **null**, not `0`, on a Basic cluster.
- [ ] `maxEcku` is **null**, not `0`, on a Dedicated cluster.
- [ ] `ageDays` is **null**, not `0`, if any key comes back without a creation
      time.
- [ ] `createdAt` / `updatedAt` are **null**, never `0001-01-01`, wherever the
      API omits them.
- [ ] `isPublic`, `hasPrivateEndpoint` and `connectionTypes` are **null**, not
      `false`/`[]`, on any cluster whose response carries no endpoints map.
      **Record whether this actually happens on a live organization** - if
      every cluster carries the map, say so, because that retires a risk in §6.
- [ ] `streamGovernancePackage` is null, not `""`, on an environment with no
      package.
- [ ] `auditLog.enabled` is **null**, not `false`, whenever the account endpoint
      answered without an audit log block, and `auditLog.topicName` is null
      alongside it. A `false` here must always be traceable to the endpoint
      saying auditing is off.
- [ ] A failed read of the account endpoint (401, 403, 404, 5xx, or a body that
      is not JSON) **errors**. It must never produce a resource at all, let
      alone one reading `enabled: false`.

### 5.3 Secret sweep must return 0

The Confluent API never returns an API key secret outside the create call, and
no field in this schema is meant to carry one. Prove it:

```shell
# with the real secrets in the environment
PROVIDERS_PATH=/tmp/pd mql run confluent -c "confluent.apiKeys" -j \
  | grep -c "$CONFLUENT_CLOUD_API_SECRET"          # must be 0
PROVIDERS_PATH=/tmp/pd mql run confluent -c "confluent.apiKeys" -j \
  | grep -c "$CONFLUENT_KAFKA_API_SECRET"          # must be 0
```

- [ ] Both counts are 0.
- [ ] Repeat over a full dump of every resource:
      `mql run confluent -c "confluent { environments kafkaClusters serviceAccounts users apiKeys roleBindings encryptionKeys networks schemaRegistryClusters }" -j | grep -c "<secret>"` is 0.
- [ ] Confirm `confluent.apiKeys` output contains the key **ID** and no field
      whose value looks like a 64-character secret.
- [ ] Confirm `topic.configs` does not carry a value for any entry the broker
      marks sensitive (the value must be the empty string, and the key must
      still be listed).

The precise guarantee to state afterwards: *no field of this provider exposes
an API key secret*. That is true. It is not the same as "the secret cannot be
obtained", which is a claim about Confluent, not about this code.

### 5.4 Composability

- [ ] `confluent.environments { kafkaClusters { acls { serviceAccount { displayName } } } }`
      walks all four levels and returns names rather than nulls.
- [ ] `confluent.kafkaClusters { topics { acls { principal } } }` returns the
      right ACLs per topic, not the whole cluster's list on every topic.
- [ ] `confluent.serviceAccounts { apiKeys { cluster { displayName } } }` -
      the round trip principal to key to cluster resolves.
- [ ] `confluent.encryptionKeys { kafkaClusters { environment { displayName } } }` -
      the reverse edge resolves and does not loop.
- [ ] `confluent.networks { kafkaClusters { id } }` - the reverse edge is
      populated for a network that has clusters and `[]` for one that does not.
- [ ] **Call-count check:** run a query that walks from many children back to
      one parent (`confluent.kafkaClusters { acls { cluster { displayName } } }`)
      and confirm it does not take time proportional to the ACL count. Every
      resolver reads from the root resource's cached listing, so this should be
      flat. If it is not, an accessor is going through `NewResource` somewhere.

---

## 6. Risk areas

**This is the most important section.** Everything below was inferred from
Confluent's published OpenAPI specifications or from the Confluent CLI source,
not observed against a live organization. They are ordered by how likely they
are to be wrong and how badly a wrong answer would mislead.

### 6.1 The audit log endpoint is a guess (highest risk) — STILL OPEN

**The SDK observes nothing here, and confirms there is nothing to observe.** All
~45 module directories of `ccloud-sdk-go-v2` were enumerated and **none of them
models audit log configuration**: there is no audit-log API, versioned or
otherwise, and no model for the `/api/me` payload. That is corroboration for the
decision to hand-roll this read, and it is not evidence that the read works. The
whole of this risk stands exactly as written.

`confluent.auditLog` reads `GET https://api.confluent.cloud/api/me` and expects
an `organization.audit_log` block with `cluster_id`, `account_id`,
`topic_name`, `service_account_id` and `service_account_resource_id`.

What is known: `confluent audit-log describe` reads exactly those five values
out of the organization the CLI's auth client returns, and that client calls
`/api/me`. What is **not** known:

- whether `/api/me` accepts **Cloud API key basic auth** at all. The CLI
  authenticates it with a session JWT obtained at login. If the endpoint only
  accepts a JWT, `confluent.auditLog` will fail with a 401 on every
  organization.
- whether the JSON key is `audit_log`, and whether the organization block sits
  at the top level or under `user`. Both positions are decoded; anywhere else
  and the field reads null.
- whether `account_id` carries `env-...` on a modern organization. Only an
  `env-` prefixed value is resolved to an environment; anything else leaves
  `environment` null.
- `/api/me` is not part of the published, versioned API surface. It can change
  without a deprecation.

**How the unknown is reported.** The field is null-when-unknown, not
false-when-unknown, and that is load-bearing rather than cosmetic:

- a failed read (401, 403, 404, 5xx, or a body that is not JSON) **errors**;
  a permission failure is not evidence of absence,
- a successful read whose payload carries no audit log block reports `enabled`
  and `topicName` as **null**,
- `false` is reported only when the endpoint returns an audit log block with no
  writing service account, which is how the CLI itself reads "audit logs are not
  enabled for this organization".

`enabled: false` on an audited organization would be a clean pass on something
nobody checked, invisible to every test, so no code path produces one from an
unknown. `resources/auditlog_test.go` pins all three directions against an
`httptest` server.

**Do not read null as reassurance.** In MQL's three-valued logic `null && null`
is `true`, so a `confluent.auditLog { enabled }` block assertion can pass
vacuously on a null. Any policy written against this field must compare
explicitly (`enabled == true`) rather than rely on the block form. Say so
wherever the field is used until §6.1 is retired.

**What to do:** run `confluent.auditLog` and `confluent audit-log describe` side
by side.

- If the values match, the risk retires and the field is trustworthy.
- If the query 401s, the endpoint does not accept Cloud API key auth. The field
  then errors on every scan, which is honest but useless. There is no versioned
  management API for audit log configuration at the time this was written, so
  the right move is likely to **drop the resource and document the gap** rather
  than ship a field that always errors.
- If the query returns null on an organization the CLI reports as audited, the
  payload shape differs from what is decoded here. Capture the raw
  `/api/me` body and fix the decode before the field ships.

### 6.2 Topics and ACLs use the per-cluster REST API, not the management API

**Half struck.** The *record shape* half is retired: `topicRecord` and
`topicConfigRecord` are now aliases of `kafkarest/v3`'s own `TopicData` and
`TopicConfigData`, so the field names, the nesting and the null-vs-empty
handling of `value` are Confluent's rather than ours and need no hand
verification. `aclRecord` deliberately stays local (see N2 in §8), but its
fields are eight flat strings with nothing to mistype.

**Still open, and unchanged:** the pagination question and the principal-format
question below. Both are server behaviours that no struct can settle, and the
pagination one is still the check most likely to find a real bug.

`topics`, `topic.configs` and `acls` are read from
`<restEndpoint>/kafka/v3/clusters/<id>/{topics,acls}` with a cluster-scoped
Kafka API key. This is the v3 Kafka REST shape from `kafkarest/v3`, which is
also used by Confluent Platform's self-hosted REST proxy, and the Cloud variant
may differ:

- **`restEndpoint` derivation.** It is taken from the `spec.endpoints` map,
  preferring the entry whose `connection_type` is `PUBLIC`, and falling back to
  the deprecated `spec.http_endpoint` when the map is absent. If Cloud returns
  the map keyed differently, or the public entry is not keyed `PUBLIC`, the
  wrong endpoint is used and both listings fail or read from the wrong place.
  **Verify `restEndpoint` against `confluent kafka cluster describe` before
  trusting anything downstream of it.**
- **Pagination.** The walk follows `metadata.next` as an absolute URL. Kafka
  REST v3 documents `next` as nullable, but whether Cloud ever populates it for
  topics and ACLs is unverified. If it does not, a cluster with more topics than
  one page holds silently reports a short list. **Create more than 100 ACLs on
  one cluster and confirm the count.** This is the check most likely to find a
  real bug.
- **`page_size` is deliberately not sent** to these endpoints, because the
  Kafka REST spec documents none. If Cloud requires one to page at all, the
  above check will show it.
- **Principal format.** The spec notes both a legacy `User:<numeric>` form and a
  `UserV2:sa-...` form, and says `UserV2:*` must be used on the *filter* to get
  the new format back. This provider sends no principal filter, so it gets
  whatever the endpoint defaults to. If ACLs come back with numeric principals,
  `acl.serviceAccount` will be null everywhere and `principalAccountID` needs a
  numeric-to-`sa-` mapping. **Record which form your organization returns.**

### 6.3 Role binding listing scope — STILL OPEN

**The SDK corroborates our reading of the spec and observes nothing.**
`mds/v2`'s generated client takes `crn_pattern` as a plain query string with the
same "partial search" description we read it from, and its `IamV2RoleBinding`
model is now what this provider decodes. Neither says how the server treats the
pattern. Whether a partial match really returns bindings *below* the
organization is a server behaviour, and it is the one that decides whether an
access review sees every grant or only the organization-wide ones.

`confluent.roleBindings` calls `/iam/v2/role-bindings` with
`crn_pattern=crn://confluent.cloud/organization=<orgId>`. The spec describes
that parameter as "a partial search of crn_pattern", which should return every
binding at or below the organization. If Confluent instead treats it as a
prefix that must match exactly, the result holds only the organization-wide
bindings and every environment- and cluster-scoped binding is **missing without
an error**. That is an under-report of exactly the grants an access review
cares about.

**Verify by count** against the CLI, per §4.

### 6.4 The `spec.endpoints` map may not be universally present — STILL OPEN

**The SDK corroborates our reading of the spec and observes nothing.**
`CmkV2ClusterSpec` types `Endpoints` as an optional `*ModelMap` and marks
`kafka_bootstrap_endpoint`, `http_endpoint` and `api_endpoint` as `Deprecated`
in favour of it. That confirms the map is the current shape and the pair is the
legacy one, which is exactly how this provider reads them. It says nothing about
how many live clusters still answer without the map, which is the only thing
that decides whether `isPublic` is usable in a policy.

The endpoints map is a newer addition to `cmk/v2`. Where it is absent this
provider falls back to the deprecated `spec.kafka_bootstrap_endpoint` /
`spec.http_endpoint` and reports `isPublic`, `hasPrivateEndpoint` and
`connectionTypes` as **null** rather than guessing from the hostname. Null is
the honest answer but it is also a field a policy cannot use.

**If every live cluster carries the map, say so and this risk retires.** If some
do not, decide whether a hostname heuristic is acceptable (it is not, in this
author's view, because `*.glb.confluent.cloud` is used for both public and
private-link endpoints) or whether the fallback should be dropped.

### 6.5 Schema Registry field names ~~(struck)~~

**Struck.** `schemaRegistryRecord` now carries `srcmv3.SrcmV3ClusterSpec` and
`srcmv3.SrcmV3ClusterStatus` directly, so `catalog_http_endpoint`,
`private_http_endpoint` and `private_networking_config.regional_endpoints` are
spelled by Confluent's own generated model rather than transcribed by us. There
is no longer a field name here that we could have got wrong.

Note the narrower thing this does *not* settle: whether `catalogHttpEndpoint` is
ever populated on a live response. A correct tag on an always-empty field still
reads empty. That is a question for the live run, not a schema risk.

**The 403 question below is NOT struck**, because it is transport rather than
shape. What `srcm/v3` answers for an environment that has no Schema Registry
remains unverified. An empty list and a 404 are both handled (the 404 skips that
environment and the walk continues), but a **403** is not, and would fail the
whole listing. If a bare environment answers 403 rather than 404, that branch
needs adding, and only after confirming the 403 really means "there is nothing
here" rather than "you may not look".

**Test it with an environment that has no Kafka cluster at all**, which is the
cheapest way to produce an environment without a registry.

### 6.6 The `/byok/v1/keys` listing is organization-wide and unfiltered ~~(struck)~~

**Struck.** `byok/v1`'s own generated client exposes the keys listing as a
single organization-wide call taking no environment parameter, which is the
shape this provider already uses. The vendor's model agrees that BYOK keys are
not environment-scoped, so there is no missing filter to add.

What that leaves is only whether the caller's key may read them at all, which is
a permissions question covered by the `OrganizationAdmin` note in §1 rather than
a modelling risk.

### 6.7 `cku` prefers the status over the spec

`kafkaCluster.cku` reports `status.cku` when present and falls back to
`spec.config.cku`. On a cluster mid-resize the two differ, and this reports the
provisioned count rather than the requested one. That is the intended reading
but it has not been observed. Confirm on a cluster that is not resizing that
both agree.

### 6.8 Pagination on the management API ~~(struck as a modelling risk)~~

**Struck as a modelling risk.** Every `*List` model in the nine adopted modules
carries the same `ListMeta` with a nullable `next`, and every list endpoint
returns exactly one page: the generated clients expose `PageSize` and
`PageToken` as request parameters and hand back a single response, with no walk
of their own. There is therefore no vendor-supplied pagination we could have
diverged from, and nothing to inherit. The envelope this provider decodes is the
one the vendor documents.

**The guards stay ours and stay necessary** (see §8). The seen-set, the page cap,
the cursor-cycle check and the foreign-host check have no counterpart in the SDK.

The live checks below are **still required**, because they are about what the
server sends rather than how it is typed:

- whether `metadata.next` is populated at all on `org/v2`, `iam/v2` and `byok/v1`
- whether the returned URL preserves the `environment=` filter for `cmk/v2`,
  `srcm/v3` and `networking/v1`. **If it does not, page two of a per-environment
  listing would return clusters from every environment**, and the `environment`
  accessor would attribute them to the wrong environment via the fallback.

Lowering `pageSize` in `connection/client.go` to 1 temporarily is still the
cheapest way to exercise every paginated walk against a small organization, and
is still strongly recommended.

### 6.9 Organization identification

`FetchOrganization` takes the **first** entry of `/org/v2/organizations`. A
Cloud API key belongs to exactly one organization, so this should always be
right, but if a key can somehow see several the asset would be named after an
arbitrary one. Confirm the listing holds exactly one entry.

### 6.10 `jit_enabled` is early access

`org.v2.Organization.jit_enabled` is documented as "Available for early access
only". It is modelled as null-when-absent, which is correct, but the field may
never be populated. If it is null everywhere, consider whether it earns its
place in the schema.

---

## 7. When this is done

- [ ] Every box above is ticked or has a written note saying why it could not be.
- [ ] Each §6 risk is either **retired** (observed correct) or **converted into
      an issue** with what was seen.
- [ ] Any field that could not be verified is named explicitly in the PR body as
      unverified. "No live organization was available" is a blocker, not a
      disclaimer to merge past.
- [ ] Nothing in §8's "why these stay local" table has been quietly adopted on
      the grounds that the SDK "has one too".

---

## 8. SDK model adoption: hazards and what stays local

The record types carry Confluent's generated models for the nested blocks, from
nine modules pinned at `org` v0.14.0, `cmk` v0.27.0, `iam` v0.17.0, `apikeys`
v0.4.0, `mds` v0.4.0, `srcm` v0.7.3, `networking` v0.14.0, `byok` v0.0.9 and
`kafkarest` v0.25.0. All nine are `v0.x` and Confluent publishes no compatibility
guarantee for them, so they are pinned exactly and a bump is a review, not a
routine dependency update.

**What was bought:** vendor-maintained struct tags on the deep, easily-mistyped
blocks. Nothing else. The transport, the pagination guards, the error
classifier, the timestamp decoder and the `/api/me` read are all still ours, on
purpose, and the rest of this section says why so that nobody later "completes"
the swap and reintroduces what it deliberately avoided.

### The three hazards, and the mitigation each one has

**N1 — a discriminated union that blanks a cluster instead of failing.**
`cmk@v0.27.0/v2/model_cmk_v2_cluster_spec_config_one_of.go` switches
`UnmarshalJSON` on the `kind` discriminator across ten cases (five tier names,
each also in `cmk.v2.`-prefixed form) and, matching none of them, falls off the
end with a bare `return nil` — every variant nil, no error. A tier Confluent
adds after this SDK release would silently blank `clusterType`, `cku`,
`maxEcku` and `zones` on every cluster of that tier.

*Mitigation:* `kafkaClusterRecord.UnmarshalJSON` keeps a sidecar decode of the
same `spec.config` bytes (`clusterConfigRaw`). `clusterConfigOf` reads the shape
from the matched variant and falls back **wholly to the sidecar** when nothing
matched, logging a warning naming the unrecognized kind. Pinned by
`TestDecodeKafkaClusterUnknownTier`, which fabricates a `"Quantum"` tier and
asserts the union matched nothing *and* that all four values still read.

**N1b — the same defect in `byok`, found by reading rather than briefed.**
`byok@v0.0.9/v1/model_byok_v1_key_key_one_of.go:138` is the identical bare
`return nil` across AwsKey / AzureKey / GcpKey. Adopting it unguarded would
blank `keyReference` on a self-managed key from a fourth cloud, and would have
regressed behaviour this provider already promised in a comment and a test.

*Mitigation:* identical. `encryptionKeyRecord.UnmarshalJSON` keeps a
`keySidecar`, `encryptionKeyDetailOf` falls back to it with a warning. Pinned by
`TestDecodeEncryptionKeys/an unrecognized kind still reports its reference`.

**N2 — an enum that fails a whole cluster's ACL listing.**
`kafkarest@v0.25.0/v3/model_acl_resource_type.go` admits exactly seven values
(`UNKNOWN`, `ANY`, `TOPIC`, `GROUP`, `CLUSTER`, `TRANSACTIONAL_ID`,
`DELEGATION_TOKEN`) and returns an error for anything else. **Kafka's own `USER`
resource type is not among them.** A single such entry would fail the decode of
the page it sits on, and the listing would report a cluster as having no access
control entries at all.

*Mitigation:* `AclData` and `AclResourceType` are **not adopted**. `aclRecord`
stays local and `ResourceType` stays a `string`. Pinned by
`TestDecodeAclUserResourceType`, which decodes a two-entry page containing a
`USER` ACL and asserts both survive — and which also asserts that the SDK enum
*does* reject `"USER"`, so the day that stops being true the test says so.

**N3 — every SDK scalar is `*T` with `omitempty`.** A naive port would read the
pointers directly and turn today's `""` into `null` across roughly forty fields,
which is a behaviour change dressed as a type change. Every adopted scalar is
therefore read through its `GetX()` accessor, which yields the zero value for an
absent field. The `GetXOk()` / pointer form is used only where null-when-absent
is the deliberate reading: `cku`, `maxEcku`, `status.cku`, `jitEnabled`, a
topic configuration's sensitive `value`, and the three endpoint-derived exposure
fields.

`cku` deserves its own note. `CmkV2Dedicated` types it as a **value**
`int32`, so the union genuinely cannot distinguish a cluster that reported no
cku from one that reported zero. That is why `clusterConfigOf` takes `cku` from
the sidecar on any decoded record, and from the variant only for a record built
in code. Pinned by `TestClusterCku/a decoded cluster with no cku reports none`.

### A fourth hazard, found while testing: the SDK's own secret scrub panics

`IamV2ApiKeySpec.Redact()` is the vendor's answer to the `secret` field, and it
**cannot be used**. It calls `recurseRedact(o.Resource)` with no nil check; a
typed-nil `*ObjectReference` satisfies the redactor interface, and
`ObjectReference.Redact()` then dereferences a nil receiver. A Cloud API key
carries `"resource": null` by definition, so `Redact()` panics on the most
common key in any organization — and a panic in a decode takes the whole scan
down, not one field.

`apiKeyRecord.UnmarshalJSON` clears `Spec.Secret` directly instead, which is
everything `Redact()` does that matters (every other value it walks is a
`*string` implementing nothing). `TestSdkRedactPanicsOnACloudApiKey` pins the
panic so that a future SDK release fixing it is visible rather than assumed.

### Why these stay local

| kept | instead of | because |
|---|---|---|
| the `/api/me` read and `auditLogRecord` | nothing | **no module in the SDK models audit log configuration.** All ~45 directories were enumerated. The null-when-undetermined behaviour from `e93b17e` is untouched: an absent block reports `enabled` as null, never `false`, because `enabled: false` on an audited organization is a clean pass on something nobody checked |
| the four pagination guards in `GetPaged` | nothing | the generated clients return **one page** and expose `PageSize`/`PageToken` as request parameters. There is no walk to inherit, so the seen-set, the page cap, the cursor-cycle check and the foreign-host check (which stops credentials being sent to a host the walk did not start on) have no vendor counterpart |
| `APIError`, `IsForbidden`, `IsNotFound`, `StatusCode` | the SDK's `GenericOpenAPIError` | classification is on the status the server returned, never on error text, so a transport failure is never mistaken for a definitive answer about absence or permission. A 403 says the caller may not look; only a 404 says the object is not there |
| `objectMeta` + `confluentTime` | `ObjectMeta` in every module | the generated metadata types time all three timestamps as `*time.Time`, which **fails the decode of the entire object** on a value Go cannot parse — one timestamp taking every other field of the record with it. `confluentTime` warns and reports null. Pinned by `TestSdkObjectMetaFailsWhereConfluentTimeDoesNot`, which asserts four SDK metadata types error on `"yesterday"` where the local one survives |
| `encryptionKeyValidationRecord` | `ByokV1KeyValidation` | its `Since` is a **bare `time.Time`**, not even a pointer, so an absent validation timestamp decodes to the zero time and reports 1 January year 1 as the moment the key was last checked. That is an invented value where the schema promises null |
| `networkStatusRecord` | `NetworkingV1NetworkStatus` | it carries `idle_since` as `*time.Time`. This provider exposes no such field, and adopting the block would still let a timestamp it never reads fail the decode of every network in the listing |
| `aclRecord` | `AclData` | N2 above |
| `OrganizationRecord`, `serviceAccountRecord`, `userRecord`, `roleBindingRecord` | the matching SDK models | these are flat scalars plus a metadata block. The only thing adoption would change is the metadata, which is the one part being kept, so it would buy nothing and cost the timestamp guarantee |

### If you are here to "finish" the adoption

Every row above is a deliberate exclusion with a test behind it. Before removing
one, make the corresponding test fail on purpose and read what it says. The
adoption is finished; what remains local is the part that adopting would have
broken.
