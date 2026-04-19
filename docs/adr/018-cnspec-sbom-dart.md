# ADR: mql SBOM Cataloger for Dart/Flutter

**Status:** Proposed
**Date:** 2026-04-19

---

## Context

cnspec needs native SBOM cataloging for Dart/Flutter packages to detect dependencies in both local and remote scanning contexts. Dart is the language behind Flutter, one of the most popular cross-platform mobile/web frameworks. Dependencies are managed via `pub` and recorded in `pubspec.lock`.

## File Formats

### pubspec.lock (Lock File)

YAML file with resolved dependency versions.

```yaml
packages:
  http:
    dependency: "direct main"
    description:
      name: http
      sha256: "abc123..."
      url: "https://pub.dev"
    source: hosted
    version: "1.2.1"
  meta:
    dependency: transitive
    description:
      name: meta
      sha256: "def456..."
      url: "https://pub.dev"
    source: hosted
    version: "1.15.0"
sdks:
  dart: ">=3.3.0 <4.0.0"
```

Key fields per package:
- `dependency`: "direct main", "direct dev", or "transitive"
- `version`: resolved version
- `source`: "hosted", "git", or "path"
- `description.name`: package name

### pubspec.yaml (Manifest)

Declares project metadata and dependencies. Used for root project info.

## Package Identification

- **PURL:** `pkg:pub/<name>@<version>`
- **CPE:** vendor=name, product=name

## Dev/Prod Classification

From `pubspec.lock`:
- `dependency: "direct main"` → direct production
- `dependency: "direct dev"` → direct dev
- `dependency: transitive` → transitive

## Verification

- Fixture-based tests with a real pubspec.lock
- Interactive: `mql run os -c "dart.packages(path: '.') { list { name version purl } }"`
