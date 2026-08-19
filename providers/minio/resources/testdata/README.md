# MinIO fixtures

Every file here is a response body captured off the wire from a running MinIO
server. Nothing was hand-written or transcribed from documentation.

The server: `minio server` built from `github.com/minio/minio@latest` through the
Go module proxy, reporting version `DEVELOPMENT.GOGET` on Go 1.24.7, four local
drives in one erasure set, region `us-east-1`, with the built-in key management
service enabled via `MINIO_KMS_SECRET_KEY`. The exact recipe for standing it back
up is in `../../TESTING-TODO.md` §1.

Two things about the capture are worth knowing:

- The admin responses came back on `/minio/admin/v3/...`. This server answers
  `426 Upgrade Required` to `/minio/admin/v4/...`, and madmin-go falls back to v3
  automatically, which costs one extra request per admin call.
- The admin API encrypts the bodies of some endpoints with the caller's secret
  key. Those four files were captured encrypted and are stored decrypted, so they
  can be replayed through an `httptest` server that re-encrypts them with
  `madmin.EncryptData` (see `../client_test.go`). They are marked below.

| file | request | notes |
|---|---|---|
| `list_buckets.xml` | `GET /` | note the absent `BucketRegion` on every entry |
| `bucket_location.xml` | `GET /private-data/?location=` | |
| `versioning_enabled.xml` | `GET /private-data/?versioning=` | |
| `versioning_unset.xml` | `GET /plain/?versioning=` | never-configured versioning: empty status, HTTP 200 |
| `encryption_sse_s3.xml` | `GET /private-data/?encryption=` | `AES256`, no key named |
| `encryption_sse_kms.xml` | `GET /kms-bucket/?encryption=` | `aws:kms` plus `KMSMasterKeyID` |
| `object_lock_governance.xml` | `GET /private-data/?object-lock=` | GOVERNANCE, 30 DAYS |
| `lifecycle.xml` | `GET /private-data/?lifecycle=&withUpdatedAt=true` | the prefix arrives inside `Filter`; `AbortIncompleteMultipartUpload` was set and did not come back |
| `tagging.xml` | `GET /private-data/?tagging=` | |
| `bucket_policy_anonymous_read.json` | `GET /public-assets/?policy=` | Allow to `{"AWS":["*"]}` |
| `bucket_policy_deny_only.json` | `GET /deny-only/?policy=` | Deny-only; note the condition value came back wrapped in a list although it was written as a bare string |
| `bucket_policy_wildcard_action.json` | `GET /wildcard-action/?policy=` | Allow `s3:*` to `{"AWS":["*"]}` |
| `server_info.json` | `GET /minio/admin/v3/info?metrics=false&no-cache=false` | environment variables are redacted by the server |
| `list_users.json` | `GET /minio/admin/v3/list-users` | **stored decrypted**; `bob` carries `updatedAt: 0001-01-01T00:00:00Z` |
| `group_devs.json` | `GET /minio/admin/v3/group?group=devs` | |
| `list_canned_policies.json` | `GET /minio/admin/v3/list-canned-policies` | `consoleAdmin` has statements with no `Resource` element |
| `info_canned_policy.json` | `GET /minio/admin/v3/info-canned-policy?name=custom-wildcard&v=2` | |
| `list_service_accounts.json` | `GET /minio/admin/v3/list-service-accounts?user=alice` | **stored decrypted** |
| `info_service_account.json` | `GET /minio/admin/v3/info-service-account?accessKey=alicesvcacct0001` | **stored decrypted**; carries the effective policy |
| `bucket_quota.json` | `GET /minio/admin/v3/get-bucket-quota?bucket=private-data` | carries both a legacy `quota` of 0 and the `size` that holds the limit |
| `config_kv_audit_webhook.txt` | `GET /minio/admin/v3/get-config-kv?key=audit_webhook` | **stored decrypted**; four target shapes at once, and the `auth_token` key is absent on the target that has one |
| `config_kv_logger_webhook.txt` | `GET /minio/admin/v3/get-config-kv?key=logger_webhook` | **stored decrypted** |
| `kms_key_status_ok.json` | `GET /minio/kms/v1/key/status?key-id=minio-default-key` | |
| `kms_key_status_missing.json` | `GET /minio/kms/v1/key/status?key-id=nonexistent-key` | HTTP 200 for a key that does not exist; the failure is in `encryption-error` |

No file here carries key material. `resources/secrets_test.go`
(`TestFixturesCarryNoCredentials`) sweeps all of them for the capture
deployment's credentials on every run.
