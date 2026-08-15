# Artifactory Provider

Query the configuration of a JFrog Artifactory instance: which repositories
exist and what a remote one proxies, which principals hold which actions on
which repository patterns, who is an administrator, which access tokens never
expire, and whether an unauthenticated caller can read or publish.

Artifactory is often the only registry a platform installs from, so a
permission mistake there reaches every consumer of the artifacts it serves.

## Prerequisites

An access token, or the legacy API key. Create an access token in the JFrog
platform under **User Menu > Edit Profile > Generate an Identity Token**, or
with `jf` (`jf atc`).

Most of what this provider reads is administrator-only:

| Resource                        | Rights needed |
| ------------------------------- | ------------- |
| `repositories`, `system`        | any account   |
| `permissionTargets`             | administrator |
| `users`, `groups`               | administrator |
| `accessTokens`                  | administrator to see every token, otherwise only the caller's own |
| `security`                      | administrator |
| `cleanupPolicies`               | administrator, and a product version that serves the endpoint |

A token without those rights reports the field as an error rather than as an
empty result, so an audit does not pass on data that was never read.

## Authentication

Arguments:

- `--url` - the JFrog platform base URL. Also accepted as a bare argument.
- `--token` - an Artifactory access token.
- `--api-key` - a legacy API key, an alternative to the token.

```shell
# flags
mql shell artifactory --url https://example.jfrog.io --token TOKEN

# bare argument
mql shell artifactory https://example.jfrog.io --token TOKEN

# environment variables (ARTIFACTORY_URL or JFROG_URL, ARTIFACTORY_TOKEN or
# JFROG_ACCESS_TOKEN, ARTIFACTORY_API_KEY)
export ARTIFACTORY_URL=https://example.jfrog.io
export ARTIFACTORY_TOKEN=TOKEN
mql shell artifactory
```

### URL shapes

Both deployment shapes are accepted, and so is the URL the web interface shows:

| What you pass                              | What is used                     |
| ------------------------------------------ | -------------------------------- |
| `https://example.jfrog.io`                 | `https://example.jfrog.io`       |
| `https://artifactory.example.com`          | `https://artifactory.example.com`|
| `https://artifactory.example.com/artifactory` | `https://artifactory.example.com` |
| `artifactory.example.com`                  | `https://artifactory.example.com`|

The provider appends the service prefix itself, so a trailing `/artifactory` is
removed rather than doubled. A bare host is assumed to be HTTPS, so the
credential is never sent in the clear.

## Usage

Open an interactive shell:

```shell
mql shell artifactory --url https://example.jfrog.io --token TOKEN
```

The connection produces one asset, the instance. It is identified by the
instance's service identifier, which survives a URL change.

## Examples

**Docker repositories that still accept unsigned schema 1 manifests**

A schema 1 manifest is not content addressable, so a tag can be moved to
different content without the digest changing.

```shell
mql> artifactory.repositories.where(packageType == "docker" && blockPushingSchema1 != true) { key type }
```

**Remote repositories that hold a credential and may leak it on a redirect**

```shell
mql> artifactory.repositories.where(hasUpstreamCredential == true && allowAnyHostAuth == true) { key url }
```

**Anonymous access, and whether it can publish**

The highest-value check. Both halves must hold: anonymous access is on
instance-wide, and a permission target gives the anonymous user a publishing
action.

```shell
mql> artifactory.security { anonymousAccessEnabled anonymousCanRead anonymousCanDeploy }
```

**Which permission targets would let an unauthenticated caller publish**

Listed whether or not anonymous access is on, so that turning it on shows what
would immediately apply.

```shell
mql> artifactory.security.anonymousDeployTargets { name repo { repositories includePatterns } }
```

**What an unauthenticated caller can do to one repository**

```shell
mql> artifactory.repositories.where(key == "example-docker") { key anonymousActions }
```

**Permission targets that cover every repository through a wildcard**

Such a target reaches repositories that did not exist when it was written.

```shell
mql> artifactory.permissionTargets.where(repo.appliesToAllRepositories) { name repo { repositories } }
```

**Grants that are not narrowed to a path**

```shell
mql> artifactory.permissionTargets.where(repo.appliesToAllPaths) { name principals.where(canDeploy) { name type actions } }
```

**Who can publish to the repositories a platform installs from**

```shell
mql> artifactory.repositories.where(key == "example-docker") {
       key
       permissionTargets { name principals.where(canDeploy) { name type actions } }
     }
```

**Principals that can widen their own access**

```shell
mql> artifactory.permissionTargets { name principals.where(canManage) { name type } }
```

**Remote repositories and the upstream they proxy**

An upstream that is not the expected registry places untrusted content behind a
trusted repository name.

```shell
mql> artifactory.repositories.where(type == "remote") { key packageType url }
```

**Remote repositories that may resolve from other hosts, or leak credentials on a redirect**

```shell
mql> artifactory.repositories.where(type == "remote" && (allowAnyHostAuth || externalDependenciesEnabled)) { key url }
```

**Repositories Xray does not index**

An unindexed repository is never scanned, so a finding on an artifact stored
there is never raised.

```shell
mql> artifactory.repositories.where(xrayIndex != true) { key type packageType }
```

Use `!= true` rather than `== false`. The setting is null on repository types
that do not carry it, and `== false` would skip those.

**Administrators, and accounts that are local to the instance**

A local account keeps working after the identity provider disables the person.

```shell
mql> artifactory.users.where(admin) { name realm email }
mql> artifactory.users.where(internal) { name lastLoggedIn }
```

**Groups that carry administrative rights, or that everyone joins**

```shell
mql> artifactory.groups.where(adminPrivileges) { name realm users { name } }
mql> artifactory.groups.where(autoJoin) { name permissionTargets { name } }
```

**Access tokens that never expire**

Such a token stays usable until somebody revokes it by hand.

```shell
mql> artifactory.accessTokens.where(expires == false) { id subject issuedAt }
```

**Access tokens that carry administrative rights**

```shell
mql> artifactory.accessTokens.where(grantsAdmin) { id subject expiry }
```

**Cleanup policies and what they delete from**

```shell
mql> artifactory.cleanupPolicies.where(enabled) { key cronExpression repositories keepLastNVersions }
```

**Cross-resource traversal**

Repositories, permission targets, users, and groups reference each other, so a
query can start from any of them:

```shell
mql> artifactory.users.where(admin) { name permissionTargets { name } }
mql> artifactory.permissionTargets { name principals { name group { adminPrivileges } } }
mql> artifactory.repositories.where(type == "virtual") { key memberRepositoryRefs { key type } }
```

## Verification

Confirm the credential works and see what it reaches:

```shell
mql> artifactory.system { version serviceId }
mql> artifactory.repositories { key type packageType }
```

An error on `permissionTargets` while `repositories` answers means the
credential is valid but not an administrator.

## Notes

**Repository, user, and group detail is read in bulk when the instance allows
it.** Repository configurations come from one call to
`/api/repositories/configurations`, which needs an administrator and a recent
product version. The user and group lists are used directly when they already
carry the full record. When neither applies, the detail is read on demand, one
call per record, so a query that asks only for `key`, `name`, `type`, and
`packageType` stays cheap either way.

A list entry is only taken as complete when it carries both of its markers, the
administrative flag and the group or member list. An entry carrying one of them
is treated as short and read in full, because taking a stale `false` would
report an administrator as an ordinary account, and taking an absent list would
report an account as belonging to no group.

**`anonymousActions` is taken at repository level.** It unions what every
permission target gives the anonymous user over a repository and does not
evaluate the targets' path patterns, so an action it lists may be limited to
part of the repository. Read `permissionTargets` on the repository for the
patterns.

**Settings that do not apply to a repository type stay null.** A local
repository carries no upstream settings, and a package format carries only its
own controls, so `allowAnyHostAuth`, `blockPushingSchema1`, and their peers are
null rather than false there. Test them with `!= true`.

**A permission target that cannot be read fails the field.** The list reports
names only, and each grant is read separately. A target that is denied fails
`permissionTargets` rather than being dropped, because a permission review that
silently skips a target is worse than one that reports it could not read it.

**Cleanup policies need a recent product version.** An instance that does not
serve the endpoint reports an error naming it, rather than an empty list that
would read as "no policies configured".
