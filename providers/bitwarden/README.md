# Bitwarden Provider

The `bitwarden` provider connects to a Bitwarden Teams or Enterprise
organization through its Public API and inventories organization governance:
security policies, members, collections, and groups. It authenticates as an
organization, not a user, and never reads vault item secrets, folders, or
Send content, since the Public API has no endpoints for them.

**Client note:** ADR-037 specifies a client generated from Bitwarden's
Public API OpenAPI v3 spec via `oapi-codegen`. This implementation instead
ships a minimal, hand-written `net/http` client (`connection/client.go`)
covering only the read endpoints the current schema needs, since generating
and vendoring the OpenAPI spec was out of scope for this pass. The
production intent per ADR-037 remains an OpenAPI-generated client; adopting
it is a drop-in replacement for `connection/client.go` behind the same
`connection.Client` interface used by `resources/`.

## Prerequisites

The Public API is available to Bitwarden Teams and Enterprise organizations
(not Free/Families). You need an organization API key (`client_id` /
`client_secret`), obtained from the organization's Settings > Organization
info page in the Bitwarden web vault.

## Authentication

Authentication uses OAuth2 client-credentials: an organization `client_id`
(of the form `organization.<uuid>`) and `client_secret` are exchanged at
Bitwarden's identity endpoint for a bearer token.

Arguments:

- `--client-id` - the organization client ID (e.g. `organization.<uuid>`).
- `--client-secret` - the organization client secret.
- `--api-url` - optional, overrides the Public API base URL (self-hosted deployments).
- `--identity-url` - optional, overrides the identity token URL (self-hosted deployments).

```shell
mql shell bitwarden --client-id organization.00000000-0000-0000-0000-000000000000 --client-secret <secret>
```

> You can also set `BITWARDEN_CLIENT_ID` and `BITWARDEN_CLIENT_SECRET`
> (and, for self-hosted deployments, `BITWARDEN_API_URL` /
> `BITWARDEN_IDENTITY_URL`) as environment variables instead of flags.

## Usage

Open an interactive shell:

```shell
mql shell bitwarden
```

## Discovery

This provider models a single asset per connection: the Bitwarden
organization itself. There are no child assets to discover.

## Examples

**Organization-wide security policies**

```shell
mql> bitwarden.policies { policyType enabled data }
```

**Members without two-factor authentication**

```shell
mql> bitwarden.members.where( status == "confirmed" ).where( ! twoFactorEnabled ) { email role }
```

**Groups and the collections they can reach**

```shell
mql> bitwarden.groups { name collections { name } }
```

## Resources

See the `.lr` schema comments in `resources/bitwarden.lr` for the full
resource reference: `bitwarden`, `bitwarden.organization`,
`bitwarden.policy`, `bitwarden.member`, `bitwarden.collection`, and
`bitwarden.group`.

## Verification

```shell
mql> bitwarden.organization
mql> bitwarden.policies.length
```

An empty `policies` list on an organization known to have policies
configured usually means the API key lacks the `api.organization` scope, or
the organization's plan doesn't include policy enforcement.

## Troubleshooting

- **`a Bitwarden organization client ID and client secret are required`**:
  set `--client-id`/`--client-secret` or the `BITWARDEN_CLIENT_ID` /
  `BITWARDEN_CLIENT_SECRET` environment variables.
- **`failed to verify Bitwarden credentials`**: the client ID/secret pair
  was rejected during token exchange, or the organization's plan doesn't
  include Public API access (Teams/Enterprise only).
- **Self-hosted deployments**: set `--api-url`/`--identity-url` (or
  `BITWARDEN_API_URL`/`BITWARDEN_IDENTITY_URL`) to your instance's
  endpoints.
