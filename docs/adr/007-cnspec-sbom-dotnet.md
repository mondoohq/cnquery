# ADR: cnspec SBOM Cataloger for .NET / NuGet

**Status:** Proposed
**Date:** 2026-04-17

---

## Context

cnspec needs native SBOM cataloging for .NET/NuGet packages to detect dependencies in both local and remote scanning contexts (SSH, WinRM, containers). .NET is dominant in enterprise and Windows environments, making it a high-priority ecosystem for SBOM coverage.

The .NET ecosystem uses NuGet as its package manager. Dependencies are declared in project files (`.csproj`, `.fsproj`) or `packages.config`, resolved versions are recorded in `packages.lock.json`, and runtime dependencies are listed in `*.deps.json` files.

---

## File Formats

### packages.lock.json (NuGet Lock File)

The NuGet lock file contains resolved dependency versions per target framework.

```json
{
  "version": 1,
  "dependencies": {
    "net8.0": {
      "Newtonsoft.Json": {
        "type": "Direct",
        "requested": "[13.0.3, )",
        "resolved": "13.0.3",
        "contentHash": "HrC5BXdl00IP9ze..."
      },
      "System.Text.Json": {
        "type": "Transitive",
        "resolved": "8.0.0",
        "contentHash": "OdrZO2WjkiE..."
      }
    }
  }
}
```

**Key fields:**
- `dependencies.<framework>.<package>`: Per-framework dependency entries
- `type`: `"Direct"` or `"Transitive"`
- `resolved`: Resolved version
- `contentHash`: SHA-512 content hash

### *.deps.json (Runtime Dependency Manifest)

Generated at build time, lists runtime dependencies.

```json
{
  "runtimeTarget": { "name": ".NETCoreApp,Version=v8.0" },
  "libraries": {
    "Newtonsoft.Json/13.0.3": {
      "type": "package",
      "serviceable": true,
      "sha512": "..."
    }
  }
}
```

**Key fields:**
- `libraries`: Map of `"name/version"` to metadata
- `type`: `"package"` for NuGet packages, `"project"` for project references

### packages.config (Legacy NuGet)

Legacy NuGet package reference format (pre-.NET Core).

```xml
<?xml version="1.0" encoding="utf-8"?>
<packages>
  <package id="Newtonsoft.Json" version="13.0.3" targetFramework="net48" />
  <package id="NUnit" version="3.13.3" targetFramework="net48" developmentDependency="true" />
</packages>
```

### *.csproj / *.fsproj (Project Files)

Modern .NET project files with `PackageReference` elements.

```xml
<Project Sdk="Microsoft.NET.Sdk">
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageReference Include="NUnit" Version="3.13.3" PrivateAssets="all" />
  </ItemGroup>
</Project>
```

---

## Parsing Strategy

### packages.lock.json Parser (Preferred)
1. Parse JSON via `encoding/json`
2. Iterate framework entries; use the first framework found (or merge across frameworks)
3. Classify `type: "Direct"` vs `type: "Transitive"`
4. Extract resolved version and contentHash

### deps.json Parser
1. Parse JSON via `encoding/json`
2. Iterate `libraries` entries
3. Split key on `/` to get name and version
4. Filter to `type: "package"` (skip project references)

### packages.config Parser
1. Parse XML via `encoding/xml`
2. Extract `id`, `version`, `targetFramework` from `<package>` elements
3. Detect dev dependencies via `developmentDependency="true"`

### csproj Parser
1. Parse XML via `encoding/xml`
2. Extract `Include` (name) and `Version` from `<PackageReference>` elements
3. Detect dev dependencies via `PrivateAssets="all"` or `IncludeAssets="..."` patterns

### Error Handling
- Malformed JSON/XML produces a warning and returns partial results
- Missing sections are skipped gracefully
- Multiple target frameworks in lock file: all frameworks are merged, deduplicated by name

---

## Package Identification

### PURL
Format: `pkg:nuget/<name>@<version>`

Examples:
- `pkg:nuget/Newtonsoft.Json@13.0.3`
- `pkg:nuget/System.Text.Json@8.0.0`

### CPE Generation
Vendor and product are both the package name (NuGet packages are uniquely named on nuget.org).

---

## Dev/Prod Classification

- **packages.lock.json**: `type: "Direct"` = production, `type: "Transitive"` = transitive
- **packages.config**: `developmentDependency="true"` = dev
- **csproj**: `PrivateAssets="all"` = dev (test/build tooling)
- **deps.json**: No dev/prod distinction (only runtime packages)

---

## Remote Scanning Considerations

All file reads go through `afero.Fs`. All formats are text-based (JSON/XML), typically small (< 100KB). No special handling needed.

---

## Verification Plan

### Fixture-Based Tests
- packages.lock.json: Direct + transitive deps, multi-framework
- deps.json: Package + project library types
- packages.config: Regular + development dependencies
- csproj: PackageReference with version, PrivateAssets

### Interactive Testing
- Local: `mql run os -c "dotnet.packages(path: '.') { list { name version purl } }"`
- Container: Scan a .NET container image (e.g., `mcr.microsoft.com/dotnet/aspnet:8.0`)
- Remote: Same query over SSH/WinRM

---

## Implementation Plan

### Phase 1 (This PR)
1. packages.lock.json parser with direct/transitive classification
2. deps.json parser for runtime dependency listing
3. packages.config parser (legacy NuGet)
4. csproj parser for PackageReference extraction
5. MQL resources: `dotnet.packages`, `dotnet.package`
6. SBOM generator integration (type: `nuget`)

### Phase 2 (Follow-up PR)
1. PE assembly metadata extraction from DLL/EXE files
2. fsproj/vbproj support
