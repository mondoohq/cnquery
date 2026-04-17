# ADR: cnspec SBOM Cataloger for Swift

**Status:** Proposed
**Date:** 2026-04-17

---

## Context

cnspec needs native SBOM cataloging for Swift packages to detect dependencies in both local and remote scanning contexts (SSH, containers). Swift is the primary language for Apple platform development (iOS, macOS, watchOS, tvOS), and increasingly used for server-side applications.

The Swift ecosystem has two package managers:
- **Swift Package Manager (SPM)**: Apple's official dependency manager, uses `Package.resolved` (JSON)
- **CocoaPods**: Legacy dependency manager, uses `Podfile.lock` (custom YAML-like format)

---

## File Formats

### Package.resolved (Swift Package Manager)

SPM lock file with resolved dependency versions. Has two format versions.

**Version 2 (current):**
```json
{
  "pins": [
    {
      "identity": "alamofire",
      "kind": "remoteSourceControl",
      "location": "https://github.com/Alamofire/Alamofire.git",
      "state": {
        "revision": "f82c23a8a7ef8dc1a49a8bfc6a96883e0a4e5c07",
        "version": "5.8.1"
      }
    }
  ],
  "version": 2
}
```

**Version 1 (legacy):**
```json
{
  "object": {
    "pins": [
      {
        "package": "Alamofire",
        "repositoryURL": "https://github.com/Alamofire/Alamofire.git",
        "state": {
          "revision": "f82c23a8a7ef8dc1a49a8bfc6a96883e0a4e5c07",
          "version": "5.8.1"
        }
      }
    ]
  },
  "version": 1
}
```

### Podfile.lock (CocoaPods)

CocoaPods lock file listing resolved pod versions.

```
PODS:
  - Alamofire (5.8.1)
  - SwiftyJSON (5.0.1)
  - Moya (15.0.3):
    - Alamofire (~> 5.0)

DEPENDENCIES:
  - Alamofire (~> 5.8)
  - SwiftyJSON (~> 5.0)
  - Moya (~> 15.0)

SPEC CHECKSUMS:
  Alamofire: 3ca42e259f7c0c813defc2c0820d4fce16e97a5d
  SwiftyJSON: 6faa0040f8b59dead0ee07436cbf76b73c08fd08
  Moya: abc123def456

PODFILE CHECKSUM: deadbeef12345678

COCOAPODS: 1.14.3
```

---

## Parsing Strategy

### Package.resolved Parser
1. Parse JSON via `encoding/json`
2. Detect version field (1 or 2) for format handling
3. V2: iterate `pins` array directly
4. V1: iterate `object.pins` array
5. Extract identity/package name, version, revision from each pin

### Podfile.lock Parser
1. Read line by line via `bufio.Scanner`
2. Parse `PODS:` section for package names and resolved versions
3. Format: `  - PackageName (version)` with optional sub-dependencies indented further
4. Parse `SPEC CHECKSUMS:` section for integrity hashes

### Error Handling
- Unknown Package.resolved version: warn and attempt v2 format
- Malformed entries: skip with debug log
- Missing sections in Podfile.lock: skip gracefully

---

## Package Identification

### PURL

SPM: `pkg:swift/<name>@<version>`
CocoaPods: `pkg:cocoapods/<name>@<version>`

Examples:
- `pkg:swift/alamofire@5.8.1`
- `pkg:cocoapods/Alamofire@5.8.1`

### CPE Generation
Vendor and product are both the package name (Swift packages are uniquely named).

---

## Dev/Prod Classification

Not available from either format — all dependencies are treated as production.

---

## Remote Scanning Considerations

Both files are small text/JSON (< 100KB). No special handling needed. All reads through `afero.Fs`.

Default search paths: project root directories, `/app`, `/usr/src/app`

---

## Verification Plan

- Package.resolved: v1 and v2 format fixtures
- Podfile.lock: pods with sub-dependencies, checksums
- Interactive: `mql run os -c "swift.packages(path: '.') { list { name version purl } }"`
