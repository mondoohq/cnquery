# OpenAI Provider

Query OpenAI account resources for AI Bill of Materials (AIBOM) generation, security auditing, and compliance. This provider inventories models, fine-tuning jobs, uploaded files, vector stores, and organization projects via the OpenAI API.

## Prerequisites

- An OpenAI API key ([platform.openai.com/api-keys](https://platform.openai.com/api-keys))
- For admin resources (users, invites, audit logs, projects, API keys, service accounts): an **admin API key** ([platform.openai.com/organization/admin-keys](https://platform.openai.com/organization/admin-keys))

### API key types

OpenAI uses two separate key types with non-overlapping scopes. The provider auto-detects the key type from its prefix and adjusts which resources are available:

| Key type | Prefix | Resources |
|---|---|---|
| Project key | `sk-proj-...` | models, files, fine-tuning jobs, vector stores |
| Admin key | `sk-admin-...` | users, invites, audit logs, projects, project API keys, service accounts |

Admin keys cannot access data-plane resources (models, files, etc.) and project keys cannot access admin resources (users, projects, etc.). Resources for which no matching key is configured gracefully return empty results.

## Authentication

```bash
# Project key — data-plane resources (models, files, fine-tuning, vector stores)
export OPENAI_API_KEY="sk-proj-..."
mql shell openai

# Admin key — organization resources (users, invites, projects, audit logs)
export OPENAI_API_KEY="sk-admin-..."
mql shell openai

# Via CLI flag
mql shell openai --token "sk-proj-..."
```

### Configuration options

| Flag             | Environment Variable   | Description                                                        |
|------------------|------------------------|--------------------------------------------------------------------|
| `--token`        | `OPENAI_API_KEY`       | API key — project key (`sk-proj-...`) or admin key (`sk-admin-...`), auto-detected |
| `--organization` | `OPENAI_ORG_ID`        | Restrict queries to an organization                                |
| `--project`      | `OPENAI_PROJECT_ID`    | Restrict queries to a project                                      |
| `--base-url`     | `OPENAI_BASE_URL`      | Custom API endpoint (e.g., proxy)                                  |

## Examples

### List all available models

```shell
mql> openai.models { id ownedBy createdAt }
```

### Find fine-tuned models

```shell
mql> openai.models.where(isFineTuned == true) { id baseModel ownedBy }
```

### List uploaded files with their purpose

```shell
mql> openai.files { id filename purpose bytes status }
```

### Audit fine-tuning jobs and their training data

```shell
mql> openai.fineTuningJobs { id model status fineTunedModel trainedTokens trainingFile { filename } }
```

### Find failed fine-tuning jobs

```shell
mql> openai.fineTuningJobs.where(status == "failed") { id model error }
```

### Inspect vector stores

```shell
mql> openai.vectorStores { id name status usageBytes fileCounts }
```

### List organization projects (requires admin key)

```shell
mql> openai.projects { id name status createdAt }
```

### List organization users and their roles (requires admin key)

```shell
mql> openai.users { id email name role addedAt apiKeyLastUsedAt }
```

### Find inactive users (no API key usage)

```shell
mql> openai.users.where(apiKeyLastUsedAt == null) { email name role addedAt }
```

### Audit pending invites (requires admin key)

```shell
mql> openai.invites { id email role status createdAt expiresAt }
```

### Review audit logs (requires admin key)

```shell
mql> openai.auditLogs { id type effectiveAt actorType actorId }
```

### List project API keys and their owners (requires admin key)

```shell
mql> openai.projects { name apiKeys { name ownerType ownerName createdAt lastUsedAt } }
```

### List project service accounts (requires admin key)

```shell
mql> openai.projects { name serviceAccounts { name role createdAt } }
```

### AIBOM inventory query

Combine model inventory with fine-tuning lineage for a complete AI supply chain view:

```shell
mql> openai.models { id ownedBy isFineTuned baseModel createdAt }
mql> openai.fineTuningJobs.where(status == "succeeded") { model fineTunedModel trainingFile { filename bytes } trainedTokens }
mql> openai.files.where(purpose == "fine-tune") { id filename bytes createdAt }
```

## Resources

| Resource                       | Description                                       |
|--------------------------------|---------------------------------------------------|
| `openai`                       | Root resource — organization, project, model list  |
| `openai.model`                 | AI model (base or fine-tuned)                     |
| `openai.file`                  | Uploaded file (training data, batch I/O, etc.)    |
| `openai.fineTuningJob`         | Fine-tuning job with training lineage             |
| `openai.vectorStore`           | Vector store for knowledge retrieval              |
| `openai.project`               | Organization project (requires admin key)         |
| `openai.project.apiKey`        | Project API key with ownership info               |
| `openai.project.serviceAccount`| Project service account                           |
| `openai.organizationUser`      | Organization member with role and usage info      |
| `openai.invite`                | Pending organization invite                       |
| `openai.auditLog`              | Organization audit log entry                      |

## AIBOM Coverage

This provider supports AI Bill of Materials generation by exposing:

- **Model inventory**: All accessible base and fine-tuned models with ownership
- **Model lineage**: Fine-tuning jobs link base model to fine-tuned model with training data references
- **Data assets**: Uploaded files (training data, validation data, batch I/O) and vector stores
- **Organization context**: Projects for access isolation and governance

The AIBOM query pack (in cnspec) maps these resources to CycloneDX ML-BOM components:

| OpenAI Resource Field          | CycloneDX Mapping                                |
|-------------------------------|--------------------------------------------------|
| `openai.model.id`             | `component.name`                                 |
| `openai.model.ownedBy`        | `component.author`                               |
| `openai.model.createdAt`      | `component.properties["mondoo:model:created"]`   |
| `openai.model.baseModel`      | `modelCard.modelParameters.baseModelId`           |
| `openai.fineTuningJob.model`  | Model lineage (base model reference)              |
| `openai.file` (training)      | Training data provenance                          |
| PURL (generated by cnspec)    | `component.purl` (`pkg:openai/<model-id>`)       |
