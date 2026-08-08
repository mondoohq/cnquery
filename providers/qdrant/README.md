# Qdrant provider (PROTOTYPE - schema design)

Wave 4 of the AI-service coverage concept. This directory currently contains
**only the resource schema** (`resources/qdrant.lr`) as a design prototype for a
net-new Qdrant provider (the vector-database archetype).

It is intentionally **not registered** in `providers.yaml`, so it is not built by
`make providers` and cannot break the build.

## Why Qdrant

Qdrant is a leading open-source vector database used as a RAG backend. Together
with the existing Weaviate provider it establishes the vector-DB coverage
pattern that Pinecone, Milvus/Zilliz, and Chroma follow. Vector stores hold
embeddings of proprietary content, so anonymous access, TLS, and RBAC are the
security-critical controls.

## Resource surface (see `resources/qdrant.lr`)

| Resource | Purpose | Security-relevant fields |
|---|---|---|
| `qdrant` | instance root | `apiKeyRequired`, `tlsEnabled`, `jwtRbacEnabled`, `readOnlyApiKeySupported` |
| `qdrant.collection` | vector collection | `replicationFactor`, `onDiskPayload`, status |

## Remaining steps to make this a working provider

1. `config/config.go` — provider metadata + `Version`
2. `connection/` — HTTP(S) connection with optional API key / JWT
3. `provider/` — `ParseCLI` / `Connect` / `GetData` / `StoreData` / `Disconnect`
4. `gen/main.go`, `main.go`, and registration in the four provider-registry sites
5. `mqlr generate providers/qdrant/resources/qdrant.lr --dist providers/qdrant/resources`
6. Implement the generated interfaces (Pattern A for the collections list; the
   root posture fields come from the instance `/` and `/telemetry` endpoints)
7. Generate `qdrant.permissions.json`, add `.lr.versions` entries
8. **Verify against a live Qdrant instance** before GA

## Planned cnspec security policy (`mondoo-qdrant-security`, Wave 4)

Maps to the AI security control framework (Tier-2 self-hosted / vector DB):

- **Authentication** — API key required (`qdrant.apiKeyRequired == true`)
- **Transport** — TLS enabled (`qdrant.tlsEnabled == true`)
- **Authorization** — JWT RBAC enabled (`qdrant.jwtRbacEnabled == true`)
- **Resilience** — collections replicated (`qdrant.collections.all(replicationFactor > 1)`)

Mirrors the shape of `mondoo-weaviate-security` (Wave 1). The policy ships once
the provider builds and the fields are verified live.
