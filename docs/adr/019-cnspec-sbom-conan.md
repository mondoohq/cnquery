# ADR: mql SBOM Cataloger for C/C++ Conan Packages

**Status:** Proposed
**Date:** 2026-04-19

---

## Context

cnspec needs native SBOM cataloging for C/C++ Conan packages. Conan is the leading open-source C/C++ package manager, widely used for managing dependencies in embedded systems, game engines, and infrastructure software.

## File Formats

### conan.lock (v2, JSON)

Conan 2.x lock files are JSON with a `requires` list of resolved package references.

```json
{
  "version": "0.5",
  "requires": [
    "zlib/1.3.1#sha256",
    "openssl/3.2.1#sha256",
    "boost/1.84.0#sha256"
  ],
  "build_requires": [
    "cmake/3.28.1#sha256"
  ],
  "python_requires": []
}
```

Each entry is a Conan reference: `name/version#revision`.

### conan.lock (v1, JSON)

Conan 1.x lock files use a graph structure with nodes.

```json
{
  "graph_lock": {
    "nodes": {
      "0": { "ref": "myproject/1.0" },
      "1": { "ref": "zlib/1.3.1#rev", "requires": ["2"] },
      "2": { "ref": "openssl/3.2.1#rev" }
    }
  },
  "version": "0.4"
}
```

## Package Identification

- **PURL:** `pkg:conan/<name>@<version>`
- **CPE:** vendor=name, product=name

## Parsing Strategy

1. Parse JSON via `encoding/json`
2. Detect version: `"version": "0.5"` = v2, `"graph_lock"` present = v1
3. V2: parse `requires` and `build_requires` arrays, split on `/` and `#`
4. V1: iterate `graph_lock.nodes`, parse `ref` field

## Dev/Prod Classification

- V2: `requires` = production, `build_requires` = build/dev
- V1: node 0 is the root project, all others are dependencies

## Verification

- Fixture-based tests with v1 and v2 lock files
- Interactive: `mql run os -c "conan.packages(path: '.') { list { name version purl } }"`
