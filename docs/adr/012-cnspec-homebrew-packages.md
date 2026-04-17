# ADR: cnspec Homebrew Package Resource

**Status:** Proposed
**Date:** 2026-04-17

---

## Context

Homebrew is the de facto package manager on macOS and increasingly used on Linux (Linuxbrew). It is not covered by the existing `packages` resource (apt/yum/rpm/apk). In enterprise macOS fleets, Homebrew packages are often the largest source of unmanaged software. Compliance teams need to answer: "Which machines have unapproved software installed via Homebrew?" and "Are any Homebrew packages at vulnerable versions?"

## Decision

Add `homebrew.packages` and `homebrew.package` as new MQL resources in the OS provider. These are standalone resources (not added to the existing `packages` resource) because Homebrew has a fundamentally different metadata schema with fields like `tap`, `installedOnRequest`, `autoUpdates`, `pinned`, and `type` (formula vs cask) that have no equivalents in OS-native packages.

## Data Gathering

### Primary: `brew info --json=v2 --installed`

A single CLI call returns all installed formulae and casks with full metadata (name, version, description, tap, outdated status, dependencies, etc.). The JSON v2 format is versioned and stable.

### Fallback: Cellar file parsing

When `brew` cannot be executed (offline disk image, container scan), parse the Cellar directory:
- `<prefix>/Cellar/<name>/<version>/INSTALL_RECEIPT.json` for formulae
- `<prefix>/Caskroom/<name>/<version>/.metadata/` for casks

### Discovery

Check known brew binary paths:
- `/opt/homebrew/bin/brew` (Apple Silicon)
- `/usr/local/bin/brew` (Intel Mac)
- `/home/linuxbrew/.linuxbrew/bin/brew` (Linux)

## Resource Schema

### `homebrew.package` (16 fields)

| Field | Type | Source |
|-------|------|--------|
| `name` | string | Formula name or cask token |
| `version` | string | Currently installed version |
| `latestVersion` | string | Latest stable version available (equals version when up-to-date) |
| `purl` | string | Package URL (pkg:brew/tap/name@version) |
| `description` | string | Package description |
| `homepage` | string | Homepage URL |
| `path` | string | Install path |
| `type` | string | "formula" or "cask" |
| `appName` | string | App name (casks only) |
| `autoUpdates` | bool | Cask manages own updates |
| `installedOnRequest` | bool | Explicitly installed (not a dependency) |
| `installedAsDependency` | bool | Installed as a dependency |
| `outdated` | bool | Newer version available |
| `pinned` | bool | Version pinned |
| `tap` | string | Homebrew tap (e.g., "homebrew/core") |
| `prefix` | string | Homebrew install prefix |

## Transport Compatibility

| Transport | Method |
|-----------|--------|
| Local | Execute `brew info` directly |
| SSH | Execute `brew info` via SSH |
| Container image | File parsing fallback (Cellar + INSTALL_RECEIPT.json) |

## Verification

- Unit tests with real `brew info --json=v2` fixture
- Test INSTALL_RECEIPT.json fallback parsing
- Interactive: `mql run os -c "homebrew.packages { list { name version type tap } }"`
