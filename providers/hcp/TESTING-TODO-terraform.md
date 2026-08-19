<!--
Copyright Mondoo, Inc. 2024, 2026
SPDX-License-Identifier: BUSL-1.1
-->

# TESTING TODO: `hcp.terraform.*` resources

**Status: NOT VERIFIED AGAINST A LIVE TARGET.** Every `hcp.terraform.*` field on
branch `claude/hcp-terraform-resources` is covered only by unit tests against
synthetic payloads. Per `new-resource` §5, that proves the decoding, not the
resource. This branch does not ship until somebody works through this file.

The attribute names are no longer guesses, which is a real change in the quality
of the evidence but not a substitute for a live call. The record types now come
from **`github.com/hashicorp/go-tfe v1.110.0`**, whose `jsonapi:"attr,…"` tags
are vendor-maintained and exercised continuously by `terraform-provider-tfe`.
Adopting them settled six of the twelve risks below outright, and proved two of
them were **wrong in a way that produced wrong answers** (see R3 and R4). What
remains unverified is what a live installation actually returns.

This document is written to be executable by someone with zero context on the
change. Work top to bottom.

---

## 0. What comes from go-tfe, and what deliberately does not

**Adopted: the record types only.** `tfe.Organization`, `tfe.Workspace`,
`tfe.VCSRepo`, `tfe.TeamAccess`, `tfe.Variable`, `tfe.Team`,
`tfe.OrganizationAccess`, `tfe.TeamToken`, `tfe.PolicySet`, `tfe.Policy`,
`tfe.Enforcement` and `tfe.AgentPool`, decoded through
`github.com/hashicorp/jsonapi` by `TfeRecord.DecodeTyped`. That is where the
value is: 132 hand-written struct tags became 6 (see below).

**Kept local, on purpose.** Each of these is worse in go-tfe, so "simplifying"
any of them onto the SDK would be a regression. If you are tempted, read this
first:

| kept | why go-tfe is worse |
|---|---|
| `List` and its page cap + stuck-cursor guards | **go-tfe has no pagination at all.** Every `List` method returns a single page and leaves the walk to the caller. There is no cap and no cursor guard anywhere in the SDK. |
| `TfeError{StatusCode}` and the three classifiers | go-tfe maps only 401 and 404 onto sentinels. **A 403 becomes a bare `errors.New(<response body>)`** with no status, so classifying it would mean matching on message text, which `TestTfeErrorClassifiersRejectNonAPIErrors` forbids. |
| `tfeTime` and `sanitizeTimestamps` | go-tfe types every timestamp as a bare `time.Time`, and its decoder **rejects the whole record** when one timestamp is malformed, empty, or numeric (`Only strings can be parsed as dates, ISO8601 timestamps`). One odd value would blind an entire collection. Unparseable timestamps are nulled before the typed decode instead. |
| `timePtr` | an absent or null timestamp decodes to the zero instant, which would report 1 January year 1 as a real date. |
| `tfeOrganizationGaps`, `tfeTeamGaps`, `tfePolicySetGaps` | the six attributes go-tfe cannot express, below. |
| the request layer generally | keeping it is what lets `show_only_configured` and the team-token endpoint be fixed at all. |

**The six fields go-tfe cannot supply**, for two distinct reasons:

1. *Not modelled at all* — `plan-expired` (organization) and `versioned`
   (policy set). Both are real: the vendor's own OpenAPI specification, which
   `go-tfe/v2` is generated from, carries `planExpired` and `versioned`.
2. *Modelled with a non-pointer type, so null collapses onto the zero value* —
   `session-timeout`, `session-remember`, `owners-team-saml-role-id`
   (organization) and `sso-team-id` (team). Reporting `0` for a session lifetime
   nobody set, or `""` for a SAML role that does not exist, would invent a value
   the API never gave.

**Why v1 and not `go-tfe/v2`.** v1's own header says it is **"NO LONGER TESTED
and SHOULD NOT BE EXTENDED"**, and new attributes will land in v2 first. It is
still the right choice here, and the reasoning has to survive or somebody will
"upgrade" us into a regression:

- **v2's models cannot be decoded without v2's transport.** They are Kiota
  generated and need a `ParseNode` from `kiota-serialization-json-go`. Using
  them with our client means building and maintaining a Kiota bridge.
- **v2's fields are unexported**, behind getters, so the `scrubSecrets`
  chokepoint in §5.3 would be impossible to write.
- **v2's enums silently drop unknown values.** Every enum attribute is an
  int-backed Go enum whose parser returns `nil, nil` on anything unrecognised,
  discarding the raw string. Eleven of our plain-`string` schema fields would
  start reading `""` when HashiCorp adds a value, including
  `teamAccess.access`, which feeds `canApply`. v1 keeps the raw string, which
  `TestTeamAccessKeepsUnknownRoleVerbatim` pins.

v2 is genuinely better on null semantics (pointers throughout) and on error
classification (`APIError` carries the status). Revisit it when the Kiota bridge
is worth building, or when v1 falls far enough behind on attributes.

---

## 1. Credentials and fixtures you need

### 1.1 Credentials

`hcp.terraform.*` reaches a **different control plane** from the rest of the
`hcp` provider. The HCP service principal (`HCP_CLIENT_ID` /
`HCP_CLIENT_SECRET`) does **not** authenticate to `app.terraform.io`, so a
separate token is required.

| variable | required | what it is |
|---|---|---|
| `HCP_CLIENT_ID` | yes | HCP service principal client id. The provider refuses to connect without it, even for a Terraform-only query. |
| `HCP_CLIENT_SECRET` | yes | HCP service principal client secret. Same reason. |
| `TFE_TOKEN` | yes | HCP Terraform API token. Equivalent flag: `--tfe-token`. |
| `TFE_ADDRESS` | no | Terraform Enterprise base URL. Defaults to `https://app.terraform.io`. Equivalent flag: `--tfe-address`. |
| `TFE_ORGANIZATION` | no | Restricts `hcp.terraformOrganizations` to one organization. Equivalent flag: `--tfe-organization`. |

**Which kind of `TFE_TOKEN`.** Use a **user token** belonging to an owner of the
test organization, or an **organization token**. Do *not* verify with a team
token: a team token cannot read `/organizations/:org/teams`,
`/organizations/:org/policies`, or `/organizations/:org/agent-pools`, and those
endpoints will 403. A 403 is classified as "unavailable" in a few places, so a
wrong token type produces **empty lists that look like a working resource** —
exactly the vacuous pass this document exists to prevent. Confirm the token
works first:

```bash
curl -sS -H "Authorization: Bearer $TFE_TOKEN" \
  https://app.terraform.io/api/v2/organizations | head -c 400
```

If the HCP service principal is genuinely unobtainable, note it here rather than
skipping: the connection currently requires it before any resource resolves, and
that is itself worth recording as a finding (see §6, risk R9).

### 1.2 Organization fixture

One HCP Terraform organization (free tier is enough for most of this; note the
tier-gated items below). Inside it:

| fixture | purpose |
|---|---|
| org with `collaborator-auth-policy = password` | the "2FA not required" state |
| **and** the same org flipped to `two_factor_mandatory` | the "2FA required" state (see §3) |
| a team named `owners` (created automatically) | `ownersTeam` |
| one extra team, e.g. `platform`, with several organization-access toggles ON | `hcp.terraform.team` permissions |
| one more team, e.g. `readers`, with all toggles OFF | the negative state |
| **two or more** team tokens on `platform`, each with a description | `hcp.terraform.teamToken` — more than one is the point: reporting only the first is the bug risk R4 describes |
| a team with **no** token, e.g. `readers` | the empty case, which must be `[]` and not an error |
| an agent pool `shared-agents`, organization-scoped | `organizationScoped == true` |
| an agent pool `prod-agents`, scoped to one workspace | `organizationScoped == false` + `allowedWorkspaces` |
| a Sentinel policy set with one `advisory` and one `hard-mandatory` policy | `enforcementLevel` / `blocking` |
| a global policy set **and** a workspace-scoped policy set | `global` both ways |

**Tier gates.** Policy sets, policies, agent pools, and SAML are not available
on every HCP Terraform plan. If the fixture organization cannot carry one of
them, say so explicitly against the relevant checklist rows below rather than
leaving them unticked with no explanation.

### 1.3 Workspace fixtures

At minimum four workspaces, chosen so every boolean has a fixture on both sides:

| workspace | settings |
|---|---|
| `ws-vcs-autoapply` | VCS-connected (any repo), `auto-apply = true`, execution mode `remote`, working directory `envs/prod`, pinned Terraform version, locked |
| `ws-cli-manual` | no VCS connection, `auto-apply = false`, execution mode `remote`, working directory empty, unlocked |
| `ws-agent` | execution mode `agent`, assigned to `prod-agents` |
| `ws-shared-state` | `global-remote-state = false`, with `ws-cli-manual` added as a remote state consumer |

Plus, on `ws-vcs-autoapply`:

- one **sensitive** `env` variable (e.g. `AWS_SECRET_ACCESS_KEY`) with a
  recognizable value, e.g. `MONDOO-CANARY-SENSITIVE-0001`
- one **non-sensitive** `terraform` variable (e.g. `region = us-east-1`) with a
  recognizable value, e.g. `MONDOO-CANARY-PLAIN-0002`
- one variable with `hcl = true`

And team access grants on `ws-vcs-autoapply`:

- `platform` at `admin` (expect `canApply == true`)
- `readers` at `read` (expect `canApply == false`)
- a third team at `custom` with `runs = apply` (expect `canApply == true`)
- a fourth team at `custom` with `runs = plan` (expect `canApply == false`)

Also set `global-remote-state = true` on **one** workspace so the true state is
observed, not just inferred.

---

## 2. Build under test

Never install over the released provider while verifying; `PROVIDERS_PATH`
replaces the search path entirely.

```bash
cd <repo root>
git checkout claude/hcp-terraform-resources

# NOTE: `make providers/install/hcp` COPIES from dist/, it does not build.
# Always build first, or you will verify a stale binary.
make providers/build/hcp
ls -l providers/hcp/dist/            # confirm the mtime is from this build

mkdir -p /tmp/pd/hcp
cp providers/hcp/dist/hcp* /tmp/pd/hcp/

export HCP_CLIENT_ID=... HCP_CLIENT_SECRET=... TFE_TOKEN=...
PROVIDERS_PATH=/tmp/pd mql shell hcp
```

Shorthand used below: `Q` means running the query in that shell, or

```bash
PROVIDERS_PATH=/tmp/pd mql run hcp -c "<query>"
```

**A/B against `main`** (to show the change is additive and broke nothing):

```bash
git checkout origin/main && make providers/build/hcp
mkdir -p /tmp/pd-main/hcp && cp providers/hcp/dist/hcp* /tmp/pd-main/hcp/
git checkout claude/hcp-terraform-resources && make providers/build/hcp
mkdir -p /tmp/pd-pr/hcp && cp providers/hcp/dist/hcp* /tmp/pd-pr/hcp/

Q='hcp.projects { id name vaultClusters { id } packerRegistry { id } }'
PROVIDERS_PATH=/tmp/pd-main mql run hcp -c "$Q" -j > /tmp/main.json
PROVIDERS_PATH=/tmp/pd-pr   mql run hcp -c "$Q" -j > /tmp/pr.json
cmp -s /tmp/main.json /tmp/pr.json && echo IDENTICAL
```

Expected: `IDENTICAL`. Anything else is a regression in the pre-existing HCP
resources and must be explained before merge.

- [ ] Build is from this branch (dist mtime checked)
- [ ] A/B against `main` reports IDENTICAL on the pre-existing resources

---

## 3. Two-state requirement

**A field that reads the same before and after you change the setting proves
nothing** — it cannot distinguish a resource bug from a fixture that never
moved. Every boolean and enum below has a designated pair. Read each field in
**both** states and record both values.

| field | state A | state B | how to flip |
|---|---|---|---|
| `organization.twoFactorRequired` | `false` (`password`) | `true` (`two_factor_mandatory`) | Organization Settings → Authentication → "Two-factor authentication required" |
| `organization.twoFactorConformant` | `false` while any member lacks 2FA | `true` when all do | enroll/unenroll a member in 2FA |
| `organization.sessionTimeoutMinutes` | null (installation default) | an explicit number | Organization Settings → Authentication → session timeout |
| `organization.costEstimationEnabled` | off | on | Organization Settings → Cost Estimation |
| `workspace.autoApply` | `ws-cli-manual` | `ws-vcs-autoapply` | workspace General Settings → Auto-apply |
| `workspace.vcsDriven` | `ws-cli-manual` (`false`) | `ws-vcs-autoapply` (`true`) | connect/disconnect a repository |
| `workspace.locked` | unlocked | locked | workspace → Lock |
| `workspace.executionMode` | `remote` | `agent` on `ws-agent` | workspace General Settings → Execution Mode |
| `workspace.globalRemoteState` | `false` on `ws-shared-state` | `true` on the other workspace | workspace Settings → Remote state sharing |
| `workspace.workingDirectory` | `""` on `ws-cli-manual` | `envs/prod` | workspace General Settings |
| `variable.sensitive` | the plain variable | the sensitive one | variable "Sensitive" checkbox |
| `variable.hcl` | plain string variable | the HCL variable | variable "HCL" checkbox |
| `teamAccess.canApply` | `readers`/`read` and `custom`+`plan` | `platform`/`admin` and `custom`+`apply` | workspace → Team Access |
| `policy.enforcementLevel` / `blocking` | `advisory` (blocking `false`) | `hard-mandatory` (blocking `true`) | policy edit → enforcement mode |
| `policySet.global` | the scoped set | the global set | policy set → "Policies enforced globally" |
| `agentPool.organizationScoped` | `prod-agents` (`false`) | `shared-agents` (`true`) | agent pool → Scope |
| `team.canManagePolicies` (and siblings) | `readers` (all false) | `platform` (toggled on) | team → Organization Access |

- [ ] Every row above read in **both** states, both values recorded

---

## 4. Per-field checklist

All rows start unchecked. Fill in "observed" and tick only when it matches
"expected".

### 4.1 Root

| # | field | query | expected | observed | ok |
|---|---|---|---|---|---|
| 1 | `hcp.terraformOrganizations` | `hcp.terraformOrganizations { name }` | every org the token can reach | | [ ] |
| 2 | scoped to one org | `--tfe-organization acme` then the same query | exactly one entry, `acme` | | [ ] |
| 3 | no token configured | unset `TFE_TOKEN`, run query 1 | **errors** with "token required"; must NOT return `[]` | | [ ] |

Row 3 is the absent case from §5 below. Treat a silent empty list as a failure.

### 4.2 `hcp.terraform.organization`

Query: `hcp.terraformOrganizations { * }`

| # | field | expected | observed | ok |
|---|---|---|---|---|
| 4 | `name` | the org name | | [ ] |
| 5 | `externalId` | `org-…` | | [ ] |
| 6 | `email` | the org notification address | | [ ] |
| 7 | `createdAt` | a real date, **not** year 1 | | [ ] |
| 8 | `collaboratorAuthPolicy` | `password` or `two_factor_mandatory` | | [ ] |
| 9 | `twoFactorRequired` | matches §3, both states | | [ ] |
| 10 | `twoFactorConformant` | matches §3, both states | | [ ] |
| 11 | `samlEnabled` | matches org SAML config | | [ ] |
| 12 | `ownersTeamSamlRoleId` | **null** when SAML is off, not `""` | | [ ] |
| 13 | `sessionTimeoutMinutes` | **null** when unset; the number when set | | [ ] |
| 14 | `sessionRememberMinutes` | as above | | [ ] |
| 15 | `costEstimationEnabled` | matches §3, both states | | [ ] |
| 16 | `assessmentsEnforced` | matches org setting | | [ ] |
| 17 | `allowForceDeleteWorkspaces` | matches org setting | | [ ] |
| 18 | `defaultExecutionMode` | `remote` / `local` / `agent` | | [ ] |
| 19 | `planExpired` | `false` on an active plan | | [ ] |
| 20 | `ownersTeam` | `{ name }` returns `owners`, not null | | [ ] |

Row 20: the `owners-team` relationship **does not exist** (risk R3, settled),
so `ownersTeam` resolves the team named `owners` by listing teams. That listing
is the only path, and it costs one `/teams` call per read: confirm the cost in a
debug log (`mql shell hcp --log-level debug`), and confirm the installation does
not permit renaming the owners team, because a rename makes this field null.

### 4.3 `hcp.terraform.workspace`

Query: `hcp.terraformOrganizations { workspaces { * } }`

| # | field | expected | observed | ok |
|---|---|---|---|---|
| 21 | `id` | `ws-…` | | [ ] |
| 22 | `name` | workspace name | | [ ] |
| 23 | `description` | text, or `""` when unset | | [ ] |
| 24 | `executionMode` | both states per §3 | | [ ] |
| 25 | `autoApply` | both states per §3 | | [ ] |
| 26 | `autoApplyRunTrigger` | matches setting | | [ ] |
| 27 | `terraformVersion` | e.g. `1.9.5`, not empty | | [ ] |
| 28 | `workingDirectory` | both states per §3 | | [ ] |
| 29 | `locked` | both states per §3 | | [ ] |
| 30 | `vcsDriven` | both states per §3 | | [ ] |
| 31 | `vcsRepoIdentifier` | `acme/infrastructure`, `""` on CLI workspace | | [ ] |
| 32 | `vcsRepoBranch` | branch name, or `""` for repo default | | [ ] |
| 33 | `vcsRepoServiceProvider` | `github` / `gitlab_hosted` / … | | [ ] |
| 34 | `vcsRepoIngressSubmodules` | matches setting | | [ ] |
| 35 | `speculativeEnabled` | matches setting | | [ ] |
| 36 | `globalRemoteState` | both states per §3 | | [ ] |
| 37 | `allowDestroyPlan` | matches setting | | [ ] |
| 38 | `fileTriggersEnabled` | matches setting | | [ ] |
| 39 | `queueAllRuns` | matches setting | | [ ] |
| 40 | `structuredRunOutputEnabled` | matches setting | | [ ] |
| 41 | `assessmentsEnabled` | matches setting | | [ ] |
| 42 | `resourceCount` | matches the workspace's resource count | | [ ] |
| 43 | `tagNames` | the tags, `[]` when none | | [ ] |
| 44 | `createdAt` | a real date | | [ ] |
| 45 | `updatedAt` | a real date; **null**, never `0001-01-01`, if the API omits it (risk R2 settled: the attribute exists) | | [ ] |
| 46 | `organization` | `{ name }` resolves to the owning org, not null | | [ ] |
| 47 | `agentPool` | `{ name }` on `ws-agent`; **null** on the others | | [ ] |
| 48 | `remoteStateConsumers` | on `ws-shared-state`: `[ws-cli-manual]` | | [ ] |
| 49 | `remoteStateConsumers` on a workspace with none | `[]`, and the query does not error | | [ ] |
| 49a | `remoteStateConsumers` on a `globalRemoteState == true` workspace with no explicit consumers | `[]` — **not** every workspace in the organization (risk NR-1, fixed) | | [ ] |
| 50 | direct selection | `hcp.terraform.workspace(id: "ws-…") { name autoApply }` returns the right workspace | | [ ] |
| 51 | bad id | `hcp.terraform.workspace(id: "ws-doesnotexist")` **errors**, does not return a blank resource | | [ ] |

### 4.4 `hcp.terraform.teamAccess`

Query: `hcp.terraformOrganizations { workspaces.where(name == "ws-vcs-autoapply") { teamAccess { * team { name } } } }`

| # | field | expected | observed | ok |
|---|---|---|---|---|
| 52 | `id` | `tws-…` | | [ ] |
| 53 | `access` | `admin`, `read`, `custom` per fixture | | [ ] |
| 54 | `canApply` | true for admin & custom+apply; false for read & custom+plan | | [ ] |
| 55 | `runs` | `apply` / `plan` on the custom grants | | [ ] |
| 56 | `variables` | matches the custom grant | | [ ] |
| 57 | `stateVersions` | matches the custom grant | | [ ] |
| 58 | `sentinelMocks` | matches the custom grant | | [ ] |
| 59 | `workspaceLocking` | matches the custom grant | | [ ] |
| 60 | `runTasks` | matches the custom grant | | [ ] |
| 61 | `team` | `{ name }` resolves, not null | | [ ] |
| 62 | `workspace` | `{ name }` resolves back to the same workspace | | [ ] |
| 63 | "who can apply" | `teamAccess.where(canApply) { team.name }` lists exactly the expected teams | | [ ] |

### 4.5 `hcp.terraform.variable`

Query: `hcp.terraformOrganizations { workspaces { variables { * } } }`

| # | field | expected | observed | ok |
|---|---|---|---|---|
| 64 | `id` | `var-…` | | [ ] |
| 65 | `key` | variable name | | [ ] |
| 66 | `category` | `env` on the sensitive one, `terraform` on the plain one | | [ ] |
| 67 | `sensitive` | `true` on the sensitive one, `false` on the plain one | | [ ] |
| 68 | `hcl` | `true` on the HCL variable, `false` otherwise | | [ ] |
| 69 | `description` | the description, `""` when unset | | [ ] |
| 70 | `workspace` | `{ name }` resolves to the owning workspace | | [ ] |
| 71 | no `value` field exists | `variables { value }` fails to compile | | [ ] |

Row 71 is intentional: the schema has no `value` field, for sensitive *or*
non-sensitive variables. See §5.3.

### 4.6 `hcp.terraform.team` and `hcp.terraform.teamToken`

Query: `hcp.terraformOrganizations { teams { * } }`

| # | field | expected | observed | ok |
|---|---|---|---|---|
| 72 | `id` / `name` | `team-…` / the team name | | [ ] |
| 73 | `usersCount` | actual member count | | [ ] |
| 74 | `visibility` | `secret` / `organization`, both observed | | [ ] |
| 75 | `ssoTeamId` | the SSO group, or **null** when unmapped | | [ ] |
| 76 | `allowMemberTokenManagement` | matches setting | | [ ] |
| 77 | `canManagePolicies` | true on `platform`, false on `readers` | | [ ] |
| 78 | `canManagePolicyOverrides` | as above | | [ ] |
| 79 | `canManageWorkspaces` | as above | | [ ] |
| 80 | `canManageVcsSettings` | as above | | [ ] |
| 81 | `canManageMembership` | as above | | [ ] |
| 82 | `canManageTeams` | as above | | [ ] |
| 83 | `canManageOrganizationAccess` | as above | | [ ] |
| 84 | `canManageProjects` | as above | | [ ] |
| 85 | `canManageRunTasks` | as above | | [ ] |
| 86 | `canManageAgentPools` | as above | | [ ] |
| 87 | `canManageProviders` | as above | | [ ] |
| 88 | `canManageModules` | as above | | [ ] |
| 89 | `canReadWorkspaces` | as above | | [ ] |
| 90 | `canReadProjects` | as above | | [ ] |
| 91 | `canAccessSecretTeams` | as above | | [ ] |
| 92 | `organization` | `{ name }` resolves | | [ ] |
| 93 | `tokens` | non-empty on `platform` | | [ ] |
| 94 | `tokens` on a team with none | `[]`, no error | | [ ] |
| 95 | `tokens { id createdAt }` | real id and date | | [ ] |
| 96 | `tokens { lastUsedAt }` | **null** on a never-used token; a date after using it | | [ ] |
| 97 | `tokens { expiredAt }` | **null** on a non-expiring token; a date on an expiring one | | [ ] |
| 98 | `tokens { description }` | the description, `""` for the legacy single token | | [ ] |
| 99 | `tokens { team { name } }` | resolves back to the owning team | | [ ] |

Rows 93/94 cover risk R4, which was **wrong and is fixed**: the endpoint the old
code listed does not exist, so a team reported at most one token. Tokens now come
from `GET /organizations/:org/team-tokens`, filtered to the team. Drive both
states deliberately: a team with **several** tokens must report all of them
(this is the case that used to under-report), and a team with none must report
`[]` without erroring.

### 4.7 `hcp.terraform.policySet` and `hcp.terraform.policy`

Query: `hcp.terraformOrganizations { policySets { * } policies { * } }`

| # | field | expected | observed | ok |
|---|---|---|---|---|
| 100 | `policySet.id` / `name` | `polset-…` / name | | [ ] |
| 101 | `policySet.description` | text, `""` when unset | | [ ] |
| 102 | `policySet.kind` | `sentinel` and (if fixtured) `opa` | | [ ] |
| 103 | `policySet.global` | both states per §3 | | [ ] |
| 104 | `policySet.policyCount` | matches the UI | | [ ] |
| 105 | `policySet.workspaceCount` | matches the UI; `0` on the global set | | [ ] |
| 106 | `policySet.versioned` | matches the set's source | | [ ] |
| 107 | `policySet.policiesPath` | path, `""` for repo root | | [ ] |
| 108 | `policySet.agentEnabled` | matches setting | | [ ] |
| 109 | `policySet.overridable` | matches setting | | [ ] |
| 110 | `policySet.createdAt` / `updatedAt` | real dates | | [ ] |
| 111 | `policySet.organization` | resolves | | [ ] |
| 112 | `policySet.policies { name }` | the set's policies | | [ ] |
| 113 | `policySet.workspaces { name }` | the attached workspaces; `[]` on the global set | | [ ] |
| 114 | `policy.id` / `name` / `description` | match the UI | | [ ] |
| 115 | `policy.kind` | `sentinel` / `opa` | | [ ] |
| 116 | `policy.enforcementLevel` | `advisory` and `hard-mandatory`, both observed | | [ ] |
| 117 | `policy.blocking` | `false` for advisory, `true` for hard-mandatory | | [ ] |
| 118 | soft-mandatory | add one: `enforcementLevel == "soft-mandatory"`, `blocking == false` | | [ ] |
| 119 | `policy.policySetCount` | matches the UI | | [ ] |
| 120 | `policy.updatedAt` | a real date | | [ ] |
| 121 | `policy.organization` | resolves | | [ ] |

Row 116 covers risk R5, now settled as a schema question: enforcement is
per-policy, and go-tfe marks the legacy `enforce` list deprecated in favour of
`enforcement-level`, which is the preference order the code already used. What
is left is purely observational: capture the raw payload (§6) and say which
shape the live API returned.

### 4.8 `hcp.terraform.agentPool`

Query: `hcp.terraformOrganizations { agentPools { * } }`

| # | field | expected | observed | ok |
|---|---|---|---|---|
| 122 | `id` / `name` | `apool-…` / name | | [ ] |
| 123 | `agentCount` | matches the UI (register an agent to see a non-zero value) | | [ ] |
| 124 | `organizationScoped` | both states per §3 | | [ ] |
| 125 | `createdAt` | a real date | | [ ] |
| 126 | `organization` | resolves | | [ ] |
| 127 | `allowedWorkspaces { name }` | on `prod-agents`: the allowed workspace | | [ ] |
| 128 | `allowedWorkspaces` on `shared-agents` | `[]` | | [ ] |
| 129 | direct selection | `hcp.terraform.agentPool(id: "apool-…") { name }` | | [ ] |

Row 127 covers risk R6: `allowed-workspaces` is confirmed against go-tfe's
`AgentPool.AllowedWorkspaces` relation tag, so the key is right. If it still
comes back empty on a pool that visibly has an allow list, the cause is that the
relationship is only served for scoped pools, not a wrong key: capture the raw
`agent-pools` payload and record which.

---

## 5. The four `new-resource` §5 checks

### 5.1 Absent case must FAIL, not pass vacuously

An organization with none of a thing must make an assertion about that thing
**fail**, never satisfy it. Run each of these and record the outcome:

```bash
# no token at all: must ERROR, not return []
unset TFE_TOKEN
PROVIDERS_PATH=/tmp/pd mql run hcp -c 'hcp.terraformOrganizations'
```
- [ ] errors with "HCP Terraform API token required"; does **not** return `[]`

```bash
# a token that cannot read teams (use a deliberately under-privileged token)
PROVIDERS_PATH=/tmp/pd mql run hcp -c 'hcp.terraformOrganizations { teams { name } }'
```
- [ ] errors or reports a permission problem; does **not** report `[]` teams as
      if the organization had none

```bash
# a nonexistent organization must error, not hydrate a blank resource
PROVIDERS_PATH=/tmp/pd mql run hcp \
  -c 'hcp.terraform.organization(name: "definitely-not-an-org") { twoFactorRequired }'
```
- [ ] errors; does **not** return a resource whose every field is null

```bash
# and the assertion form: this must FAIL on a nonexistent org
PROVIDERS_PATH=/tmp/pd mql run hcp \
  -c 'hcp.terraform.organization(name: "definitely-not-an-org").twoFactorRequired == true'
```
- [ ] fails or errors. **If it passes, stop and file a bug**: MQL three-valued
      logic makes `null == true` unreliable, and a policy written on this field
      would silently pass everywhere.

```bash
# a workspace with no team access grants and no variables
PROVIDERS_PATH=/tmp/pd mql run hcp \
  -c 'hcp.terraform.workspace(id: "<empty-ws>") { teamAccess variables }'
```
- [ ] both return `[]` (a genuinely empty collection, which is a true answer
      here) and a `.none(canApply)` assertion over an empty list is understood
      to be vacuously true by design — note it, do not tick it as "verified
      protection"

### 5.2 Null over invented defaults

No field may report a value the API did not give. Specifically confirm:

- [ ] `organization.sessionTimeoutMinutes` is **null**, not `0`, when the org
      uses the installation default
- [ ] `organization.sessionRememberMinutes` likewise
- [ ] `organization.ownersTeamSamlRoleId` is **null**, not `""`, when SAML is off
- [ ] `teamToken.lastUsedAt` is **null**, not `0001-01-01`, on a never-used token
- [ ] `teamToken.expiredAt` is **null** on a non-expiring token
- [ ] `workspace.updatedAt` is **null**, not year 1, if the API omits it
- [ ] `workspace.agentPool` is **null**, not an empty resource, on a non-agent
      workspace
- [ ] no timestamp anywhere reads `0001-01-01` — sweep it:
      `PROVIDERS_PATH=/tmp/pd mql run hcp -c 'hcp.terraformOrganizations { workspaces { * } teams { tokens { * } } }' -j | grep -c '0001-01-01'` must be **0**

Fields that *are* documented as reporting an effective default: none in this
change. If verification finds one, document it in the field's `.lr` doc comment
and pick the fail-safe direction.

### 5.3 Secret sweep must return 0

This provider handles sensitive workspace variables and team tokens. The schema
deliberately carries **no** variable value and **no** token secret.

**This became more important with go-tfe** (risk NR-3): the vendor types carry
four credential fields the hand-written record types simply did not have —
`Variable.Value`, `TeamToken.Token`, `VCSRepo.OAuthTokenID` and
`VCSRepo.WebhookURL`. The API returned those values before too, so the exposure
is not new, but a populated `Value` now sits one line away from anybody adding a
field.

Two guards, in order:

1. **Structural.** `scrubSecrets` clears all four at the single decode
   chokepoint (`decodeTfeRecord` in `resources/terraform.go`), so nothing
   downstream has a populated secret to reach for. A field added later reads
   `""` and fails its own test rather than shipping a workspace credential.
2. **Behavioural.** `TestTerraformResourcesCarryNoSecrets` builds every
   SDK-backed record through the production code from payloads whose every
   credential position holds a distinct canary — including an *undeclared*
   attribute, to catch a future `dict` passthrough — then renders every field of
   every resource, including unexported cache fields, and asserts no canary
   survives. It carries a **negative control** asserting the non-secret values
   *did* come through, so it cannot pass by decoding nothing.
   `TestTerraformVariableHasNoValueField` separately asserts the schema refuses
   to resolve `value`, `token`, `secret`, `password` or `oauthToken`.

Neither guard replaces the live sweep. Run it:

```bash
PROVIDERS_PATH=/tmp/pd mql run hcp \
  -c 'hcp.terraformOrganizations { workspaces { variables { * } } teams { tokens { * } } }' -j \
  > /tmp/hcp-tf.json

grep -c 'MONDOO-CANARY-SENSITIVE-0001' /tmp/hcp-tf.json   # must be 0
grep -c 'MONDOO-CANARY-PLAIN-0002'     /tmp/hcp-tf.json   # must be 0
grep -c 'atlasv1\.'                     /tmp/hcp-tf.json   # must be 0 (token prefix)
grep -ci "$TFE_TOKEN"                   /tmp/hcp-tf.json   # must be 0
```

- [ ] canary value of the **sensitive** variable appears 0 times
- [ ] canary value of the **non-sensitive** variable appears 0 times
- [ ] no `atlasv1.` token material appears
- [ ] the connecting `TFE_TOKEN` itself does not appear anywhere in the output
- [ ] same sweep over the shell's non-JSON output

State the guarantee precisely in the PR: *"these resources carry no variable
value and no token secret"* is true. *"the secrets are not exposed"* is **not**
a claim this change can make — anyone with the same token can read the
non-sensitive values straight from the API.

### 5.4 Composability

Cross-resource traversal is the reason the typed accessors exist. Each of these
must return real data, not empty:

```bash
# who can apply, across the whole estate
hcp.terraformOrganizations { workspaces { name teamAccess.where(canApply) { team { name } } } }

# unpinned auto-apply workspaces reachable from VCS
hcp.terraformOrganizations { workspaces.where(autoApply && vcsDriven) { name terraformVersion } }

# workspaces sharing state with anyone
hcp.terraformOrganizations { workspaces.where(globalRemoteState) { name } }
hcp.terraformOrganizations { workspaces { name remoteStateConsumers { name } } }

# unmarked secrets
hcp.terraformOrganizations { workspaces { name variables.where(category == "env" && !sensitive) { key } } }

# blocking policy coverage
hcp.terraformOrganizations { policySets { name global workspaces { name } policies.where(blocking) { name } } }

# agent pools reachable by every workspace
hcp.terraformOrganizations { agentPools.where(organizationScoped) { name } }

# round trip: workspace -> org -> workspaces
hcp.terraformOrganizations { workspaces { organization { name workspaces { name } } } }
```

- [ ] every query above returns non-empty, correct data
- [ ] the round-trip query does not loop, error, or blow up the runtime
- [ ] no `llx: encountered a primitive with no type information` anywhere
- [ ] no `provider returned no data and no error for a field` anywhere

---
## 6. Risk areas

**This is the highest-value section.** Adopting go-tfe's types settled a lot of
it: where a risk was "we guessed this attribute name from documentation", the
vendor's maintained tags now answer it. Two of the guesses turned out to be
**wrong in a way that produced wrong answers**, and both are fixed on this
branch (R3, R4). One more wrong answer surfaced that the register never
contemplated (NR-1).

What the SDK cannot settle is what a live installation actually returns. Capture
raw payloads while verifying so the remaining items can be closed with evidence:

```bash
for p in organizations organizations/$ORG/workspaces organizations/$ORG/teams \
         organizations/$ORG/policies organizations/$ORG/policy-sets \
         organizations/$ORG/agent-pools workspaces/$WS/vars \
         "team-workspaces?filter%5Bworkspace%5D%5Bid%5D=$WS" \
         "workspaces/$WS/relationships/remote-state-consumers?show_only_configured=true" \
         organizations/$ORG/team-tokens \
         teams/$TEAM/authentication-token; do
  echo "=== $p"
  curl -sS -H "Authorization: Bearer $TFE_TOKEN" \
    "https://app.terraform.io/api/v2/$p" | python3 -m json.tool | head -60
done
```

### Summary

| risk | status |
|---|---|
| R2 workspace `updated-at` | **struck** — attribute exists |
| R5 policy enforcement level | **struck** — per policy, closed vocabulary |
| R7 `filter[workspace][id]` | **struck** — was correct all along |
| R11 `resource-count` / `agent-count` types | **struck** — numeric |
| R3 `owners-team` relationship | **rewritten** — does not exist; was dead code |
| R4 team token endpoint | **rewritten and fixed** — endpoint did not exist |
| R6 relationship key names | **mostly closed** — one key was wrong (R3) |
| R8 pagination | **half closed** — parameters confirmed; >100 records still unproven |
| R1, R9, R10, R12 | **open**, unchanged by this work |
| NR-1 remote state consumers | **new, and fixed** |
| NR-2 go-tfe v1 is frozen | **new**, accepted |

---

## Struck: settled by the SDK, and our guess was right

### R2 — Workspace `updated-at` (STRUCK)

The attribute exists. go-tfe v1 carries
`UpdatedAt time.Time \`jsonapi:"attr,updated-at,iso8601"\``, and the vendor's
OpenAPI specification carries `updatedAt` alongside a *separate* `latestChangeAt`.
The field was correctly pointed and does not need re-pointing at
`latest-change-at`.

One thing did change: go-tfe's bare `time.Time` makes an absent value the zero
instant, so `timePtr` maps the zero instant back to null. `workspace.updatedAt`
still reports null when the API omits it, and the `0001-01-01` sweep in §5.2 is
now structurally unfailable rather than merely expected to pass.

### R5 — Policy enforcement level (STRUCK)

Three separate questions, all answered:

- **Is it per policy or per policy set?** Per policy. `EnforcementLevel` sits on
  go-tfe's `Policy`, not on the policy-set-to-policy edge. The field is modelled
  at the right level and does not belong somewhere else.
- **Is the legacy `enforce` list still a thing?** go-tfe marks
  `Policy.Enforce` **`// Deprecated: Use EnforcementLevel instead.`** — which is
  exactly the preference order `terraformEnforcementLevel` already used. It is
  still read deliberately, so a policy on an older Terraform Enterprise that
  reports only the legacy list does not read as unenforced. That is the
  documented exception in CLAUDE.md §6 for a deprecated SDK field.
- **Is the OPA vocabulary really just two values?** The whole `EnforcementLevel`
  set is closed at four: `advisory`, `hard-mandatory`, `soft-mandatory`,
  `mandatory`. `terraformPolicyBlocking` treats `hard-mandatory` and `mandatory`
  as blocking, which is complete against that set.

Still worth observing live which shape the API returns, but nothing here is a
schema decision any more.

### R7 — `filter[workspace][id]` on `/team-workspaces` (STRUCK)

**This was called the worst risk in the batch, and it does not exist.** The
parameter name was correct all along.

Two independent confirmations:

- go-tfe v1: `type TeamAccessListOptions struct { ListOptions; WorkspaceID string \`url:"filter[workspace][id]"\` }`
- the vendor OpenAPI specification:
  `Filterworkspaceid *string "uriparametername:\"filter%5Bworkspace%5D%5Bid%5D\""`,
  documented as *"The workspace ID to list team access for."*

No workspace was ever reporting another workspace's `canApply` grants. Because
the request layer stays local (§0), the filter is still ours to get right rather
than the SDK's, so it is now pinned by `TestTeamAccessIsScopedToTheWorkspace`,
which asserts the encoded query string rather than only the decoded answer.

Still worth the live cross-check that two different workspaces return different
grant sets, but this is no longer a design risk.

### R11 — `resource-count` and `agent-count` field types (STRUCK)

Numeric, not strings. go-tfe v1 types them `ResourceCount int` and
`AgentCount int`; the vendor OpenAPI specification types them `*int32`. The
Packer `version-count` string problem does not repeat here, and no
`parseVersionCount`-style helper is needed.

---

## Rewritten: settled by the SDK, and our guess was wrong

### R3 — `owners-team` relationship (REWRITTEN: it does not exist)

**The relationship is not real.** It appears neither in go-tfe v1's
`Organization` type nor in the vendor's OpenAPI specification, whose
organization relationships are exactly: `auditTrailsAuthenticationToken`,
`authenticationToken`, `dataRetentionPolicy`, `defaultAgentPool`,
`defaultProject`, `entitlementSet`, `moduleProducers`, `oauthTokens`,
`primaryHyokConfiguration`, `providerProducers`, `stacksDefaultAgentPool`,
`subscription`.

So `relOneID(rec, "owners-team")` returned `""` **on every call**, the
"preferred" branch never executed once, and the name-based fallback was in
reality the only code path. R3's predicted failure mode #1 was not a
possibility, it was what always happened.

**Changed on this branch:** the dead relationship read is gone and
`ownersTeam` resolves the team named `owners` directly, which is documented in
the function rather than presented as a fallback.

**Still open, and still worth recording:**

1. It costs one team listing per `ownersTeam` read. If that hurts on a large
   organization, `?filter[names]=owners` is the cheaper call.
2. If an installation ever permits renaming the owners team, this returns
   **null** and an audit asserting "the owners team enforces 2FA" finds nothing
   to assert against. `TestOwnersTeamIsNullWhenNoTeamIsNamedOwners` pins that it
   is null rather than a blank resource, so the audit errors instead of passing
   vacuously — but confirm the rename restriction on a live installation.

### R4 — Team token endpoint (REWRITTEN AND FIXED)

**`GET /teams/:id/authentication-tokens` does not exist**, so the old code could
never have worked as intended:

- go-tfe builds `teams/{id}/authentication-tokens` for a **POST** only, to
  create a token with a description. Its `TeamTokens.List` takes an
  *organization* id and hits `organizations/{org}/team-tokens`.
- The vendor OpenAPI specification gives `teams/{id}/authentication-token`
  Get/Post/Delete, gives `teams/{id}/authentication-tokens` **no GET at all**,
  and gives `organizations/{org}/team-tokens` a GET with a `q` team-name filter.

So the list call always 404ed and always fell through to the singular endpoint,
which returns **at most one token**. A team holding three descriptive tokens
reported one, and an audit asserting *"this organization issues no long-lived
team tokens"* could pass on an organization that had several. That is exactly
the dangerous case R4 named, and it was not hypothetical.

**Behaviour delta, deliberate and reviewed.** `team.tokens` now lists
`GET /organizations/:org/team-tokens` and filters to the team by its `team`
relationship. A token whose team relationship is missing is skipped rather than
attributed, because over-reporting a credential to the wrong team is worse than
under-reporting it. The singular endpoint is kept as a fallback for
installations that predate the organization-wide listing, and
`TestTeamTokensFallBackToTheSingleTokenEndpoint` exercises that path
deliberately so it does not ship unverified.

**Verify live:** a team with several tokens reports all of them, a team with
none reports `[]` without erroring, and no request is made to the plural path
(`TestTeamTokensUseTheOrganizationListing` asserts the last one already).

### R6 — Relationship key names (MOSTLY CLOSED)

Read off go-tfe's `jsonapi:"relation,…"` tags and the vendor specification:

| resource | key | verdict |
|---|---|---|
| workspace | `organization`, `agent-pool` | correct, both SDKs |
| organization | `owners-team` | **wrong, does not exist** (R3) |
| team-workspaces | `team`, `workspace` | correct |
| team | `organization` | correct (present in the vendor spec; absent from go-tfe v1's `Team`) |
| policy-set | `policies`, `workspaces` | correct |
| policy | `organization` | correct |
| agent-pool | `organization`, `allowed-workspaces` | correct |

Seven of eight were right. The eighth is R3.

One caveat worth a live check: go-tfe v1 does not model `team.organization`,
though the vendor specification does. `team.organization` is populated from the
parent when a team is reached through `organization.teams`, so the common path
is safe; a team reached by id alone (`teamAccess.team.organization`) depends on
the relationship actually being served.

---

## Open

### R1 — The control-plane split (OPEN, narrowed)

`hcp.terraform.*` uses a separate bearer token against `app.terraform.io/api/v2`
rather than the HCP service principal. **No client library can settle this** —
it is a question about what the API accepts, not about how we call it.

Narrowing: the earlier framing said go-tfe had been rejected because the HCP
service principal is not accepted. That conflated auth with transport. go-tfe
takes a TFE/HCP Terraform API token, exactly what `--tfe-token` supplies; its
`Config` is `Address` + `BasePath` + `Token`. The token model is unchanged and
correct as far as anyone knows.

Still to settle live: whether an HCP service-principal-derived token
authenticates to the Terraform API at all, and whether HCP Terraform
organizations are discoverable from an HCP organization (if so, `hcp.organization`
may deserve a link to them). Impact if wrong: a needless required flag, and a
schema shape that would be breaking to change later.

### R8 — Pagination (HALF CLOSED)

**Closed:** `workspaces/:id/relationships/remote-state-consumers` does accept
`page[number]` and `page[size]` — the vendor specification lists both. It also
accepts `show_only_configured`, which turned out to matter far more (NR-1).

**Open:** whether any endpoint caps `page[size]` below 100, and what a real
second page looks like. `workspaces/:id/vars` has *no* pagination parameters in
the specification at all, so the ones we send are presumably ignored there; the
stuck-cursor guard covers that, but it is unobserved.

Confirm with an organization holding **more than 100 workspaces**. If no such
organization exists, say so: pagination is then unverified in production
conditions and covered only by the httptest unit tests in
`connection/terraform_test.go`.

### R9 — HCP service principal required for Terraform-only queries (OPEN)

`NewHcpConnection` still fails without `--client-id`/`--client-secret`, so a user
who only wants HCP Terraform data must supply HCP credentials they may not have.
Inherited from the existing connection, not introduced here, but it will be the
first thing a user hits — and it is concrete enough that the secret-sweep test
has to pass placeholder HCP credentials to build a connection that never calls
an HCP endpoint. Decide whether the connection should accept a Terraform-only
configuration.

### R10 — N+1 API calls on relationship resolution (OPEN, unchanged)

`policySet.policies`, `policySet.workspaces`, `agentPool.allowedWorkspaces`,
`teamAccess.team` and `teamAccess.workspace` each resolve one API call per
referenced record, because `NewResource` runs the target's `init` before the
runtime cache is consulted. `workspaces { teamAccess { team } }` is one
`GET /teams/:id` per grant.

**The type swap does not address this**, and should not be mistaken for having
done so. Measure the request count on the fixture organization and, if it is
bad, resolve against a once-fetched list per CLAUDE.md §1.5.

`organization.ownersTeam` adds one team listing per read (R3).

### R12 — Pre-existing duplicate field path (OPEN, unchanged)

`mqlr generate` warns `duplicate field paths detected: ["hcp.organization"]`.
This predates the change and is not fallout from this branch; `terraformOrganizations`
was named specifically to avoid adding a second one. Somebody should eventually
check whether `hcp.organization` is affected by the empty-husk failure mode in
CLAUDE.md.

---

## New risks

### NR-1 — `remoteStateConsumers` returned the whole organization (NEW, FIXED)

The vendor specification documents a `show_only_configured` parameter on
`workspaces/:id/relationships/remote-state-consumers`: *"When true, return only
explicitly configured remote state consumers **even if global-remote-state is
enabled**."*

We never sent it. So on a workspace with `globalRemoteState == true`, the
endpoint enumerates **every workspace in the organization** as a consumer — and
the field's own doc comment claimed the opposite, that the list is empty in that
case. That is a wrong answer rather than an empty one, it grows as the square of
the estate, and the register never contemplated it.

**Behaviour delta, deliberate and reviewed.** `show_only_configured=true` is now
sent, which makes the field mean what its doc comment already said: the
workspaces somebody explicitly granted. Pinned by
`TestRemoteStateConsumersAsksOnlyForConfiguredOnes`.

**Verify live:** on a workspace with `globalRemoteState == true` and no explicit
consumers the list is `[]`, not the whole organization; on `ws-shared-state` it
is exactly `[ws-cli-manual]`.

### NR-2 — go-tfe v1 is frozen (NEW, accepted)

v1's header: *"the final version of the go-tfe (v1) package … NO LONGER TESTED
and SHOULD NOT BE EXTENDED."* New HCP Terraform attributes will appear in
`go-tfe/v2` first, so the six-field gap list in §0 can only grow.

Accepted, because v2 is worse for this provider on three counts that §0 spells
out (Kiota transport dependency, unexported fields blocking the secret scrub,
and enums that silently drop unrecognised values). Watch the gap list: if it
grows past a handful, the Kiota bridge starts paying for itself.

### NR-3 — the SDK types carry secrets ours omitted (NEW, mitigated)

`tfe.Variable.Value`, `tfe.TeamToken.Token`, `tfe.VCSRepo.OAuthTokenID` and
`tfe.VCSRepo.WebhookURL` exist on the vendor types where the hand-written record
types had no such field. The API returned these values before too — the exposure
is not new — but adopting the types puts a populated `Value` one
`llx.StringData(v.Value)` away from a future contributor.

Mitigated structurally by `scrubSecrets`, which clears all four at the single
decode chokepoint (`decodeTfeRecord`), so a field added later reads `""` and
fails its own test rather than shipping a credential. See §5.3.

## 7. Sign-off

- [ ] every checklist row above ticked, or explicitly marked unverifiable with a reason
- [ ] every remaining open risk (R1, R6 caveat, R8, R9, R10, R12, NR-2) either closed with evidence, or restated in the PR body as an open blocker. R2, R5, R7 and R11 are struck; R3, R4 and NR-1 are fixed on this branch and need live confirmation, not re-investigation
- [ ] the PR body carries the **observed values table**, not a description of the schema
- [ ] `go test ./...` green inside `providers/hcp/` (includes the automated secret sweep in §5.3)
- [ ] nothing credential-shaped committed (the secret scanner reads every commit, not the final diff)
