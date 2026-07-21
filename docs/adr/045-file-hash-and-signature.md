# ADR: cnspec File Hash and Code Signature Fields

**Status:** Proposed
**Date:** 2026-07-20

---

## Context

Two foundational endpoint-security questions are currently unanswerable in MQL, and
both are properties of a specific file:

1. **File integrity** — "Does `/usr/bin/sudo` match the vendor binary?" or "Does any
   script under `/usr/local/bin` match a known-bad SHA-256?" Verifying a content
   digest is the basis of tamper detection and cross-referencing against threat-intel
   hash feeds. cnspec today cannot compute a digest of a file.

2. **Code-signing trust** — "Is this executable genuinely from who it claims, and has
   the OS's trust machinery accepted it?" This underpins publisher allow-listing
   (only trust binaries signed by an approved Apple Team ID or Windows publisher),
   tamper detection (a patched binary breaks its seal), and supply-chain checks
   (flag unsigned or ad-hoc-signed tools). cnspec has no equivalent today.

Both facts are intrinsic to a file that already has identity (`path`) and a byte
stream (`content`, `size`, `permissions`), so the existing `file` resource is their
natural home — not a distinct, system-wide table. osquery exposes analogous data via
its `hash`, `signature`, and `authenticode` tables.

The governing constraint: adding fields to `file` must not regress the many existing
`file(...)` queries that never ask for a hash or a signature. Hashing a large binary
and running a platform verification tool are both expensive, so these fields must be
**lazy** — computed only when selected — which is exactly how MQL computed fields
already behave.

## Decision

Extend the existing `file` resource with two independent capabilities.

**Content digests** — three computed fields on `file`:

- `md5(path) string`
- `sha1(path) string`
- `sha256(path) string`

Each declares a dependency on `path`, so it resolves only when selected; existing
queries that read `path`/`size`/`permissions` pay nothing. One filesystem read serves
all three: the first digest getter that runs streams the file once and caches
`md5`/`sha1`/`sha256` on the resource, so `file(p) { md5 sha1 sha256 }` reads the bytes
a single time. Digests are lowercase hex strings. Unreadable files, directories, and
non-existent paths resolve to the **empty string** (`""`), never an error — so
`files.find(...).map(sha256)` never aborts a walk because one entry was a directory or
permission-denied. Weak digests (`md5`, `sha1`) are included deliberately: threat-intel
feeds and legacy allow/deny lists remain overwhelmingly MD5/SHA-1 keyed. Policy authors
choose `sha256` for integrity assertions.

**Code signature** — a single computed field on `file` returning a shared element
resource:

```lr
signature(path) file.signature
```

backed by a `private file.signature` element whose fields **unify** the macOS
`codesign` and Windows Authenticode concepts behind one schema. A single query
(`file(p).signature.verified`) then works on both platforms; the `format` field tells
the author which backend produced the result. Declaring `signature(path)` makes it
lazy — the verification tool runs only when the field is selected.

`signed` vs `verified` is a deliberate two-field split: a binary can be
`signed == true` (a signature blob exists) yet `verified == false` (seal broken, cert
untrusted/revoked/expired, or ad-hoc self-signed). Security policy almost always wants
`verified`.

| unified field | macOS source | Windows source |
|---|---|---|
| `signed` | signature present | `Status != NotSigned` |
| `verified` | passes `codesign --verify --deep --strict` | `Status == Valid` (`WinVerifyTrust` trusted) |
| `authority` | leaf `Authority=` (e.g. `Developer ID Application: Acme (TEAMID)`) | signer `Subject` common name |
| `teamId` | `TeamIdentifier=` | *(empty — no equivalent)* |
| `issuer` | *(empty — chain is on `authority`)* | signing cert `Issuer` |
| `timestamp` | `Timestamp=` | countersignature / timestamp |
| `format` | `"codesign"` | `"authenticode"` |

### OS support

Content digests are pure byte-stream computation over the connection's filesystem, so
they work identically on every platform cnspec can read a file on — no platform
branching, one code path for all transports.

| Capability | Linux | macOS | Windows |
|---|---|---|---|
| `md5` / `sha1` / `sha256` | Yes | Yes | Yes |
| `signature` | N/A (null) | Yes (`codesign`) | Yes (Authenticode) |

Signature verification is inherently OS-specific — it depends on each platform's native
trust store and verification API. **There is no universal, per-file ELF code-signing
model on Linux**, so `file.signature` is honestly marked N/A there and resolves to a
null resource. Two adjacent Linux mechanisms are a *different trust model* and out of
scope:

- **IMA/EVM appraisal** — kernel-enforced file integrity via `security.ima`/
  `security.evm` extended attributes against a system keyring. This is a host-integrity
  policy, not a per-binary publisher signature.
- **Package-level GPG signatures** (RPM/DEB) — sign the *package*, not the on-disk
  file, and are verified at install time via the package manager's keyring (already
  reachable through the `packages` resource).

On Linux, `file(p).signature` is queryable everywhere but yields null rather than
erroring; policies gate on platform via `filters:`
(`asset.platform.family.contains("darwin")` / `"windows"`).

## Data Gathering

### Content digests (all OSes)

Open the file through `conn.FileSystem()` (the afero abstraction the `file` resource
already uses for `content`) and stream it through the Go standard-library hashers
(`crypto/md5`, `crypto/sha1`, `crypto/sha256`) via a single `io.Copy` into an
`io.MultiWriter`. Streaming bounds memory by the copy buffer, not the file size — a
multi-gigabyte file is hashed without being loaded into memory. Because one read feeds
all three algorithms, the read is done once and all three digests are cached on the
resource on first access.

No offline fallback is required: `conn.FileSystem()` is already backed by the target's
filesystem, so on a mounted disk image, container image, or tarball connection the
digests compute without any command execution.

**Edge cases**

- **Directory** → `""` (no meaningful content digest).
- **Non-existent path** → `""` (consistent with `exists == false`).
- **Unreadable file** (permission denied, special/device file, broken symlink) → `""`;
  the error is swallowed so list maps do not abort (optionally logged at debug level).
- **Empty regular file** → the well-known digest of zero bytes (e.g. sha256
  `e3b0c442…b855`), *not* `""` — an empty regular file is readable and has a defined
  hash. Only unreadable/non-file cases yield `""`.
- **Large files** → streamed, never buffered; no size cap, but scope `files.find`
  before mapping a digest over a whole tree.
- **Symlinks** → follow the filesystem's resolution (afero `Open` follows), matching
  `content` behavior.

### Code signature — macOS

1. **Verify:** `codesign --verify --deep --strict <path>` — exit 0 ⇒ `verified = true`;
   non-zero ⇒ `verified = false`.
2. **Display metadata:** `codesign -dvvv <path>` — note this writes to **stderr**. Parse:
   - `Authority=…` lines (first is the leaf ⇒ `authority`, e.g.
     `Developer ID Application: Acme Inc (AB12CD34EF)`),
   - `TeamIdentifier=…` ⇒ `teamId` (empty / `not set` for ad-hoc),
   - `Timestamp=…` ⇒ `timestamp`,
   - presence of a `CodeDirectory` / `Signature` block ⇒ `signed = true`
     (`Signature=adhoc` ⇒ signed but not from a real identity, so `teamId` empty).
   - "code object is not signed at all" on stderr ⇒ `signed = false`.
3. `format = "codesign"`.

Gatekeeper posture (`spctl -a`) is deliberately **not** folded into `verified` —
`verified` reflects the cryptographic seal, keeping semantics parallel with Windows.
Gatekeeper acceptance could be a future `notarized`/`gatekeeperAccepted` field.

### Code signature — Windows

Run PowerShell:

```powershell
Get-AuthenticodeSignature -FilePath '<path>' |
  Select-Object Status, StatusMessage,
    @{n='Subject';   e={$_.SignerCertificate.Subject}},
    @{n='Issuer';    e={$_.SignerCertificate.Issuer}},
    @{n='TimeStamp'; e={$_.TimeStamperCertificate.NotBefore}} |
  ConvertTo-Json
```

Map results:

- `Status == Valid` ⇒ `signed = true`, `verified = true`.
- `Status == NotSigned` ⇒ `signed = false`, `verified = false`.
- `Status ∈ {HashMismatch, NotTrusted, UnknownError, Incompatible}` ⇒ `signed = true`,
  `verified = false`.
- `Subject` ⇒ `authority`; `Issuer` ⇒ `issuer`; timestamp ⇒ `timestamp`;
  `format = "authenticode"`.
- `Get-AuthenticodeSignature` transparently resolves **catalog** signatures — many OS
  binaries are catalog-signed rather than embedded — matching `WinVerifyTrust` behavior.

### Code signature — offline / mounted images

Both backends require executing the platform's verification tool against a live OS of
the matching family (`codesign` and `WinVerifyTrust` need the host trust store and
APIs). On a mounted-image, container, or Linux connection, `signature` resolves to
**null** (with a debug log) rather than returning a misleadingly "unsigned" result.
Parsing the embedded PKCS#7 / Mach-O `LC_CODE_SIGNATURE` blob directly to report
`signed` + certificate subject offline (without trust evaluation) is explicitly
deferred.

**Signature edge cases**

- **Unsigned file** ⇒ `signed = false`, `verified = false`, other fields empty.
- **Ad-hoc signature** (macOS) ⇒ `signed = true`, `teamId` empty, `verified` reflects
  `--verify`.
- **Broken seal / tampered binary** ⇒ `signed = true`, `verified = false`.
- **Catalog-signed** (Windows) ⇒ handled; `authority`/`issuer` come from the catalog
  signer.
- **Expired or revoked cert** ⇒ `signed = true`, `verified = false`.
- **Non-executable / directory / missing path** ⇒ null resource (nothing to verify).

## Resource Schema

`os.lr` additions to the existing `file` block, plus the new `file.signature` element:

```lr
file @defaults("path size permissions.string") {
  init(path string)
  path string
  basename(path) string
  dirname(path) string
  content(path, exists) string
  exists(path) bool
  permissions(path) file.permissions
  size(path) int
  user() user
  group() group
  empty(path) bool
  // NEW — content digests (computed lazily; empty string for dirs /
  // unreadable / missing files). One read serves all three.
  // MD5 digest of the file content, lowercase hex
  md5(path) string
  // SHA-1 digest of the file content, lowercase hex
  sha1(path) string
  // SHA-256 digest of the file content, lowercase hex (preferred for integrity)
  sha256(path) string
  // NEW — code signature (macOS codesign + Windows Authenticode; null on
  // Linux and on offline/mounted connections). Computed lazily.
  signature(path) file.signature
}

// Code-signing information for a file, unifying macOS codesign and Windows
// Authenticode. Populated only on macOS and Windows.
private file.signature @defaults("signed verified authority") {
  // Whether a code signature is present on the file
  signed bool
  // Whether the signature passes the platform's trust verification
  verified bool
  // Leaf signer identity (macOS Authority / Windows subject common name)
  authority string
  // Apple Developer Team Identifier (macOS only; empty otherwise)
  teamId string
  // Issuer of the signing certificate (Windows only; empty otherwise)
  issuer string
  // Signing / countersignature timestamp, if present
  timestamp time
  // Signature backend: "codesign" (macOS) or "authenticode" (Windows)
  format string
}
```

Digest fields:

| Field | Type | Depends on | Source | Empty when |
|-------|------|-----------|--------|-----------|
| `md5` | string | `path` | stream + `crypto/md5`, hex | dir / unreadable / missing |
| `sha1` | string | `path` | stream + `crypto/sha1`, hex | dir / unreadable / missing |
| `sha256` | string | `path` | stream + `crypto/sha256`, hex | dir / unreadable / missing |

Signature fields:

| Field | Type | macOS | Windows | Linux |
|-------|------|-------|---------|-------|
| `signed` | bool | signature present | `Status != NotSigned` | — |
| `verified` | bool | `codesign --verify` passes | `Status == Valid` | — |
| `authority` | string | leaf `Authority=` | signer `Subject` | — |
| `teamId` | string | `TeamIdentifier=` | (empty) | — |
| `issuer` | string | (empty) | cert `Issuer` | — |
| `timestamp` | time | `Timestamp=` | timestamp cert | — |
| `format` | string | `"codesign"` | `"authenticode"` | — |

## Transport Compatibility

Digests read via the afero filesystem abstraction, so they are available on every
transport where `file` already resolves `content` — no command execution required.
Signature verification is command-driven, so it requires a live target of the matching
OS family.

| Transport | Digests | Signature |
|-----------|---------|-----------|
| Local (macOS) | Yes | Yes (`codesign`) |
| Local (Windows) | Yes | Yes (PowerShell) |
| Local / SSH (Linux) | Yes | N/A (null) |
| SSH (macOS) | Yes | Yes (`codesign` over SSH) |
| WinRM (Windows) | Yes | Yes (`Get-AuthenticodeSignature` over WinRM) |
| Container image | Yes (offline) | No (null — no live trust store) |
| Mounted device / disk image | Yes (offline) | No (null — needs a running OS) |

## Implementation

**Digests — target Go file:** `providers/os/resources/file.go` (extend the existing
`mqlFile`). Add a shared helper that reads the file once and fills all three digest
`TValue`s, plus three thin getters that call it. The getters follow the existing
`content`/`size` getter shape (`conn := s.MqlRuntime.Connection.(shared.Connection)`),
so no new dependency injection is needed.

```go
func (s *mqlFile) md5(path string) (string, error)    // returns cached, else computes all
func (s *mqlFile) sha1(path string) (string, error)
func (s *mqlFile) sha256(path string) (string, error)
func (s *mqlFile) computeDigests(path string) error   // reads once, fills md5/sha1/sha256
```

```go
func (s *mqlFile) sha256(path string) (string, error) {
	if !s.Sha256.IsSet() {
		if err := s.computeDigests(path); err != nil {
			return "", err
		}
	}
	return s.Sha256.Data, nil
}

func (s *mqlFile) computeDigests(path string) error {
	setEmpty := func() { // dir / unreadable / missing -> "" on all three
		for _, v := range []*plugin.TValue[string]{&s.Md5, &s.Sha1, &s.Sha256} {
			*v = plugin.TValue[string]{Data: "", State: plugin.StateIsSet}
		}
	}

	conn := s.MqlRuntime.Connection.(shared.Connection)
	f, err := conn.FileSystem().Open(path)
	if err != nil {
		setEmpty() // missing or permission denied — not a query-breaking error
		return nil
	}
	defer f.Close()

	if fi, err := f.Stat(); err == nil && fi.IsDir() {
		setEmpty()
		return nil
	}

	m, s1, s256 := md5.New(), sha1.New(), sha256.New()
	if _, err := io.Copy(io.MultiWriter(m, s1, s256), f); err != nil {
		setEmpty() // e.g. device file that errors mid-read
		return nil
	}

	s.Md5 = plugin.TValue[string]{Data: hex.EncodeToString(m.Sum(nil)), State: plugin.StateIsSet}
	s.Sha1 = plugin.TValue[string]{Data: hex.EncodeToString(s1.Sum(nil)), State: plugin.StateIsSet}
	s.Sha256 = plugin.TValue[string]{Data: hex.EncodeToString(s256.Sum(nil)), State: plugin.StateIsSet}
	return nil
}
```

`io.MultiWriter` hashes all three algorithms in the single `io.Copy`, so the file is
read exactly once regardless of how many digests the query selects. The `md5`/`sha1`
getters are identical in shape to `sha256` but return the matching cached field.

**Signature — target Go file:** new `providers/os/resources/file_signature.go` (getter
on the existing `mqlFile`, keeping `file.go` focused). The element type is generated as
`mqlFileSignature`.

```go
func (s *mqlFile) signature(path string) (*mqlFileSignature, error)         // platform branch
func codesignSignature(rt *plugin.Runtime, conn shared.Connection, path string) (*mqlFileSignature, error)
func authenticodeSignature(rt *plugin.Runtime, conn shared.Connection, path string) (*mqlFileSignature, error)
```

```go
func (s *mqlFile) signature(path string) (*mqlFileSignature, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	pf := conn.Asset().Platform
	switch {
	case pf != nil && pf.IsFamily("darwin"):
		return codesignSignature(s.MqlRuntime, conn, path)
	case pf != nil && pf.IsFamily("windows"):
		return authenticodeSignature(s.MqlRuntime, conn, path)
	default:
		// Linux / offline: no per-file signature model -> null resource
		s.Signature = plugin.TValue[*mqlFileSignature]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}
}

func codesignSignature(rt *plugin.Runtime, conn shared.Connection, path string) (*mqlFileSignature, error) {
	verify, _ := conn.RunCommand("codesign --verify --deep --strict " + shellQuote(path))
	disp, err := conn.RunCommand("codesign -dvvv " + shellQuote(path))
	if err != nil {
		return nil, err
	}
	meta, _ := io.ReadAll(disp.Stderr) // codesign -dvvv writes to stderr
	sig := parseCodesign(string(meta)) // -> signed, authority, teamId, timestamp
	sig.verified = verify != nil && verify.ExitStatus == 0
	res, err := CreateResource(rt, "file.signature", map[string]*llx.RawData{
		"signed":    llx.BoolData(sig.signed),
		"verified":  llx.BoolData(sig.verified),
		"authority": llx.StringData(sig.authority),
		"teamId":    llx.StringData(sig.teamID),
		"issuer":    llx.StringData(""),
		"timestamp": llx.TimeDataPtr(sig.timestamp),
		"format":    llx.StringData("codesign"),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlFileSignature), nil
}
```

The Windows path is analogous:
`RunCommand("powershell -c \"Get-AuthenticodeSignature … | ConvertTo-Json\"")`,
`json.Unmarshal`, map `Status`→`signed`/`verified`, `Subject`→`authority`,
`Issuer`→`issuer`, `format = "authenticode"`. Keep the `parseCodesign` line scanner and
the Authenticode JSON struct local to this file with focused unit tests.

**`os.lr` edit + regeneration:** add the three digest lines and the `signature` line to
the `file` block, add the `private file.signature` block, then run
`make providers/build/os` to regenerate `os.lr.go` / `os.lr.json` / the manifest (which
declares `Md5`/`Sha1`/`Sha256` as `plugin.TValue[string]`, `Signature` as
`plugin.TValue[*mqlFileSignature]` on `mqlFile`, and the new `mqlFileSignature` type).

## Verification

**Unit tests + fixtures**

Digests (`providers/os/resources/file_test.go`, mock connection over a fixture tree):

- Known-content file → assert exact hex `md5`/`sha1`/`sha256`.
- Empty regular file → assert the zero-byte digests (not `""`).
- Directory path → all three return `""`.
- Missing path → all three return `""`.
- Single-read caching → selecting all three triggers exactly one filesystem `Open`
  (spy/counter on the mock filesystem).

Signature (`providers/os/resources/file_signature_test.go`, mock connection returning
canned command output):

- macOS Developer-ID binary: fixture of `codesign -dvvv` stderr ⇒ assert `signed`,
  `authority`, `teamId`, `format == "codesign"`; verify exit 0 ⇒ `verified == true`.
- macOS unsigned: "not signed at all" ⇒ `signed == false`, `verified == false`.
- macOS ad-hoc: `Signature=adhoc` ⇒ `signed == true`, `teamId == ""`.
- Windows: `Get-AuthenticodeSignature` JSON with `Status: Valid` ⇒ `verified == true`,
  `authority`/`issuer` populated, `format == "authenticode"`; `Status: HashMismatch` ⇒
  `signed == true`, `verified == false`; `Status: NotSigned` ⇒ `signed == false`.
- Linux: platform=linux ⇒ `signature` resolves null.

**Interactive**

```sh
cnquery run os -c 'file("/usr/bin/sudo").sha256'
cnquery run os -c 'files.find(from: "/usr/local/bin", type: "file").map(sha256)'
cnquery run os -c 'file("/etc").sha256'   # directory -> ""
cnquery run os -c 'file("/Applications/Firefox.app").signature { signed verified authority teamId }'
cnquery run os -c 'file("C:/Windows/System32/cmd.exe").signature { verified authority issuer }'
cnquery run os -c 'file("/usr/bin/ls").signature'   # Linux -> null
```

**Content checks**

```yaml
# Known-good binary integrity
mql: file("/usr/bin/docker").sha256 == "3f7b8c…"
```

```yaml
# Require a trusted signature on a deployed agent binary (macOS)
filters: asset.platform.family.contains("darwin")
mql: file("/usr/local/bin/agent").signature.verified == true
```
</content>
</invoke>
