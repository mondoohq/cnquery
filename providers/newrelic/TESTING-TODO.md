# New Relic provider: live verification TODO

**Status: NOT VERIFIED against a live New Relic account.**

Every resource and field in this provider was written from the NerdGraph schema
as the official `newrelic/newrelic-client-go` client and the New Relic docs
describe it. Nothing here has been read back from a real account. The unit
tests prove the decoding, the pagination walks and the error classifiers, but a
fixture written from documentation reproduces the documentation, so they cannot
prove that the queries return what the schema claims.

This file is the handover. Work through it in order. Everything is unchecked.

**What has been proven already**, so you do not re-spend the time:

- The provider loads through the real plugin boundary, parses its flags, builds
  the asset and dispatches queries. `PROVIDERS_PATH=/tmp/pd ./mql run newrelic
  --api-key … --account-id … -c 'newrelic.currentAccount'` reaches the resource
  and fails at the network, not before it.
- Every query in section 2 and section 4.4 **compiles** against the generated
  schema, including the nested and cross-resource ones. That rules out the two
  schema traps that produce silent nulls (a list field colliding with its
  element resource type, and a field colliding with a resource's dotted path).
- The connection refuses a non-numeric account ID and an unrecognized region
  before any request is made.

None of that says anything about whether the queries return the right values.
That is what the rest of this file is for.

---

## 1. Credentials and fixtures needed

### 1.1 Environment

```shell
export NEW_RELIC_API_KEY=NRAK-…      # a user key, NOT a license key
export NEW_RELIC_ACCOUNT_ID=…        # numeric account ID
export NEW_RELIC_REGION=us           # or eu
```

The key must be a **user key** (its value starts with `NRAK-`). An ingest or
license key cannot query NerdGraph at all and will fail at the first request.

### 1.2 Permissions the key's user needs

| Needed for | Permission |
|---|---|
| `newrelic.currentAccount`, `newrelic.accounts` | read access to the named account |
| `newrelic.apiKeys` | account: API keys read |
| `newrelic.alertPolicies`, `newrelic.alertConditions` | account: alerts read |
| `newrelic.notificationDestinations`, `newrelic.notificationChannels` | account: applied intelligence / notifications read |
| `newrelic.dropRules` | account: NRQL drop rules read, with drop rules enabled on the subscription |
| `newrelic.dataRetentionRules` | account: data retention read |
| `newrelic.org`, `newrelic.users`, `newrelic.groups`, `newrelic.roles`, `newrelic.accessGrants`, `newrelic.authenticationDomains` | organization read |
| `newrelic.authenticationDomain.authenticationType` / `ssoEnabled` / `passwordLoginEnabled` | **organization administration** (the `customerAdministration` NerdGraph root) |

- [ ] Run the whole checklist once with a **full-access organization admin key**,
      to prove the fields work at all.
- [ ] Run it a second time with a **deliberately narrow key** (account read
      only, no organization access). Every organization-scoped field must report
      an **error**, never an empty list and never `false`. This is the check
      that separates "no one has admin access" from "I could not read who
      does", and it is the single most valuable run in this document.

### 1.3 What must exist in the account before you start

| Fixture | Why |
|---|---|
| At least two authentication domains, **one with SAML or OIDC single sign-on enforced and one without** | `ssoEnabled` two-state check |
| One of those domains provisioned over **SCIM**, one **MANUAL** | `scimEnabled` two-state check |
| A user with a **restricted (non-admin) role** and a user with an admin role | `accessGrants`, `roles`, `group.users` |
| At least one group with an **organization-wide** grant and one with an **account-scoped** grant | `organizationWide` two-state check |
| A user who has **never signed in** (freshly invited) | `lastActive` must be **null**, not 1970 |
| A user with a **pending upgrade request** | `pendingUpgrade` / `requestedType` |
| An **ingest (license) key** and a **user key** | `type` / `ingestType` split |
| An **API key older than 90 days** and one created today | `createdAt` age check |
| At least one of the **original account keys** (created with the account, undeletable) | `createdAt` must be **null**, not 1970 |
| One **NRQL drop rule** with action `DROP_DATA`, one with `DROP_ATTRIBUTES` | `action` two-state check |
| A **custom event retention rule** on one namespace, plus a **deleted** retention rule on the same namespace | `active` two-state check |
| An **alert policy with conditions** and one **with none** | `conditions.length` |
| At least one **disabled** alert condition | `enabled` two-state check |
| A **notification destination that has delivered** and one that never has | `lastSent` null vs set |
| A notification **channel** bound to one of those destinations | `destination` reference |

> If the account cannot supply a fixture, say so next to that row rather than
> ticking it. An unticked row is information; a ticked row that was never
> exercised is not.

### 1.4 Build under test

`make providers/install/newrelic` **copies from `dist/`, it does not build.** A
failed or skipped build leaves the previous binary in place and install ships
it, so the verification silently tests the wrong code. Always build first, and
never install over your working provider while verifying.

```shell
cd <repo root>
git checkout claude/provider-newrelic

make providers/build/newrelic          # builds into providers/newrelic/dist/
ls -l providers/newrelic/dist/newrelic # check the mtime is from this build

mkdir -p /tmp/pd/newrelic
cp providers/newrelic/dist/newrelic \
   providers/newrelic/dist/newrelic.json \
   providers/newrelic/dist/newrelic.resources.json \
   /tmp/pd/newrelic/

make mql/build                         # ./mql in the repo root

PROVIDERS_PATH=/tmp/pd ./mql shell newrelic
```

`PROVIDERS_PATH` replaces the provider search path entirely, so nothing here
touches `~/.config/mondoo/providers`.

Run a query non-interactively with:

```shell
PROVIDERS_PATH=/tmp/pd ./mql run newrelic -c '<query>'
```

Add `-j` for JSON.

- [ ] Provider builds and the `dist/` mtime is from this build
- [ ] `PROVIDERS_PATH=/tmp/pd ./mql run newrelic -c 'newrelic.currentAccount { id name }'` returns the expected account

---

## 2. Per-field checklist

Every field, the query that reads it, and what to expect. Tick a row only after
reading the value and agreeing with it.

### 2.1 Connection and account

| # | Field | Query | Expected | ✓ |
|---|---|---|---|---|
| 1 | `newrelic.currentAccount` | `newrelic.currentAccount { id name }` | the account named in `NEW_RELIC_ACCOUNT_ID`, with its real name | ☐ |
| 2 | `newrelic.account.id` | as above | equals `$NEW_RELIC_ACCOUNT_ID` | ☐ |
| 3 | `newrelic.account.name` | as above | matches the New Relic UI | ☐ |
| 4 | `newrelic.accounts` | `newrelic.accounts { id name }` | every account the key can read, matching the account switcher in the UI | ☐ |
| 5 | `newrelic.account(id:)` selector | `newrelic.account(id: <ID>) { name }` | resolves that account; an ID the key cannot see must **error**, not return a blank account | ☐ |
| 6 | `newrelic.org` | `newrelic.org { id name }` | the organization the account belongs to | ☐ |
| 7 | `newrelic.organization.id` / `.name` | as above | match the UI's organization settings page | ☐ |

### 2.2 Authentication domains

| # | Field | Query | Expected | ✓ |
|---|---|---|---|---|
| 8 | `newrelic.authenticationDomains` | `newrelic.authenticationDomains { id name }` | **every** domain, count matches the UI | ☐ |
| 9 | `.id` | as above | matches the domain ID in the UI URL | ☐ |
| 10 | `.name` | as above | matches the UI | ☐ |
| 11 | `.provisioningType` | `newrelic.authenticationDomains { name provisioningType }` | `SCIM` / `MANUAL` / `DISABLED` matching each domain's setting | ☐ |
| 12 | `.scimEnabled` | `newrelic.authenticationDomains { name scimEnabled }` | `true` **only** on the SCIM domain | ☐ |
| 13 | `.authenticationType` | `newrelic.authenticationDomains { name authenticationType }` | `SAML_SSO` / `OIDC_SSO` / `PASSWORD` / `DISABLED` matching each domain | ☐ |
| 14 | `.ssoEnabled` | `newrelic.authenticationDomains { name ssoEnabled }` | `true` on the SSO domain, `false` on the password one | ☐ |
| 15 | `.passwordLoginEnabled` | `newrelic.authenticationDomains { name passwordLoginEnabled }` | `true` only on the `PASSWORD` domain | ☐ |
| 16 | `.users` | `newrelic.authenticationDomains { name users.length }` | per-domain user counts match the UI | ☐ |
| 17 | `.groups` | `newrelic.authenticationDomains { name groups.length }` | per-domain group counts match the UI | ☐ |

### 2.3 Users

| # | Field | Query | Expected | ✓ |
|---|---|---|---|---|
| 18 | `newrelic.users` | `newrelic.users.length` | total user count across all domains, matching the UI | ☐ |
| 19 | `.id` | `newrelic.users { id email }` | matches the user ID in the UI | ☐ |
| 20 | `.name` | as above | full name | ☐ |
| 21 | `.email` | as above | email address | ☐ |
| 22 | `.type` | `newrelic.users { email type }` | `Basic` / `Core` / `Full platform` matching each user's tier | ☐ |
| 23 | `.emailVerificationState` | `newrelic.users { email emailVerificationState }` | `Verified` / `Pending` / `Not Verifiable` | ☐ |
| 24 | `.timeZone` | `newrelic.users { email timeZone }` | an IANA zone such as `Europe/London` | ☐ |
| 25 | `.lastActive` | `newrelic.users { email lastActive }` | a real date for an active user, and **null** for the never-signed-in user (never 1970, never year 1) | ☐ |
| 26 | `.pendingUpgrade` | `newrelic.users.where(pendingUpgrade) { email requestedType }` | exactly the user with the open upgrade request | ☐ |
| 27 | `.requestedType` | as above | the tier requested; empty for everyone else | ☐ |
| 28 | `.authenticationDomain` | `newrelic.users { email authenticationDomain { name } }` | each user's real domain, **never null** for a user returned by the same query | ☐ |
| 29 | `.groups` | `newrelic.users { email groups { displayName } }` | matches the group membership shown on the user in the UI | ☐ |

### 2.4 Groups, roles and access grants

| # | Field | Query | Expected | ✓ |
|---|---|---|---|---|
| 30 | `newrelic.groups` | `newrelic.groups { id displayName }` | every group across every domain | ☐ |
| 31 | `.displayName` | as above | matches the UI | ☐ |
| 32 | `.authenticationDomain` | `newrelic.groups { displayName authenticationDomain { name } }` | the domain the group is defined in, never null | ☐ |
| 33 | `.users` | `newrelic.groups { displayName users { email } }` | matches the membership shown in the UI, **in both directions** (cross-check against row 29) | ☐ |
| 34 | `.accessGrants` | `newrelic.groups { displayName accessGrants { roleName accountId } }` | matches the group's "Manage access" panel | ☐ |
| 35 | `newrelic.roles` | `newrelic.roles { id name type scope }` | every role, both `STANDARD` and any `CUSTOM` ones | ☐ |
| 36 | `.type` | as above | `STANDARD` for built-in roles, `CUSTOM` for authored ones | ☐ |
| 37 | `.scope` | as above | `ACCOUNT` or `ORGANIZATION` matching the role | ☐ |
| 38 | `.displayName` | as above | matches the UI | ☐ |
| 39 | `newrelic.accessGrants` | `newrelic.accessGrants.length` | equals the sum of `newrelic.groups { accessGrants.length }` | ☐ |
| 40 | `.roleName` / `.roleDisplayName` / `.roleType` | `newrelic.accessGrants { roleName roleDisplayName roleType }` | match the granted role | ☐ |
| 41 | `.groupId` | `newrelic.accessGrants { groupId roleName }` | never empty | ☐ |
| 42 | `.accountId` | `newrelic.accessGrants { accountId organizationWide }` | the granted account, or `0` for an organization-wide grant | ☐ |
| 43 | `.organizationWide` | `newrelic.accessGrants.where(organizationWide) { roleName groupId }` | exactly the org-wide grants, and **nothing else** | ☐ |
| 44 | `.roleRef` | `newrelic.accessGrants { roleName roleRef { name scope } }` | resolves; `roleRef.name` matches `roleName` on every row | ☐ |
| 45 | `.groupRef` | `newrelic.accessGrants { groupRef { displayName } }` | resolves to the holding group, never null | ☐ |
| 46 | `.account` | `newrelic.accessGrants { account { name } organizationWide }` | resolves for account grants; **null** for org-wide grants | ☐ |
| 47 | grant identity | `newrelic.accessGrants { __id }` then `newrelic.accessGrants.length` vs `newrelic.accessGrants.map(__id).unique.length` | the two lengths are **equal**: no two grants share a cache key | ☐ |

### 2.5 API and ingest keys

| # | Field | Query | Expected | ✓ |
|---|---|---|---|---|
| 48 | `newrelic.apiKeys` | `newrelic.apiKeys.length` | matches the key count in the API keys UI for this account | ☐ |
| 49 | `.id` | `newrelic.apiKeys { id name }` | matches the key ID in the UI (not the keystring) | ☐ |
| 50 | `.name` / `.notes` | as above | match the UI | ☐ |
| 51 | `.type` | `newrelic.apiKeys { name type }` | `INGEST` and `USER` both appear | ☐ |
| 52 | `.ingestType` | `newrelic.apiKeys.where(type == "INGEST") { name ingestType }` | `LICENSE` or `BROWSER`; **empty** on every `USER` key | ☐ |
| 53 | `.createdAt` | `newrelic.apiKeys { name createdAt }` | real dates; **null** on the original account key; never 1970 | ☐ |
| 54 | `.account` | `newrelic.apiKeys { name account { id } }` | the account the key belongs to | ☐ |
| 55 | `.userRef` | `newrelic.apiKeys.where(type == "USER") { name userRef { email } }` | resolves to the owning user; **null** on every `INGEST` key | ☐ |
| 56 | key identity | `newrelic.apiKeys.length` vs `newrelic.apiKeys.map(__id).unique.length` | equal | ☐ |

### 2.6 Alerting

| # | Field | Query | Expected | ✓ |
|---|---|---|---|---|
| 57 | `newrelic.alertPolicies` | `newrelic.alertPolicies.length` | matches the policy count in the UI | ☐ |
| 58 | `.name` / `.id` | `newrelic.alertPolicies { id name }` | match the UI | ☐ |
| 59 | `.incidentPreference` | `newrelic.alertPolicies { name incidentPreference }` | `PER_POLICY` / `PER_CONDITION` / `PER_CONDITION_AND_TARGET` | ☐ |
| 60 | `.account` | `newrelic.alertPolicies { name account { id } }` | the scanned account | ☐ |
| 61 | `.conditions` | `newrelic.alertPolicies { name conditions.length }` | matches each policy's condition count; the empty policy reports `0` | ☐ |
| 62 | `newrelic.alertConditions` | `newrelic.alertConditions.length` | equals the sum of the per-policy counts in row 61 | ☐ |
| 63 | `.enabled` | `newrelic.alertConditions { name enabled }` | `false` on the deliberately disabled condition, `true` elsewhere | ☐ |
| 64 | `.type` | `newrelic.alertConditions { name type }` | `STATIC` / `BASELINE` / `OUTLIER` | ☐ |
| 65 | `.nrql` | `newrelic.alertConditions { name nrql }` | the real NRQL query text, not empty | ☐ |
| 66 | `.description` / `.runbookUrl` | `newrelic.alertConditions { name description runbookUrl }` | match the UI; empty where unset | ☐ |
| 67 | `.violationTimeLimitSeconds` | as above | matches the condition's incident-close setting | ☐ |
| 68 | `.terms` | `newrelic.alertConditions { name terms }` | one dict per threshold, with `operator`, `priority`, `threshold`, `thresholdDuration`, `thresholdOccurrences` all populated | ☐ |
| 69 | `.policy` | `newrelic.alertConditions { name policy { name } }` | resolves to the owning policy, never null for a condition in the list | ☐ |

### 2.7 Notifications

| # | Field | Query | Expected | ✓ |
|---|---|---|---|---|
| 70 | `newrelic.notificationDestinations` | `newrelic.notificationDestinations.length` | matches the destination count in the UI | ☐ |
| 71 | `.name` / `.id` | `newrelic.notificationDestinations { id name }` | match the UI | ☐ |
| 72 | `.type` | `newrelic.notificationDestinations { name type }` | `EMAIL` / `WEBHOOK` / `SLACK` / … matching each destination | ☐ |
| 73 | `.active` | `newrelic.notificationDestinations { name active }` | matches the UI toggle | ☐ |
| 74 | `.status` | as above | `DEFAULT` / `DRAFT` / `ERROR` / `TESTING` / `THROTTLED` | ☐ |
| 75 | `.userAuthenticated` | `newrelic.notificationDestinations { name userAuthenticated }` | `true` for a destination holding a personal authorization | ☐ |
| 76 | `.createdAt` / `.updatedAt` | as above | real dates | ☐ |
| 77 | `.lastSent` | `newrelic.notificationDestinations { name lastSent }` | a real date on the used destination, **null** on the never-used one | ☐ |
| 78 | `.account` | `newrelic.notificationDestinations { name account { id } }` | the scanned account | ☐ |
| 79 | `newrelic.notificationChannels` | `newrelic.notificationChannels { id name type product }` | matches the UI | ☐ |
| 80 | `.product` | as above | `IINT` / `NTFC` / `ALERTS` / … | ☐ |
| 81 | `.active` / `.status` | `newrelic.notificationChannels { name active status }` | match the UI | ☐ |
| 82 | `.createdAt` / `.updatedAt` | as above | real dates | ☐ |
| 83 | `.destination` | `newrelic.notificationChannels { name destination { name type } }` | resolves to the bound destination, never null | ☐ |
| 84 | `.account` | `newrelic.notificationChannels { name account { id } }` | the scanned account | ☐ |

### 2.8 Data handling

| # | Field | Query | Expected | ✓ |
|---|---|---|---|---|
| 85 | `newrelic.dropRules` | `newrelic.dropRules.length` | matches the drop rule count in the UI | ☐ |
| 86 | `.id` | `newrelic.dropRules { id action }` | matches the UI | ☐ |
| 87 | `.action` | `newrelic.dropRules { action nrql }` | both `DROP_DATA` and `DROP_ATTRIBUTES` appear | ☐ |
| 88 | `.nrql` | as above | the real selection query, not empty | ☐ |
| 89 | `.description` | `newrelic.dropRules { description }` | matches the UI; empty where unset | ☐ |
| 90 | `.source` | `newrelic.dropRules { source }` | whatever registered the rule; empty is acceptable | ☐ |
| 91 | `.createdAt` | `newrelic.dropRules { createdAt }` | a real date, never 1970 | ☐ |
| 92 | `.account` | `newrelic.dropRules { account { id } }` | the scanned account | ☐ |
| 93 | `.creator` | `newrelic.dropRules { creator { email } }` | resolves for a rule created by a person; **null** for a system-created rule | ☐ |
| 94 | `newrelic.dataRetentionRules` | `newrelic.dataRetentionRules { namespace retentionInDays active }` | includes the custom rule you created | ☐ |
| 95 | `.namespace` | as above | `APM` / `Log` / `Metric` / … matching the UI | ☐ |
| 96 | `.retentionInDays` | as above | matches the window set in the UI | ☐ |
| 97 | `.active` | `newrelic.dataRetentionRules { namespace active deletedAt }` | `true` on the live rule, `false` on the deleted one **for the same namespace** | ☐ |
| 98 | `.createdAt` | as above | real date | ☐ |
| 99 | `.deletedAt` | as above | **null** on the live rule, a real date on the deleted one | ☐ |
| 100 | `.createdBy` / `.deletedBy` | `newrelic.dataRetentionRules { createdBy { email } deletedBy { email } }` | resolve where the user still exists; null otherwise | ☐ |
| 101 | retention identity | `newrelic.dataRetentionRules.length` vs `.map(__id).unique.length` | equal (the live and deleted rules on one namespace must not collide) | ☐ |

---

## 3. Two-state requirement

A field that reads the same before and after you change the setting is either a
resource bug or a fixture that never moved, and one state cannot tell those
apart. Each of these must be read **twice**, with the setting flipped between
the readings, and both readings recorded.

| # | Field | State A | State B | ✓ |
|---|---|---|---|---|
| 102 | `authenticationDomain.ssoEnabled` | a domain with SAML/OIDC enforced → `true` | a domain on `PASSWORD` → `false` | ☐ |
| 103 | `authenticationDomain.passwordLoginEnabled` | password domain → `true` | SSO domain → `false` | ☐ |
| 104 | `authenticationDomain.scimEnabled` | SCIM domain → `true` | MANUAL domain → `false` | ☐ |
| 105 | `dataRetentionRule.active` | live rule → `true` | deleted rule → `false` | ☐ |
| 106 | `dropRule` presence | rule created → appears in the list | rule deleted → disappears | ☐ |
| 107 | `alertCondition.enabled` | enabled condition → `true` | disabled condition → `false` | ☐ |
| 108 | `notificationDestination.active` | active → `true` | deactivated → `false` | ☐ |
| 109 | `notificationDestination.lastSent` | after a test notification → a date | never used → `null` | ☐ |
| 110 | `accessGrant.organizationWide` | org-wide grant → `true` | account grant → `false` | ☐ |
| 111 | `user.pendingUpgrade` | request open → `true` | no request → `false` | ☐ |
| 112 | `user.lastActive` | user who signed in → a date | invited, never signed in → `null` | ☐ |
| 113 | `apiKey.ingestType` | ingest key → `LICENSE`/`BROWSER` | user key → empty | ☐ |
| 114 | `apiKey.userRef` | user key → the owner | ingest key → `null` | ☐ |

> Flipping SSO on a production authentication domain locks people out. Use a
> **dedicated test domain** with no real users in it, or a trial organization.

---

## 4. The four checks from `new-resource` §5

### 4.1 The absent case must FAIL, not pass vacuously

A check that reads null as satisfied is worse than no check. Prove each of these
**errors or reports false**, and never quietly passes.

- [ ] **Key without organization access.** With an account-only key:
      `newrelic.authenticationDomains` must **error**, not return `[]`.
- [ ] **Key without organization administration.** With an organization-read but
      not organization-admin key:
      `newrelic.authenticationDomains { ssoEnabled }` must **error**, not report
      `false`. Confirm the message names the domain.
- [ ] **Wrong account.** `--account-id` set to an account the key cannot read:
      `newrelic.currentAccount` must **error**, not return a blank account.
- [ ] **Drop rules not enabled.** On an account without the drop rules feature:
      `newrelic.dropRules` must **error** carrying the API's reason, not return
      `[]`. An empty list would assert that nothing is being discarded.
- [ ] **Null combination.** `newrelic.authenticationDomains.all(ssoEnabled && scimEnabled)`
      on a key that cannot read either must **not** pass. (MQL's three-valued
      logic makes `null && null` evaluate to `true`, so this is exactly the
      shape that passes vacuously if a field returns null instead of erroring.)

### 4.2 Null over invented defaults

- [ ] `apiKey.createdAt` on the original account key is **null**, not
      `1970-01-01` and not `0001-01-01`.
- [ ] `user.lastActive` on a never-signed-in user is **null**.
- [ ] `notificationDestination.lastSent` on a never-used destination is **null**.
- [ ] `dataRetentionRule.deletedAt` on a live rule is **null**.
- [ ] `dropRule.createdAt` is a real date, and null rather than 1970 if the API
      omits it.
- [ ] No field reports a value the API did not supply. Spot-check with
      `-j` and compare against the raw NerdGraph response from
      <https://api.newrelic.com/graphiql>.

### 4.3 Secret sweep: must return 0

The provider's guarantee is that it never asks NerdGraph for key material. Prove
it from the outside, not by reading the code:

```shell
# 1. Take a real ingest key's keystring from the New Relic UI.
KEYSTRING='<paste the real keystring here>'

# 2. Dump everything the provider can produce.
PROVIDERS_PATH=/tmp/pd ./mql run newrelic -j -c '
  newrelic.apiKeys { * }
' > /tmp/nr-keys.json

PROVIDERS_PATH=/tmp/pd ./mql run newrelic -j -c '
  newrelic.notificationDestinations { * }
  newrelic.notificationChannels { * }
' > /tmp/nr-notifications.json

# 3. Both counts must be 0.
grep -c "$KEYSTRING" /tmp/nr-keys.json
grep -c "$KEYSTRING" /tmp/nr-notifications.json
```

- [ ] `grep -c` returns **0** for the ingest keystring
- [ ] `grep -c` returns **0** for a user keystring (`NRAK-…`)
- [ ] `grep -Ec 'NRAK-|NRJS-|NRII-|NRBR-'` over both dumps returns **0**
      (New Relic's key prefixes; a match means a keystring leaked)
- [ ] For a webhook destination whose URL carries a token in its path, `grep -c`
      for that token returns **0**
- [ ] Delete `/tmp/nr-*.json` afterwards

State the guarantee precisely when reporting: *no field of `newrelic.apiKey`
carries key material, because the keystring is never requested.* Do not write
"the secret is not exposed" more broadly than that.

### 4.4 Composability

The point of the typed references is that a policy can express a join in one
query. Each of these must return real data, not empty lists:

- [ ] `newrelic.users.where(lastActive == null) { email groups { displayName accessGrants { roleName } } }`
      (invited-but-never-active users and what they were already granted)
- [ ] `newrelic.apiKeys.where(type == "USER") { name userRef { email authenticationDomain { ssoEnabled } } }`
      (user keys owned by people in a domain without single sign-on)
- [ ] `newrelic.accessGrants.where(organizationWide) { roleName groupRef { users { email } } }`
      (who actually holds organization-wide access, by name)
- [ ] `newrelic.alertPolicies.where(conditions.all(enabled == false)) { name }`
      (policies whose every condition is switched off)
- [ ] `newrelic.notificationChannels { name destination { name lastSent } }`
      (channels pointing at destinations that have never delivered)

---

## 5. Risk areas, ranked by damage

Everything below was inferred from the schema and the documentation, not
observed. The ranking is by **how wrong the answer would be**, not by how likely
the problem is. A query that fails loudly is near the bottom; a filter that
returns the wrong rows is at the top, because nothing downstream can tell.

### Rank 1: WRONG ANSWER, silent: `apiAccess.keySearch` scope filter

`resources/api.go`, `apiKeysQuery`:

```graphql
keySearch(query: { types: [INGEST, USER], scope: { accountIds: [$accountId] } }, cursor: $cursor)
```

Two ways this returns a confidently wrong answer:

1. **The scope narrows more than intended.** If `scope.accountIds` also
   implicitly filters user keys (the schema documents `userIds` as user-key-only
   and `ingestTypes` as ingest-key-only, but says nothing about `accountIds`),
   the result could omit user keys that belong to the organization rather than
   to the account. The audit then reports "no user keys" on an account that has
   them.
2. **`types` is mandatory and the list must be complete.** If New Relic adds a
   third key type, this query silently stops returning it.

Also: New Relic's docs state that *"user keys belonging to other users will be
obfuscated in the results"*. It is **not established** what obfuscation does to
`id`, `name`, `createdAt` and `userId`. If those are blanked, `apiKeys` may
report other people's keys with null owners and null creation times, which reads
as a fleet of unattributable keys.

- [ ] Compare `newrelic.apiKeys.length` against the API keys UI, counted by hand,
      for **both** key types, with a key that owns some and does not own others.
- [ ] Confirm `userRef` resolves for a user key owned by **somebody else**.

### Rank 2: WRONG ANSWER, silent: `customerAdministration.authenticationDomains(filter: {})`

`resources/api.go`, `adminAuthDomainsQuery`. The filter argument is non-null in
the schema (`OrganizationAuthenticationDomainFilterInput!`), so an empty object
is passed. Every field inside it is optional, so `{}` *should* mean "no filter".

If New Relic instead treats an empty filter as matching nothing, the map comes
back empty and **every** domain's `authenticationType` lookup fails with "did
not report a login method". That is at least a loud failure. The dangerous
variant is the opposite: if the filter silently scopes to a *default*
organization in a multi-organization account, the returned domains would be the
wrong ones, and a domain ID that happened to match would report **another
organization's** SSO setting as this one's.

- [ ] Compare the domain IDs returned by
      `newrelic.authenticationDomains { id name }` against the IDs
      `customerAdministration` reports, by running the raw query in GraphiQL.
      The two sets must be identical.
- [ ] Confirm `authenticationType` per domain against the UI, domain by domain,
      not just "at least one domain says SAML_SSO".

### Rank 3: WRONG ANSWER, silent: authentication domain identity across the two views

The domain list comes from `actor.organization.userManagement.authenticationDomains`
and the login method from `customerAdministration.authenticationDomains`, joined
on the domain **ID**. This assumes the two views number domains identically. If
they do not, `ssoEnabled` would be attributed to the wrong domain, which is the
worst possible failure for this provider: it would report the password-only
domain as SSO-protected.

- [ ] For every domain, check that `name` from the user-management view and
      `name` from `customerAdministration` (via GraphiQL) agree **for the same
      ID**. Not just that both lists have the same length.

### Rank 4: WRONG ANSWER, silent: `organizationWide` on an access grant

`isOrganizationWideGrant` requires `organizationId != ""` and `accountId <= 0`.
That mapping was read off a mutation's sample response, not off a real read
query. If a real read returns both an account ID and an organization ID on an
account-scoped grant, org-wide grants would read as `false` and the broadest
access in the organization would be invisible to a policy checking for it.

- [ ] Create one org-wide grant and one account grant, then read
      `newrelic.accessGrants { roleName accountId organizationWide }` and confirm
      the flag matches the UI's "Access" panel for each.

### Rank 5: WRONG ANSWER, silent: alert conditions joined to policies by `policyId`

`alertPolicy.conditions` filters the account-wide condition list on
`condition.policyId == policy.id`. Both are strings in the schema. If either
side is returned as a number and serialized differently (`"3455"` vs `3455`), or
if `policyId` is scoped per account in a way `id` is not, every policy would
report **zero** conditions and `alertPolicies.where(conditions.length == 0)`
would flag every policy in the account.

- [ ] `newrelic.alertPolicies { name conditions.length }` must sum to
      `newrelic.alertConditions.length`, and each policy's count must match the
      UI.

### Rank 6: WRONG ANSWER, silent: user and group membership read from one side

`group.users` is computed by scanning the user list for users whose `groups`
contain the group ID, rather than by asking for the group's members. If New
Relic ever populates one direction and not the other (for instance for a group
provisioned by SCIM), `group.users` would report an empty group.

- [ ] Cross-check rows 29 and 33: every membership visible from the user side
      must be visible from the group side and vice versa.

### Rank 7: LOUD FAILURE: the `cursor` argument may not exist on every collection

Every paginated query declares `$cursor: String` and passes it. This is
confirmed by New Relic's own documentation for `policiesSearch`,
`aiNotifications.destinations`, `userManagement.authenticationDomains` and
`userManagement…users`, and by the official Go client for
`nrqlConditionsSearch`. It is **inferred** for:

- `apiAccess.keySearch`
- `customerAdministration.authenticationDomains`
- `authorizationManagement.authenticationDomains`, its nested `groups`, and
  `authorizationManagement.roles`

If any of those does not accept `cursor`, GraphQL rejects the whole query at
validation and the corresponding list errors. That is a loud failure, not a
wrong answer, which is why it ranks below the filter risks.

- [ ] Run each list once and confirm none of them errors with an unknown
      argument.

### Rank 8: LOUD FAILURE: fields inferred but not confirmed

These field selections come from the generated client's types rather than from a
documented query, so a name could be wrong. A wrong field name fails the whole
query at validation.

| Query | Inferred selection |
|---|---|
| `authorizationManagement.roles` | `roles { id name displayName type scope }` |
| `authorizationManagement…groups.roles` | `roles { id name displayName type roleId accountId organizationId groupId }` |
| `dataManagement.eventRetentionRules` | `createdById` / `deletedById` types (documented as fields, type unconfirmed) |
| `aiNotifications.destinations` | `isUserAuthenticated`, `lastSent`, `accountId` |
| `nrqlDropRules.list` | `source`, `creator { id name email }` |

- [ ] Run every top-level list once with a full-access key and record which, if
      any, fail at validation.

### Rank 9: UNDER-REPORTING, loud: unpageable nested collections

Two nested cursors cannot be followed, because no documented follow-up query
exists for them. Both are reported as errors rather than truncated, so the
failure is loud and nothing under-reports:

- A group holding more access grants than fit on one page
  (`fetchGroupsWithGrants` returns "more access grants than one page returns").
- A user belonging to more groups than fit on one page
  (`fetchAuthDomainsWithUsers` returns "belongs to more groups than one page
  returns").

- [ ] Confirm no group in the fixture organization trips the grants error.
- [ ] Confirm no user trips the groups error. If either fires in a real
      organization, a follow-up query has to be found for that nesting level.

### Rank 10: timestamp shapes

`nrTime` accepts ISO 8601 strings and epoch seconds/milliseconds, and **errors**
on anything else rather than reporting null. API keys are documented as epoch
seconds; everything else is documented as the ISO 8601 `DateTime` scalar. The
retention rules' `createdAt` / `deletedAt` types are not documented at all.

- [ ] Confirm no list errors with "could not decode the New Relic timestamp".
      If one does, the raw shape is in the GraphiQL response and
      `resources/nrtime.go` needs the layout added.

### Rank 11: region and endpoint

Only `us` and `eu` are supported. The FedRAMP endpoint (`gov-api.newrelic.com`)
is not.

- [ ] If a FedRAMP account is in scope, this is a gap to file, not a bug.

---

## 6. What to do with the results

1. Record the **observed value** for each row, not just a tick. A table of what
   the queries returned is what lets a reviewer disagree with you about
   something real.
2. Any row where the observed value differs from the expectation is a provider
   bug: fix it on `claude/provider-newrelic` and add a unit test that contains
   the shape that produced it. A fix whose fixture does not carry the triggering
   shape leaves the suite silent about the thing you just fixed.
3. Any row that could not be exercised stays unticked, and the reason goes next
   to it. "No fixture available" is a blocker to record, not a disclaimer to
   merge past.
4. Re-run `go test ./...` inside `providers/newrelic/` after any fix.
