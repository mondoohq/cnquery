# DigitalOcean Provider

Query and assess your DigitalOcean account: Droplets, Kubernetes (DOKS), managed databases, Spaces, load balancers, firewalls, VPCs, App Platform, Functions, the container registry, GradientAI, and more.

## Prerequisites

You need a DigitalOcean **personal access token (PAT)**:

1. Go to **DigitalOcean Control Panel > API > Tokens** and generate a new token.
2. A **read-only** token is sufficient for inventory and assessment. DigitalOcean also supports [custom-scoped tokens](https://docs.digitalocean.com/reference/api/create-personal-access-token/) if you want to restrict the token to specific resource types.

Auditing **Spaces** buckets additionally requires a Spaces access key/secret pair (see below), because Spaces uses the S3-compatible API rather than the DigitalOcean API.

## Authentication

### Environment variables (recommended)

```bash
export DIGITALOCEAN_TOKEN="<your-api-token>"

# Optional — only needed to audit Spaces buckets (S3-compatible API):
export DIGITALOCEAN_SPACES_KEY="<spaces-access-key>"
export DIGITALOCEAN_SPACES_SECRET="<spaces-secret>"
# Optional — restrict Spaces bucket listing to a single region; otherwise
# the provider iterates the known Spaces regions:
export DIGITALOCEAN_SPACES_REGION="nyc3"
```

### CLI flag

```bash
cnspec shell digitalocean --token <api-token>
```

If `DIGITALOCEAN_TOKEN` is set, you can omit the flag:

```bash
cnspec shell digitalocean
```

## Discovery

The provider connects to the account as a single asset and can expand it into child assets. Pass discovery targets with `--discover`:

| Target | Expands into |
|---|---|
| `auto` (default) | The account asset |
| `all` | Everything below |
| `databases` | Managed database clusters |
| `kubernetes` | DOKS clusters |
| `loadbalancers` | Load balancers |
| `firewalls` | Cloud firewalls |
| `spaces-buckets` | Spaces buckets (requires Spaces credentials) |

```bash
cnspec shell digitalocean --discover all
```

## Notes

- **Spaces** resources return an empty list unless `DIGITALOCEAN_SPACES_KEY` and `DIGITALOCEAN_SPACES_SECRET` are set.
- **Functions** are backed by Apache OpenWhisk. Listing the functions (actions) deployed in a namespace reaches the namespace's Functions API host using credentials retrieved from the DigitalOcean API.
- Resources the token cannot read (for example, a scoped token) are handled gracefully where possible.

## Examples

Launch an interactive shell and run queries:

```bash
cnspec shell digitalocean
```

List Droplets and flag any without a firewall:

```mql
digitalocean.droplets.where(missingFirewall)
```

Find internet-reachable Droplets:

```mql
digitalocean.droplets.where(exposure.internetReachable)
```

Check managed databases reachable from the internet:

```mql
digitalocean.databases.where(internetReachable)
```

Review App Platform deployment history and configured alerts:

```mql
digitalocean.apps { name deployments { phase cause createdAt } alerts { rule disabled } }
```

Audit web-exported Functions that don't require an API key:

```mql
digitalocean.functionNamespaces.functions.where(webExported && requiresApiKey == false)
```

Inspect Kubernetes clusters and their node pools:

```mql
digitalocean.kubernetesClusters { name version nodePools { name size count } }
```

You can also run a policy scan:

```bash
cnspec scan digitalocean
```
