# ADR: cnspec SBOM Cataloger for Rust

**Status:** Proposed
**Date:** 2026-04-16

---

## Context

cnspec needs native SBOM cataloging for Rust/Cargo packages to detect dependencies in both local and remote scanning contexts (SSH, WinRM, containers). Rust is increasingly adopted for security-critical systems, CLI tools, and cloud-native infrastructure, making it important for SBOM coverage.

Rust uses Cargo as its package manager and build system. Dependencies are declared in `Cargo.toml` and resolved versions are recorded in `Cargo.lock`. Both files use TOML format, making parsing straightforward.

---

## File Formats

### Cargo.lock (Lock File)

The `Cargo.lock` file contains resolved dependency versions with checksums.

```toml
[[package]]
name = "serde"
version = "1.0.188"
source = "registry+https://github.com/rust-lang/crates.io-index"
checksum = "abc123..."
dependencies = [
  "serde_derive",
]

[[package]]
name = "serde_derive"
version = "1.0.188"
source = "registry+https://github.com/rust-lang/crates.io-index"
checksum = "def456..."
dependencies = [
  "proc-macro2",
  "quote",
  "syn",
]
```

**Key fields per `[[package]]`:**
- `name`: Crate name
- `version`: Resolved version
- `source`: Registry URL or git source (optional for path dependencies)
- `checksum`: SHA-256 hash of the crate archive
- `dependencies`: List of dependency names (optional)

The first `[[package]]` entry is typically the root project (no `source` field).

### Cargo.toml (Manifest)

The `Cargo.toml` file declares the project and its dependencies.

```toml
[package]
name = "myproject"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = "1.0"
tokio = { version = "1.32", features = ["full"] }

[dev-dependencies]
criterion = "0.5"

[build-dependencies]
cc = "1.0"
```

**Key sections:**
- `[package]`: Project name, version, edition
- `[dependencies]`: Production dependencies
- `[dev-dependencies]`: Test/development dependencies
- `[build-dependencies]`: Build script dependencies

Dependency values can be simple version strings or tables with `version`, `features`, `optional`, `git`, `path` fields.

---

## Parsing Strategy

### Cargo.lock Parser

1. Read file via `afero.Fs` and decode with TOML parser (`BurntSushi/toml`)
2. Iterate `[[package]]` entries
3. Root package: the entry without a `source` field (local project)
4. All other entries: transitive dependencies with resolved versions
5. Extract checksum for integrity verification

### Cargo.toml Parser

1. Read file via `afero.Fs` and decode with TOML parser
2. Extract `[package]` for root project name and version
3. Extract `[dependencies]` as direct production dependencies
4. Extract `[dev-dependencies]` as dev dependencies
5. Handle both string (`"1.0"`) and table (`{ version = "1.0" }`) dependency formats

### Error Handling

- Malformed TOML produces a warning and returns partial results
- Missing sections are skipped (e.g., no `[dev-dependencies]` is valid)
- Path/git dependencies without versions are included with empty version strings

---

## Package Identification

### PURL

Format: `pkg:cargo/<name>@<version>`

Examples:
- `pkg:cargo/serde@1.0.188`
- `pkg:cargo/tokio@1.32.0`

### CPE Generation

Format: `cpe:2.3:a:<name>:<name>:<version>:*:*:*:*:*:*:*`

For Cargo crates, vendor and product are both the crate name (Rust crates are uniquely named in crates.io).

---

## Dev/Prod Classification

### From Cargo.toml
- `[dependencies]` → production
- `[dev-dependencies]` → dev
- `[build-dependencies]` → build (treated as production for SBOM purposes)

### From Cargo.lock
No dev/prod distinction — all resolved dependencies are listed together.

---

## Dependency Relationships

### From Cargo.lock
Each `[[package]]` has an optional `dependencies` array listing crate names. This provides the full dependency tree with direct parent-child relationships.

### From Cargo.toml
Only direct dependencies are listed. Transitive resolution requires `Cargo.lock`.

---

## Remote Scanning Considerations

### afero.Fs Usage
All file reads go through `afero.Fs` via `afero.Afero.ReadFile()`.

### File Size
- `Cargo.lock`: Typically 10-100KB, can be 500KB+ for large projects
- `Cargo.toml`: Typically < 10KB
No special size handling needed.

---

## Verification Plan

### Fixture-Based Tests
- `Cargo.lock` with multiple packages, root detection, checksums
- `Cargo.toml` with dependencies, dev-dependencies, table-format deps
- Edge cases: path dependencies, git dependencies, workspace projects

### Interactive Testing
- Local: `mql run os -c "rust.packages(path: '.') { list { name version purl } }"`
- Container: Scan a Rust-based container image
- Remote: Same query over SSH

---

## Implementation Plan

### Phase 1 (This PR)
1. Cargo.lock parser with root detection and checksum extraction
2. Cargo.toml parser with dev/prod classification
3. MQL resources: `rust.packages`, `rust.package`
4. SBOM generator integration (type: `cargo`)

### Phase 2 (Follow-up PR)
1. Rust binary analysis via `cargo auditable` `.dep-v0` ELF section
