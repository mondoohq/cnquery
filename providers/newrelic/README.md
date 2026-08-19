# New Relic Provider

The `newrelic` provider inventories a New Relic account and the organization it
belongs to: who can sign in and how, which groups hold which roles over which
accounts, which API and ingest keys exist and how old they are, and which drop
rules and retention settings decide whether telemetry survives long enough to
investigate. It talks to NerdGraph, New Relic's GraphQL API, over read-only
queries.

The provider never requests the keystring of any key it reports, and never
requests the credentials or endpoint URLs attached to a notification
destination.

## Prerequisites

- A New Relic **user key** (the value starts with `NRAK-`). A license key or
  browser key cannot query NerdGraph.
- The numeric **account ID** you want to scan. New Relic scopes keys, alert
  policies, drop rules and retention rules to one account, so an organization
  with several accounts is scanned one account at a time.
- Network access to `api.newrelic.com` (US region) or `api.eu.newrelic.com`
  (EU region).

Some resources read organization-wide data rather than account data and need a
key belonging to a user with organization read permissions:

| Resource | Scope |
|---|---|
| `newrelic.org`, `newrelic.users`, `newrelic.groups`, `newrelic.roles`, `newrelic.accessGrants`, `newrelic.authenticationDomains` | organization |
| `newrelic.authenticationDomain.authenticationType` and the `ssoEnabled` / `passwordLoginEnabled` fields derived from it | organization administration |
| `newrelic.apiKeys`, `newrelic.alertPolicies`, `newrelic.alertConditions`, `newrelic.notificationDestinations`, `newrelic.notificationChannels`, `newrelic.dropRules`, `newrelic.dataRetentionRules` | account |

A key that lacks a permission makes the corresponding field report an error. It
never reports an empty list, because "nobody has access" and "I could not read
who has access" are opposite findings.

## Authentication

Arguments:

- `--api-key` - New Relic user key. Defaults to `$NEW_RELIC_API_KEY`.
- `--account-id` - numeric account ID. Defaults to `$NEW_RELIC_ACCOUNT_ID`.
- `--region` - `us` or `eu`. Defaults to `$NEW_RELIC_REGION`, then `us`.

With environment variables:

```shell
export NEW_RELIC_API_KEY=NRAK-EXAMPLE
export NEW_RELIC_ACCOUNT_ID=1234567
mql shell newrelic
```

With flags, against the EU data region:

```shell
mql shell newrelic --api-key NRAK-EXAMPLE --account-id 1234567 --region eu
```

> Create a user key in the New Relic UI under **API keys**, or with the
> `apiAccessCreateKeys` NerdGraph mutation.

The region is not guessed. An account in the EU region queried against the US
host answers with an empty organization, which would read as an account with no
users and no keys, so an unrecognized region is refused rather than defaulted.

## Usage

Open an interactive shell:

```shell
mql shell newrelic
```

## Examples

**Authentication domains that do not enforce single sign-on**

Every domain has to be checked. An organization commonly runs more than one, and
one domain enforcing single sign-on says nothing about the others.

```shell
mql> newrelic.authenticationDomains.where(ssoEnabled == false) { name authenticationType provisioningType }
```

**Domains that do not provision from a directory**

Without SCIM, a person removed from the corporate directory keeps New Relic
access until somebody deletes the account by hand.

```shell
mql> newrelic.authenticationDomains.where(scimEnabled == false) { name provisioningType users.length }
```

**Ingest keys older than 90 days**

`createdAt` is what a rotation audit reads. New Relic reports no last-used time
for a key, so age is the available signal.

```shell
mql> newrelic.apiKeys.where(type == "INGEST" && createdAt < time.now - 90 * time.day) { name ingestType createdAt }
```

**User keys and who they act as**

An unscoped user key acts with everything its owner can do, so a key belonging
to a departed employee is a standing grant of that person's access.

```shell
mql> newrelic.apiKeys.where(type == "USER") { name createdAt userRef { email lastActive } }
```

**Groups holding organization-wide access**

An organization-wide grant reaches every account under the contract.

```shell
mql> newrelic.accessGrants.where(organizationWide) { roleName groupRef { displayName users.length } }
```

**Drop rules discarding whole events**

A drop rule with the `DROP_DATA` action discards matching telemetry on ingest.
The data is not queryable afterwards and lengthening retention does not bring it
back.

```shell
mql> newrelic.dropRules.where(action == "DROP_DATA") { nrql description creator { email } }
```

**Retention windows actually in force**

New Relic returns deleted rules alongside live ones, so `active` is what
separates a window that applies from a record of one that was removed.

```shell
mql> newrelic.dataRetentionRules.where(active) { namespace retentionInDays }
```

**Alert policies with no conditions**

A policy holding no conditions raises nothing at all.

```shell
mql> newrelic.alertPolicies.where(conditions.length == 0) { name incidentPreference }
```

**Notification destinations that have never delivered anything**

```shell
mql> newrelic.notificationDestinations.where(lastSent == null) { name type active createdAt }
```

## Resources

The resource reference is generated from the schema in
[resources/newrelic.lr](resources/newrelic.lr). The top-level `newrelic`
resource is the entry point to all of them.

## Verification

Confirm the connection and the account scope:

```shell
mql> newrelic.currentAccount { id name }
mql> newrelic.org { id name }
```

`newrelic.currentAccount` fails when the supplied key cannot read the account
named on the command line. That failure is worth acting on: without it, every
account-scoped list below would come back empty and read as a clean account.

## Troubleshooting

**`the supplied New Relic key cannot read account N`** - the key is valid but is
not scoped to that account, or `--account-id` names the wrong account.

**`the New Relic API did not report a login method for authentication domain X`**
- the key lacks organization administration access, so the single sign-on state
of that domain cannot be read. It is reported as an error rather than as
"no single sign-on".

**`the New Relic API could not list drop rules: FEATURE_FLAG_DISABLED`** - drop
rules are not enabled on the account. The list reports the refusal rather than
an empty list, because an empty list would say nothing is being discarded.

**Everything is empty and the account is not** - check `--region`. A US key and
an EU account (or the reverse) authenticate but see nothing.
