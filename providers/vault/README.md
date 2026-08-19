# HashiCorp Vault Provider

The `vault` provider connects to a HashiCorp Vault server and inventories its
security posture through read-only queries: whether the server is sealed and how
it unseals, which audit devices record access, which authentication methods let
callers in, which ACL policies grant them capabilities, which secret engines are
mounted, and which Enterprise namespaces divide the server.

It works against self-managed Vault (Community or Enterprise) and against HCP
Vault through its public endpoint. It complements the `hcp` provider, which
reports the HCP control plane's view of a cluster (tier, version, network
exposure) but not how Vault itself is configured.

## Prerequisites

A Vault token, or an AppRole role ID and secret ID, with read access to the
`sys/` endpoints this provider queries:

| Endpoint | Used for |
| --- | --- |
| `sys/health` | `version`, `clusterName`, `clusterId`, `initialized` |
| `sys/seal-status` | `vault.seal` |
| `sys/audit` | `vault.auditDevices` |
| `sys/auth` | `vault.authMethods` |
| `sys/mounts` | `vault.secretEngines` |
| `sys/policies/acl` and `sys/policies/acl/:name` | `vault.policies` |
| `sys/namespaces` | `vault.namespaces` (Enterprise only) |

A policy granting exactly that:

```hcl
path "sys/health"         { capabilities = ["read"] }
path "sys/seal-status"    { capabilities = ["read"] }
path "sys/audit"          { capabilities = ["read", "sudo"] }
path "sys/auth"           { capabilities = ["read"] }
path "sys/mounts"         { capabilities = ["read"] }
path "sys/policies/acl"   { capabilities = ["read", "list"] }
path "sys/policies/acl/*" { capabilities = ["read"] }
path "sys/namespaces"     { capabilities = ["list"] }
```

> `sys/audit` requires the `sudo` capability. Without it the server answers 403
> and `vault.auditDevices` returns an error rather than an empty list, so a
> missing permission cannot be mistaken for a server with no audit device.

## Authentication

Arguments:

- `--address` - Vault API address. Defaults to `$VAULT_ADDR`.
- `--token` - Vault token. Defaults to `$VAULT_TOKEN`.
- `--role-id` / `--secret-id` - AppRole credentials, used instead of a token.
  The secret ID defaults to `$VAULT_SECRET_ID`.
- `--namespace` - Enterprise namespace to scan. Defaults to `$VAULT_NAMESPACE`,
  and to the root namespace when unset.
- `--ca-cert` - path to a certificate authority to trust. Defaults to
  `$VAULT_CACERT`.
- `--tls-skip-verify` - skip certificate verification. Lab servers only.

With a token:

```shell
export VAULT_ADDR=https://vault.example.com:8200
export VAULT_TOKEN=hvs.example
mql shell vault
```

With AppRole:

```shell
mql shell vault --address https://vault.example.com:8200 \
  --role-id ROLE-ID --secret-id SECRET-ID
```

Against an Enterprise namespace:

```shell
mql shell vault --namespace team-a/
```

## Usage

Open an interactive shell:

```shell
mql shell vault
```

## Examples

Confirm the server is unsealed and check how it unseals:

```coffee
> vault.seal { type sealed autoUnseal threshold shares }
vault.seal: {
  type: "awskms"
  sealed: false
  autoUnseal: true
  threshold: 1
  shares: 1
}
```

Find a server with no audit device, which keeps no record of secret access:

```coffee
> vault.auditDevices.length > 0
```

Find audit devices writing secrets to disk in plaintext:

```coffee
> vault.auditDevices.where(logRaw)
```

Find policies that grant across every mount, or that carry `sudo`:

```coffee
> vault.policies.where(grantsRootPath || grantsSudo) { name paths }
```

Inspect the parsed grants of one policy:

```coffee
> vault.policies.where(name == "default") { paths }
vault.policies.where: [
  0: {
    paths: [
      0: { path: "auth/token/lookup-self", capabilities: ["read"] }
      1: { path: "sys/renew/*", capabilities: ["update"] }
    ]
  }
]
```

Find key/value engines still on version 1, which keep no version history:

```coffee
> vault.secretEngines.where(type == "kv" && kvVersion == 1) { path }
```

Find auth methods issuing tokens that never expire:

```coffee
> vault.authMethods.where(maxLeaseTtl == 0) { path type }
```

Find auth methods advertised on the unauthenticated login form:

```coffee
> vault.authMethods.where(listedInLoginForm) { path type }
```

## Verification

After a change, rebuild and install the provider, then walk the resources
against a real server. A Vault dev server is enough for everything except
namespaces, which need Enterprise:

```shell
vault server -dev -dev-root-token-id=root &
export VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN=root
vault audit enable file file_path=/tmp/vault-audit.log

make providers/build/vault && make providers/install/vault
mql shell vault
```

## Troubleshooting

**`a Vault address is required`** - set `VAULT_ADDR` or pass `--address`. A bare
host is assumed to be HTTPS; use an explicit `http://` for a dev server.

**`permission denied` on `vault.auditDevices`** - the token's policy is missing
`sudo` on `sys/audit`. See Prerequisites.

**`vault.namespaces` is empty** - namespaces are a Vault Enterprise feature.
Community edition answers 404 on `sys/namespaces`, which the provider reports as
an empty list. A token that lacks permission on the endpoint gets an error
instead, so an empty list always means "no namespaces" rather than "could not
look".

**Certificate errors against a private authority** - pass `--ca-cert` with the
path to the authority's PEM rather than disabling verification.
