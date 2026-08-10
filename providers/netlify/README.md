# Netlify Provider

Query the configuration and security posture of your Netlify accounts and sites:
who can administer an account, what a build is allowed to do, which environment
variables are exposed to runtime code, what is injected into served pages, and
what the account's DNS zones point at.

## Prerequisites

A Netlify personal access token. Create one under **User settings >
Applications > Personal access tokens** in the Netlify app.

The token inherits the roles of the account that issued it. A token issued by a
member rather than an owner cannot read the account roster or some site
settings, and those fields come back null rather than failing the scan.

## Authentication

Arguments:

- `--token` - the Netlify personal access token.
- `--account` - optional, restrict the scan to one account (slug or ID).

```shell
# flag
mql shell netlify --token TOKEN

# environment variable (NETLIFY_AUTH_TOKEN or NETLIFY_TOKEN)
export NETLIFY_AUTH_TOKEN=TOKEN
mql shell netlify
```

## Usage

Open an interactive shell:

```shell
mql shell netlify --token TOKEN
```

## Discovery

Connecting to `netlify` discovers two kinds of assets:

- **`netlify-account`** - every account the token can access (one asset per account).
- **`netlify-site`** - every site within those accounts.

Scope discovery with `--discover`:

```shell
cnspec scan netlify --token TOKEN --discover accounts
cnspec scan netlify --token TOKEN --discover sites
```

Restrict the scan to one account with `--account` (slug or ID). The flag
narrows both discovery and plain queries, so the same connection sees the same
accounts either way:

```shell
cnspec scan netlify --token TOKEN --account acme
```

## Examples

**Accounts that do not enforce multi-factor authentication**

```shell
mql> netlify.accounts.where(enforceMfa == "not_enforced") { slug typeName }
```

**Members without a second factor, and owners in particular**

```shell
mql> netlify.accounts { slug members.where(mfaEnabled == false) { email role } }
mql> netlify.accounts { slug owners.where(mfaEnabled == false) { email } }
```

**Invitations still outstanding**

```shell
mql> netlify.accounts { slug members.where(pending == true) { email role } }
```

**Email domains that can join the team without an invitation**

```shell
mql> netlify.accounts.where(teamRegistrationDomains.length > 0) { slug teamRegistrationDomains }
```

**SAML sessions that stay valid for longer than a day**

```shell
mql> netlify.accounts.where(samlEnabled == true && samlSessionExpiration > 86400) { slug samlSessionExpiration }
```

**Accounts where Netlify support may administer the team**

```shell
mql> netlify.accounts.where(supportAdministrationEnabled == true) { slug }
```

**Sites that let an outside pull request build unreviewed**

```shell
mql> netlify.sites.where(untrustedFlow != "review") { name untrustedFlow repoUrl }
```

**Sites that do not redirect visitors to HTTPS**

```shell
mql> netlify.sites.where(forceSsl == false) { name url ssl }
```

**Production deploys that can be published outside of git**

```shell
mql> netlify.sites.where(preventNonGitProdDeploys == false) { name repoUrl }
```

**Environment variables exposed to runtime code but not stored as secrets**

```shell
mql> netlify.sites {
       name
       environmentVariables.where(isSecret == false && scopes.contains("runtime")) { key }
     }
```

**Sites with public build logs**

```shell
mql> netlify.sites.where(privateLogs == false) { name repoUrl }
```

**Scripts injected into served pages outside the repository**

```shell
mql> netlify.sites { name snippets { title generalPosition general } }
```

**Build hooks that deploy production without authentication**

```shell
mql> netlify.sites { name buildHooks.where(branch == "main") { title createdAt } }
```

**Account owners**

```shell
mql> netlify.accounts { slug owners { email role } }
```

**DNS records delegating a name somewhere else**

```shell
mql> netlify.dnsZones { name records.where(type == "CNAME" && managed == false) { hostname value } }
```

**Deploy keys no site clones with**

```shell
mql> netlify.deployKeys.where(sites.length == 0) { id createdAt }
```

**Cross-resource traversal**

Sites, accounts, DNS zones, and deploy keys reference each other, so a query can
start from any of them:

```shell
mql> netlify.dnsZones { name site { name forceSsl } }
mql> netlify.sites { name account { slug typeName } }
mql> netlify.sites { name deployKey { id createdAt } }
```

## Verification

Confirm the token works and see what it can reach:

```shell
mql> netlify.currentUser { email fullName loginProviders }
mql> netlify.accounts { slug name typeName }
```

An empty account list means the token was accepted but is not a member of any
account. A null `members` list on an account means the token lacks the
administrative rights to read the roster.

## Troubleshooting

**`a valid Netlify token is required`** - neither `--token` nor
`NETLIFY_AUTH_TOKEN` / `NETLIFY_TOKEN` was set.

**`netlify API /accounts: 401`** - the token is expired or was revoked. Issue a
new one under User settings > Applications.

**Fields come back null on an account or site** - the token's account role does
not grant the read. Reads that are denied stay null rather than reporting an
empty result, so an audit does not pass on data that was never read.

## Notes

Some values the API returns are bearer secrets and are deliberately not exposed
as fields: the trigger URL of a build hook, the delivery settings of a
notification hook, and the account's site JWT secret.

**Site password protection is reported per account, not per site.** The API does
not return a site's visitor password, or any per-site flag derived from it, on
either the site list or the site detail response. The account-level
`hasSitePassword` is the only readable signal, so a per-site field would have
reported every protected site as unprotected.

**A build control the site has never set reports null**, not false. `privateLogs`,
`skipPrs`, and `skipAutomaticBuilds` are absent from the API until they are
configured, and the site follows the team default until then.
