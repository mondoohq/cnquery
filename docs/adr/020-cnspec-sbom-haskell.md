# ADR: mql SBOM Cataloger for Haskell

**Status:** Proposed
**Date:** 2026-04-19

---

## Context

cnspec needs native SBOM cataloging for Haskell packages to detect dependencies in both local and remote scanning contexts. Haskell has two main build tools — Stack and Cabal — each with its own lock file format.

## File Formats

### stack.yaml.lock (Stack)

YAML file with resolved package snapshots.

```yaml
packages:
- completed:
    hackage: aeson-2.2.1.0@sha256:abc123
    pantry-tree:
      sha256: def456
      size: 12345
  original:
    hackage: aeson-2.2.1.0
- completed:
    hackage: text-2.1.1@sha256:ghi789
    pantry-tree:
      sha256: jkl012
      size: 6789
  original:
    hackage: text-2.1.1
snapshots:
- completed:
    sha256: mno345
    size: 123456
    url: https://raw.githubusercontent.com/commercialhaskell/stackage-snapshots/master/lts/22/7.yaml
  original:
    url: https://raw.githubusercontent.com/commercialhaskell/stackage-snapshots/master/lts/22/7.yaml
```

Key: `packages[].completed.hackage` contains `name-version@sha256`.

### cabal.project.freeze (Cabal)

Custom text format with `constraints:` block listing pinned versions.

```
active-repositories: hackage.haskell.org:merge
constraints: any.aeson ==2.2.1.0,
             any.base ==4.19.0.0,
             any.text ==2.1.1,
             aeson +ordered-keymap
```

Key: `any.<name> ==<version>` entries, one per line. Lines with flags (no `==`) are skipped.

## Package Identification

- **PURL:** `pkg:hackage/<name>@<version>`
- **CPE:** vendor=name, product=name

## Verification

- Fixture-based tests for both formats
- Interactive: `mql run os -c "haskell.packages(path: '.') { list { name version purl } }"`
