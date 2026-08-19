# MinIO provider: live verification TODO

**Status: NOT VERIFIED against a live target through `mql`.** Every field in this
provider was written against payloads captured from a real MinIO server (see
`resources/testdata/README.md` for the provenance of each fixture) and is covered
by unit tests, but no field has been read back through the `mql` query path. This
document is the handoff for that work. It assumes no prior context.

Do not merge this branch on the strength of the unit tests. They prove the
parsers and the mappings; they cannot prove that a field resolves at all through
the runtime, which is the failure mode that stays green in every automated test
(`new-resource` §5: "on a resource created from a path alone they never were --
every one returned null with *provider returned no data and no error*, while the
build and the parser tests stayed green").

---

## 1. Standing the target up

`dl.min.io` is egress-blocked from the container this branch was developed in, and
`docker pull minio/minio` fails: the Docker Hub manifest resolves but the blob CDN
(`production.cloudfront.docker.com`) answers `403 Forbidden` through the egress
proxy. **Do not spend time on either.**

What does work, and is what produced every fixture in `resources/testdata/`:
building the server from source through the Go module proxy.

### 1a. Docker daemon (only needed if you take the container route)

The daemon is not running at session start but the binary is present:

```bash
nohup dockerd > /tmp/dockerd.log 2>&1 &
sleep 8 && docker info      # this worked; the image pull is what fails
```

### 1b. Build MinIO from source (the route that worked)

```bash
mkdir -p /tmp/miniobin
cd /tmp && GOBIN=/tmp/miniobin go install github.com/minio/minio@latest
/tmp/miniobin/minio --version
# -> minio version DEVELOPMENT.GOGET / Runtime: go1.24.7 linux/amd64
```

Takes roughly 6 minutes on a cold module cache.

### 1c. Run it

Four drives in one erasure set. **This matters**: versioning, object locking and
replication are unavailable on a single-drive (FS backend) deployment, so a
one-drive server cannot exercise most of the bucket schema.

`MINIO_CI_CD=1` is required here because `/tmp` sits on the root filesystem and
MinIO otherwise refuses the drives with `drive not found`.

`MINIO_KMS_SECRET_KEY` is required for **any** bucket encryption, SSE-S3 included:
without it `SetBucketEncryption` fails with "Server side encryption specified but
KMS is not configured". This contradicts the common assumption that SSE-S3 works
out of the box.

```bash
mkdir -p /tmp/mdata/{1,2,3,4}
cd /tmp && MINIO_CI_CD=1 \
  MINIO_ROOT_USER=minioadmin \
  MINIO_ROOT_PASSWORD=minioadmin123 \
  MINIO_REGION=us-east-1 \
  MINIO_KMS_SECRET_KEY='minio-default-key:Jr8EBqxOKo3fUJerHzTS3Yz4N3dRmdUDkAAjSkJeMgA=' \
  nohup /tmp/miniobin/minio server /tmp/mdata/1 /tmp/mdata/2 /tmp/mdata/3 /tmp/mdata/4 \
  --address :9000 --console-address :9001 > /tmp/minio-server.log 2>&1 &
sleep 8 && tail -5 /tmp/minio-server.log
```

The KMS key above is a throwaway generated for the fixture run. Generate your own
with `head -c 32 /dev/urandom | base64`.

**Client calls must bypass the egress proxy**, or every request to 127.0.0.1 is
sent to the proxy and fails:

```bash
env -u HTTPS_PROXY -u HTTP_PROXY -u https_proxy -u http_proxy NO_PROXY='*' <command>
```

### 1d. Load the fixture data

The Go program that created the fixture deployment is not committed (it was
scratch), so recreate it. Either write the equivalent with `minio-go` +
`madmin-go`, or use `mc`. The state it must reach is in §3 below, because the
two-state requirement is what the whole checklist depends on.

`mc` is easiest if it can be obtained (`go install github.com/minio/mc@latest`
through the module proxy; the `dl.min.io` binary is blocked):

```bash
mc alias set fx http://127.0.0.1:9000 minioadmin minioadmin123
```

---

## 2. Build the provider under test

`make providers/install/minio` **copies from `dist/`, it does not build**. A failed
or skipped build leaves the previous binary in place and install ships it, which
looks exactly like "the change did not work". Always build first, and use
`PROVIDERS_PATH` so the installed provider set is not clobbered:

```bash
cd <repo root>
make providers/build/minio
ls -la providers/minio/dist/minio       # check the mtime is from this build
mkdir -p /tmp/pd/minio && cp providers/minio/dist/minio* /tmp/pd/minio/

export MINIO_ENDPOINT=http://127.0.0.1:9000
export MINIO_ROOT_USER=minioadmin
export MINIO_ROOT_PASSWORD=minioadmin123

env -u HTTPS_PROXY -u HTTP_PROXY NO_PROXY='*' \
  PROVIDERS_PATH=/tmp/pd mql run minio -c "minio.buckets { name }"
```

If a result looks wrong, rebuild before concluding anything. `strings` on the
binary will not settle it (Go concatenates literals); use a control instead, for
example a field you know changed alongside the one you are checking.

---

## 3. Two-state fixture requirement

**A field that reads the same in both states cannot distinguish a resource bug
from a fixture that never moved.** Every row of the checklist in §4 names the
bucket or identity that supplies each state. Set the deployment up to this shape
before starting:

| bucket | object lock | versioning | encryption | policy | other |
|---|---|---|---|---|---|
| `public-assets` | off | off | none | Allow `Principal: {"AWS":["*"]}` on `s3:GetObject` for `arn:aws:s3:::public-assets/*` | |
| `private-data` | on, GOVERNANCE 30 DAYS | on | SSE-S3 (`AES256`) | none | tags `env=prod owner=platform`; quota 1 GiB hard; lifecycle rule `expire-old` (Enabled, prefix `logs/`, expire 90d, noncurrent 30d) |
| `kms-bucket` | off | off | SSE-KMS with key `minio-default-key` | none | |
| `deny-only` | off | off | none | Deny-only: `Effect: Deny`, `Principal: {"AWS":["*"]}`, `s3:*`, condition `Bool aws:SecureTransport false` | |
| `wildcard-action` | off | off | none | Allow `Principal: {"AWS":["*"]}` on `s3:*` | |
| `plain` | off | off | none | none | nothing configured at all: the absent case for every field |

Identities:

| object | state |
|---|---|
| user `alice` | enabled, policy `scoped-read` attached, member of group `devs`, has a service account |
| user `bob` | **disabled**, no policy, no groups, no service accounts, `updatedAt` never stamped |
| group `devs` | enabled, member `alice`, policy `custom-wildcard` |
| policy `custom-wildcard` | `Allow s3:* on arn:aws:s3:::*` (wildcard action + wildcard resource, no admin) |
| policy `scoped-read` | `Allow s3:GetObject on arn:aws:s3:::private-data/*` (no wildcards) |
| policy `consoleAdmin` | built in; grants `admin:*` -- the admin-access state |
| service account `alicesvcacct0001` | parent `alice`, named `ci-pipeline`, implied policy, **with** an expiry |
| a second service account | parent `alice`, **no** expiry -- the null-timestamp state |

Webhooks (the audit/logger schema needs all four shapes):

| target | state |
|---|---|
| `audit_webhook` (unnamed) | endpoint set, enabled -- reports as name `_` |
| `audit_webhook:primary` | endpoint set, `auth_token` set, enabled |
| `audit_webhook:noauth` | endpoint set, no `auth_token`, enabled |
| `audit_webhook:offtarget` | endpoint set, `enable=off` |
| `logger_webhook:primary` | endpoint set |

For the `status` two-state, run a listener on the target port
(`python3 -m http.server 18080`) and re-read: targets flip `offline` -> `online`
within a few seconds. This was observed, and is the only way to tell a working
`status` from a hard-coded one.

---

## 4. Per-field checklist

All unchecked. Tick a row only after reading the value and comparing it to the
expected column. Where a row names two buckets, run it against **both**.

Run everything with the `env -u HTTPS_PROXY ... PROVIDERS_PATH=/tmp/pd mql run minio -c "..."`
prefix from §2.

### 4.1 Deployment (`minio`)

| # | field | query | expected |
|---|---|---|---|
| [ ] | `version` | `minio.version` | matches `/tmp/miniobin/minio --version` (`DEVELOPMENT.GOGET` on a source build) |
| [ ] | `deploymentMode` | `minio.deploymentMode` | `standalone` on the one-server fixture. **Second state:** stand up a 2-node distributed deployment and confirm `distributed`. Until then this field is untested in one direction. |
| [ ] | `region` | `minio.region` | `us-east-1` (from `MINIO_REGION`). **Second state:** restart without `MINIO_REGION`, expect empty |
| [ ] | `deploymentId` | `minio.deploymentId` | a UUID, stable across restarts of the same drives |
| [ ] | `endpoint` | `minio.endpoint` | `127.0.0.1:9000` |
| [ ] | `tlsEnabled` | `minio.tlsEnabled` | `false` on the http fixture. **Second state:** re-run against an https endpoint (see §6 risk 4) and expect `true` |
| [ ] | `backendType` | `minio.backendType` | `Erasure`. **Second state:** a single-drive server reports `FS` |
| [ ] | `onlineDrives` | `minio.onlineDrives` | `4` |
| [ ] | `offlineDrives` | `minio.offlineDrives` | `0`. **Second state:** `chmod 000` one drive dir, restart, expect non-zero |
| [ ] | `kmsConfigured` | `minio.kmsConfigured` | `true` with `MINIO_KMS_SECRET_KEY` set. **Second state:** restart without it, expect `false` |
| [ ] | `consoleHstsSeconds` | `minio.consoleHstsSeconds` | `0` by default. **Second state:** `mc admin config set fx browser hsts_seconds=31536000`, expect `31536000` |
| [ ] | `consoleHstsIncludeSubdomains` | `minio.consoleHstsIncludeSubdomains` | `false`; flip with `hsts_include_subdomains=on` |
| [ ] | `consoleHstsPreload` | `minio.consoleHstsPreload` | `false`; flip with `hsts_preload=on` |
| [ ] | `corsAllowOrigin` | `minio.corsAllowOrigin` | `["*"]` by default. **Second state:** `mc admin config set fx api cors_allow_origin="https://a.example,https://b.example"`, expect two entries |
| [ ] | `servers` | `minio.servers { endpoint state version }` | one row, `127.0.0.1:9000`, `online` |
| [ ] | `buckets` | `minio.buckets { name }` | all six fixture buckets |
| [ ] | `users` | `minio.users { name }` | `alice`, `bob` |
| [ ] | `groups` | `minio.groups { name }` | `devs` |
| [ ] | `serviceAccounts` | `minio.serviceAccounts { accessKey }` | both fixture accounts |
| [ ] | `policies` | `minio.policies { name }` | the five built-ins plus `custom-wildcard` and `scoped-read` |
| [ ] | `auditWebhooks` | `minio.auditWebhooks { name }` | four targets including `_` |
| [ ] | `loggerWebhooks` | `minio.loggerWebhooks { name }` | `primary` |

### 4.2 Server (`minio.server`)

| # | field | query | expected |
|---|---|---|---|
| [ ] | `endpoint` | `minio.servers { endpoint }` | `127.0.0.1:9000` |
| [ ] | `state` | `minio.servers { state }` | `online` |
| [ ] | `version` | `minio.servers { version }` | same as `minio.version` |
| [ ] | `commitId` | `minio.servers { commitId }` | `DEVELOPMENT.GOGET` on a source build; a real commit on a release build |
| [ ] | `uptime` | `minio.servers { uptime }` | grows between two runs a minute apart |
| [ ] | `poolNumber` | `minio.servers { poolNumber }` | `1` |
| [ ] | `totalDrives` | `minio.servers { totalDrives }` | `4` |
| [ ] | `onlineDrives` | `minio.servers { onlineDrives }` | `4`; drops when a drive is taken out |

### 4.3 Bucket (`minio.bucket`)

| # | field | query | expected |
|---|---|---|---|
| [ ] | `name` | `minio.buckets { name }` | the six names |
| [ ] | `createdAt` | `minio.buckets { name createdAt }` | a real timestamp on every bucket, never null and never year 1 |
| [ ] | `location` | `minio.buckets { name location }` | `us-east-1` on all. **This is the field most likely to be empty**: the bucket listing carries no region, so it takes a separate call |
| [ ] | `tags` | `minio.bucket(name: "private-data").tags` vs `minio.bucket(name: "plain").tags` | `{env: prod, owner: platform}` vs `{}` |
| [ ] | `versioning` | `... "private-data" .versioning` vs `"plain"` | `{Status: "Enabled", MFADelete: ""}` vs `{Status: "", MFADelete: ""}` |
| [ ] | `versioningEnabled` | same two buckets | `true` vs `false` |
| [ ] | `objectLockEnabled` | `"private-data"` vs `"plain"` | `true` vs `false` |
| [ ] | `defaultRetentionMode` | `"private-data"` vs `"plain"` | `GOVERNANCE` vs `""` |
| [ ] | `defaultRetentionValidity` | `"private-data"` vs `"plain"` | `30` vs `0` |
| [ ] | `defaultRetentionUnit` | `"private-data"` vs `"plain"` | `DAYS` vs `""` |
| [ ] | `encrypted` | `"private-data"`, `"kms-bucket"` vs `"plain"` | `true`, `true` vs `false` |
| [ ] | `encryptionRules` | `... { sseAlgorithm }` on the three | `AES256`, `aws:kms`, `[]` |
| [ ] | `kmsKey` | `"kms-bucket".kmsKey { name }` vs `"private-data".kmsKey` vs `"plain".kmsKey` | `minio-default-key` vs **null** vs **null**. SSE-S3 names no key, so a non-null here would be wrong |
| [ ] | `quotaBytes` | `"private-data"` vs `"plain"` | `1073741824` vs `0` |
| [ ] | `quotaType` | `"private-data"` vs `"plain"` | `hard` vs `""` |
| [ ] | `lifecycleRules` | `"private-data"` vs `"plain"` | one rule vs `[]` |
| [ ] | `replicationRules` | needs a remote target configured (see §6 risk 2) | rows vs `[]` |
| [ ] | `policyDocument` | `"public-assets"` vs `"plain"` | the JSON vs `""` |
| [ ] | `policyStatements` | `"public-assets" { effect actions principals }` | one Allow, `["s3:GetObject"]`, `["*"]` |
| [ ] | `hasAnonymousAccess` | `public-assets` / `wildcard-action` vs `deny-only` / `plain` | `true`, `true` vs `false`, `false` |
| [ ] | `hasWildcardPrincipal` | `deny-only` vs `plain` | `true` vs `false` (a Deny to `*` still names the wildcard) |
| [ ] | `hasWildcardAction` | `wildcard-action` vs `public-assets` vs `deny-only` | `true` vs `false` vs `false` |
| [ ] | `enforceSslOnly` | `deny-only` vs `public-assets` | `true` vs `false` |

### 4.4 Bucket sub-resources

| # | field | query | expected |
|---|---|---|---|
| [ ] | `encryptionRule.sseAlgorithm` | `minio.bucket(name:"kms-bucket").encryptionRules { sseAlgorithm }` | `aws:kms` |
| [ ] | `encryptionRule.kmsKey` | same, `{ kmsKey { name } }` | `minio-default-key`; **null** on the SSE-S3 bucket |
| [ ] | `lifecycleRule.id` | `minio.bucket(name:"private-data").lifecycleRules { id }` | `expire-old` |
| [ ] | `lifecycleRule.status` | " | `Enabled`; **second state:** set a Disabled rule |
| [ ] | `lifecycleRule.prefix` | " | `logs/`. Note the prefix arrives inside `Filter`, not on the rule |
| [ ] | `lifecycleRule.expirationDays` | " | `90` |
| [ ] | `lifecycleRule.noncurrentVersionExpirationDays` | " | `30` |
| [ ] | `lifecycleRule.abortIncompleteMultipartUploadDays` | " | **Expected 7, observed absent.** The value set through `SetBucketLifecycle` did not come back in the GET. Determine whether MinIO drops it or minio-go fails to send it, and fix or document |
| [ ] | `lifecycleRule.expiredObjectDeleteMarker` | needs a rule with `ExpiredObjectDeleteMarker` | `true` vs `false` |
| [ ] | `lifecycleRule.transitionDays` / `transitionStorageClass` | needs a tier configured (`mc ilm tier add`) | non-zero / tier name vs `0` / `""` |
| [ ] | `replicationRule.*` (8 fields) | all untested; see §6 risk 2 | |

### 4.5 Identity

| # | field | query | expected |
|---|---|---|---|
| [ ] | `user.name` | `minio.users { name }` | `alice`, `bob` |
| [ ] | `user.status` | `minio.users { name status }` | `enabled` / `disabled` |
| [ ] | `user.enabled` | " | `true` / `false` |
| [ ] | `user.updatedAt` | `minio.users { name updatedAt }` | a timestamp for `alice`, **null** for `bob`. A value of `0001-01-01` here is a bug |
| [ ] | `user.policies` | `minio.user(name:"alice").policies { name }` vs `bob` | `["scoped-read"]` vs `[]` |
| [ ] | `user.groups` | `minio.user(name:"alice").groups { name }` vs `bob` | `["devs"]` vs `[]` |
| [ ] | `user.serviceAccounts` | `minio.user(name:"alice").serviceAccounts { accessKey }` vs `bob` | two vs `[]` |
| [ ] | `group.name` / `status` / `enabled` | `minio.groups { name status enabled }` | `devs` / `enabled` / `true`. **Second state:** `mc admin group disable fx devs` |
| [ ] | `group.updatedAt` | `minio.groups { updatedAt }` | a timestamp, never year 1 |
| [ ] | `group.members` | `minio.group(name:"devs").members { name }` | `["alice"]` |
| [ ] | `group.policies` | `minio.group(name:"devs").policies { name }` | `["custom-wildcard"]` |
| [ ] | `serviceAccount.accessKey` | `minio.serviceAccounts { accessKey }` | both |
| [ ] | `serviceAccount.name` / `description` | " | `ci-pipeline` / `used by CI`; empty on an unnamed one |
| [ ] | `serviceAccount.status` | " | `on`. **Second state:** disable one, expect `off` |
| [ ] | `serviceAccount.impliedPolicy` | " | `true` on an inheriting account; **second state:** create one with `--policy`, expect `false` |
| [ ] | `serviceAccount.expiresAt` | " | a timestamp on the expiring one, **null** on the other |
| [ ] | `serviceAccount.parentUser` | `... { parentUser { name } }` | `alice`. **Second state:** an account whose parent is not a built-in user must be **null**, not an error |
| [ ] | `serviceAccount.policyDocument` | " | non-empty JSON |
| [ ] | `serviceAccount.policyStatements` | `... { policyStatements { effect actions } }` | two statements on the fixture account |
| [ ] | `policy.name` | `minio.policies { name }` | all seven |
| [ ] | `policy.document` | `minio.policy(name:"custom-wildcard").document` | the JSON |
| [ ] | `policy.createdAt` / `updatedAt` | `minio.policy(name:"custom-wildcard") { createdAt updatedAt }` | timestamps on a user-created policy; **check what a built-in reports** and confirm it is null rather than year 1 |
| [ ] | `policy.statements` | `... .statements { effect actions resources principals }` | principals **empty** on a named policy |
| [ ] | `policy.hasWildcardAction` | `custom-wildcard` vs `scoped-read` | `true` vs `false` |
| [ ] | `policy.hasWildcardResource` | `custom-wildcard` vs `scoped-read` | `true` vs `false` |
| [ ] | `policy.grantsAdminAccess` | `consoleAdmin` vs `custom-wildcard` | `true` vs `false` |
| [ ] | `policy.users` | `minio.policy(name:"scoped-read").users { name }` vs `readonly` | `["alice"]` vs `[]` |
| [ ] | `policy.groups` | `minio.policy(name:"custom-wildcard").groups { name }` vs `readonly` | `["devs"]` vs `[]` |
| [ ] | `policyStatement.*` (8 fields) | `minio.bucket(name:"deny-only").policyStatements { sid effect actions notActions resources notResources principals conditions }` | `DenyInsecure` / `Deny` / `["s3:*"]` / `[]` / `[...]` / `[]` / `["*"]` / `{Bool: {aws:SecureTransport: ["false"]}}` |

### 4.6 Webhooks and KMS

| # | field | query | expected |
|---|---|---|---|
| [ ] | `webhook.type` | `minio.auditWebhooks { type }` / `minio.loggerWebhooks { type }` | `audit` / `logger` |
| [ ] | `webhook.name` | `minio.auditWebhooks { name }` | `_`, `primary`, `noauth`, `offtarget` |
| [ ] | `webhook.endpoint` | " `{ endpoint }` | the configured URLs |
| [ ] | `webhook.enabled` | " `{ name enabled }` | `true` for all but `offtarget`, which is `false` |
| [ ] | `webhook.status` | " `{ name status }` | `offline` with no listener; start `python3 -m http.server 18080` and re-run: `online`. `offtarget` reports `""` |
| [ ] | `webhook.queueSize` | " | `100000` |
| [ ] | `webhook.queueDir` | " | `""`; **second state:** set `queue_dir` and expect the path |
| [ ] | `webhook.clientCertConfigured` | " | `false`; **second state:** set `client_cert` and expect `true` |
| [ ] | `webhook.maxRetry` | " | `0`; **second state:** set `max_retry=3` |
| [ ] | `webhook.httpTimeout` | " | `5s` |
| [ ] | `kmsKey.name` | `minio.kmsKey(name: "minio-default-key").name` | `minio-default-key` |
| [ ] | `kmsKey.healthy` | `minio.kmsKey(name:"minio-default-key").healthy` vs `minio.kmsKey(name:"nope").healthy` | `true` vs `false` |
| [ ] | `kmsKey.encryptionError` | same two | `""` vs `key with given key ID does not exist` |
| [ ] | `kmsKey.decryptionError` | same two | `""` vs whatever the deployment reports |
| [ ] | `kmsKey.version` | `minio.kmsKey(name:"minio-default-key").version` | **Observed empty on the built-in KMS.** Confirm whether a KES-backed deployment fills it; if it is always empty, remove the field |

---

## 5. The four checks from `new-resource` §5

### 5a. The absent case must FAIL, not pass vacuously

Point the provider at something that is not MinIO and confirm the queries error
rather than returning satisfying nulls.

```bash
# 1. no server at all -- must error, must not return an empty bucket list
python3 -c "import socket;s=socket.socket();s.bind(('127.0.0.1',9099));s.listen(1)" &
env -u HTTPS_PROXY NO_PROXY='*' PROVIDERS_PATH=/tmp/pd \
  mql run minio --endpoint http://127.0.0.1:9099 --access-key a --secret-key bbbbbbbb \
  -c "minio.buckets"

# 2. wrong credentials -- must error, must not report zero buckets
env -u HTTPS_PROXY NO_PROXY='*' PROVIDERS_PATH=/tmp/pd \
  mql run minio --endpoint http://127.0.0.1:9000 --access-key wrong --secret-key wrongwrong \
  -c "minio.buckets"

# 3. an under-scoped key. Create a user with the built-in `readonly` policy and
#    connect as them. Every field that cannot be read must ERROR, and
#    hasAnonymousAccess must come back NULL. It must NOT come back false: a
#    false there is indistinguishable from "verified private".
env -u HTTPS_PROXY NO_PROXY='*' PROVIDERS_PATH=/tmp/pd \
  mql run minio --access-key readonlyuser --secret-key ... \
  -c "minio.buckets { name hasAnonymousAccess encrypted }"
```

- [ ] no server: errors
- [ ] wrong credentials: errors
- [ ] under-scoped key: `hasAnonymousAccess` is **null**, not false
- [ ] under-scoped key: `encrypted` **errors**, does not report false
- [ ] a check written as `minio.buckets.all(hasAnonymousAccess == false)` FAILS
      under an under-scoped key rather than passing. (MQL three-valued logic:
      `null && null` is `true`, so verify this explicitly.)

### 5b. Null over invented values

- [ ] `minio.bucket(name:"plain").kmsKey` is **null**, not an empty resource
- [ ] `minio.bucket(name:"private-data").kmsKey` is **null** (SSE-S3 names no key)
- [ ] `minio.user(name:"bob").updatedAt` is **null**, not `0001-01-01T00:00:00Z`
- [ ] a service account with no expiry reports `expiresAt == null`
- [ ] `minio.serviceAccounts { parentUser }` is **null** for a non-built-in parent
- [ ] `minio.bucket(name:"plain").quotaType` is `""`, and `quotaBytes` is `0` --
      confirm that is what the deployment actually says and not a substituted default
- [ ] `minio.policy(name:"readonly").createdAt` -- confirm null on a built-in
      rather than year 1

### 5c. Secret sweep must return 0

```bash
env -u HTTPS_PROXY NO_PROXY='*' PROVIDERS_PATH=/tmp/pd \
  mql run minio -c "minio { buckets { * } users { * } groups { * } serviceAccounts { * } policies { * } auditWebhooks { * } loggerWebhooks { * } }" -j > /tmp/minio-scan.json

grep -c 'minioadmin123'          /tmp/minio-scan.json   # must be 0
grep -c 'alicesvcacctsecret0001' /tmp/minio-scan.json   # must be 0
grep -c 'supersecrettoken'       /tmp/minio-scan.json   # must be 0 (webhook auth_token)
grep -ci 'secretkey'             /tmp/minio-scan.json   # must be 0
grep -c 'Jr8EBqxOKo3fUJerHzTS3Yz4N3dRmdUDkAAjSkJeMgA=' /tmp/minio-scan.json  # must be 0
```

- [ ] every count is 0

Note the guarantee is precise: **no field in this schema carries key material**.
It is not "the deployment's secrets are unreachable" -- `minio.policy.document`
and `minio.bucket.policyDocument` return policy documents verbatim, so a policy
that embeds a credential in a condition value is reported as written.

### 5d. Composability

- [ ] `minio.buckets { name kmsKey { name healthy } }` reaches the key from every
      encrypted bucket
- [ ] `minio.users { name groups { name policies { name grantsAdminAccess } } }`
      traverses user -> group -> policy in one query
- [ ] `minio.policies.where(grantsAdminAccess) { users { name } groups { members { name } } }`
      resolves the reverse edges
- [ ] `minio.serviceAccounts { parentUser { name policies { name } } }`
- [ ] Re-run each of the above and confirm the request count does not grow with
      the number of rows (see §6 risk 6)

---

## 6. Risk areas, ranked by damage

Ordered by how badly a defect here would mislead someone. Everything above the
line produces a **confidently wrong answer**; everything below produces an empty
or missing one.

### WRONG-ANSWER RISKS

**1. `hasAnonymousAccess` reporting `false` on an exposed bucket.** This is the
flagship field and the whole reason the provider exists. It is `false` when the
policy is readable and grants nothing to `*`, and **null** when the policy could
not be read or parsed -- but that null path has only been exercised in unit
tests, never through the runtime. If `plugin.StateIsNull` is not honoured the way
`buckets.go: hasAnonymousAccess` assumes, an unreadable policy degrades to
`false` and a world-readable bucket is reported as private. **Verify §5a's
under-scoped-key case before anything else.** Related: a policy shape not covered
by `resources/policy_test.go` (a `NotPrincipal`, a `Principal` object keyed by
something other than `AWS`, an `arn:aws:iam::*:root` wildcard) would parse to no
principals and read as private. Only the `{"AWS":["*"]}` and bare-`"*"` spellings
have been seen on a real deployment.

**2. Replication is entirely unexercised.** No fixture had a replication target,
so all eight `minio.bucket.replicationRule` fields and the `replicationRules`
list ran against a `{Rules: null}` response only. A bucket replicating to an
unexpected destination is exactly the finding this resource is for, and a mapping
bug would report the destination bucket, priority or `deleteMarkerReplicationEnabled`
wrongly rather than emptily. Set up a second MinIO and `mc replicate add` before
trusting any of it.

**3. `enforceSslOnly` and the condition parser.** The single real deny-only
policy used `Bool` / `aws:SecureTransport` / `["false"]`. A deployment writing
`BoolIfExists`, or `"Null"`, or nesting the condition differently, would report
`false` -- "does not enforce TLS" -- on a bucket that does. That direction is a
false alarm rather than a false pass, so it ranks below 1 and 2, but a policy
using `Deny` with `NotAction` would go the other way.

**4. `tlsEnabled` describes the connection, not the deployment.** It reports the
scheme this connection used. A deployment that serves both HTTP and HTTPS, scanned
over HTTPS, reports `true` while still accepting plaintext. The doc comment says
so, but an audit author may read it as "the deployment requires TLS". Decide
whether that is acceptable or whether the field needs renaming before release --
renaming after release is breaking.

**5. `deploymentMode` is derived from the server count, not reported.** MinIO's
`mode` field is health (`online`), not topology. `len(Servers) > 1` was never
tested against a real distributed deployment. If a single-node multi-drive
deployment ever reports more than one server entry, every such deployment would
be labelled `distributed`.

**6. `webhook.enabled` inverts an absence.** MinIO omits the `enable` key on a
target it is using and writes `enable=off` on one it is not, so the mapping treats
an absent key as enabled. That was verified in both directions on one server
version. If a future release starts writing `enable=on` explicitly the mapping
still works, but if it stops writing `enable=off` on a disabled target, a
switched-off audit trail would be reported as active.

**7. `quotaBytes` reads `Size`, not `Quota`.** The real payload carries both, with
the legacy `quota` field at 0 and `size` holding the limit. madmin v4's struct
only exposes `Size`, so this is correct today; a madmin downgrade or an older
server that populates only `quota` would report every bucket as unlimited.

### EMPTY-ANSWER RISKS

**8. `location` on every bucket.** The listing carries no region, so this depends
entirely on a per-bucket `GetBucketLocation`. If that fails or is denied, every
bucket reports empty. Low damage (an empty region is obviously empty) but it will
be the first thing anyone notices.

**9. `lifecycleRule.abortIncompleteMultipartUploadDays`.** Set to 7 in the
fixture and **absent in the response**. Either MinIO drops it or minio-go does not
send it. It currently reports 0, which reads as "no cleanup configured" on a
bucket that has it -- so this is arguably a wrong answer, but a conservative one.
Resolve it.

**10. `kmsKey.version` was empty** against the built-in KMS. If it is always empty
outside a KES deployment, the field is dead schema and should be removed before
release (removing it after is breaking).

**11. `minio.serviceAccounts` iterates every user.** It issues one
`ListServiceAccounts` per user plus one for the deployment's own account. A
deployment with thousands of users makes that field slow enough to time out. There
is no bulk endpoint in madmin v4 for the built-in identity store
(`ListAccessKeysBulk` exists but targets LDAP/OpenID); confirm before optimizing.

**12. Nothing in the API surface paginates.** `ListBuckets`, `ListUsers`,
`ListGroups`, `ListCannedPolicies` and `ListServiceAccounts` all return complete
listings, so there is no cursor to get wrong -- but equally no guard if a
deployment ever truncates. Verify against a deployment with more than a thousand
buckets that the count matches `mc ls`.

**13. Admin API version fallback.** madmin-go v4 addresses `/minio/admin/v4/`;
the server built for the fixtures answers `426 Upgrade Required` and madmin retries
against `/minio/admin/v3/`. That doubles the request count for every admin call
on any server predating the v4 admin API. Measure it, and consider whether
`MADMIN_API_VERSION=v3` should be set for the provider process.

**14. `minio.bucket(name: ...)` init walks the whole bucket listing.** Correct but
O(n) per lookup; and it returns a not-found error rather than a blank resource,
which is what §5a check 1 should confirm.

---

## 7. When this is done

Update the top of this file with what was verified and what was not, and record
any field that could not be exercised as a named gap rather than a disclaimer.
`new-resource` §5: a resource that has never run against a live instance does not
ship, and "no live host was available" belongs in the PR body as a blocker.
