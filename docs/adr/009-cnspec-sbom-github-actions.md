# ADR: cnspec SBOM Cataloger for GitHub Actions

**Status:** Proposed
**Date:** 2026-04-17

---

## Context

cnspec needs native SBOM cataloging for GitHub Actions to detect action dependencies in CI/CD workflows. GitHub Actions are a supply chain vector — pinning actions to specific commits or versions is a security best practice, and SBOM generation enables auditing which actions (and versions) a repository depends on.

GitHub Actions workflows are defined in YAML files under `.github/workflows/`. Each `uses:` directive references an action by owner, repository, and ref (tag, branch, or commit SHA).

---

## File Formats

### Workflow YAML Files

Located at `.github/workflows/*.yml` (or `.yaml`).

```yaml
name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: docker/build-push-action@v5.1.0
      - uses: github/codeql-action/init@v3
      - run: echo "not an action"
```

**Key fields:**
- `jobs.<job>.steps[].uses`: Action reference in `owner/repo@ref` format
- Nested actions: `owner/repo/path@ref` (e.g., `github/codeql-action/init@v3`)
- Local actions: `./path/to/action` (skipped — not external dependencies)
- Docker actions: `docker://image:tag` (skipped — separate ecosystem)

---

## Parsing Strategy

1. Read YAML files via `afero.Fs` using `gopkg.in/yaml.v3` (already in go.mod)
2. Walk the `jobs` map, iterate each job's `steps` array
3. Extract `uses` field from each step
4. Parse `uses` value into owner, repo, optional path, and ref
5. Skip local actions (`./`) and Docker actions (`docker://`)
6. Deduplicate by `owner/repo@ref`

### Error Handling
- Malformed YAML produces a warning and returns partial results
- Steps without `uses` (e.g., `run:` steps) are silently skipped
- Invalid `uses` format produces a debug log and is skipped

---

## Package Identification

### PURL
Format: `pkg:github/<owner>/<repo>@<ref>`

Examples:
- `pkg:github/actions/checkout@v4`
- `pkg:github/docker/build-push-action@v5.1.0`

For actions with sub-paths (e.g., `github/codeql-action/init@v3`), the PURL uses the repo only: `pkg:github/github/codeql-action@v3`

### CPE Generation
Not applicable for GitHub Actions — they don't map to CPE entries. CPEs will be empty.

---

## Dev/Prod Classification

Not applicable — all referenced actions are production CI/CD dependencies.

---

## Remote Scanning Considerations

YAML files are small (< 50KB typically). No special handling needed. All reads through `afero.Fs`.

Default search path: `.github/workflows/`

### Known Limitations

- Only `.github/workflows/` is scanned by default. Composite actions in `.github/actions/` or reusable workflows in other directories are not discovered unless explicitly targeted via `githubactions.packages(path: ".github/actions")`
- YAML files that are not workflow definitions (e.g., composite action metadata) are silently skipped (no `jobs` key)

---

## Verification Plan

- Fixture-based tests with a multi-job workflow file
- Edge cases: sub-path actions, Docker actions, local actions, reusable workflows
- Interactive: `mql run os -c "githubactions.packages(path: '.') { list { name version purl } }"`
