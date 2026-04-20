# ADR: mql SBOM Cataloger for Jenkins Plugins

**Status:** Proposed
**Date:** 2026-04-18

---

## Context

cnspec needs native SBOM cataloging for Jenkins plugins to detect which plugins (and versions) are installed on Jenkins instances. Jenkins plugins are a supply chain vector — knowing installed plugins and their versions is critical for vulnerability management (e.g., Jenkins security advisories frequently target specific plugin versions).

Jenkins plugins are distributed as `.jpi` or `.hpi` files (both are ZIP archives) containing a `META-INF/MANIFEST.MF` with plugin metadata.

## Decision

Add `jenkins.packages` and `jenkins.package` as new MQL resources in the OS provider. The parser scans a directory for `.jpi`/`.hpi` files and extracts metadata from their `MANIFEST.MF` files, reusing the existing Java archive scanning infrastructure.

## Data Gathering

### Primary: MANIFEST.MF parsing from plugin archives

Jenkins plugins are stored at `<JENKINS_HOME>/plugins/`. Each `.jpi`/`.hpi` file contains:

```
META-INF/MANIFEST.MF with:
  Plugin-Version: 2.387.3
  Short-Name: git
  Long-Name: Git plugin
  Url: https://github.com/jenkinsci/git-plugin
  Plugin-Dependencies: credentials:1289.vb,git-client:4.7.0,...
```

Key MANIFEST.MF headers:
- `Short-Name`: Plugin identifier
- `Plugin-Version`: Installed version
- `Long-Name`: Human-readable name
- `Url`: Plugin homepage/repository URL
- `Plugin-Dependencies`: Comma-separated dependency list

### Discovery

Check known Jenkins plugin directories:
- `/var/lib/jenkins/plugins/` (Linux default)
- `/var/jenkins_home/plugins/` (Docker official image)
- Configurable via `jenkins.packages(path: "/custom/path")`

## Resource Schema

### `jenkins.package`

| Field | Type | Source |
|-------|------|--------|
| `name` | string | `Short-Name` header |
| `version` | string | `Plugin-Version` header |
| `purl` | string | `pkg:jenkins-plugin/<name>@<version>` |
| `longName` | string | `Long-Name` header |
| `url` | string | `Url` header |
| `dependencies` | []string | `Plugin-Dependencies` header (parsed) |

## Verification

- Fixture-based tests with a real Jenkins plugin MANIFEST.MF
- Interactive: `mql run os -c "jenkins.packages(path: '/var/lib/jenkins/plugins') { list { name version } }"`
