# HashiCorp Consul Provider

The `consul` provider connects to a self-managed HashiCorp Consul agent over its
HTTP API and reports the runtime security configuration of the datacenter it
serves: whether the ACL system is switched on and what it does with an unmatched
request, whether the gossip pool between agents is encrypted, how the
agent-to-agent TLS layer verifies its peers, and which service mesh intentions
decide who may talk to whom. It also inventories the ACL tokens, policies and
roles defined in the datacenter.

This is a different asset from `hcp.consul.cluster`, which reports HCP
control-plane metadata (version, tier, endpoint) about a managed cluster. This
provider reads the running agent's own configuration, which HCP does not expose.

## Prerequisites

- Network access to a Consul agent's HTTP API (`8500` for plaintext, `8501` for
  TLS by default).
- An ACL token, when the datacenter has ACLs enabled. Without one the provider
  reads whatever the anonymous token may read, which on a datacenter with a
  default policy of `deny` is usually nothing.

## Authentication

Arguments:

- `--address` - the Consul HTTP API address. Defaults to `$CONSUL_HTTP_ADDR`,
  then to `http://127.0.0.1:8500`.
- `--token` - the ACL token. Defaults to `$CONSUL_HTTP_TOKEN`.
- `--ca-cert` - a certificate authority to trust, as a PEM file path. Defaults
  to `$CONSUL_CACERT`. An inventory file may carry the PEM inline instead.
- `--tls-skip-verify` - skip certificate verification. For lab agents only.

Query the agent on the local host:

```shell
mql shell consul
```

Query a remote agent over TLS:

```shell
mql shell consul --address https://consul.example.com:8501 --token ACL_TOKEN
```

Or through the environment, which the Consul CLI uses the same way:

```shell
export CONSUL_HTTP_ADDR=https://consul.example.com:8501
export CONSUL_HTTP_TOKEN=00000000-0000-0000-0000-000000000000
export CONSUL_CACERT=/etc/consul.d/tls/consul-agent-ca.pem
mql shell consul
```

> Create a read-only token with
> `consul acl token create -policy-name mondoo-read -description "Mondoo scanning"`,
> after writing the policy below.

### Least-privilege policy

```hcl
agent_prefix "" {
  policy = "read"
}
acl      = "read"
operator = "read"
service_prefix "" {
  policy     = "read"
  intentions = "read"
}
```

- `agent:read` serves the agent configuration, which is where every setting on
  the `consul` resource comes from.
- `acl:read` serves the token, policy and role inventory. Without it those
  fields report an error rather than an empty list, so a missing permission
  cannot be mistaken for a datacenter with no tokens.
- `intentions:read` on the service prefix serves the mesh intentions.

The provider never writes, and never reads the gossip keyring or the secret half
of any ACL token.

## Usage

Open an interactive shell:

```shell
mql shell consul
```

## Examples

**Is the datacenter locked down at all?**

The three settings that decide whether an unauthenticated caller can do anything
at all.

```shell
mql> consul { datacenter version server acl { enabled defaultDeny } gossipEncrypted }
```

**Are the agents verifying each other on the RPC channel?**

`verifyServerHostname` is the one that stops a client certificate impersonating
a server.

```shell
mql> consul { verifyIncoming verifyOutgoing verifyServerHostname }
```

**Which TLS layer is weaker than the others?**

Consul configures the internal RPC channel, the HTTPS API and the gRPC/xDS
stream separately, and they routinely disagree.

```shell
mql> consul.tlsProfiles { scope verifyIncoming verifyOutgoing tlsMinVersion }
```

**Does an unauthenticated caller get any access?**

Every request arriving without a token resolves to the anonymous token, so any
grant on it is a grant to the whole network.

```shell
mql> consul.acl.anonymousToken { hasGrants policies { name } }
```

**Which tokens hold unrestricted access, and which never expire?**

```shell
mql> consul.acl.tokens.where(isGlobalManagement) { accessorId description }
mql> consul.acl.tokens.where(expirationTime == null) { accessorId description }
```

**Which intentions open a service to the whole mesh?**

```shell
mql> consul.mesh.intentions.where(hasWildcard && action == "allow") { sourceName destinationName }
```

**Is the mesh an allowlist or a blocklist?**

```shell
mql> consul.mesh { enabled defaultIntentionPolicy defaultDeny }
```

## Resources

- `consul` - the agent and the datacenter it serves.
- `consul.tlsProfile` - TLS settings per traffic layer.
- `consul.aclSystem` - the ACL system, reached as `consul.acl`.
- `consul.acl.token`, `consul.acl.policy`, `consul.acl.role` - the ACL
  inventory.
- `consul.serviceMesh` - the mesh, reached as `consul.mesh`.
- `consul.mesh.intention` - one mesh authorization rule.

The full field reference is generated from the schema comments in
`resources/consul.lr`.

## Verification

```shell
mql run consul -c "consul.datacenter"
```

A datacenter name confirms the address, the token and `agent:read`.

```shell
mql run consul -c "consul.acl.enabled"
```

If this is `true` but `consul.acl.tokens` errors, the token lacks `acl:read`.
An empty token list is only ever reported when the ACL system is switched off.

## Troubleshooting

**`Unexpected response code: 403 (ACL not found)`** - the token is not one this
datacenter knows. Check `CONSUL_HTTP_TOKEN` and that you are pointed at the right
datacenter.

**`Unexpected response code: 403 (Permission denied: ... lacks permission 'agent:read')`** -
the token authenticates but may not read the agent configuration. Add the policy
above.

**`a Consul address must use http or https`** - a `unix://` socket address is
not supported, because it carries no host for the asset identity. Point at the
agent's TCP listener instead.

**Certificate errors against an agent using the cluster's own authority** - pass
`--ca-cert /etc/consul.d/tls/consul-agent-ca.pem`, or set `CONSUL_CACERT`.
