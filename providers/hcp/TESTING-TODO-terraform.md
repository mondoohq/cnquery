<!--
Copyright Mondoo, Inc. 2024, 2026
SPDX-License-Identifier: BUSL-1.1
-->

# TESTING TODO: `hcp.terraform.*` resources

**Status: NOT VERIFIED AGAINST A LIVE TARGET.** Every `hcp.terraform.*` field on
branch `claude/hcp-terraform-resources` was written from the HCP Terraform /
Terraform Enterprise API v2 documentation and is covered only by unit tests
against synthetic payloads. Per `new-resource` §5, that proves the decoding, not
the resource: a fixture built from the documentation reproduces the
documentation. This branch does not ship until somebody works through this file.

This document is written to be executable by someone with zero context on the
change. Work top to bottom.

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
| a team token on `platform` | `hcp.terraform.teamToken` |
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

Row 20 has a known fallback worth exercising deliberately (risk R3): the
`owners-team` relationship is not present on every response shape, in which case
the code falls back to the team literally named `owners`. Confirm which branch
ran, e.g. by checking whether `hcp.terraformOrganizations { ownersTeam { id } }`
issues a `/teams` list call in a debug log (`mql shell hcp --log-level debug`).

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
| 45 | `updatedAt` | a real date, **null** if the API omits it (see risk R2) | | [ ] |
| 46 | `organization` | `{ name }` resolves to the owning org, not null | | [ ] |
| 47 | `agentPool` | `{ name }` on `ws-agent`; **null** on the others | | [ ] |
| 48 | `remoteStateConsumers` | on `ws-shared-state`: `[ws-cli-manual]` | | [ ] |
| 49 | `remoteStateConsumers` on a workspace with none | `[]`, and the query does not error | | [ ] |
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

Row 93/94 exercise a fallback that will otherwise ship untested — see risk R4,
and drive it deliberately.

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

Row 116 covers risk R5: confirm whether the live API returns a top-level
`enforcement-level`, the legacy `enforce[].mode`, or both. Capture the raw
payload (§6) and say which branch actually ran.

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

Row 127 covers risk R6: the `allowed-workspaces` relationship name is inferred
from the docs. If it comes back empty on a pool that visibly has an allow list,
capture the raw `agent-pools` payload and correct the relationship key.

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
deliberately carries **no** variable value and **no** token secret. Prove it:

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

**This is the highest-value section.** Everything below is inferred from API
documentation rather than observed against a live installation. Each item names
what could be wrong, what the wrong answer would look like, and how to settle
it. Capture raw payloads while verifying so these can be closed with evidence:

```bash
for p in organizations organizations/$ORG/workspaces organizations/$ORG/teams \
         organizations/$ORG/policies organizations/$ORG/policy-sets \
         organizations/$ORG/agent-pools workspaces/$WS/vars \
         "team-workspaces?filter%5Bworkspace%5D%5Bid%5D=$WS" \
         workspaces/$WS/relationships/remote-state-consumers \
         teams/$TEAM/authentication-tokens; do
  echo "=== $p"
  curl -sS -H "Authorization: Bearer $TFE_TOKEN" \
    "https://app.terraform.io/api/v2/$p" | python3 -m json.tool | head -60
done
```

### R1 — The whole control-plane split is an assumption

`hcp.terraform.*` uses a **separate bearer token** against
`app.terraform.io/api/v2`, not the HCP service principal that authenticates
every other resource in this provider. That decision was made because the HCP
service principal OAuth token is not accepted by the Terraform API — but this
was never tested against a live account. **Settle this first**: if an HCP
service-principal-derived token *does* authenticate to the Terraform API, the
entire `--tfe-token` flag surface is unnecessary ceremony and should be
reconsidered before it ships and becomes a compatibility obligation. Also check
whether HCP Terraform organizations are discoverable from an HCP org at all; if
they are, `hcp.organization` may deserve a link to them.

Impact if wrong: a needless required flag, and a schema shape (`hcp.terraform.*`
hanging off the root rather than off `hcp.organization`) that would be breaking
to change later.

### R2 — Workspace `updated-at` may not exist

`updated-at` is not documented on the workspace attributes object in every API
version; `latest-change-at` is the field the UI uses. If `updated-at` is absent,
`workspace.updatedAt` reports **null** on every workspace — which is safe but
useless, and the field should be re-pointed at `latest-change-at` (a schema
change, so do it before release, not after).

Check: `curl … /workspaces/$WS | python3 -c 'import json,sys; print(sorted(json.load(sys.stdin)["data"]["attributes"]))'`

### R3 — `owners-team` relationship and the "owners" name fallback

`organization.ownersTeam` first reads an `owners-team` relationship from the
organization record. That relationship is **not documented** on the
organizations endpoint; the code therefore falls back to the team literally
named `owners`. Two things can go wrong:

1. If the relationship never exists, the fallback runs on every call, which
   lists every team in the organization to find one. That is a hidden extra API
   call per `ownersTeam` read.
2. If an installation ever allows renaming the owners team, the fallback returns
   **null** and an audit on "the owners team has 2FA" silently finds nothing.

Determine which branch actually runs (debug log) and record it. If the
relationship does not exist, consider reading the owners team through
`/organizations/:org/teams?filter[names]=owners` instead of listing all teams.

### R4 — Team token endpoint shape and the 404 fallback

`team.tokens` calls `GET /teams/:id/authentication-tokens` (multiple team
tokens) and falls back to `GET /teams/:id/authentication-token` (singular,
legacy) **only on a 404**. Unverified assumptions:

- that the plural endpoint exists on HCP Terraform at all,
- that it 404s (rather than 403s or 200-with-empty) when unsupported,
- that the singular endpoint 404s when no token has been issued.

If the plural endpoint returns 403 on an under-privileged token, the fallback
does not run and the error propagates — acceptable. If it returns **200 with an
empty list** where a legacy single token exists, the token is invisible and a
"no long-lived team tokens" audit passes on an organization that has one. That
is the dangerous case. **Exercise both branches deliberately**: a team with a
token, and a team with none.

### R5 — Policy enforcement level: two shapes, one field

`policy.enforcementLevel` prefers the top-level `enforcement-level` attribute
and falls back to `enforce[0].mode`. Unverified:

- whether current HCP Terraform still returns `enforce` at all,
- whether a policy in **multiple** policy sets can carry **different** modes per
  set, in which case `enforce[0].mode` picks an arbitrary one and
  `blocking` may under-report.

If enforcement is genuinely per-policy-set rather than per-policy, this field is
modeled at the wrong level and belongs on the policy-set-to-policy edge instead.
That is a schema decision, so settle it before release. Also confirm the OPA
level vocabulary: the code treats `mandatory` as blocking and `advisory` as not,
which assumes OPA has exactly those two.

### R6 — Relationship key names

Every relationship key is a documentation guess. A wrong key silently yields an
**empty list or a null typed ref** — never an error. The keys in use:

| resource | key | field it feeds |
|---|---|---|
| workspace | `organization` | `workspace.organization` |
| workspace | `agent-pool` | `workspace.agentPool` |
| organization | `owners-team` | `organization.ownersTeam` |
| team-workspaces | `team`, `workspace` | `teamAccess.team` / `.workspace` |
| team | `organization` | `team.organization` |
| policy-set | `policies`, `workspaces` | `policySet.policies` / `.workspaces` |
| policy | `organization` | `policy.organization` |
| agent-pool | `organization`, `allowed-workspaces` | `agentPool.organization` / `.allowedWorkspaces` |

Confirm each against a raw payload. In particular `allowed-workspaces` may be
named `allowed-workspaces` on some versions and only be present when the pool is
scoped.

### R7 — `filter[workspace][id]` on `/team-workspaces`

`workspace.teamAccess` lists `GET /team-workspaces?filter[workspace][id]=ws-…`.
If the filter parameter name is wrong, the API is likely to **ignore it** and
return every team-workspace grant in the organization — so a workspace would
report grants belonging to other workspaces, and `canApply` audits would fire
against the wrong workspace. This is a wrong-answer failure, not an empty one,
so it is the most damaging item on this list. Verify by comparing the returned
grant ids against the workspace's Team Access page, and by checking that two
different workspaces return **different** grant sets.

### R8 — Pagination parameters and the remote-state-consumers endpoint

`List` always sends `page[number]` and `page[size]=100`. Unverified:

- whether `/workspaces/:id/relationships/remote-state-consumers` accepts them
  (it may ignore them, which the stuck-cursor guard handles, or reject them,
  which would surface as an error),
- whether any endpoint caps `page[size]` below 100 and silently returns fewer,
  which is harmless, versus returning an error, which is not.

Confirm with an organization holding **more than 100 workspaces** if one is
available — that is the only way to observe a real second page. If no such
organization exists, say so: pagination is then unverified in production
conditions and only covered by the httptest unit tests.

### R9 — HCP service principal is required even for Terraform-only queries

`NewHcpConnection` fails without `--client-id`/`--client-secret`, so a user who
only wants HCP Terraform data must still supply HCP credentials they may not
have. This is a usability defect inherited from the existing connection, not
introduced here, but it will be the first thing a user hits. Decide during
verification whether the connection should accept a Terraform-only
configuration.

### R10 — N+1 API calls on relationship resolution

`policySet.policies`, `policySet.workspaces`, `agentPool.allowedWorkspaces`,
`teamAccess.team`, and `teamAccess.workspace` each resolve **one API call per
referenced record** (`NewResource` runs the target's `init` before the runtime
cache is consulted). On a large organization, `workspaces { teamAccess { team } }`
is one `GET /teams/:id` per grant. Measure the request count on the fixture
organization and, if it is bad, switch these to resolve against a
once-fetched list (the pattern in CLAUDE.md §1.5) before release.

### R11 — `resource-count` and `agent-count` field types

Both are decoded as `int64` from a JSON number. If the API returns them as
**strings** (HCP does this elsewhere, e.g. Packer's `version-count`, which this
provider already has a `parseVersionCount` helper for), the decode fails and the
whole record errors out. Check the raw payload types before assuming.

### R12 — Pre-existing duplicate field path

`mqlr generate` warns `duplicate field paths detected: ["hcp.organization"]`.
This predates the change (the `hcp` root has an `organization` field and there is
an `hcp.organization` resource) and is **not** introduced here — the new
`terraformOrganizations` field was named specifically to avoid adding a second
one. It is listed so nobody mistakes it for fallout from this branch, and so
somebody eventually checks whether `hcp.organization` is affected by the
empty-husk failure mode described in CLAUDE.md.

---

## 7. Sign-off

- [ ] every checklist row above ticked, or explicitly marked unverifiable with a reason
- [ ] every risk R1–R12 either closed with evidence, or restated in the PR body as an open blocker
- [ ] the PR body carries the **observed values table**, not a description of the schema
- [ ] `go test ./...` green inside `providers/hcp/`
- [ ] nothing credential-shaped committed (the secret scanner reads every commit, not the final diff)
