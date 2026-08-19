# Rancher provider: live verification hand-off

**Status: no field in this provider has ever been read from a running Rancher
server.** Everything below is unverified. The schema was derived from the
generated Rancher v3 API client (see
`resources/testdata/README.md`), which is machine generated from the schema the
server actually serves, so the *field names* are on solid ground. The *values*,
and every derived predicate computed from them, are not.

This document is written to be executed by somebody with no prior context. Work
top to bottom.

---

## 0. What a mock server already established (and what it did not)

Before writing this, the provider was run end to end through the real `mql`
runtime against a **mock Rancher server** that replays the fixtures in
`resources/testdata/`. That is not live verification and does not tick a single
box in §2, because a mock built from the same source as the schema agrees with
the schema by construction. It does, however, rule out a whole class of failure
that would otherwise waste the first hour of a live session:

- every resource resolves through the plugin boundary; no field came back as
  `provider returned no data and no error`, and no `__id` collided;
- the pagination walk crossed a two-page `/v3/clusters` and returned all three
  clusters in order;
- every typed accessor resolves: `cluster.podSecurityAdmissionTemplate`,
  `cluster.defaultClusterRoleForProjectMembers`, `project.cluster`,
  `token.user`, `token.cluster`, `roleTemplate.inheritedRoleTemplates`,
  `globalRole.bindings`, `podSecurityAdmissionConfigurationTemplate.clusters`,
  `registryCredential.project.cluster`;
- reverse edges filter correctly rather than returning everything;
- **no N+1**: a query resolving users through two bindings issued exactly one
  `GET /v3/users`; the whole surface cost 11 requests;
- nulls land where they should: `kubernetesVersion` on a provisioning cluster,
  `enableNetworkPolicy` on an imported cluster, `expiresAt` on a non-expiring
  token, `lastUsedAt` on an unused one, `enforceVersion` on an unpinned
  template, `user` on a binding naming a group;
- the secret sweep over `tokens { * } users { * } registryCredentials { * }
  clusters { * } settings { * } authProviders { * }` returned **0** for every
  planted secret;
- a `/v3/clusterTemplates` that answers 404 makes
  `clusterTemplateEnforcement` read null even with the setting at `true`, and
  `rancher.clusterTemplateEnforcement == false` **fails** rather than passing;
- an unreachable server makes `rancher.clusters.length == 0` **error** rather
  than report true.

What the mock cannot tell you is whether any of those field *values* match what
a real Rancher sends. That is all of §2.

If you want the mock for a first smoke test after a change, it is roughly 150
lines of stdlib Go that reads the fixture files and serves them at the `/v3/…`
paths; rebuilding it is faster than finding it.

---

## 1. Stand the target up

### 1.1 The docker prerequisite

**This was attempted and failed.** In the environment the provider was written
in, `dockerd` was running and the Docker Hub *manifest* was reachable, but the
CDN that serves image *blobs* (`production.cloudfront.docker.com`) is blocked by
the outbound proxy, so the pull died partway:

```
failed to copy: httpReadSeeker: failed open: ... production.cloudfront.docker.com ... Forbidden
```

`registry.suse.com/rancher/rancher` was blocked at the manifest stage too. So
the first thing to check on a new machine is simply whether the image comes
down:

```bash
docker info >/dev/null 2>&1 || { sudo dockerd >/tmp/dockerd.log 2>&1 & sleep 8; }
docker pull rancher/rancher:v2.11.3     # ~1.5 GB compressed
```

If that fails, use a machine with unrestricted registry access, or a cloud VM.

### 1.2 Run Rancher

`rancher/rancher` boots an embedded k3s and needs a privileged container. Budget
5 to 10 minutes to first login and roughly 8 GB of RAM.

```bash
docker run -d --restart=unless-stopped \
  --privileged \
  -p 8443:443 \
  --name rancher \
  rancher/rancher:v2.11.3

# wait for it, then read the bootstrap password
docker logs rancher 2>&1 | grep -i "Bootstrap Password"
```

Open `https://localhost:8443`, complete setup, and **set the server URL** when
prompted (an unset `server-url` is itself one of the two states in §3).

> Pick **2.11.x**, not 2.12+. Cluster templates were removed in 2.12, and this
> checklist needs a version that still has them. Once 2.11 is fully checked,
> repeat §2 against a 2.12 server to confirm the absent-feature behavior in §4.

### 1.3 Mint an API token

Console → top-right avatar → **Account & API Keys** → **Create API Key**, with
**no scope** and **no expiry**. Copy the whole `token-xxxxx:yyyyy` value; it is
shown once.

```bash
export RANCHER_URL=https://localhost:8443
export RANCHER_TOKEN='token-xxxxx:yyyyy'
```

The container serves a self-signed certificate, so add `--tls-skip-verify` to
every command below (and see §4 for why that flag itself needs testing).

### 1.4 The minimum fixture

The checklist needs all of these to exist before it can distinguish a working
field from an empty one:

| # | fixture | how |
|---|---|---|
| F1 | **a downstream cluster** | Cluster Management → Create → **Custom**. Register one extra node (a second VM, or a nested docker host). A single-node import is enough; it does not need to reach `active` for most fields, but `kubernetesVersion` and `nodeCount` need it to. |
| F2 | **a project other than the defaults** | Cluster → Projects/Namespaces → Create Project, name it `audit-test`. |
| F3 | **a project resource quota** | On `audit-test`, set a CPU limit and a pod limit. Without one, `project.resourceQuota` cannot be told from a bug. |
| F4 | **a custom role template with escalating rules** | Users & Authentication → Role Templates → Create, context **Project**, with a rule granting `escalate` on `roles` in `rbac.authorization.k8s.io`. |
| F5 | **a custom role template with wildcard rules** | Same, but `*` verbs on `*` resources in `*` API groups. |
| F6 | **a role template that inherits another** | A template whose "Inherit from" names F4. |
| F7 | **a non-expiring API token** | The key from §1.3. |
| F8 | **an expiring API token** | A second key with a 1-day expiry. |
| F9 | **a group binding** | Needs an external auth provider (F12). Otherwise skip and mark `subjectKind == "group"` unverified. |
| F10 | **a pod security admission template applied to a cluster** | Cluster Management → Advanced → Pod Security Admissions. Apply `rancher-restricted` to the downstream cluster from F1. Also create a **custom** one with an `enforce-version` pinned and an exempt namespace, so §2's version fields have something to read. |
| F11 | **a cluster template with two revisions** | Cluster Management → RKE1 Configuration → Cluster Templates. One revision `enabled`, one disabled, one marked default, and at least one question. |
| F12 | **an external auth provider** | Easiest realistic option is a throwaway GitHub OAuth app, or Keycloak in a second container. Needed for `authProvider` values other than `local`, and for F9. |
| F13 | **a registry credential** | Project → Storage → Secrets → Registry Credential. Create one **project-scoped** and one **namespace-scoped**, each with a username and password, against `harbor.example.com`. Both are needed for §2's `scope` field and for the secret sweep in §4. |
| F14 | **a second cluster** | Any second cluster, even an import that never connects, so that fleet-wide `.where(...)` queries return a subset rather than everything. |

---

## 2. Per-field checklist

Build the provider under test first (§5). Every row is **unchecked**. Fill in
the observed value; a row is only done when the observed value has been
*compared to what the console shows*, not merely seen to be non-null.

Run the whole surface once first, to catch anything that errors outright:

```bash
PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify -j -c '{
  version serverUrl telemetryOptIn clusterTemplateEnforcement
  authTokenMaxTtlMinutes kubeconfigDefaultTokenTtlMinutes passwordMinLength
  localAuthEnabled externalAuthEnabled
}'
```

### 2.1 `rancher` (root)

| ✓ | field | query | expected |
|---|---|---|---|
| ☐ | `version` | `rancher.version` | matches the version in the console footer, e.g. `v2.11.3` |
| ☐ | `serverUrl` | `rancher.serverUrl` | matches Global Settings → `server-url` |
| ☐ | `telemetryOptIn` | `rancher.telemetryOptIn` | **null** on 2.10+; `"in"`/`"out"`/`""` on 2.9 |
| ☐ | `clusterTemplateEnforcement` | `rancher.clusterTemplateEnforcement` | `false` on 2.11 default; see §3 for the on-state |
| ☐ | `authTokenMaxTtlMinutes` | `rancher.authTokenMaxTtlMinutes` | `129600` by default |
| ☐ | `kubeconfigDefaultTokenTtlMinutes` | `rancher.kubeconfigDefaultTokenTtlMinutes` | `43200` by default |
| ☐ | `passwordMinLength` | `rancher.passwordMinLength` | `12` by default |
| ☐ | `localAuthEnabled` | `rancher.localAuthEnabled` | `true` |
| ☐ | `externalAuthEnabled` | `rancher.externalAuthEnabled` | `false` before F12, `true` after |
| ☐ | `settings` | `rancher.settings.length` | roughly 90 to 130 entries; not 0 |
| ☐ | `clusters` | `rancher.clusters { id name }` | `local` plus F1 and F14 |
| ☐ | `projects` | `rancher.projects { id name }` | Default + System per cluster, plus F2 |
| ☐ | `globalRoles` | `rancher.globalRoles.length` | roughly 15 to 30 |
| ☐ | `globalRoleBindings` | `rancher.globalRoleBindings.length` | at least 1 |
| ☐ | `roleTemplates` | `rancher.roleTemplates.length` | roughly 25 to 40, plus F4/F5/F6 |
| ☐ | `clusterRoleTemplateBindings` | `... .length` | at least 1 per cluster |
| ☐ | `projectRoleTemplateBindings` | `... .length` | at least 1 per project |
| ☐ | `clusterTemplates` | `rancher.clusterTemplates { name }` | F11 on 2.11; **empty** on 2.12+ |
| ☐ | `podSecurityAdmissionConfigurationTemplates` | `... { name }` | `rancher-restricted`, `rancher-privileged`, plus F10's custom one |
| ☐ | `authProviders` | `rancher.authProviders { id type enabled }` | every provider Rancher knows, most disabled |
| ☐ | `tokens` | `rancher.tokens.length` | F7 + F8 + the console session |
| ☐ | `users` | `rancher.users { id username }` | `admin` plus the system accounts |
| ☐ | `registryCredentials` | `rancher.registryCredentials { name scope }` | both halves of F13 |

### 2.2 `rancher.cluster`

Query: `rancher.clusters { <field> }`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id` | `local` for the management cluster, `c-m-…` for downstream |
| ☐ | `name` | the **display** name from the console, not the `c-m-…` id |
| ☐ | `description` | matches the console |
| ☐ | `state` | `active` |
| ☐ | `transitioning` | `no` when settled, `yes` mid-provision |
| ☐ | `transitioningMessage` | empty when settled; non-empty mid-provision |
| ☐ | `driver` | `imported` / `rke2` / `k3s` / `EKS` … |
| ☐ | `provider` | `k3s` for the local cluster of a docker install |
| ☐ | `kubernetesVersion` | e.g. `v1.31.4+k3s1`; **null**, not `""`, on a cluster still provisioning |
| ☐ | `isLocalCluster` | `true` only for `local` |
| ☐ | `internal` | `true` for `local` |
| ☐ | `created` | a real date, matching the console |
| ☐ | `nodeCount` | matches the node list |
| ☐ | `enableNetworkPolicy` | **null** on an imported cluster; `true`/`false` on an RKE2 one. Toggle it (§3) |
| ☐ | `appliedEnableNetworkPolicy` | lags `enableNetworkPolicy` while a change rolls out |
| ☐ | `localClusterAuthEndpointEnabled` | toggle it (§3) |
| ☐ | `localClusterAuthEndpointFqdn` | the FQDN typed into the console, `""` when unset |
| ☐ | `fleetWorkspaceName` | `fleet-local` for `local`, `fleet-default` otherwise |
| ☐ | `apiEndpoint` | the `…/k8s/clusters/<id>` proxy URL |
| ☐ | `labels` | includes `provider.cattle.io` |
| ☐ | `annotations` | includes `field.cattle.io/description` when a description was set |
| ☐ | `podSecurityAdmissionTemplate` | resolves to F10's template on the cluster it was applied to; **null** elsewhere |
| ☐ | `defaultClusterRoleForProjectMembers` | resolves to a role template, or null |
| ☐ | `projects` | only the projects of *that* cluster, not every project |
| ☐ | `roleTemplateBindings` | only the bindings of *that* cluster |

### 2.3 `rancher.project`

Query: `rancher.projects { <field> }`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id` | `<clusterId>:<projectId>` |
| ☐ | `name` | display name (`Default`, `System`, `audit-test`) |
| ☐ | `description`, `state`, `created` | match the console |
| ☐ | `backingNamespace` | a namespace that exists in the local cluster |
| ☐ | `labels`, `annotations` | match the console |
| ☐ | `resourceQuota` | populated on F3, **null** on a project with no quota |
| ☐ | `containerDefaultResourceLimit` | **null** when unset |
| ☐ | `cluster` | resolves, and `cluster.id` is the first half of `project.id` |
| ☐ | `roleTemplateBindings` | only this project's bindings |

### 2.4 `rancher.globalRole`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id` | `admin`, `user`, `restricted-admin`, … |
| ☐ | `name` | `Administrator`, `Standard User`, … (display names) |
| ☐ | `description`, `created` | match the console |
| ☐ | `builtin` | `true` for shipped roles |
| ☐ | `newUserDefault` | `true` for exactly the roles the console marks as default |
| ☐ | `rules` | for `admin`, one rule of `*`/`*`/`*` |
| ☐ | `namespacedRules`, `inheritedNamespacedRules` | null on most roles |
| ☐ | `grantsFullAdmin` | `true` for `admin`, `false` for `user` |
| ☐ | `grantsPrivilegeEscalation` | `true` for `admin` |
| ☐ | `inheritedClusterRoleTemplates` | resolves for `user` (usually `cluster-member` or similar) |
| ☐ | `bindings` | only the bindings of that role |

### 2.5 `rancher.globalRoleBinding`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id`, `created` | match the console |
| ☐ | `subjectKind` | `user` for the admin binding; `group` after F9 |
| ☐ | `subjectName` | `u-…` for a user; the group principal for a group |
| ☐ | `userPrincipalId` | `local://u-…` for a local user |
| ☐ | `globalRole` | resolves; `globalRole.id` matches the console |
| ☐ | `user` | resolves for a user binding; **null** for a group binding |

### 2.6 `rancher.roleTemplate`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id`, `name`, `description`, `created` | match the console |
| ☐ | `context` | `cluster` or `project`; `""` on templates only ever inherited |
| ☐ | `builtin` | `true` for shipped templates, `false` for F4/F5 |
| ☐ | `external` | `false` on all of the above |
| ☐ | `hidden`, `locked` | match the console's Role Templates list |
| ☐ | `clusterCreatorDefault` | `true` for `cluster-owner` |
| ☐ | `projectCreatorDefault` | `true` for `project-owner` |
| ☐ | `rules` | for `cluster-owner`, includes `*`/`*`/`*` |
| ☐ | `externalRules` | empty unless a template is external |
| ☐ | `administrative` | deprecated; note what it reads on F5, which grants everything |
| ☐ | `grantsFullAdmin` | `true` on F5 and `cluster-owner`, `false` on F4 |
| ☐ | `grantsPrivilegeEscalation` | `true` on F4, `false` on `cluster-member` |
| ☐ | `inheritedRoleTemplates` | resolves to F4 from F6 |

### 2.7 `rancher.clusterRoleTemplateBinding` / `rancher.projectRoleTemplateBinding`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id` | `<clusterId>:crtb-…` / `<projectId>:prtb-…` |
| ☐ | `created` | matches the console |
| ☐ | `subjectKind`, `subjectName`, `userPrincipalId` | as in §2.5 |
| ☐ | `cluster` (CRTB) | resolves to the right cluster |
| ☐ | `project` (PRTB) | resolves to the right project |
| ☐ | `roleTemplate` | resolves; matches the role shown in the console's Members tab |
| ☐ | `user` | resolves, or null for a group binding |

### 2.8 `rancher.clusterTemplate` and `.revision` (Rancher ≤ 2.11 only)

| ✓ | field | expected |
|---|---|---|
| ☐ | `id`, `name`, `description`, `created` | match F11 |
| ☐ | `members` | one entry per member, with `accessType` |
| ☐ | `revisions` | both revisions of F11, and none from another template |
| ☐ | `defaultRevision` | the revision marked default; **null** when none is |
| ☐ | revision `id`, `name`, `created`, `state` | match the console |
| ☐ | revision `enabled` | `false` on the disabled revision; check whether the enabled one reads `true` or **null** (Rancher may only write the field when disabling) |
| ☐ | revision `questions` | one entry per question, with `variable` and `default` |
| ☐ | revision `clusterTemplate` | resolves back to the parent |

### 2.9 `rancher.podSecurityAdmissionConfigurationTemplate`

| ✓ | field | expected |
|---|---|---|
| ☐ | `name`, `description`, `created` | match the console |
| ☐ | `enforce` | `restricted` on `rancher-restricted`, `privileged` on `rancher-privileged` |
| ☐ | `enforceVersion` | **the pinned version** on F10's custom template; **null** when unpinned. This is the hyphenated-key trap; if it reads null on a template that pins a version, the tag is wrong |
| ☐ | `audit`, `auditVersion`, `warn`, `warnVersion` | same |
| ☐ | `exemptNamespaces` | matches F10's exemption list |
| ☐ | `exemptUsernames`, `exemptRuntimeClasses` | empty unless set |
| ☐ | `clusters` | only the clusters F10 was applied to; **empty** for the template that was not applied |

### 2.10 `rancher.authProvider`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id` | `local`, `activedirectory`, `github`, `oidc`, … |
| ☐ | `type` | `localConfig`, `githubConfig`, `activeDirectoryConfig`, … |
| ☐ | `enabled` | `true` for `local` and F12; `false` for the rest |
| ☐ | `accessMode` | `unrestricted` by default; change it to `restricted` and re-read |
| ☐ | `allowedPrincipalIds` | populated once the access mode is restricted |
| ☐ | `isLocalProvider` | `true` for exactly one entry |
| ☐ | `logoutAllSupported` | `true` on SAML/OIDC providers |

### 2.11 `rancher.token`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id` | `token-…` for an API key; `kubeconfig-…` for a downloaded kubeconfig |
| ☐ | `userId` | `u-…` |
| ☐ | `description` | the description typed at creation (F7) |
| ☐ | `authProvider` | `local` |
| ☐ | `isDerived` | `false` on F7/F8, `true` on the console session |
| ☐ | `enabled` | `true`; check whether an absent flag reads null |
| ☐ | `expired` | `false` on F7/F8 |
| ☐ | `current` | `true` on exactly one token, the one this connection used |
| ☐ | `ttlMillis` | `0` on F7; the configured lifetime in **milliseconds** on F8 |
| ☐ | `expiresAt` | **null** on F7; a real date on F8. If it reads year 1 anywhere, the parse is wrong |
| ☐ | `lastUsedAt` | a recent date on the token this run authenticated with; **null** on a token never used |
| ☐ | `activityLastSeenAt` | null unless token activity tracking is on |
| ☐ | `created` | matches the console |
| ☐ | `neverExpires` | `true` on F7, `false` on F8 |
| ☐ | `user` | resolves to `admin` |
| ☐ | `cluster` | **null** on an unscoped token; resolves on a downloaded kubeconfig token |

### 2.12 `rancher.user`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id`, `username`, `name`, `description` | match the console |
| ☐ | `enabled` | `true`; disable a test user and re-read |
| ☐ | `mustChangePassword` | `true` on a freshly created local user |
| ☐ | `created` | matches the console |
| ☐ | `principalIds` | `local://u-…`; a second entry after signing in through F12 |
| ☐ | `isSystemUser` | `true` for the `u-…` service accounts, `false` for `admin` |
| ☐ | `globalRoleBindings` | only that user's bindings |

### 2.13 `rancher.registryCredential`

| ✓ | field | expected |
|---|---|---|
| ☐ | `id`, `name`, `description`, `created` | match F13 |
| ☐ | `scope` | `project` on the project-scoped one, `namespace` on the other |
| ☐ | `namespaceName` | `""` on the project-scoped one, the namespace on the other |
| ☐ | `registries` | `["harbor.example.com"]` |
| ☐ | `registryUsernames` | the username typed at creation |
| ☐ | `project` | resolves to the right project |

### 2.14 `rancher.setting`

| ✓ | field | expected |
|---|---|---|
| ☐ | `name` | the setting key |
| ☐ | `value` | the value in force; `""` on a setting nobody changed |
| ☐ | `defaultValue` | the shipped default |
| ☐ | `customized` | `true` on `server-url` after setup |
| ☐ | `source` | `db` on a changed setting; `env` on one pinned by an environment variable |

---

## 3. Two-state requirement

A field that reads the same before and after the setting is changed cannot tell
a resource bug from a fixture that never moved. Drive each of these both ways
and record **both** readings.

| # | control | off state | on state | field |
|---|---|---|---|---|
| S1 | Global Settings → `cluster-template-enforcement` | `false` | `true` | `rancher.clusterTemplateEnforcement` |
| S2 | Auth provider F12 | disabled | enabled | `rancher.externalAuthEnabled`, `authProviders.where(id == "github").enabled` |
| S3 | local auth alongside F12 | local only | local **and** F12 both enabled | `rancher.localAuthEnabled` must stay `true` while `externalAuthEnabled` becomes `true` |
| S4 | API key expiry | F7 (no expiry) | F8 (1 day) | `token.neverExpires`, `token.ttlMillis`, `token.expiresAt` |
| S5 | Authorized cluster endpoint | off | on, with an FQDN | `cluster.localClusterAuthEndpointEnabled`, `…Fqdn` |
| S6 | Project network isolation | off | on | `cluster.enableNetworkPolicy`, `cluster.appliedEnableNetworkPolicy` |
| S7 | Pod security template on a cluster | none | `rancher-restricted` | `cluster.podSecurityAdmissionTemplate`, `template.clusters` |
| S8 | Auth provider access mode | `unrestricted` | `restricted` with a principal | `authProvider.accessMode`, `allowedPrincipalIds` |
| S9 | Cluster template revision | enabled | disabled | `revision.enabled` |
| S10 | A user account | enabled | disabled | `user.enabled` |
| S11 | Project quota | none | CPU + pod limits | `project.resourceQuota` |
| S12 | `server-url` | unset (pre-setup) | set | `rancher.serverUrl`, `setting.customized` |

**S1 deserves extra care.** It is the field whose wrong answer is worst (see
§6). Confirm the reading flips, and confirm that on a **2.12** server it reads
**null** rather than `false`.

---

## 4. The four checks from `new-resource` §5

### 4.1 Absent case must FAIL, not pass vacuously

Point the provider at something that is not Rancher and confirm every check
errors rather than passing:

```bash
# a server that answers, but is not Rancher
PROVIDERS_PATH=/tmp/pd mql run rancher --url https://example.com --token 'token-x:y' \
  -c 'rancher.clusters.length == 0'          # must ERROR, not report true

# a Rancher that refuses the token
PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify --token 'token-bogus:bogus' \
  -c 'rancher.globalRoles.none(grantsFullAdmin)'   # must ERROR, not pass

# an unreachable host
PROVIDERS_PATH=/tmp/pd mql run rancher --url https://127.0.0.1:1 --token 'token-x:y' \
  -c 'rancher.clusterTemplateEnforcement'    # must ERROR
```

Also confirm on a **2.12** server:

```bash
PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify \
  -c 'rancher.clusterTemplateEnforcement == false'
```

This must **not** be `true`. The field is null on 2.12, so the comparison should
report null / fail. If it reports `true`, the null handling regressed and a
fleet with no such control is being reported as one that has it and left it off.

### 4.2 Null over invented defaults

Confirm each of these is **null**, not a zero value:

```bash
PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify -j -c \
  'rancher.tokens.where(neverExpires) { id expiresAt lastUsedAt }'
```

- `expiresAt` on a non-expiring token: `null`, **never** `0001-01-01T00:00:00Z`
- `lastUsedAt` on a never-used token: `null`
- `kubernetesVersion` on a provisioning cluster: `null`, not `""`
- `enableNetworkPolicy` on an imported cluster: `null`, not `false`
- `enforceVersion` on an unpinned pod security template: `null`
- `project.resourceQuota` on a project with no quota: `null`, not `{}`
- `telemetryOptIn` on 2.10+: `null`

### 4.3 Secret sweep must return 0

```bash
PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify -j -c \
  'rancher { tokens { * } users { * } registryCredentials { * } clusters { * } }' \
  > /tmp/rancher-sweep.json

# the API key you authenticated with
grep -c "${RANCHER_TOKEN#*:}"  /tmp/rancher-sweep.json     # must be 0
# the registry password typed into F13
grep -c '<the F13 password>'   /tmp/rancher-sweep.json     # must be 0
# nothing that looks like a kubeconfig or a private key
grep -ciE 'BEGIN (RSA |EC )?PRIVATE KEY|apiVersion: v1' /tmp/rancher-sweep.json  # must be 0
```

State the guarantee precisely afterwards: *this provider models no field that
carries a secret*. It is not "Rancher does not return secrets" — Rancher does
return several through endpoints this provider does not read.

### 4.4 Composability

Reference chains must resolve end to end, not just one hop:

```bash
PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify -c \
  'rancher.projects { name cluster { name podSecurityAdmissionTemplate { enforce } } }'

PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify -c \
  'rancher.globalRoles.where(grantsFullAdmin) { name bindings { user { username } } }'

PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify -c \
  'rancher.roleTemplates { name inheritedRoleTemplates { name grantsPrivilegeEscalation } }'

PROVIDERS_PATH=/tmp/pd mql run rancher --tls-skip-verify -c \
  'rancher.clusters { name roleTemplateBindings { roleTemplate { grantsFullAdmin } user { username } } }'
```

Also check the **API call count** while doing it. Every listing is cached once
per connection, so the whole run above should issue on the order of ten requests
plus two per project for registry credentials. Watch with:

```bash
docker logs -f rancher 2>&1 | grep -c "GET /v3/"
```

If a reference resolution turns out to be one call per record, that is the
N+1 the caching was meant to prevent and it needs fixing.

---

## 5. Build-under-test recipe

`make providers/install/rancher` **copies from `dist/`, it does not build**. A
failed build leaves the previous binary in place and install ships it happily,
which reads exactly like "the fix did not work". Always build first, and use
`PROVIDERS_PATH` so the installed provider is never touched:

```bash
cd /path/to/mql
git checkout claude/provider-rancher

make providers/build/rancher            # BUILD FIRST
ls -l providers/rancher/dist/rancher    # check the mtime is from just now

mkdir -p /tmp/pd/rancher
cp providers/rancher/dist/rancher* /tmp/pd/rancher/

export RANCHER_URL=https://localhost:8443
export RANCHER_TOKEN='token-xxxxx:yyyyy'
PROVIDERS_PATH=/tmp/pd mql shell rancher --tls-skip-verify
```

After any source change, repeat `make providers/build/rancher` **and** the `cp`.
Forgetting the `cp` is the same trap one step later.

Unit tests, which must stay green throughout:

```bash
cd providers/rancher && go test ./...
```

---

## 6. Risk areas, ranked by damage

Ranked by how badly a wrong reading misleads, not by how likely it is. The top
three produce a **wrong answer**, which is far worse than an empty one, because
an empty field is visible and a wrong one is not.

### Rank 1: `clusterTemplateEnforcement` reported as governing when it governs nothing

`cluster-template-enforcement` still **exists as a setting** on Rancher 2.12 and
newer, but nothing in the 2.12 codebase reads it any more: cluster templates were
removed with RKE1. A fleet on 2.12 with the setting left at `true` from an
earlier upgrade would be reported as enforcing cluster templates while anyone
can create an ad-hoc cluster.

The provider guards this by asking whether `/v3/clusterTemplates` exists at all
before it reads the setting: a 404 means the feature is gone and the field
reports **null** whatever the setting says. That guard is **untested against a
real 2.12 server**, and it rests on an assumption that has not been confirmed:
that Rancher 2.12 answers 404 rather than an empty collection for a schema it no
longer registers. If it answers `200` with `{"data":[]}` instead, the guard
fails open and this becomes the wrong answer it was meant to prevent.

**Verify, and treat this as the single most important row in this document:**

1. On a **2.11** server, set `cluster-template-enforcement=true`, confirm the
   field reads `true`; set it back to `false`, confirm it reads `false`.
2. On a **2.12** server, set `cluster-template-enforcement=true`.
3. `curl -k -H "Authorization: Bearer $RANCHER_TOKEN" "$RANCHER_URL/v3/clusterTemplates" -o /dev/null -w '%{http_code}\n'`
   — record the status code.
4. `rancher.clusterTemplateEnforcement` must read **null**. If it reads `true`,
   step 3 explains why, and the guard needs to key off something else (the
   schema listing at `/v3/schemas`, or the server version).

Do not ship a policy that depends on this field until steps 3 and 4 are done.

### Rank 2: `grantsFullAdmin` and `grantsPrivilegeEscalation` under-reporting

These are computed from `rules`, and a role that grants everything but does not
match the predicate reads as safe. Three known ways that can happen:

- an **external** role template takes its rules from a Kubernetes cluster role
  that Rancher does not report. `effectiveRules` falls back to `externalRules`,
  but Rancher only populates that when the `external-rules` feature flag is on.
  On a server without it, an external template's permissions are **invisible**
  and both predicates read `false`. F5 as an external template is the test.
- wildcard permissions **split across two rules** (`*`/`*` with `get`, plus a
  narrow rule with `*` verbs) do not match `grantsFullAdmin`, by design. That is
  arguably correct, but it is a limit worth stating.
- a global role's `inheritedClusterRoles` grant power in downstream clusters
  that `grantsFullAdmin` does not see, because it reads only `rules`. A role
  that inherits `cluster-owner` everywhere is a fleet administrator and reads
  `grantsFullAdmin == false`.

**Verify with F4, F5, F6, and an external template.** Compare each predicate
against reading the rules by hand.

### Rank 3: pod security `enforceVersion` reading null on a template that pins one

The upstream admission configuration spells the keys `enforce-version`,
`audit-version`, `warn-version`, with hyphens. A camel-cased tag decodes to `""`
and this provider reports **unpinned**. A policy checking "the enforce level is
pinned to a reviewed version" would fail on a compliant template, which is the
safe direction, but a policy checking "the level is not pinned to something old"
would pass on a template pinned to a version from three releases ago.

The unit test pins the tag. **Verify against a real template that pins a
version** (F10's custom one) — that is the only thing that proves the tag
matches what the server sends rather than what the generated client declares.

### Rank 4: `tokens` under-reporting because of the caller's permissions

`/v3/tokens` returns only the calling user's tokens unless the caller is an
administrator. A fleet-wide "no non-expiring API keys" check run under a
standard user would pass on a server full of them. The doc comment says so; a
policy cannot read a doc comment.

**Verify:** run `rancher.tokens.length` as an administrator and as a standard
user against the same server. If the counts differ, consider whether the
provider should fail rather than under-report.

### Rank 5: reference resolution silently returning null

Every typed accessor resolves through a listing and returns null when nothing
matches. A binding whose role template was deleted, a project whose cluster is
gone, a cluster whose pod security template was renamed: all read null. That is
correct, but it is indistinguishable from a resolution bug (a wrong id key, a
listing the token cannot read). §4.4 is the check; a null on a reference that
the console clearly shows is a bug, not a dangling pointer.

### Rank 6: `registryCredentials` cost and coverage

It is two API calls per project. On a fleet with hundreds of projects that is a
slow field, and any project the token cannot read is skipped only on a 404 —
a 403 fails the whole listing. Confirm the behavior against a token that can
read some projects but not others, and decide whether a partial listing with a
count is better than a failure.

---

## Fields I am not sure earn their place

Stated plainly, so whoever verifies can cut rather than verify:

- **`rancher.settings`** as a whole collection. It is 100-odd entries of which
  maybe eight matter, and the eight that matter are already lifted onto the root
  resource. It is cheap (one call, already made) and occasionally the only way
  to reach a setting we did not model, but it is also a lot of schema surface
  for a browsing aid.
- **`cluster.transitioningMessage`**, **`cluster.apiEndpoint`**,
  **`cluster.fleetWorkspaceName`**. Operational detail, not posture. Kept
  because they are free once the record is decoded, but no policy will read
  them.
- **`clusterTemplate` and `clusterTemplate.revision`** entirely. They are dead
  on Rancher 2.12+, which is the version a new install lands on. If the customer
  base is on 2.12+, this whole branch of the schema is inert and could be
  dropped rather than maintained. The `questions` field in particular is a
  `[]dict` blob that nothing will query.
- **`registryCredential.registryUsernames`**. A username is not a credential and
  is mildly useful ("which identity do we pull as"), but it is the one field in
  this provider that sits closest to the secret line. Dropping it costs almost
  nothing.
- **`authProvider.logoutAllSupported`**. A capability of the provider software,
  not a configuration choice; `logoutAllEnabled` would have been the interesting
  one, and it lives on the provider-specific config objects rather than the
  shared `authConfig`, so it is not reachable from the polymorphic listing.

## Whether this provider earns its place at all

The honest position: **that depends on how many customers run Rancher rather
than EKS, AKS, or GKE.** Rancher's share is real but concentrated in on-premise
and edge fleets, and the parts of it that are most worth auditing (RKE1 cluster
templates) are the parts SUSE has been removing. The durable value is the
governance layer that survives: global roles, role templates, pod security
admission templates, non-expiring tokens, and whether local auth is still open
next to single sign-on. If this provider is cut back later, that is the core to
keep.
