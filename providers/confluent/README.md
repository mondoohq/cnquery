# Confluent Cloud Provider

The `confluent` provider inventories a Confluent Cloud organization: its
environments, Apache Kafka clusters and how they are reachable, the topic ACLs
and RBAC role bindings that decide who may read and write, the service accounts
and API keys those grants are attached to, the Schema Registry clusters, the
self-managed encryption keys protecting data at rest, and the audit log
destination. Kafka is where sensitive data actually moves, and its access model
drifts quietly, so this is the inventory an access review is written against.

## Prerequisites

- A Confluent Cloud **Cloud API key** with organization-wide read access. The
  `OrganizationAdmin` role covers everything this provider reads; a narrower
  set of reader roles works too, but any environment the key cannot see is
  simply missing from the results.
- To read **topics and ACLs**, a **cluster-scoped Kafka API key** for each
  cluster you want covered. Those two listings are served by the cluster's own
  REST endpoint, which does not accept a Cloud API key.

Create both with the Confluent CLI:

```shell
confluent api-key create --resource cloud                # Cloud API key
confluent api-key create --resource lkc-abc123           # Kafka API key
```

## Authentication

The Cloud API key is required. Pass it as flags or through the environment.

```shell
export CONFLUENT_CLOUD_API_KEY=<KEY>
export CONFLUENT_CLOUD_API_SECRET=<SECRET>
mql shell confluent
```

```shell
mql shell confluent --api-key <KEY> --api-secret <SECRET>
```

The Kafka API key is optional and only needed for `topics` and `acls`:

```shell
export CONFLUENT_KAFKA_API_KEY=<KEY>
export CONFLUENT_KAFKA_API_SECRET=<SECRET>
```

With more than one cluster, set a pair per cluster by appending the cluster ID
in upper case with hyphens replaced by underscores. A per-cluster pair wins over
the connection-wide one:

```shell
export CONFLUENT_KAFKA_API_KEY_LKC_ABC123=<KEY>
export CONFLUENT_KAFKA_API_SECRET_LKC_ABC123=<SECRET>
```

> Querying `topics` or `acls` without a usable Kafka API key fails with an error
> naming the cluster and the variables to set. It deliberately does not return
> an empty list, because "no ACLs" is an answer an audit would act on.

Flags:

- `--api-key` / `--api-secret` - the Cloud API key pair.
- `--kafka-api-key` / `--kafka-api-secret` - a cluster-scoped Kafka API key pair.
- `--base-url` - management API root, for deployments not served from
  `api.confluent.cloud`.

## Usage

```shell
mql shell confluent
```

The provider emits a single asset: the Confluent Cloud organization the API key
belongs to.

## Examples

**Kafka clusters reachable from the internet**

```shell
mql> confluent.kafkaClusters.where(isPublic) { id displayName cloud region clusterType }
```

**Clusters that are not in a customer network**

```shell
mql> confluent.kafkaClusters.where(network == null) { id displayName }
```

**ACL entries that grant every principal access to every topic**

```shell
mql> confluent.kafkaClusters.first.acls.where(
       permission == "ALLOW" && grantsWildcardResource && resourceType == "TOPIC"
     ) { principal operation resourceName patternType }
```

**Service accounts holding an organization-wide role**

```shell
mql> confluent.roleBindings.where(isOrganizationScoped) { principal roleName serviceAccount { displayName } }
```

**API keys older than a year**

```shell
mql> confluent.apiKeys.where(ageDays > 365) { id displayName ageDays ownerKind isCloudKey }
```

**Topics nothing grants access to**

```shell
mql> confluent.kafkaClusters.first.topics.where(acls.length == 0) { name partitionsCount }
```

**Topics that survive fewer than two broker failures**

```shell
mql> confluent.kafkaClusters.first.topics.where(replicationFactor < 3) { name replicationFactor configs["min.insync.replicas"] }
```

**Clusters whose data at rest is protected by a Confluent-managed key**

```shell
mql> confluent.kafkaClusters.where(customerManagedEncryption == false) { id displayName clusterType }
```

**Self-managed keys Confluent can no longer reach**

```shell
mql> confluent.encryptionKeys.where(validationPhase == "INVALID") { id provider validationMessage kafkaClusters { id } }
```

**Human accounts that sign in with a local password rather than through SSO**

```shell
mql> confluent.users.where(authType == "AUTH_TYPE_LOCAL") { email fullName }
```

**Audit log destination**

```shell
mql> confluent.auditLog { enabled topicName cluster { id } serviceAccount { displayName } }
```

## Resources

Every resource and field carries its documentation in
`resources/confluent.lr`, which is the source the resource reference is
generated from. The top-level entry point is `confluent`.

## Verification

```shell
mql> confluent.organizationName
mql> confluent.environments { id displayName }
```

An empty environment list means the Cloud API key holds no read access to any
environment, not that the organization has none. Permission failures surface as
errors rather than as empty results, so an empty list is a real answer about
what the key can see.

## Troubleshooting

**`a Confluent Cloud API key and secret are required`** - neither the flags nor
`CONFLUENT_CLOUD_API_KEY` / `CONFLUENT_CLOUD_API_SECRET` supplied a complete
pair.

**`a cluster-scoped Kafka API key is required to read lkc-...`** - `topics` or
`acls` was queried without a Kafka API key for that cluster. The message names
the exact environment variables that satisfy it.

**`confluent API /iam/v2/role-bindings: 403`** - the Cloud API key lacks the
role needed to list role bindings at the organization scope. Grant it
`OrganizationAdmin`, or a role that can read RBAC bindings.

**A cluster reports `isPublic` as null** - the cluster's response carried no
endpoints map, so nothing said how it is reached. The exposure is reported as
unknown rather than guessed from the endpoint hostname.

**`confluent.auditLog.enabled` is null** - the account endpoint answered without
an audit log block, so the question could not be put. It is deliberately not
reported as `false`, which would read as "this organization is not audited" on
an organization nobody checked. Compare the field explicitly
(`confluent.auditLog.enabled == true`) rather than relying on a block
assertion: in MQL's three-valued logic a null satisfies a bare `{ enabled }`
block.
