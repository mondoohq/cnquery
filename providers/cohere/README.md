# Cohere provider (PROTOTYPE - schema design)

Wave 3 of the AI-service coverage concept. This directory currently contains
**only the resource schema** (`resources/cohere.lr`) as a design prototype for a
net-new Cohere provider (a Tier-1 foundation-model SaaS).

It is intentionally **not registered** in `providers.yaml`, so it is not built by
`make providers` and cannot break the build.

## Why Cohere

Cohere is a widely adopted enterprise foundation-model API with no mql provider
today. Its retrieval **connectors** (which reach into third-party systems for
RAG) are a distinctive AI-specific security surface that generic cloud posture
tools miss.

## Resource surface (see `resources/cohere.lr`)

| Resource | Purpose | Security-relevant fields |
|---|---|---|
| `cohere` | account root | models, datasets, fineTunedModels, connectors |
| `cohere.organization` | org root (admin key) | members, apiKeys |
| `cohere.organization.member` | RBAC | `role` (admin/member), email |
| `cohere.organization.apiKey` | credentials | `keyType` (trial/production), createdAt, lastUsedAt |
| `cohere.model` | model inventory | endpoints, isFinetuned (AIBOM) |
| `cohere.dataset` | uploaded data | datasetType, validationStatus, keepOriginalFile |
| `cohere.fineTunedModel` | model lineage | status, baseModel |
| `cohere.connector` | RAG connectors | `authType` (none/oauth/service-token), `active`, url |

## Remaining steps to make this a working provider

1. `config/config.go` — provider metadata + `Version` (initial release version)
2. `connection/` — API-key connection (Cohere `Authorization: Bearer` + admin key)
3. `provider/` — `ParseCLI` / `Connect` / `GetData` / `StoreData` / `Disconnect`
4. `gen/main.go`, `main.go`, and registration in the four provider-registry sites
5. `mqlr generate providers/cohere/resources/cohere.lr --dist providers/cohere/resources`
6. Implement the generated interfaces (Pattern A immediate mapping for list APIs;
   Pattern B init for lazy/keyed lookups) per the repo mql CLAUDE.md
7. Generate `cohere.permissions.json`, add `.lr.versions` entries at the initial
   version
8. **Verify against the live Cohere API** with a real admin key before GA

## Planned cnspec security policy (`mondoo-cohere-security`, Wave 3)

Maps to the normalized AI security control framework:

- **Identity** — limit org admins (`cohere.organization.members.where(role == "admin").length <= N`)
- **Credential hygiene** — no unused production keys; production keys rotated
  (`cohere.organization.apiKeys.where(keyType == "production").all(lastUsedAt > time.now - 90*time.day)`)
- **Connector security** — no unauthenticated active connectors
  (`cohere.connectors.where(active == true).all(authType != "none")`); prefer per-user OAuth
- **Data governance** — datasets do not retain original files unnecessarily
  (`cohere.datasets.all(keepOriginalFile == false)`)

The policy ships once the provider builds and the fields are verified live.
