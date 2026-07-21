# ADR: cnspec Network Host, Protocol, and Service Databases

**Status:** Proposed
**Date:** 2026-07-20

---

## Context

Three classic name-resolution/lookup config files ship on essentially every
operating system and are load-bearing for both networking behavior and security
posture:

- **hosts** — static hostname → IP mappings, consulted before DNS.
- **protocols** — IP protocol number ↔ name table (`tcp` 6, `udp` 17, `icmp` 1, …).
- **services** — network service name ↔ port/protocol table (`ssh` 22/tcp, …).

The OS provider has no first-class resource for any of them today. Authors must
fall back to `file("/etc/hosts").content` plus hand-rolled MQL string parsing,
which is brittle (comment handling, whitespace, aliases) and does not work
uniformly across platforms. (osquery exposes the equivalents as `etc_hosts`,
`etc_protocols`, and `etc_services`.)

**Security value:**

- **hosts** — the hosts file is a classic tampering target. Malware and adware
  redirect update/telemetry/security domains (e.g. pointing
  `updates.company.com` or an AV vendor's domain at `127.0.0.1` / `0.0.0.0` to
  blackhole them), and unexpected static mappings can silently override DNS.
- **protocols / services** — the canonical port/protocol mapping used to reason
  about exposed services; unexpected edits (a renamed/added service, a hijacked
  well-known port name) are a low-noise integrity signal.

## Decision

Expose the three databases through **getters on the existing `network` resource**:

```
network.hosts       // -> networkHosts
network.protocols   // -> networkProtocols
network.services    // -> networkServices
```

so authors write `network.hosts.where(ip == "0.0.0.0")`,
`network.services.where(port == 443)`, etc. This groups them with the network
resource's existing `interfaces`, `routes`, and `neighbors`, and — importantly —
avoids colliding `services` with the unrelated system-services resource
(`services`), mirroring osquery's own `etc_services`-vs-`services` split.

**Resource names.** The backing list resources are `networkHosts`,
`networkProtocols`, and `networkServices` (with `networkHosts.entry`, etc.), in
camelCase to match the existing `networkInterface` / `networkRoutes` /
`networkNeighbor` resources. Dotted `network.*` names are **not** used for the
resources themselves because that namespace is owned by the imported `network`
provider pack; the clean `network.hosts` access path is provided by the getter,
not by naming the resource `network.hosts`.

**Cross-OS.** The default paths are resolved per platform (never hard-coded to
`/etc`) because all three files exist on Windows too, under
`%SystemRoot%\System32\drivers\etc\`.

**`.where()` compatibility (design constraint).** A list resource whose fields
include a *resource-typed* field (`file`) fails when filtered with `.where()`:
the filter round-trips the resource's fields through `StoreData`, which cannot
rehydrate a resource pointer (only scalars survive). Therefore `file()` and
`content()` are **pure getters** (not `list()` dependencies), and `list()` takes
no arguments and reads the file itself. Only `path` (a string) is a stored field,
so `.where()` works.

## Data Gathering

**Default path resolution.** The `network.hosts/protocols/services` getters
resolve the default path from the platform family and build the resource with an
explicit id and path (mirroring `network.routes()`):

| Resource | Unix | Windows (`%SystemRoot%\System32\drivers\etc\`) |
|----------|------|------------------------------------------------|
| `networkHosts` | `/etc/hosts` | `hosts` |
| `networkProtocols` | `/etc/protocols` | `protocol` (singular, no `s`) |
| `networkServices` | `/etc/services` | `services` |

The resources also accept an explicit `init(path)` for a non-default location.
On Windows the path uses forward slashes (`C:/Windows/System32/drivers/etc/…`),
which the filesystem layer accepts. Note the Windows *protocols* file is named
`protocol` (singular), unlike Unix `protocols`.

**Reading.** `list()` reads content via `conn.FileSystem()` (afero). Because the
data is read through the filesystem abstraction — not a shell command — the
resources work over local, SSH, WinRM, docker/container-image, and mounted-disk
connections, including fully offline image scans.

**Parse rules (common to all three).** Classic whitespace-delimited, `#`-comment
format:

- Everything from `#` to end-of-line is a comment; captured on the entry
  (`comment`), stripped before field parsing.
- Blank / comment-only lines are skipped.
- Fields split on runs of whitespace.
- 1-based source `line` number preserved on each entry.

Per-file field layout:

- **hosts**: `IP  hostname [alias …]` → `ip` + `hostnames[]`.
- **protocols**: `name  number  [alias …]` → `name` + `number` + `aliases[]`.
- **services**: `name  port/protocol  [alias …]` → `name` + `port` + `protocol`
  (splitting the `port/proto` token) + `aliases[]`.

## Resource Schema

`.lr` — getters on `network` plus three list resources (hosts shown; the other
two are identical modulo names/fields):

```lr
network {
  // … existing interfaces/routes/neighbors …
  // Static hostname-to-IP mappings (the hosts file)
  hosts() networkHosts
  // IP protocol registry (the protocols file)
  protocols() networkProtocols
  // Network service/port registry (the services file)
  services() networkServices
}

networkHosts {
  []networkHosts.entry
  init(path string)
  path string       // path to the hosts file
  file() file       // the underlying file (pure getter)
  content() string  // raw file content (pure getter)
}

networkHosts.entry @defaults("ip hostnames") {
  line int              // 1-based line number
  ip string             // IPv4/IPv6 address
  hostnames []string    // canonical name + aliases
  comment string        // trailing # comment, if any
}
```

Entry field tables:

| `networkHosts.entry` | `networkProtocols.entry` | `networkServices.entry` |
|---|---|---|
| `line` int | `line` int | `line` int |
| `ip` string | `name` string | `name` string |
| `hostnames` []string | `number` int | `port` int |
| `comment` string | `aliases` []string | `protocol` string |
| | `comment` string | `aliases` []string, `comment` string |

## Transport Compatibility

| Transport | Method |
|-----------|--------|
| Local | Read via `conn.FileSystem()` |
| SSH | Read via `conn.FileSystem()` (SFTP) |
| WinRM | Read via `conn.FileSystem()` |
| Container image | Read from the image filesystem |
| Mounted disk image (offline) | Read from the mounted filesystem |

No command execution is required, so every filesystem-exposing connection works,
including offline image scans.

## Implementation

**Target Go file:** `providers/os/resources/network_databases.go` — the three
`network` getters, the three list resources (`mqlNetworkHosts`,
`mqlNetworkProtocols`, `mqlNetworkServices`) + their `.entry` types, and a shared
comment/whitespace tokenizer (`parseEtcTable`).

The `network` getter builds the resource with `NewResource` + explicit `__id` and
platform-resolved `path` (the `network.routes()` pattern):

```go
func (c *mqlNetwork) hosts() (*mqlNetworkHosts, error) {
	r, err := c.networkDBResource("networkHosts", "/etc/hosts", "C:/Windows/System32/drivers/etc/hosts")
	if err != nil {
		return nil, err
	}
	return r.(*mqlNetworkHosts), nil
}
```

`list()` is argless and reads the file itself; `file()`/`content()` are pure
getters (see the `.where()` design note above):

```go
func (x *mqlNetworkHosts) list() ([]any, error) {
	res := []any{}
	content, exists, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	if err != nil || !exists {
		return res, err
	}
	for _, r := range parseEtcTable(content) {
		entry, _ := CreateResource(x.MqlRuntime, "networkHosts.entry", map[string]*llx.RawData{
			"line":      llx.IntData(int64(r.line)),
			"ip":        llx.StringData(r.fields[0]),
			"hostnames": llx.ArrayData(llx.TArr2Raw[string](r.fields[1:]), "string"),
			"comment":   llx.StringData(r.comment),
		})
		res = append(res, entry)
	}
	return res, nil
}
```

`networkProtocols`/`networkServices` differ only in field extraction. Build with
`make providers/build/os` after editing `os.lr`.

## Verification

**Unit tests** (`security_parsers_internal_test.go`): `parseEtcTable` covers
comment-only lines, inline comments, tab/space separators, multiple aliases, and
1-based line numbers.

**Runtime** — verified across all three platforms (native macOS, Linux via a
container-image scan, and native Windows + Windows-over-SSH):

```mql
network.hosts.where(ip == "127.0.0.1") { ip hostnames }
network.services.where(name == "https") { name port protocol }
network.protocols.where(name == "tcp").first.number     // 6
network.hosts.path                                       // default path
network.hosts.file.permissions.string                   // -rw-r--r--
```

On Windows, `network.services` parsed the 277-entry `…\drivers\etc\services`
file with `https 443/tcp`; the hosts default path resolved to the Windows
location.

**Content check** (hosts file must not redirect a security-relevant domain):

```yaml
checks:
  - uid: hosts-no-security-domain-redirect
    title: The hosts file does not blackhole or hijack security update domains
    mql: |
      network.hosts.
        where(hostnames.any(_ == "updates.company.com")).
        all(ip == "10.0.0.5")
```
