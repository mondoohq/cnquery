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
- Most records now decode into **`newrelic-client-go` v2.93.1** types, so their
  struct tags are vendor-maintained and continuously exercised by
  `terraform-provider-newrelic` rather than hand-written here. That removes the
  mistyped-tag class of bug from those records. It removes nothing else: the
  queries, the pagination walks and the client are still this provider's own,
  for the reasons in §6 and §7.

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

Most of this section was originally inferred from the schema and the
documentation. It has since been checked against the official
`newrelic/newrelic-client-go` client at **v2.93.1**, whose types this provider
now decodes into for most collections. Where the vendor's generated types settle
a question, the entry says so and is struck or downgraded; where they only
corroborate an assumption, it stays open.

Adopting the vendor's types does **not** make any of the remaining entries safe.
They are questions about what the *server* returns, and a type cannot answer
those.

The ranking is by **how wrong the answer would be**, not by how likely the
problem is. A query that fails loudly is near the bottom; a filter that returns
the wrong rows is at the top, because nothing downstream can tell.

### Rank 1: WRONG ANSWER, silent: authentication domain identity across the two views

**Still open. The SDK corroborates the assumption rather than settling it.**

The domain list comes from `actor.organization.userManagement.authenticationDomains`
and the login method from `customerAdministration.authenticationDomains`, joined
on the domain **ID**. This assumes the two views number domains identically. If
they do not, `ssoEnabled` would be attributed to the wrong domain, which is the
worst possible failure for this provider: it would report the password-only
domain as SSO-protected.

What the SDK tells us: it models the two views as **two distinct types** —
`usermanagement.UserManagementAuthenticationDomain` (which carries **no**
`authenticationType` field at all) and
`customeradministration.OrganizationAuthenticationDomain` (which does). Both are
keyed by `id`. So the vendor's own client, and `terraform-provider-newrelic` on
top of it, makes exactly the same join this provider makes. That is evidence the
join is the intended one. It is **not** proof the ID spaces coincide, and no
amount of type-reading can make it proof.

- [ ] For every domain, check that `name` from the user-management view and
      `name` from `customerAdministration` (via GraphiQL) agree **for the same
      ID**. Not just that both lists have the same length.

### Rank 2: WRONG ANSWER, silent: `apiAccess.keySearch` scope filter

**Still open, and the SDK makes one half of it worse rather than better.**

`resources/api.go`, `apiKeysQuery`:

```graphql
keySearch(query: { types: [INGEST, USER], scope: { accountIds: [$accountId] } }, cursor: $cursor)
```

1. **The scope may narrow more than intended.** If `scope.accountIds` also
   implicitly filters user keys, the result could omit user keys that belong to
   the organization rather than to the account. The audit then reports "no user
   keys" on an account that has them. The SDK's
   `APIAccessKeySearchScope{AccountIDs, IngestTypes, UserIDs}` documents the
   shape but says nothing about the server-side semantics.
2. **`types` is mandatory.** Confirmed by the SDK:
   `APIAccessKeySearchQuery.Types` is tagged `json:"types"` with **no**
   `omitempty`, where `Scope` has it. If New Relic adds a third key type, this
   query silently stops returning it.

Also: New Relic's docs state that *"user keys belonging to other users will be
obfuscated in the results"*. It is **not established** what obfuscation does to
`id`, `name`, `createdAt` and `userId`.

- [ ] Compare `newrelic.apiKeys.length` against the API keys UI, counted by hand,
      for **both** key types, with a key that owns some and does not own others.
- [ ] Confirm `userRef` resolves for a user key owned by **somebody else**.

### Rank 3: WRONG ANSWER, silent: `customerAdministration.authenticationDomains(filter: {})`

**Downgraded. The vendor's client sends the same empty filter.**

The filter argument is non-null in the schema
(`OrganizationAuthenticationDomainFilterInput!`), so an empty object is passed.

The SDK confirms this is the intended construction:
`OrganizationAuthenticationDomainFilterInput` has exactly three fields (`ID`,
`Name`, `OrganizationId`), all optional pointers with `omitempty`, and
`GetAuthenticationDomainsWithContext` passes whatever filter it is given
straight through — callers routinely pass the zero value. So `{}` meaning "no
filter" is what the vendor assumes too.

What remains unobserved is the server-side behaviour in a
**multi-organization** account: if an empty filter scopes to a *default*
organization, the returned domains would be the wrong ones, and a domain ID that
happened to match would report another organization's SSO setting as this one's.

- [ ] Compare the domain IDs returned by
      `newrelic.authenticationDomains { id name }` against the IDs
      `customerAdministration` reports, by running the raw query in GraphiQL.
      The two sets must be identical.
- [ ] Confirm `authenticationType` per domain against the UI, domain by domain.

### Rank 4: WRONG ANSWER, silent: `organizationWide` on an access grant

**Downgraded. Confirmed against the read type, not a mutation sample.**

`isOrganizationWideGrant` requires `organizationId != ""` and `accountId <= 0`.
The original worry was that this mapping had been read off a *mutation's* sample
response.

The SDK settles that much: `authorizationmanagement.AuthorizationManagementGrantedRole`
is the type returned by the **read** path (`AuthorizationManagementGrantedRoleSearch`),
and it carries `AccountID int` and `OrganizationId string`, both `omitempty`,
alongside `GroupId` and `RoleId`. The field shape is right and this provider now
decodes into that exact type.

What is still unobserved is whether a real read returns them **mutually
exclusively**. If an account-scoped grant carries both an account ID and an
organization ID, org-wide grants would read as `false` and the broadest access
in the organization would be invisible to a policy checking for it.

- [ ] Create one org-wide grant and one account grant, then read
      `newrelic.accessGrants { roleName accountId organizationWide }` and confirm
      the flag matches the UI's "Access" panel for each.

### ~~Rank 5: alert conditions joined to policies by `policyId`~~ — STRUCK

Settled by the SDK. `alerts.NrqlAlertCondition.PolicyID` is a `string`
(`json:"policyId,omitempty"`) and `alerts.AlertsPolicy.ID` is a `string`
(`json:"id"`). Both sides of the join are strings in the vendor's generated
types, which `terraform-provider-newrelic` exercises continuously. This provider
now decodes into those exact types, so the string/number mismatch that would
have made every policy report zero conditions cannot occur.

The per-policy count is still worth eyeballing once against the UI, but it is no
longer a ranked risk.

### Rank 6: WRONG ANSWER, silent: user and group membership read from one side

**Unchanged — the SDK has nothing to say about it.**

`group.users` is computed by scanning the user list for users whose `groups`
contain the group ID, rather than by asking for the group's members. If New
Relic ever populates one direction and not the other (for instance for a group
provisioned by SCIM), `group.users` would report an empty group.

- [ ] Cross-check rows 29 and 33: every membership visible from the user side
      must be visible from the group side and vice versa.

### Rank 7: the `cursor` argument — mostly confirmed, one case INVERTED

Previously "the cursor argument may not exist on every collection". The SDK
settles most of it and **reverses** one case, which is now the most important
thing in this entry.

Confirmed to accept a cursor: `policiesSearch`, `nrqlConditionsSearch`
(`alerts.SearchNrqlConditionsQueryWithContext` pages on it),
`aiNotifications.destinations` and `channels` (`GetChannels` /
`GetDestinationsAccount` both take `cursor *string`),
`userManagement.authenticationDomains` and its nested `users`.

**Inverted — `customerAdministration.authenticationDomains` does NOT accept a
cursor.** See §7 below. This provider passes one anyway, which is the right call
if the argument exists and harmless if it does not, but it means the domain list
cannot be paged past the first page until this is confirmed against a real
organization with more domains than one page returns.

Still inferred, because the SDK offers no read path for them:
`authorizationManagement.authenticationDomains`, its nested `groups`, and
`authorizationManagement.roles`.

- [ ] Run each list once and confirm none of them errors with an unknown
      argument.
- [ ] Specifically confirm whether `customerAdministration.authenticationDomains`
      accepts `cursor`. If it does not, this provider's query is wrong in the
      same way the SDK's is, and the walk has to be driven some other way.

### Rank 8: fields inferred but not confirmed — mostly STRUCK

Field names are now confirmed against the vendor's generated types for:

| Collection | Confirmed by |
|---|---|
| `authorizationManagement.roles` | `AuthorizationManagementRole{ID,Name,DisplayName,Type,Scope}` |
| `authorizationManagement…groups.roles` | `AuthorizationManagementGrantedRole{ID,Name,DisplayName,Type,RoleId,AccountID,OrganizationId,GroupId}` |
| `nrqlDropRules.list` | `NRQLDropRulesDropRule{Source,Creator,CreatedBy,…}` and `NRQLDropRulesError{Description,Reason}` |
| `aiNotifications.destinations` | `AiNotificationsDestination{IsUserAuthenticated,LastSent,AccountID,…}` |
| notification destination types | `AiNotificationsDestinationTypeTypes` lists exactly the 14 values the `.lr` doc-comment enumerates |

**Still unconfirmed:** `dataManagement.eventRetentionRules` — its
`createdById` / `deletedById` field types. The SDK does not model event
retention rules at all (see §6), so there is nothing to check them against.

- [ ] Run every top-level list once with a full-access key and record which, if
      any, fail at validation.

### Rank 9: UNDER-REPORTING, loud: unpageable nested collections

**Unchanged.** Two nested cursors cannot be followed, because no documented
follow-up query exists for them. Both are reported as errors rather than
truncated, so the failure is loud and nothing under-reports:

- A group holding more access grants than fit on one page.
- A user belonging to more groups than fit on one page.

- [ ] Confirm no group in the fixture organization trips the grants error.
- [ ] Confirm no user trips the groups error.

### Rank 10: timestamp shapes

**Unchanged, and the SDK is no help here — see §7.**

`nrTime` accepts ISO 8601 strings and epoch seconds/milliseconds, and **errors**
on anything else rather than reporting null.

- [ ] Confirm no list errors with "could not decode the New Relic timestamp".

### Rank 11: region and endpoint

Only `us` and `eu` are supported. The FedRAMP endpoint (`gov-api.newrelic.com`)
is not.

- [ ] If a FedRAMP account is in scope, this is a gap to file, not a bug.

### Rank 12: NEW — an absent alert threshold is reported as `0`

Surfaced by the swap. `alerts.NrqlConditionTerm.Threshold` is a **`*float64`**,
which means New Relic can return a term with no threshold at all. This provider
flattens that to `0` in the `terms` dict, which is the behaviour it had before
the swap and was deliberately preserved rather than changed inside a
type-only change.

A threshold of `0` is a meaningful value for some operators, so "absent" and
"zero" are not safely interchangeable. Whether an absent threshold should report
as null instead is a **schema decision**, not a drive-by fix: it changes an
observable value.

- [ ] Determine whether New Relic ever returns a term with no threshold, and if
      so, decide whether `terms[].threshold` should be null in that case.

---

## 6. Why three collections keep local types

This provider decodes into `newrelic-client-go` types for most collections, so
the struct tags are vendor-maintained. **Three deliberately do not.** Each is
pinned by a test, because each is exactly the kind of thing a later reader would
"complete" without realising what it costs.

Do not swap these to the SDK's types without reading this section first.

### 6.1 API keys — the SDK's types and query both carry the keystring

`apiaccess.APIAccessKey`, `APIAccessIngestKey` and `APIAccessUserKey` all carry
a `Key string` field, and the SDK's only read query selects it —
**four times**, at `pkg/apiaccess/keys.go:220`
(`graphqlAPIAccessKeyBaseFields`): once at the top level and once inside each of
the three inline fragments.

This provider's `.lr` schema states, of `newrelic.apiKeys`: *"The keystring is
never requested, so no field on the returned keys carries usable key material."*
Adopting the SDK's query would make that shipped sentence false. Not asking for
the secret is the only guarantee that cannot be undone by a later change to the
mapping code.

**Additionally, the SDK's key search cannot paginate at all.** The query at
`pkg/apiaccess/keys.go:292` takes no cursor argument, and the response struct at
`:203` has **no `nextCursor` field** — pagination is not merely unimplemented,
it is un-modelled. Adopting it would silently truncate the key list to one page,
and a short key list reads as "no unrotated keys".

Pinned by: `TestAPIKeysQueryNeverAsksForTheKeystring`,
`TestNoDecodeTargetCarriesCredentialFields`.

### 6.2 Notification destinations and channels — unmasked credential fields

`notifications.AiNotificationsDestination` carries `Auth ai.AiNotificationsAuth`,
`Properties []AiNotificationsProperty` and `SecureURL AiNotificationsSecureURL`,
and the SDK's destinations query selects all three (with inline fragments for
Basic, OAuth2, Token and CustomHeaders auth). The channels query selects
`properties`.

The auth values are masked server-side, but **`AiNotificationsProperty.Value` is
a plain unmasked `string`** and routinely holds a webhook URL complete with the
token in its path, along with channel IDs and recipient lists.
`ai.AiNotificationsAuth` additionally carries a `Password` field.

Pinned by: `TestNotificationQueriesNeverAskForCredentials`,
`TestNoDecodeTargetCarriesCredentialFields`.

### 6.3 Event retention rules — the SDK does not model them at all

There is nothing to adopt. Neither `RetentionInDays` nor `eventRetentionRule`
appears **anywhere** in `newrelic-client-go` v2.93.1, and `pkg/datamanagement`
models account limits only (`DataManagementAccountLimit` and friends). Do not go
looking for a type to swap in; there isn't one.

### 6.4 Also local, for a lesser reason: `apiOrganization`

`organization.Organization` is a stitched-fields root carrying every namespace
hung off an organization (`AccountManagement`, `AccountShares`,
`AuthorizationManagement`, `Administrator`, …). This provider reads two scalars
from it, so a two-field local struct is used instead. This one is a size
judgement, not a safety one.

---

## 7. Hazards in `newrelic-client-go` that this provider does NOT inherit

Found while adopting the vendor types at v2.93.1, verified against the source in
the module cache. Recorded here because anyone who later reaches for the SDK's
*queries* or *client* rather than its types inherits all of them.

### 7.1 `customerAdministration.GetAuthenticationDomains` accepts a cursor and silently discards it

`pkg/customeradministration/customeradministration_api.go:190-226`. The method
signature takes `cursor string` and places it in the variables map:

```go
vars := map[string]interface{}{"cursor": cursor, "filter": filter, "sort": sort}
```

but the query it sends declares only two variables and references neither a
cursor nor a page argument:

```graphql
query($filter: OrganizationAuthenticationDomainFilterInput!,
      $sort: [OrganizationAuthenticationDomainSortInput!]) {
  customerAdministration { authenticationDomains(filter: $filter, sort: $sort) {
    items { authenticationType id name organizationId provisioningType }
    nextCursor
  } } }
```

`$cursor` is never declared and never used. The method therefore returns **page
one forever**, while still handing the caller a `nextCursor` that makes it look
like paging is working. Anyone who adopts this method inherits a silent
truncation of the authentication domain list — and the login method is read from
that list, so a truncated result means domains whose `ssoEnabled` cannot be
resolved.

This provider sends its own query with a declared `$cursor` and drives it
through `walkPages`, whose repeated-cursor guard turns a server that ignores the
cursor into a loud error rather than a short list.

### 7.2 `SearchAPIAccessKeys` does not paginate

`pkg/apiaccess/keys.go:292` and `:203`. No cursor argument on the query, no
`nextCursor` on the response struct, a single request and no loop. See §6.1.

### 7.3 `SearchNrqlConditions` loops with no cycle guard and no page cap

`alerts.SearchNrqlConditionsQueryWithContext` loops while `nextCursor != nil`,
with no seen-cursor set and no maximum page count. A server that repeats a
cursor spins **unbounded** inside the SDK. This provider's `walkPages` errors on
a repeated cursor and caps at `maxPages`.

### 7.4 The SDK's timestamps cannot express "absent"

`nrtime.DateTime` is declared `type DateTime string` — a bare, **unparsed**
ISO 8601 string with no decoding whatsoever. Separately,
`apiaccess.APIAccessKey.CreatedAt` is a plain `int64`, where epoch `0` is
indistinguishable from "no creation time".

Mapping either naively reports the original account keys — which genuinely have
no creation time — as **1970-01-01**, the exact invented default the `.lr`
doc-comment promises not to produce (*"Null when New Relic reports no creation
time for the key"*).

This provider keeps `resources/nrtime.go` and shadows the SDK's timestamp fields
with it. Go's `encoding/json` gives the shallower field precedence, so the outer
`nrTime` wins in both directions and null stays null.

### 7.5 The SDK's error type is unimportable and cannot be classified

The GraphQL error type lives in `internal/http`, so `errors.As` cannot name it
from outside the module. `GraphQLErrorResponse.Error()` flattens every error to
`strings.Join(messages, ", ")`, and **`IsNotFound()` returns `false`
unconditionally**.

Routing this provider's queries through the SDK's client would reduce
`IsForbidden` / `IsNotFound` to matching on message prose, which New Relic is
free to reword. This provider keeps `connection/client.go`, which classifies on
the HTTP status and on `extensions.errorClass`, and never reads a transport
failure as an absence.

---
## 8. What to do with the results

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
