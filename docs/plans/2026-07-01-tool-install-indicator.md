# Tool Install Indicator Implementation Plan

Implements [ADR 028](../adr/028-tool-install-indicator.md): a `package()` accessor on
each tool resource that resolves to the real system package when it can be attributed,
and to an abstract package (reusing the `package` resource) when it cannot.

**Status: implemented in this branch.** The `package()` accessor, shared resolver, and
abstract-package synthesis are in place for all 26 tools (`providers/os/resources/tool_package.go`).
Binary-ownership detection remains the ADR follow-up.

## Scope

- Add `package() package` to the 26 AI/LLM tool resources in the `os` provider.
- Add a shared Go resolver + per-tool spec table that returns a real or abstract
  `package`.
- Document the abstract-package conventions on the `package` resource's existing fields.
- No new field on `package`; no new resource.

Non-goals for this PR: rolling the pattern out to non-AI tools, NVD mapping for
`pkg:generic` PURLs, richer origin taxonomies. These are ADR 028 follow-ups.

## Current implementation anchors

- `providers/os/resources/os.lr` — `package` resource (lines 875–962); the 27 tool
  resources (lines ~9453–~10450). `claude.code` at ~9457, `openai.codex` at ~9547.
- `providers/os/resources/packages.go` — `id()` (line 28), `initPackage()` name lookup
  (line 42), `list()` (line 162), `refreshCache()` (line 275). Template for both the
  managed-package lookup and the abstract-package synthesis (the not-found stub in
  `initPackage` is the shape of an `installed == false` package).
- `providers/os/resources/ai_coding_tools.go` — `initConfigPath`, `targetHomeDir`,
  `connectionAfs`, `readJSONFileAfero`, `dirHasFilesAfero` — reused for presence checks.
- `providers/os/resources/openai_codex.go:55` — `version()` reading `version.json`; the
  version-inference precedent.
- `providers/os/resources/ai_agents.go` — the simple tools that will share one spec.
- `providers/os/config/config.go:17` — provider `Version: "13.31.0"`.

## Task 1 — Schema (`providers/os/resources/os.lr`)

1. Add the accessor to each tool resource, right after its `configPath` field:

   ```
   // Package that installed this tool
   //
   // Resolves to the system-package-manager entry that installed the tool when
   // one can be identified; otherwise an abstract package that records the
   // tool's presence and, where possible, an inferred version. Read
   // `.installed` for the install signal, or any package field (`version`,
   // `purl`, `cpes`, `vendor`) for detail.
   package() package
   ```

   Apply to all 27: `claude.code`, `openai.codex`, `cursor`, `github.copilot`, `goose`,
   `gemini`, `windsurf`, `zed`, `roo`, `cline`, `kiro`, `continuedev`, `trae`,
   `opencode`, `pi`, `mistral.vibe`, `antigravity`, `ibm.bob`, `openclaw`,
   `snowflake.cortex`, `junie`, `augment`, `warp`, `kilocode`, `openhands`, `qwen.code`.

2. Update the doc-comments on `package.format` (os.lr:903) and `package.origin`
   (os.lr:914) to describe the abstract-package convention — `origin == "unknown"` and
   an empty `format` mean the package was not attributed to a system package manager and
   does not appear in `packages`. Comment-only edits; **no version bump** for these.

3. **Parser check:** confirm the `.lr` parser accepts a field literally named `package`
   of type `package` — no existing resource uses the `package` type as a typed field.
   Run codegen (Task 3) early against one tool to verify. If it clashes, the fallback
   accessor name is `installedPackage()`, but `package` is the target for the
   `tool.package.installed` UX in ADR 028.

## Task 2 — Go resolver (`providers/os/resources/tool_package.go`, new)

```go
// toolPackageSpec describes how to resolve a tool's backing package.
type toolPackageSpec struct {
    // canonical package name used for the abstract package and for lookup
    packageName string
    // candidate names to try against the system package manager, in order
    managerCandidates []string
    // producing vendor/tool identity for the abstract package (optional)
    vendor string
    // presence + version detection; installed=false means the tool is absent
    detect func(runtime *plugin.Runtime, configPath string) (installed bool, version string, err error)
}

// resolveToolPackage returns the real managed package when the tool can be
// attributed to a system package manager, otherwise an abstract package.
func resolveToolPackage(runtime *plugin.Runtime, configPath string, spec toolPackageSpec) (*mqlPackage, error) {
    // (a) try the real package manager via the existing name lookup
    for _, name := range spec.managerCandidates {
        raw, err := CreateResource(runtime, "package", map[string]*llx.RawData{
            "name": llx.StringData(name),
        })
        if err != nil {
            return nil, err
        }
        pkg := raw.(*mqlPackage)
        if pkg.Installed.State.IsSet() && pkg.Installed.Data {
            return pkg, nil // real, manager-tracked, also in `packages`
        }
    }

    // (b) not attributable — detect presence + infer version
    installed, version, err := false, "", error(nil)
    if spec.detect != nil {
        installed, version, err = spec.detect(runtime, configPath)
        if err != nil {
            return nil, err
        }
    }

    args := map[string]*llx.RawData{
        "__id":        llx.StringData("tool://" + spec.packageName),
        "name":        llx.StringData(spec.packageName),
        "installed":   llx.BoolData(installed),
        "origin":      llx.StringData("unknown"), // abstract discriminator
        "format":      llx.StringData(""),        // no package-manager format
        "version":     versionData(version),      // llx.StringData or llx.NilData
        "vendor":      vendorData(spec.vendor),
        // available/purl/cpes left null unless known
    }
    raw, err := CreateResource(runtime, "package", args)
    if err != nil {
        return nil, err
    }
    return raw.(*mqlPackage), nil
}
```

Notes:
- The synthetic `__id` (`tool://<name>`) keeps abstract packages from colliding with the
  real `format://name/version/arch` scheme (`packages.go:28`) and is never exposed as a
  field, per the `__id` guidance in CLAUDE.md.
- Verify that fields declared as computed methods (`origin()`, `status()`) can be seeded
  through `CreateResource` args so the generated method short-circuits — this is how the
  runtime handles pre-populated computed fields elsewhere. If a computed field cannot be
  seeded, set it via the returned `*mqlPackage`'s `TValue` directly (as `initPackage`
  does for the stub).
- A per-tool spec table (`map[string]toolPackageSpec`) holds candidate names, vendor,
  and detector. Simple tools default to: `detect = configDirPresent` (uses
  `dirHasFilesAfero`/`Stat` on `configPath`) with no version.

Each tool's generated `package()` method is a thin wrapper:

```go
func (r *mqlClaudeCode) package_() (*mqlPackage, error) {
    return resolveToolPackage(r.MqlRuntime, r.ConfigPath.Data, toolPackageSpecs["claude.code"])
}
```

(Confirm the generated Go method name for a `package` field — likely `package_` due to
the Go keyword — during codegen.)

## Task 3 — Codegen

```bash
make providers/mqlr
./mqlr generate providers/os/resources/os.lr --dist providers/os/resources
```

Run a second time if a `mql<Resource>Internal` struct is added for caching.

## Task 4 — `.lr.versions`

The `os` provider is at **13.31.0** (`providers/os/config/config.go:17`), so each new
`package()` accessor entry is **13.31.1** (next patch). Prefer the version tool, then
verify — do not trust the highest number already in the file:

```bash
go run providers-sdk/v1/util/version/version.go update providers/os/
git diff providers/os/resources/os.lr.versions   # new package() entries at 13.31.1
```

The `package.format`/`package.origin` doc-comment edits keep their existing versions.

## Verification

```bash
make providers/build/os && make providers/install/os

# abstract case (tool present, no manager package)
mql run local -c 'cursor.package { name installed version origin format }'
mql run local -c 'claude.code.package.installed'
mql run local -c 'openai.codex.package { version origin }'   # version via version.json

# managed case: install a tool through a package manager (brew/apt), then
mql run local -c 'brew_installed_tool.package { format origin installed }'  # real format, not "unknown"
mql run local -c 'packages.where(name == "<tool>").length'                  # 1 — same instance

# negative case
mql run local -c 'roo.package.installed'   # false when the tool is absent

# invariant: abstract packages never appear in the packages list
mql run local -c 'packages.where(origin == "unknown").length'   # 0
```

Expected: `installed` reflects real presence; abstract packages carry
`origin == "unknown"` and empty `format`; manager-tracked tools return the same object
that appears in `packages`; no abstract package leaks into `packages`.

## Pre-PR checklist

- `gofmt -w` on new/changed Go files.
- `make mql/generate` clean (`git diff --exit-code`).
- `providers/os/resources/os.lr.versions` and `*.permissions.json` committed if changed.
- `make test/lint` and `make test/go/plain`.
- Spell-check terms (new words in `.lr`/`.md`) added to
  `.github/actions/spelling/expect.txt` if flagged.
