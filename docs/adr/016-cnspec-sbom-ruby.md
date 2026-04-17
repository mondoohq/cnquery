# ADR: mql SBOM Cataloger for Ruby

**Status:** Proposed
**Date:** 2026-04-18

---

## Context

cnspec needs native SBOM cataloging for Ruby/Gem packages to detect dependencies in both local and remote scanning contexts. Ruby is widely used in web development (Rails), DevOps tooling (Chef, Puppet, Vagrant), and security tools.

Ruby uses Bundler as its dependency manager. Dependencies are declared in `Gemfile` and resolved versions are recorded in `Gemfile.lock`.

## File Formats

### Gemfile.lock

Custom text format with distinct sections:

```
GEM
  remote: https://rubygems.org/
  specs:
    actioncable (7.1.3)
      actionpack (= 7.1.3)
      nio4r (~> 2.0)
    nio4r (2.7.0)
    rack (3.0.8)

PLATFORMS
  ruby
  x86_64-linux

DEPENDENCIES
  actioncable (~> 7.1)
  rack

BUNDLED WITH
   2.5.6
```

**Sections:**
- `GEM > specs:` — All resolved gems with versions and sub-dependencies
- `DEPENDENCIES` — Direct dependencies (explicitly listed in Gemfile)
- `PLATFORMS` — Target platforms
- `BUNDLED WITH` — Bundler version used

### Parsing Strategy

1. Read line by line via `afero.Fs`
2. Track current section (GEM specs, DEPENDENCIES, BUNDLED WITH)
3. In GEM specs: parse `    name (version)` lines (4-space indent = top-level gem)
4. Sub-dependencies are indented 6+ spaces — skip them (they're listed as top-level elsewhere)
5. In DEPENDENCIES: collect direct dependency names for direct/transitive classification

## Package Identification

- **PURL:** `pkg:gem/<name>@<version>`
- **CPE:** vendor=name, product=name

## Dev/Prod Classification

`Gemfile.lock` does not distinguish dev from prod dependencies in the lock file itself. All resolved gems appear in the GEM section. To distinguish, cross-reference with `Gemfile` (future enhancement).

## Verification

- Fixture-based tests with a real Gemfile.lock
- Interactive: `mql run os -c "ruby.packages(path: '.') { list { name version purl } }"`
