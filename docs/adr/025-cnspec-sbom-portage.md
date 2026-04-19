# ADR: mql Portage (Gentoo) SBOM Enhancement

**Status:** Proposed
**Date:** 2026-04-19

---

## Context

The existing Gentoo package manager parser (`qlist -Iv`) provides basic name+version extraction but lacks PURL generation and filesystem-based fallback. For complete SBOM coverage, we need:

1. **PURL generation** using the `pkg:ebuild` type per the PURL specification
2. **Filesystem fallback** for offline/image scanning via `/var/db/pkg/`
3. **Richer metadata**: description from `/var/db/pkg/CATEGORY/NAME-VERSION/DESCRIPTION`

## Current State

- `qlist -Iv --format '%{CATEGORY}/%{PN}:%{PVR}'` parser: name (with category), version
- No PURL generation
- No filesystem fallback
- No description extraction

## Enhancement

### PURL Generation

Portage packages use the `ebuild` PURL type, which is defined in the PURL specification:

```
pkg:ebuild/net-misc/curl@8.4.0
```

The namespace is the Gentoo category (e.g., `net-misc`), and the name is the package name (e.g., `curl`).

### Filesystem Fallback: `/var/db/pkg/`

The Portage installed package database uses a directory structure:

```
/var/db/pkg/
├── net-misc/
│   ├── curl-8.4.0/
│   │   ├── CATEGORY        → "net-misc"
│   │   ├── PF              → "curl-8.4.0"
│   │   ├── DESCRIPTION     → "A Client and Server ..."
│   │   ├── EAPI            → "8"
│   │   └── ...
│   └── dhcpcd-10.0.5-r1/
│       └── ...
├── acct-group/
│   └── audio-0-r2/
│       └── ...
```

Each subdirectory under a category represents an installed package. The directory name encodes the package name and version (e.g., `curl-8.4.0`). The version is separated from the name by the last hyphen before a version-like segment (starting with a digit).

### Changes

- Add `TypeEbuild` PURL type
- Add PURL generation to `ParseGentooPackages`
- Add `listFromFS()` method with `/var/db/pkg/` fallback
- Parse `DESCRIPTION` file for package description
- Use CLI primary, filesystem fallback (same pattern as Homebrew, ALPM)
- Pass platform to parser for PURL generation

## Verification

- Fixture-based tests with `qlist` output and `/var/db/pkg/` directory structure
- Interactive: `mql run os -c "packages { list.where(format == 'gentoo') { name version purl } }"`
