# Consul provider: live verification handoff

**Status: this provider has never been run through `mql`.** Everything below is
outstanding. The Go unit tests pass and the schema was designed against real
`/v1/agent/self`, `/v1/acl/*` and `/v1/connect/intentions` payloads captured from
a Consul 1.20.1 agent (those captures are checked in under
`resources/testdata/`), but no MQL query has ever executed against this code.
Until somebody works through this document, every field is unverified end to end.

This document is written to be executable with no prior context. Work top to
bottom. Tick the boxes as you go and record what you actually saw, not what you
expected.

---

## 1. Stand up the targets

You need **two** Consul agents, because a field that reads the same before and
after you change a setting cannot distinguish a resource bug from a fixture that
never moved. Agent A is stock; agent B has ACLs, gossip encryption and TLS
verification switched on.

### Get the binary

```bash
mkdir -p /tmp/consul-test && cd /tmp/consul-test
curl -sSLo consul.zip https://releases.hashicorp.com/consul/1.20.1/consul_1.20.1_linux_amd64.zip
unzip -o -q consul.zip
./consul version    # expect: Consul v1.20.1
```

(The URL was confirmed reachable, HTTP 200, while this branch was written.)

### Agent A: state OPEN (nothing switched on)

```bash
cd /tmp/consul-test
./consul agent -dev -client 127.0.0.1 \
  -http-port 8700 -server-port 9300 -serf-lan-port 9301 -serf-wan-port 9302 \
  -grpc-port 9502 -grpc-tls-port 9503 -dns-port 9600 \
  -node dev-open -data-dir /tmp/consul-test/data-open > /tmp/consul-test/open.log 2>&1 &

# wait for it
until curl -sf http://127.0.0.1:8700/v1/agent/self -o /dev/null; do sleep 1; done
echo "agent A up on http://127.0.0.1:8700"
```

Do **not** use the default DNS port 8600 for the HTTP port; the agent refuses to
start with `bind: address already in use` because its own DNS server has it.

### Agent B: state HARDENED

Generate certificates first, because `verify_outgoing = true` without a CA makes
the agent exit with `VerifyOutgoing set but no CA certificates were provided`:

```bash
cd /tmp/consul-test
./consul tls ca create
./consul tls cert create -server -dc dc-secure -additional-ipaddress=127.0.0.1
./consul keygen        # copy the output into `encrypt` below
```

```bash
cat > /tmp/consul-test/secure.hcl <<'EOF'
datacenter  = "dc-secure"
node_name   = "secure-node"
server      = true
bootstrap   = true
bind_addr   = "127.0.0.1"
client_addr = "127.0.0.1"
data_dir    = "/tmp/consul-test/data-secure"

encrypt = "PASTE-THE-consul-keygen-OUTPUT-HERE"

acl {
  enabled                  = true
  default_policy           = "deny"
  down_policy              = "extend-cache"
  enable_token_persistence = true
  tokens {
    initial_management = "11111111-2222-3333-4444-555555555555"
    agent              = "11111111-2222-3333-4444-555555555555"
  }
}

connect { enabled = true }

tls {
  defaults {
    ca_file         = "/tmp/consul-test/consul-agent-ca.pem"
    cert_file       = "/tmp/consul-test/dc-secure-server-consul-0.pem"
    key_file        = "/tmp/consul-test/dc-secure-server-consul-0-key.pem"
    verify_incoming = false
    verify_outgoing = true
  }
  internal_rpc {
    verify_incoming        = true
    verify_server_hostname = true
  }
}

ports { http = 8500 }
EOF

cd /tmp/consul-test
./consul agent -config-file=secure.hcl > /tmp/consul-test/secure.log 2>&1 &
export CONSUL_HTTP_TOKEN=11111111-2222-3333-4444-555555555555
until curl -sf -H "X-Consul-Token: $CONSUL_HTTP_TOKEN" http://127.0.0.1:8500/v1/agent/self -o /dev/null; do sleep 1; done
echo "agent B up on http://127.0.0.1:8500"
```

### Seed agent B with ACL and mesh objects

Without this the token, policy, role and intention inventories are nearly empty
and prove almost nothing.

```bash
A=http://127.0.0.1:8500/v1
H="X-Consul-Token: $CONSUL_HTTP_TOKEN"

# a policy, a role that uses it, and a token that uses both
curl -s -X PUT -H "$H" -d '{"Name":"web-read","Description":"read web","Rules":"service \"web\" { policy = \"read\" }"}' $A/acl/policy
curl -s -X PUT -H "$H" -d '{"Name":"web-role","Description":"web role","Policies":[{"Name":"web-read"}],"ServiceIdentities":[{"ServiceName":"web"}]}' $A/acl/role
curl -s -X PUT -H "$H" -d '{"Description":"app token","Policies":[{"Name":"web-read"}],"Roles":[{"Name":"web-role"}],"ExpirationTTL":"24h"}' $A/acl/token
# a LOCAL token, and one carrying a templated policy and a node identity
curl -s -X PUT -H "$H" -d '{"Description":"local token","Local":true,"NodeIdentities":[{"NodeName":"secure-node","Datacenter":"dc-secure"}]}' $A/acl/token
curl -s -X PUT -H "$H" -d '{"Description":"templated token","TemplatedPolicies":[{"TemplateName":"builtin/service","TemplateVariables":{"Name":"web"}}]}' $A/acl/token

# grant the anonymous token something, so hasGrants can be seen moving
curl -s -X PUT -H "$H" -d '{"Description":"Anonymous Token","Policies":[{"Name":"builtin/global-read-only"}]}' $A/acl/token/00000000-0000-0000-0000-000000000002

# intentions: connection-level allow, wildcard deny, and an L7 one
curl -s -X PUT -H "$H" -d '{"Kind":"service-defaults","Name":"api","Protocol":"http"}' $A/config
curl -s -X PUT -H "$H" -d '{"Kind":"service-intentions","Name":"db","Sources":[{"Name":"web","Action":"allow","Description":"web to db"},{"Name":"*","Action":"deny"}]}' $A/config
curl -s -X PUT -H "$H" -d '{"Kind":"service-intentions","Name":"api","Sources":[{"Name":"web","Permissions":[{"Action":"allow","HTTP":{"PathPrefix":"/public"}}]}]}' $A/config
```

### Optional third target: a CLIENT agent

Needed for two fields whose behaviour on a client agent was **inferred, never
observed**: `server` reading `false`, and `wanGossipEncrypted` reading **null**
because a client runs no WAN gossip pool.

```bash
cd /tmp/consul-test
./consul agent -client 127.0.0.1 -bind 127.0.0.1 -node client-1 \
  -data-dir /tmp/consul-test/data-client -retry-join 127.0.0.1 \
  -http-port 8800 -server-port 9400 -serf-lan-port 9401 -serf-wan-port 9402 \
  -grpc-port 9602 -grpc-tls-port 9603 -dns-port 9700 \
  -encrypt "SAME-KEY-AS-AGENT-B" -datacenter dc-secure > /tmp/consul-test/client.log 2>&1 &
```

Confirm the assumption directly before trusting the resource:

```bash
curl -s http://127.0.0.1:8800/v1/agent/self | python3 -c 'import json,sys; print(sorted(json.load(sys.stdin)["Stats"]))'
```

- [ ] the client agent's `Stats` block contains `serf_lan` but **no** `serf_wan`

If it *does* contain `serf_wan`, the null case in `consul.go:wanGossipEncrypted`
is unreachable and the doc comment on the field is wrong. Say so.

---

## 2. Build the provider under test

`PROVIDERS_PATH` replaces the provider search path entirely, so this never
touches `~/.config/mondoo/providers`.

```bash
cd <repo root>
make providers/build/consul
mkdir -p /tmp/pd/consul && cp providers/consul/dist/consul* /tmp/pd/consul/
ls -la providers/consul/dist/consul     # check the mtime is from THIS build
```

`make providers/install/consul` **copies from `dist/`, it does not build.** A
failed build leaves the previous binary in place and install ships it happily.
Always build first and check the mtime.

Every query below is run as:

```bash
PROVIDERS_PATH=/tmp/pd mql run consul --address http://127.0.0.1:8700 -c '<QUERY>'          # state OPEN
PROVIDERS_PATH=/tmp/pd mql run consul --address http://127.0.0.1:8500 --token $CONSUL_HTTP_TOKEN -c '<QUERY>'   # state HARDENED
```

Shorthand below: `OPEN` and `HARDENED`.

- [ ] `PROVIDERS_PATH=/tmp/pd mql shell consul --address http://127.0.0.1:8700` opens a shell
- [ ] the asset line names the datacenter, e.g. `Consul 127.0.0.1:8700 (dc1)`
- [ ] `mql run consul ... -c 'asset.platform'` reports `consul`

---

## 3. Two-state matrix

These are the fields whose whole purpose is to move. **A field reading the same
in both states is a finding, not a pass** - report it even if the value looks
plausible.

| query | OPEN (agent A) | HARDENED (agent B) |
|---|---|---|
| `consul.acl.enabled` | [ ] `false` | [ ] `true` |
| `consul.acl.defaultPolicy` | [ ] `"allow"` | [ ] `"deny"` |
| `consul.acl.defaultDeny` | [ ] `false` | [ ] `true` |
| `consul.gossipEncrypted` | [ ] `false` | [ ] `true` |
| `consul.wanGossipEncrypted` | [ ] `false` | [ ] `true` |
| `consul.verifyIncoming` | [ ] `false` | [ ] `true` |
| `consul.verifyOutgoing` | [ ] `false` | [ ] `true` |
| `consul.verifyServerHostname` | [ ] `false` | [ ] `true` |
| `consul.mesh.defaultIntentionPolicy` | [ ] `"allow"` | [ ] `"deny"` |
| `consul.mesh.defaultDeny` | [ ] `false` | [ ] `true` |
| `consul.acl.tokens.length` | [ ] `0` | [ ] `>= 5` |
| `consul.acl.policies.length` | [ ] `0` | [ ] `>= 3` |
| `consul.acl.roles.length` | [ ] `0` | [ ] `>= 1` |
| `consul.acl.anonymousToken` | [ ] `null` | [ ] a token |
| `consul.acl.anonymousToken.hasGrants` | [ ] n/a (null) | [ ] `true` |
| `consul.mesh.intentions.length` | [ ] `0` | [ ] `3` |

One more toggle worth doing on agent B, because `downPolicy` is otherwise stuck
at its default: restart it with `acl { down_policy = "deny" }` and confirm
`consul.acl.downPolicy` moves from `"extend-cache"` to `"deny"`.

- [ ] `consul.acl.downPolicy` moves when `down_policy` is changed

And `tokenReplication` / `enableKeyListPolicy`, which were only ever observed as
`false`:

- [ ] add `acl { enable_key_list_policy = true }`, confirm
      `consul.acl.enableKeyListPolicy` becomes `true`

---

## 4. Per-field checklist

Run each query and record the value. Unless stated otherwise, use HARDENED.

### `consul`

| field | query | expected |
|---|---|---|
| [ ] `datacenter` | `consul.datacenter` | `"dc-secure"` |
| [ ] `primaryDatacenter` | `consul.primaryDatacenter` | `"dc-secure"` |
| [ ] `nodeName` | `consul.nodeName` | `"secure-node"` |
| [ ] `nodeId` | `consul.nodeId` | a UUID, stable across two runs |
| [ ] `version` | `consul.version` | `"1.20.1"` |
| [ ] `revision` | `consul.revision` | a short git sha |
| [ ] `buildDate` | `consul.buildDate` | a 2024 timestamp, **not** year 1 |
| [ ] `server` | `consul.server` | `true`; `false` on the client agent |
| [ ] `gossipEncrypted` | `consul.gossipEncrypted` | see matrix |
| [ ] `wanGossipEncrypted` | `consul.wanGossipEncrypted` | see matrix; **null** on the client agent |
| [ ] `verifyIncoming` | `consul.verifyIncoming` | see matrix |
| [ ] `verifyOutgoing` | `consul.verifyOutgoing` | see matrix |
| [ ] `verifyServerHostname` | `consul.verifyServerHostname` | see matrix |
| [ ] `autoEncryptTls` | `consul.autoEncryptTls` | `false`; see below |
| [ ] `autoEncryptAllowTls` | `consul.autoEncryptAllowTls` | `false`; see below |

Both auto-encrypt fields were only ever observed as `false`. Drive them:
add `auto_encrypt { allow_tls = true }` to agent B and `auto_encrypt { tls = true }`
to the client agent.

- [ ] `consul.autoEncryptAllowTls` reads `true` on agent B with `allow_tls` set
- [ ] `consul.autoEncryptTls` reads `true` on the client agent with `tls` set

### `consul.tlsProfile`

```
consul.tlsProfiles { scope verifyIncoming verifyOutgoing verifyServerHostname tlsMinVersion caFile caPath certFile useAutoCert cipherSuites }
```

| check | expected on HARDENED |
|---|---|
| [ ] exactly three rows | `internalRpc`, `https`, `grpc` |
| [ ] `internalRpc.verifyIncoming` | `true` |
| [ ] `internalRpc.verifyServerHostname` | `true` |
| [ ] `https.verifyOutgoing` | `true` (inherited from `defaults`) |
| [ ] `grpc.verifyOutgoing` | **`false`** - it does not inherit; this is the point of the three rows |
| [ ] `tlsMinVersion` | `"TLSv1_2"` |
| [ ] `caFile` | the CA path from `secure.hcl` |
| [ ] `certFile` | the server cert path |
| [ ] `caPath` | `""` - never observed non-empty; set `ca_path` to move it |
| [ ] `cipherSuites` | `[]` - never observed non-empty; set `tls.defaults.tls_cipher_suites` to move it |
| [ ] `useAutoCert` | `false` - never observed `true`; set `tls.grpc.use_auto_cert = true` to move it |
| [ ] no row carries a private key | grep the JSON for `-----BEGIN` and for `"KeyFile"` |

The last one matters: the agent reports `KeyFile` as the literal string
`"hidden"`, and the schema does not carry it at all. Confirm neither the path nor
any key material appears.

### `consul.aclSystem` (reached as `consul.acl`)

```
consul.acl { enabled defaultPolicy downPolicy defaultDeny tokenReplication enableKeyListPolicy }
```

Covered by the matrix above, plus:

- [ ] on OPEN, `consul.acl.tokens` returns `[]` and does **not** error
- [ ] on HARDENED with a token lacking `acl:read`, `consul.acl.tokens` **errors**
      and does not return `[]` (see §5, the absent case)

### `consul.acl.token`

```
consul.acl.tokens { accessorId description local authMethod expirationTime createTime hasGrants isGlobalManagement policies { name } roles { name } serviceIdentities nodeIdentities templatedPolicies }
```

| check | expected |
|---|---|
| [ ] `accessorId` | a UUID per token, all distinct |
| [ ] `description` | matches what was seeded |
| [ ] `local` | `true` for the "local token", `false` for the rest |
| [ ] `authMethod` | `""` for all of these - never observed non-empty, see risks |
| [ ] `expirationTime` | a timestamp on "app token"; **null** on the others |
| [ ] `createTime` | a real timestamp, not year 1 |
| [ ] `hasGrants` | `true` for seeded tokens; create a token with no grants at all and confirm `false` |
| [ ] `isGlobalManagement` | `true` only for the Initial Management Token |
| [ ] `policies` | resolves to `consul.acl.policy` rows with real names, not empty |
| [ ] `roles` | `["web-role"]` on "app token" |
| [ ] `serviceIdentities` | `[]` here; seed one to see `{serviceName, datacenters}` |
| [ ] `nodeIdentities` | `[{nodeName: "secure-node", datacenter: "dc-secure"}]` on "local token" |
| [ ] `templatedPolicies` | `[{templateName: "builtin/service", templateVariables: {name: "web"}, datacenters: []}]` on "templated token" |

The typed-reference resolution is the part most likely to be broken, because it
never ran:

- [ ] `consul.acl.tokens { policies { name rules } }` returns real policy
      documents, not empty rows
- [ ] `consul.acl.tokens { roles { policies { name } } }` traverses two hops
- [ ] the resolution costs **one** `PolicyList` call, not one per token. Run the
      agent with `-log-level=debug` and count `GET /v1/acl/policies` in the log
      while running a query over all tokens. More than one is a regression
      against the caching in `mqlConsulAclSystemInternal`.

### `consul.acl.policy`

```
consul.acl.policies { id name description datacenters isGlobalManagement rules }
```

| check | expected |
|---|---|
| [ ] `id` / `name` | `global-management`, `builtin/global-read-only`, `web-read` all present |
| [ ] `isGlobalManagement` | `true` **only** for `global-management` |
| [ ] `rules` on `web-read` | `service "web" { policy = "read" }` |
| [ ] `rules` on `global-management` | a long HCL document - the built-ins **do** carry rules; an empty string here is a bug |
| [ ] `datacenters` | `[]`, not null |

### `consul.acl.role`

```
consul.acl.roles { id name description policies { name } serviceIdentities nodeIdentities templatedPolicies }
```

- [ ] `web-role` present with `policies` resolving to `web-read`
- [ ] `serviceIdentities` shows `{serviceName: "web", datacenters: []}`

### `consul.serviceMesh` (reached as `consul.mesh`)

```
consul.mesh { enabled defaultIntentionPolicy defaultDeny }
```

- [ ] `enabled` is `true` on both agents (dev mode enables Connect; `secure.hcl`
      sets it explicitly). Restart agent B with `connect { enabled = false }` and
      confirm it moves to `false`.

### `consul.mesh.intention`

```
consul.mesh.intentions { sourceName sourceNamespace sourcePartition sourcePeer sourceType destinationName destinationNamespace destinationPartition action description precedence hasWildcard hasL7Permissions permissions meta createdAt updatedAt }
```

| check | expected |
|---|---|
| [ ] three rows | `web->db`, `*->db`, `web->api` |
| [ ] `web->db` | `action: "allow"`, `precedence: 9`, `description: "web to db"` |
| [ ] `*->db` | `action: "deny"`, `precedence: 8`, `hasWildcard: true` |
| [ ] `web->api` | `action: ""`, `hasL7Permissions: true` |
| [ ] `web->api.permissions` | `[{Action: "allow", HTTP: {PathPrefix: "/public"}}]` |
| [ ] `sourceNamespace` / `destinationNamespace` | `"default"` |
| [ ] `sourcePartition` / `sourcePeer` | `""` on Community Edition |
| [ ] `sourceType` | `"consul"` |
| [ ] `createdAt` / `updatedAt` | **null** - the listing carries no timestamps; a year 1 date is a bug |
| [ ] `meta` | `{}` |
| [ ] all three `__id` values are distinct | `consul.mesh.intentions { __id }` |

The `__id` dimensions matter more than they look: if two intentions collide, the
second silently reports the first one's `action`, and a deny reads as an allow.
The unit tests cover the dimensions in isolation; confirm the ids are distinct
against real data too.

---

## 5. The four checks from new-resource §5

### 5.1 The absent case must FAIL, not pass vacuously

Three separate absences to drive. In each, the check must **error or report
false**, never return an empty result that a policy would read as satisfied.

**(a) Nothing listening.**

```bash
PROVIDERS_PATH=/tmp/pd mql run consul --address http://127.0.0.1:9999 -c 'consul.acl.enabled'
```

- [ ] this fails with a connection error, and does **not** report `false`

**(b) ACLs switched off** (agent A). The endpoints genuinely do not exist, so an
empty inventory is honest here - but the *enabled* flag must be the thing that
fails a check:

```bash
PROVIDERS_PATH=/tmp/pd mql run consul --address http://127.0.0.1:8700 -c 'consul.acl.enabled && consul.acl.defaultDeny'
```

- [ ] reports `false` (not null, not true)
- [ ] `consul.acl.tokens` returns `[]` with no error

**(c) A token that authenticates but may not read.** This is the one that
matters most: a permission failure must **never** degrade to an empty list,
because a policy written as `consul.acl.tokens.none(isGlobalManagement)` would
then pass on a datacenter full of management tokens.

```bash
# a token with no acl:read
WEAK=$(curl -s -X PUT -H "X-Consul-Token: $CONSUL_HTTP_TOKEN" \
  -d '{"Description":"weak","Policies":[{"Name":"web-read"}]}' \
  http://127.0.0.1:8500/v1/acl/token | python3 -c 'import json,sys; print(json.load(sys.stdin)["SecretID"])')

PROVIDERS_PATH=/tmp/pd mql run consul --address http://127.0.0.1:8500 --token "$WEAK" -c 'consul.acl.tokens'
```

- [ ] this **errors** (403 Permission denied), and does **not** return `[]`
- [ ] `consul.acl.policies` with the weak token also errors
- [ ] `consul.datacenter` with the weak token errors too (it lacks `agent:read`)

**(d) An unrecognized token.**

```bash
PROVIDERS_PATH=/tmp/pd mql run consul --address http://127.0.0.1:8500 --token 99999999-9999-9999-9999-999999999999 -c 'consul.acl.tokens'
```

- [ ] errors with `ACL not found`, does **not** return `[]`

### 5.2 Null over invented defaults

- [ ] `consul.acl.anonymousToken` is **null** on agent A, not a blank token
- [ ] `expirationTime` is **null** on a non-expiring token, not year 1
- [ ] `createdAt` / `updatedAt` on intentions are **null**, not year 1
- [ ] `buildDate` is a real 2024 date, not year 1
- [ ] `wanGossipEncrypted` is **null** on a client agent, not `false`
- [ ] `consul.acl.policies { rules }` on a policy whose document was deleted
      mid-query reports null rather than `""`

Record any field where the provider reports a value the agent never sent.

### 5.3 Secret sweep must return 0

The Consul API hands the **secret half of every ACL token** to a caller holding a
management token, in plaintext, from `/v1/acl/tokens`. It also serves the gossip
encryption key in plaintext from `/v1/operator/keyring`. Neither may appear
anywhere in mql output.

```bash
# the management token itself
PROVIDERS_PATH=/tmp/pd mql run consul --address http://127.0.0.1:8500 --token $CONSUL_HTTP_TOKEN \
  -c 'consul { acl { tokens { * } policies { * } roles { * } } }' -j > /tmp/consul-test/dump.json

grep -c "$CONSUL_HTTP_TOKEN" /tmp/consul-test/dump.json          # must be 0
grep -c "$WEAK" /tmp/consul-test/dump.json                        # must be 0
grep -ci "secretid" /tmp/consul-test/dump.json                    # must be 0
grep -c "$(grep encrypt /tmp/consul-test/secure.hcl | cut -d'"' -f2)" /tmp/consul-test/dump.json   # gossip key: must be 0
```

- [ ] all four counts are 0
- [ ] a full dump (`consul { * }` with every sub-resource expanded) also returns 0
- [ ] no private key path or PEM appears: `grep -c -e '-----BEGIN' -e 'KeyFile' /tmp/consul-test/dump.json` is 0

State the guarantee precisely afterwards. "The `consul.acl.token` resource
carries no secret" is true. "The token secret cannot be obtained" is false: a
management token still reaches the Consul API directly.

### 5.4 Composability

- [ ] `consul.acl.tokens { policies { rules } }` - two hops, real documents
- [ ] `consul.acl.tokens { roles { policies { name } } }` - three hops
- [ ] `consul.acl.tokens.where(isGlobalManagement) { accessorId }` - filter on a
      derived predicate
- [ ] `consul.mesh.intentions.where(hasWildcard && action == "allow")` - filter
      on two derived fields
- [ ] `consul.tlsProfiles.where(scope == "grpc") { verifyOutgoing }` - selection
      by key
- [ ] `consul.acl.tokens.all(expirationTime != null)` - reports `false` here,
      which is the direction that catches a never-expiring token

---

## 6. Risk areas

Ranked by how likely a bug is to be both present and silent. This is the
section to read first if you are short on time.

### 6.1 Nothing has run through mql at all

No MQL query has executed against this provider. The whole `.lr` schema to Go
resource wiring is unverified: null-state handling (`plugin.StateIsSet |
StateIsNull` on `wanGossipEncrypted`, `buildDate`, `anonymousToken`, `rules`),
`[]dict` rendering of the identity blocks, typed-reference resolution, and
`__id` uniqueness in the runtime cache. Every one of these produces a *plausible
wrong value or a null*, not an error, when it is wrong. Start at §3.

### 6.2 The Consul < 1.12 fallback is inferred, not observed

`agentself.go` falls back to top-level `VerifyIncoming` / `VerifyOutgoing` /
`VerifyServerHostname` / `ACLDefaultPolicy` / `ACLDownPolicy` in `DebugConfig`
when the per-layer `TLS` block is absent. **Only Consul 1.20.1 was ever
observed**, and it has the `TLS` block, so the fallback never runs against
anything I saw. The key names for older agents come from the Consul changelog,
not from a payload.

If those names are wrong on an old agent, `verifyIncoming` and friends read
`false` and `defaultPolicy` reads `""` - silently, with no error, on exactly the
agents most likely to be misconfigured.

- [ ] run against a Consul 1.10.x or 1.11.x agent and confirm the three verify
      fields and `defaultPolicy` are correct. If you cannot, at minimum capture
      its `/v1/agent/self` and diff the `DebugConfig` key names against
      `agentself.go`.

### 6.3 `mesh.defaultIntentionPolicy` is derived, not read

Consul Community Edition has no setting for the mesh default; it follows the ACL
default policy. That is what `defaultIntentionPolicy` computes. I confirmed
directly that Consul **1.20.1 CE rejects** `DefaultIntentionPolicy` on the `mesh`
config entry (`invalid config key "DefaultIntentionPolicy"`), and the
`consul/api` SDK v1.34.4 `MeshConfigEntry` type has no such field.

**On Consul Enterprise, `default_intention_policy` on the `mesh` config entry
does exist and overrides the ACL default policy.** Against such a cluster this
field is simply wrong, and wrong in the unsafe direction if the operator set it
to `allow` while ACLs deny.

- [ ] against Consul Enterprise, set `default_intention_policy` on the mesh
      config entry and check whether `consul.mesh.defaultIntentionPolicy`
      disagrees with reality. If it does, the fix is to read the mesh config
      entry raw (the SDK type drops the field) and prefer it when present.

There is also no test that the *actual* authorization behaviour matches: no
sidecar proxy was ever run.

### 6.4 The `wanGossipEncrypted` null case is inferred

Only server agents were observed, and both reported `serf_wan`. The claim that a
client agent has no `serf_wan` block - and therefore that the field should be
null rather than `false` - comes from how Consul's WAN pool works, not from a
captured payload. §1 has the command to check it in one line. If a client agent
*does* report `serf_wan`, the null branch is dead code and the field doc is
wrong.

### 6.5 Fields never observed with a non-default value

Each of these was only ever seen empty or `false`, so a mapping bug in them is
invisible so far. §4 says how to drive each one:

`authMethod`, `templatedPolicies`, `nodeIdentities`, `serviceIdentities` on
tokens, `local`, `autoEncryptTls`, `autoEncryptAllowTls`, `tokenReplication`,
`enableKeyListPolicy`, `tlsProfile.caPath`, `tlsProfile.cipherSuites`,
`tlsProfile.useAutoCert`, `intention.meta`, `intention.createdAt`/`updatedAt`,
`policy.datacenters`, and `sourceType` (only ever `"consul"`).

### 6.6 Enterprise tenancy is not modelled

Namespaces, admin partitions and cluster peers appear as fields on intentions
(`sourceNamespace`, `sourcePartition`, `sourcePeer`) and are part of the
intention `__id`, but no Enterprise cluster was available and all three were
always `"default"` or `""`. The connection layer has no `--namespace` or
`--partition` flag, so an Enterprise cluster is read in its default namespace
only. Tokens, policies and roles in other namespaces are invisible, and the
provider says nothing about that.

- [ ] against Consul Enterprise, confirm whether the default-namespace-only
      reading is acceptable or whether it needs to be surfaced as a limitation
      in the README

### 6.7 The ACL-disabled classifier matches on a message body

`isACLSystemDisabled` requires HTTP 401 **and** a body containing
`ACL support disabled`. The status and the exact string were observed on 1.20.1.
Requiring the body is deliberate: a bare 401 from a proxy in front of the agent
must not read as "ACLs are off, here is an empty inventory". The cost is that if
Consul changes the wording, the provider errors instead of returning `[]` on an
ACL-less agent. That is the safe direction, but it is a real behaviour change to
watch for on new Consul versions.

Note also that the classifier is belt and braces: `tokens()`, `policies()` and
`roles()` first check `consul.acl.enabled` from the agent configuration and
return `[]` without calling the endpoint at all. The classifier only fires if
those two sources disagree.

- [ ] confirm on the newest Consul release you can reach that a disabled ACL
      system still answers 401 with that body

### 6.8 Address handling edge cases

`NormalizeAddress` fills in the default port (8500 for http, 8501 for https) so
two spellings of one agent produce one asset. It **rejects `unix://`**, which is
a legitimate Consul deployment, because a socket path carries no host for the
asset identity. It keeps a path prefix in the address (Consul's client supports a
reverse-proxy prefix) but derives the host from `host:port` only - untested
against an actual reverse-proxied agent.

- [ ] confirm behaviour behind a reverse proxy, e.g.
      `--address https://proxy.example.com/consul`

### 6.9 The request timeout may be dropped for unusual addresses

The 60s timeout is attached by building the HTTP client before `api.NewClient`.
`NewClient` replaces that client entirely for a `unix://` address - unreachable
today because `NormalizeAddress` rejects unix, but it would become reachable the
moment somebody adds unix support. Not exercised.

### 6.10 Policy rules cost one call per policy

`consul.acl.policy.rules` calls `PolicyRead` per policy, because the list
endpoint does not carry rules. This is lazy - it only happens when `rules` is
actually selected - but a query over a datacenter with hundreds of policies makes
hundreds of calls.

- [ ] measure `consul.acl.policies { rules }` against a datacenter with many
      policies and decide whether it needs batching

---

## 7. Reporting back

For each section, record what you ran and what came back, not what you expected.
A table of observed values is what lets a reviewer disagree with you about
something real.

Call out, specifically:

1. Any field in §3 that read the same in both states.
2. Any check in §5.1 that returned an empty result instead of an error.
3. Any non-zero count in §5.3.
4. Whether the client-agent assumption in §6.4 held.
5. Anything in §6 you were unable to test, and why.

### Tear down

```bash
pkill -f 'consul agent'
rm -rf /tmp/consul-test /tmp/pd
```
