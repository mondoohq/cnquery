# ADR: mql ALPM (Arch Linux) SBOM Enhancement

**Status:** Proposed
**Date:** 2026-04-19

---

## Context

The existing pacman package manager parser (`pacman -Q`) provides basic name+version extraction with PURL generation. For complete SBOM coverage, we need to also support filesystem-based parsing of `/var/lib/pacman/local/*/desc` files, which enables:

1. **Offline/image scanning** without the `pacman` CLI
2. **Richer metadata**: description, URL, architecture, dependencies
3. **Dependency relationship extraction** for SBOM dependency graphs

## Current State

- `pacman -Q` parser: name, version, PURL (`pkg:alpm/arch/<name>@<version>`)
- No filesystem fallback
- No description, architecture, or dependency extraction

## Enhancement

### Filesystem fallback: `/var/lib/pacman/local/*/desc`

The `desc` file uses a section-based format:

```
%NAME%
zlib

%VERSION%
1:1.2.13-3

%DESC%
Compression library implementing the deflate compression method

%URL%
https://www.zlib.net/

%ARCH%
x86_64

%DEPENDS%
glibc
```

Sections are `%KEY%` followed by value lines until the next `%KEY%` or end of file.

### Changes

- Add `listFromFS()` method to `PacmanPkgManager` for filesystem-based parsing
- Parse `desc` files for name, version, description, URL, arch
- Use CLI primary, filesystem fallback (same pattern as Snap, Homebrew)
- Populate `Description` and `Arch` fields on `Package` struct

## Verification

- Fixture-based tests with real desc files
- Interactive: `mql run os -c "packages { list.where(format == 'pacman') { name version } }"`
