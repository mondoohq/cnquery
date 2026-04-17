# ADR: mql Chocolatey Package Resource

**Status:** Proposed
**Date:** 2026-04-17

---

## Context

Chocolatey is the leading open-source Windows package manager, widely deployed in enterprise environments for automated software management. It is not covered by the existing `packages` resource. Knowing which Chocolatey packages are installed, at what versions, and whether they are pinned is critical for software inventory and compliance on Windows.

## Decision

Add `chocolatey.packages` and `chocolatey.package` as new MQL resources in the OS provider. These are standalone resources because Chocolatey packages have metadata (dependencies, tags, license URL, pin status, project URL) that has no equivalent in the generic `packages` resource.

## Data Gathering

### Primary: `.nuspec` file parsing (no CLI required)

Chocolatey stores installed packages at `C:\ProgramData\chocolatey\lib\<name>\<name>.nuspec`. Each `.nuspec` is an XML file with the NuGet package specification containing name, version, authors, description, license, dependencies, and tags. This approach works on offline scans, container images, and remote connections without requiring `choco.exe`.

### Fallback: `choco list` CLI

When `.nuspec` files cannot be read, fall back to `choco list --local-only --limit-output` which produces `name|version` pipe-delimited output. This provides only name and version.

### Pin Detection

Chocolatey marks pinned packages with a `.pin` file at `<lib>\<name>\.pin`. Check file existence.

### Discovery

1. Check `ChocolateyInstall` environment variable
2. Default to `C:\ProgramData\chocolatey`
3. Verify `lib\` directory exists

## Resource Schema

### `chocolatey.package` (13 fields)

| Field | Type | Source |
|-------|------|--------|
| `name` | string | `metadata.id` |
| `version` | string | `metadata.version` |
| `purl` | string | `pkg:chocolatey/<name>@<version>` |
| `summary` | string | `metadata.summary` |
| `description` | string | `metadata.description` |
| `author` | string | `metadata.authors` |
| `license` | string | Derived from `licenseUrl` |
| `licenseUrl` | string | `metadata.licenseUrl` |
| `path` | string | Package directory path |
| `pinned` | bool | `.pin` file existence |
| `dependencies` | []string | `metadata.dependencies.dependency[].id` |
| `tags` | []string | `metadata.tags` (space-separated) |
| `projectUrl` | string | `metadata.projectUrl` |

## Transport Compatibility

| Transport | Method |
|-----------|--------|
| Local | Read `.nuspec` files directly |
| SSH/WinRM | Read `.nuspec` files via SFTP/WinRM file transfer |
| Container image | Read `.nuspec` files via `afero.Fs` |

## Verification

- Unit tests with real `.nuspec` XML fixtures (git, 7zip, chocolatey)
- Pin detection test
- Interactive (Windows): `mql run os -c "chocolatey.packages { list { name version pinned } }"`
