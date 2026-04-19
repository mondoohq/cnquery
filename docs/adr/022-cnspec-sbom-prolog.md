# ADR: mql SBOM Cataloger for SWI-Prolog Packs

**Status:** Proposed
**Date:** 2026-04-19

---

## Context

cnspec needs native SBOM cataloging for SWI-Prolog packs to detect installed packages. SWI-Prolog uses a pack system where each pack contains a `pack.pl` file with metadata in Prolog term format.

## File Format

### pack.pl

Simple Prolog term format with `key(value).` entries:

```prolog
name(mavis).
version('1.0.5').
title('Access Prolog packs from SICStus').
author('Per Mildner', 'per.mildner@sics.se').
home_url('https://github.com/example/mavis').
download_url('https://example.com/mavis-1.0.5.zip').
```

Key fields: `name(...)` and `version(...)`.

## Parsing Strategy

Line-oriented regex matching `key('value')` or `key(value)` patterns. No full Prolog parser needed.

## Package Identification

- **PURL:** `pkg:swi-prolog/<name>@<version>`

## Verification

- Fixture-based tests with a pack.pl file
- Interactive: `mql run os -c "prolog.packages(path: '/path/to/packs') { list { name version purl } }"`
