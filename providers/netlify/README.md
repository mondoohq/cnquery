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
notification hook, and the site's basic-auth password (surfaced as the derived
`passwordProtected` flag instead).
