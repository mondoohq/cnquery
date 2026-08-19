# Vault provider verification record

Verified against **HashiCorp Vault 1.20.4** (Community, dev server, in-memory
storage) on 2026-08-19. Every resource and field below was queried live and the
returned value read. This provider does not need the `TESTING-TODO.md` handoff
the other new providers carry.

## How it was run

```bash
vault server -dev -dev-root-token-id=root -dev-listen-address=127.0.0.1:8200
export VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN=root

make providers/build/vault && make providers/install/vault
PROVIDERS_PATH=$HOME/.config/mondoo/providers mql run vault -c "<query>"
```

Fixtures seeded to exercise every branch: two audit devices (one with
`log_raw=true`), three auth methods, a v1 and a v2 kv engine plus pki, and four
policies covering root-path, sudo, scoped-wildcard and deny-only rules.

## Observed results

| Query | Observed |
|---|---|
| `vault { version clusterName clusterId initialized }` | `1.20.4`, `vault-cluster-dd4b99ae`, `188f68c8-…`, `true` — matches `vault status` |
| `vault.seal { type sealed autoUnseal threshold shares }` | `shamir`, `false`, `false`, `1`, `1` |
| `vault.auditDevices { path type logRaw options }` | 2 devices; `file/` → `logRaw: false`, `raw-file/` → `logRaw: true` |
| `vault.authMethods { path type tokenType listedInLoginForm }` | 3 methods: `approle/`, `token/`, `userpass-lab/` |
| `vault.secretEngines { path type kvVersion }` | 6 engines; `kv1/` → 1, `secret/` → 2, non-kv → 0 |
| `vault.policies { name grantsSudo grantsRootPath grantsWildcardPath }` | 7 policies; `lab-admin` and `root` flagged, `lab-scoped` correctly not |
| `vault.namespaces` | `[]` — Community edition, 404 handled as absence |

## Failure modes exercised

| Case | Result |
|---|---|
| Bad token | errors `permission denied` / `invalid token` |
| Unreachable server | errors `connection refused` |
| No address | `a Vault address is required (set VAULT_ADDR or use --address)` |
| No token | `a Vault token is required (…or use --role-id with --secret-id)` |
| AppRole via flags | resolved, returned `1.20.4` |
| AppRole via `VAULT_SECRET_ID` | resolved |
| AppRole, wrong secret | errors `invalid role or secret ID` |
| `sys/audit` without `sudo` | **errors** rather than reporting zero devices |
| `sys/namespaces` without permission | **errors** rather than reporting zero namespaces |

The last two matter most: a false "no audit device configured" or "no
namespaces" would be a silent audit pass on data that was never read.

## Bugs live verification caught

1. **The built-in `root` policy reported `grantsSudo: false`.** Vault's root
   policy carries an *empty document* because the server grants it everything
   implicitly, so parsing the text reported the most privileged policy on the
   server as granting nothing. `vault.policies.where(grantsSudo)` would have
   missed it. Now answered for what it does.
2. **A 403 on `sys/namespaces` returned an empty list.** On Enterprise a token
   missing one permission would report no namespaces and let every
   namespace-scoped audit pass. The classifier now treats only 404/405 as
   absence; denial errors.
3. **The documented least-privilege policy did not work.** The client uses
   `sys/policies/acl`, not `sys/policy`. Caught only by running the policy that
   had been written into the README.

## Performance

Policy documents lazy-load behind a shared double-checked fetch. Confirmed
against Vault's own audit log:

- `vault.policies { name }` over 7 policies → **1** API call (was 8).
- All five document-derived fields over 7 policies → **8** calls (would be 35
  without the shared fetch).

## Not verified

- **Vault Enterprise namespaces.** `vault.namespaces` returns `[]` on Community
  and the 404 path is confirmed, but the populated shape (`path`, `id`,
  `customMetadata`) has never been read against an Enterprise server.
- **TLS options.** `--ca-cert` and `--tls-skip-verify` are unit-tested against a
  real `vaultapi.Config`, but never exercised against an HTTPS Vault with a
  private authority.
- **Auto-unseal seal types.** `autoUnseal` is derived from `type != "shamir"` and
  unit-tested across eight values, but only `shamir` was observed live.
