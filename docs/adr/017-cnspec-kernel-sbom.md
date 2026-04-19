# ADR: mql Linux Kernel SBOM Enhancement

**Status:** Proposed
**Date:** 2026-04-19

---

## Context

The Linux kernel is a critical component for vulnerability management. Kernel CVEs are high-severity (privilege escalation, container escapes), yet the current `kernel` resource returns untyped data (`dict`) and the kernel is not emitted as a proper SBOM package.

### Current State

- `kernel.info()` returns `dict` with version, path, device — no type safety, no discoverability
- `kernel.installed()` returns `[]dict` — also untyped
- SBOM generator only stores running kernel version as an asset label, not as a package
- No PURL or CPE generation for the kernel

## Decision

### Unified `kernel.version` typed resource

Introduce a single `kernel.version` resource used by both `kernel.running` and `kernel.installedVersions`. This replaces the untyped `dict` returns with a proper typed resource including SBOM identifiers.

```lr
kernel @defaults("info") {
  info() @maturity("deprecated") dict   // deprecated, use running instead
  running() kernel.version              // active kernel, typed
  parameters() map[string]string
  modules() []kernel.module
  installed() @maturity("deprecated") []dict  // deprecated, use installedVersions
  installedVersions() []kernel.version  // all installed kernels, typed
}

kernel.version @defaults("name version running") {
  name string         // kernel package name (e.g., "linux-image-6.8.0-40-generic")
  version string      // version string (e.g., "6.8.0-40-generic")
  running bool        // whether this is the currently running kernel
  path string         // boot path (available for running kernel)
  device string       // boot device (available for running kernel)
  arch string         // architecture (available for running kernel)
  fullVersion string  // raw /proc/version (available for running kernel)
  purl string         // pkg:generic/linux-kernel@<version>
  cpes []core.cpe     // CPE entries for vulnerability matching
}
```

### Why one resource, not two

`kernel.running` and installed kernel versions share most fields (name, version, purl, cpes). The running kernel adds path, device, arch, and fullVersion. Using a single `kernel.version` type means:
- Consistent field access whether querying running or installed kernels
- `kernel.installedVersions.where(running == true)` works naturally
- No near-duplicate resource types to maintain

## Data Sources

### Running kernel (`kernel.running`)

| Field | Source |
|-------|--------|
| `version` | `/proc/version` (parsed by existing `kernel.Info()`) |
| `path` | `/proc/cmdline` (boot path) |
| `device` | `/proc/cmdline` (boot device) |
| `arch` | Platform info (`conn.Asset().Platform.Arch`) |
| `fullVersion` | Raw `/proc/version` string |
| `purl` | Generated: `pkg:generic/linux-kernel@<version>` |
| `cpes` | Generated: `cpe:2.3:a:linux:linux_kernel:<version>:*:*:*:*:*:*:*` |

### Installed kernels (`kernel.installedVersions`)

Reuses the existing `kernel.installed()` logic which queries OS packages per distro:
- **Debian**: filters `linux-image-*` packages
- **RHEL/CentOS/Amazon Linux**: filters `kernel` packages
- **Oracle Linux**: filters `kernel` and `kernel-uek` packages
- **SUSE**: filters `kernel-*` packages
- **Photon**: filters `linux*` packages

Each installed kernel gets a PURL and CPE for vulnerability matching.

## Package Identification

### PURL

Format: `pkg:generic/linux-kernel@<version>`

Examples:
- `pkg:generic/linux-kernel@6.8.0-40-generic`
- `pkg:generic/linux-kernel@5.15.0-1054-aws`

### CPE

Format: `cpe:2.3:a:linux:linux_kernel:<version>:*:*:*:*:*:*:*`

## SBOM Generator Enhancement

Installed kernels are emitted as proper SBOM packages:
- Type: `kernel`
- PURL: `pkg:generic/linux-kernel@<version>`
- The existing asset label (`mondoo.com/os/kernel-running`) is kept for backwards compatibility

## Backwards Compatibility

- `kernel.info()` stays as-is (returns `dict`, marked deprecated via `@maturity`)
- `kernel.installed()` stays as-is (returns `[]dict`, marked deprecated via `@maturity`)
- New fields (`running`, `installedVersions`) are additions, not replacements
- Existing policies using `kernel.info` continue to work unchanged

## Usage Examples

```coffeescript
# Running kernel with SBOM identifiers
kernel.running { version arch purl cpes fullVersion }

# All installed kernels
kernel.installedVersions { name version running purl }

# Check for vulnerable kernel version
kernel.running.version != "6.1.0-vulnerable"

# Find non-running installed kernels
kernel.installedVersions.where(running == false) { name version }
```

## Remote Scanning

All data sources (`/proc/version`, `/proc/cmdline`, OS packages) work across local, SSH, and WinRM transports via the existing connection abstraction.

## Phase 2 (Follow-up)

- Add `version` field to `kernel.module` via `.modinfo` ELF section parsing
- Generate PURLs for modules: `pkg:generic/linux-kernel-module/<name>@<version>`
- Third-party modules (Nvidia, ZFS, VirtualBox) are the high-value targets
