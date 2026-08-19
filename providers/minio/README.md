# MinIO Provider

Query the security posture of a [MinIO](https://min.io) object storage deployment:
buckets and how each one is exposed, encrypted, versioned and retained; the users,
groups, service accounts and named policies that can reach them; and the webhook
targets the deployment ships its audit trail and server logs to.

The provider talks to two APIs on the same endpoint: the S3-compatible API for
bucket configuration, and the MinIO admin API for the deployment, identity and
logging configuration.

## Authentication

The provider authenticates with an access key and secret key.

```bash
export MINIO_ENDPOINT=https://minio.example.com:9000
export MINIO_ROOT_USER=minioadmin
export MINIO_ROOT_PASSWORD='<SECRET>'
mql shell minio
```

`MINIO_ACCESS_KEY` and `MINIO_SECRET_KEY` are read when the `MINIO_ROOT_*` pair is
not set. Flags override the environment:

```bash
mql shell minio --endpoint https://minio.example.com:9000 \
  --access-key ACCESS_KEY --secret-key SECRET_KEY
```

| flag | environment | meaning |
|---|---|---|
| `--endpoint` | `MINIO_ENDPOINT` | S3 and admin API endpoint. A bare host is assumed to be HTTPS. |
| `--access-key` | `MINIO_ROOT_USER`, then `MINIO_ACCESS_KEY` | Access key |
| `--secret-key` | `MINIO_ROOT_PASSWORD`, then `MINIO_SECRET_KEY` | Secret key |
| `--region` | `MINIO_REGION` | Region to sign requests for |
| `--ca-cert` | `MINIO_CACERT` | Certificate authority to trust, as a PEM file path or the PEM itself |
| `--tls-skip-verify` | | Skip certificate verification, for lab deployments only |

Supplying `--ca-cert` or `--tls-skip-verify` against an `http://` endpoint is an
error rather than a no-op, so a TLS option cannot be silently ignored on a
connection that carries no TLS.

### Permissions

The key pair needs, on the admin API: `admin:ServerInfo`, `admin:ConfigUpdate`
(the read half), `admin:ListUsers`, `admin:ListGroups`,
`admin:ListServiceAccounts`, `admin:GetPolicy` and `admin:KMSKeyStatus`. On the
S3 API it needs `s3:ListAllMyBuckets` plus the `s3:GetBucket*` and
`s3:GetEncryptionConfiguration` actions on every bucket in scope. The deployment's
root credentials cover all of it.

A key pair that is missing one of these makes the affected field report an error
rather than an empty value, deliberately: an access key that may not read a
setting tells you nothing about what the setting is, and reporting "not
configured" there would turn a missing permission into a clean audit pass.

## Examples

```coffee
# every bucket an unauthenticated caller can reach
minio.buckets.where(hasAnonymousAccess)

# buckets with no default encryption
minio.buckets.where(encrypted == false) { name location }

# buckets that are neither versioned nor object locked
minio.buckets.where(versioningEnabled == false && objectLockEnabled == false) { name }

# which SSE-KMS key each encrypted bucket uses, and whether it is usable
minio.buckets.where(encrypted) { name kmsKey { name healthy encryptionError } }

# policies that grant the admin API, and who holds them
minio.policies.where(grantsAdminAccess) { name users { name } groups { name } }

# service accounts that never expire
minio.serviceAccounts.where(expiresAt == null) { accessKey parentUser { name } }

# disabled users that still hold policies
minio.users.where(enabled == false) { name policies { name } }

# is the audit trail being shipped anywhere, and is the target reachable
minio.auditWebhooks { name endpoint enabled status }

# deployment posture
minio { version deploymentMode backendType tlsEnabled onlineDrives offlineDrives }
```

## Notes on the API

Behaviours worth knowing about, all observed against a running deployment rather
than taken from documentation:

- **`ListBuckets` reports no region.** Every entry comes back with an empty
  `BucketRegion`, so `location` costs one `GetBucketLocation` per bucket.
- **A bucket with no access policy answers with an empty body and no error.**
  An empty `policyDocument` therefore means "no policy", not "could not read".
- **Server-side encryption needs a KMS, even for SSE-S3.** MinIO refuses
  `PutBucketEncryption` with "Server side encryption specified but KMS is not
  configured" until one is attached. `minio.kmsConfigured` reports whether one is.
- **The key management service answers HTTP 200 for a key it does not hold**,
  reporting the failure in `encryption-error`. `minio.kmsKey.healthy` is derived
  from those fields rather than from the request succeeding.
- **The authentication token on a webhook target is withheld.** A target with a
  token configured comes back with the `auth_token` key removed entirely, while
  one without reports it as present and empty. Nothing in the schema derives
  from that, because either spelling gets the answer backwards the moment the
  behaviour changes.
- **Timestamps default to the zero time.** A user record the deployment never
  stamped reports `0001-01-01T00:00:00Z`; the schema reports null instead, so a
  "changed since" comparison cannot be silently satisfied.
- **`ListServiceAccounts` for a user that does not exist succeeds and returns
  nothing.** A typo in a user name is not an error.
- **The admin API is versioned.** madmin-go v4 addresses `/minio/admin/v4/` and
  falls back to `/minio/admin/v3/` when the deployment answers `426 Upgrade
  Required`, which costs one extra request per admin call on older servers.
  Setting `MADMIN_API_VERSION=v3` in the provider's environment skips the
  fallback.

## Secrets

No field in this schema carries key material. The user listing can carry a
`secretKey` and MinIO returns service account secrets in plaintext when one is
created; neither is read. `resources/secrets_test.go` sweeps the whole schema and
the mapping functions to keep it that way, and sweeps the recorded fixtures for
credentials.

`minio.policy.document` and `minio.bucket.policyDocument` return the policy
documents verbatim. A policy that embeds a credential in a condition value would
be reported as written, because the document is the thing being audited.
