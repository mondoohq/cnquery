# ADR: cnspec User knownHosts and shellHistory Fields

**Status:** Proposed
**Date:** 2026-07-20

---

## Context

Two per-user, on-disk artifacts carry high security signal but are not yet
exposed by the OS provider: the OpenSSH `known_hosts` file and interactive shell
history. Both are naturally properties of a user account — they live under the
account's home directory — and the `user` resource already parses the sibling
`~/.ssh/authorized_keys` via `authorizedkeys(home)` and discovers private keys
via `sshkeys()`. These two fields follow that same pattern. (osquery exposes the
equivalent data through its `known_hosts` and `shell_history` tables.)

**`known_hosts`** is the client-side trust anchor for SSH host authentication:
each entry says "this hostname/address is expected to present this public key."
Auditing it matters for:

- **Trust hygiene** — stale or attacker-planted entries silently suppress the
  "host key changed" warning, enabling undetected MITM after a key rotation.
- **Weak algorithms** — `ssh-rsa` (SHA-1) host keys that are still trusted.
- **Privacy / lateral-movement mapping** — a cleartext `known_hosts` reveals the
  inventory of hosts a compromised account can reach. CIS/DISA benchmarks
  recommend `HashKnownHosts yes` so the file cannot be mined; auditing that it is
  actually hashed is a real control.
- **CA and revocation posture** — `@cert-authority` lines delegate trust to a CA
  key; `@revoked` lines blocklist keys. Both change the trust model materially.

**Shell history** is a forensic and posture artifact:

- **Secret hygiene** — passwords, API tokens, and `AWS_SECRET…` values typed on a
  command line are written verbatim to history and often persist for months.
- **Incident response / threat hunting** — `curl … | sh`, `base64 -d`,
  reverse-shell one-liners, and privilege-escalation attempts stay recoverable
  from history long after the session ends.
- **Compliance** — several benchmarks call for history hygiene and for detecting
  credentials committed to disk.

## Decision

Add two lazily-computed fields to the existing **`user`** resource, mirroring the
existing `authorizedkeys(home)` / `sshkeys()` fields:

- `knownHosts(home) knownhosts` — the parsed `~/.ssh/known_hosts` for the account.
- `shellHistory() []shellHistory.command` — a flat list of every command across
  the account's shell history files.

Both attach to `user` rather than becoming free-floating tables because each
artifact belongs to exactly one account. Attaching reuses the account's `home`
and `name` for path resolution and attribution, and keeps queries natural
(`user(name: "root").knownHosts`, `user(name: "deploy").shellHistory`). Both are
backward compatible: they are lazy and only run when selected.

Introduce three supporting resources:

- `knownhosts` — a list resource wrapping a single `known_hosts` file. It has
  `init(path string)` so it can be selected directly by absolute path, which
  covers the **system-wide** file `/etc/ssh/ssh_known_hosts` that has no owning
  user.
- `private knownhosts.entry` — one parsed line, carrying
  `{line, host, isHashed, type, key, file}`.
- `private shellHistory.command` — one history command, carrying
  `{user, command, time, file}`.

Field-name rationale:

- `host` (not `hostname`) matches the mixed host/address/pattern nature of the
  field.
- `isHashed` is a first-class boolean because a hashed entry's host is **not
  recoverable**; downstream queries must branch on it rather than string-matching
  a `|1|…` blob.
- `type` mirrors `authorizedkeys.entry.type` (`ssh-ed25519`,
  `ecdsa-sha2-nistp256`, …).
- On `shellHistory.command`, `user` keeps entries attributable when lists are
  flattened across accounts, `file` identifies the source history file (and thus
  the originating shell), and `time` uses cnspec's `time` type so timestamp-aware
  shells can be range-queried — with the important caveat that `time` is often
  empty (see Data Gathering).

## Data Gathering

All reads go through `conn.FileSystem()` — no command execution anywhere — so
local, SSH, WinRM, container-image, and mounted-disk scans all behave
identically. This is the offline path by construction.

### known_hosts

**Per-user (via `user.knownHosts`)** — resolve `<home>/.ssh/known_hosts` from the
account's `home`. On Linux/macOS this is a POSIX join; on Windows the home is
already an absolute `C:\Users\<name>` path, joined with `\.ssh\known_hosts`.

**System file (via `knownhosts(path: "/etc/ssh/ssh_known_hosts")`)** — selected
explicitly by absolute path; identical parser. Default system paths are
`/etc/ssh/ssh_known_hosts` on Linux/macOS and `%ProgramData%\ssh\ssh_known_hosts`
on Windows. On Windows, per-user data lives at `%USERPROFILE%\.ssh\known_hosts`.
The byte format is identical across all platforms; only path resolution branches.

Line format — one record per line, space-separated:
`markers? host[,host2,…] type key [comment]`.

- **Hashed hosts** — with `HashKnownHosts yes`, the host field is
  `|1|<base64 salt>|<base64 SHA1 HMAC>`. Set `isHashed = true` and store the raw
  token in `host`; the original hostname is **cryptographically unrecoverable**
  (a candidate name can only be tested against the salt+HMAC), so it is never
  reversed. Consumers assert on `type`/`isHashed`, not on the opaque `host`.
- **Marker lines** — a line may be prefixed with `@cert-authority` (the host key
  is a CA that signs host certificates) or `@revoked` (the key is blocklisted).
  The marker is parsed off the front and the entry is kept, never dropped.
- **Multiple host patterns** — the host field can be a comma-separated list
  (`host1,host2,192.0.2.1`) and can include negations (`!badhost`) and wildcards
  (`*.example.com`). One entry is emitted per line, with `host` holding the raw
  (possibly comma-joined) pattern; splitting is left to the consumer so the
  "these hosts share one key" grouping is preserved.
- **Non-standard ports** — bracketed form `[host.example.com]:2222`.
- **Comments / blanks** — lines starting with `#` and empty lines are skipped;
  `line` numbers stay correct across skips.

### shellHistory

For each user, probe the known history-file paths under `home` and parse each one
that exists, emitting a flat list. A single account may have both a
`.bash_history` and a `.zsh_history`, and all matching entries are merged with
`file` distinguishing origin.

**POSIX shells** (under `home`):

| Shell | File | Timestamp support |
|-------|------|-------------------|
| bash  | `~/.bash_history` | None by default (see caveat) |
| zsh   | `~/.zsh_history` | Epoch, only when `EXTENDED_HISTORY` is set |
| fish  | `~/.local/share/fish/fish_history` | Always (`when:` epoch) |
| ksh   | `~/.sh_history` | None |

**Windows (PowerShell)** — PSReadLine writes plain newline-delimited command
lines, with no timestamps, to
`<home>\AppData\Roaming\Microsoft\Windows\PowerShell\PSReadLine\ConsoleHost_history.txt`.

Per-format parsing:

- **bash / ksh — plain lines.** One command per line; `time` empty. When the user
  has `HISTTIMEFORMAT` set, bash writes a preceding `#<epoch>` comment line before
  each command; parse it into `time` when present, otherwise leave `time` empty.
- **zsh — extended format.** Lines are either a bare `command` or the extended
  form `: <start-epoch>:<elapsed-seconds>;<command>`. Parse the `: ts:dur;` prefix
  into `time` (from `start-epoch`) and strip it to recover `command`. Multi-line
  commands use a trailing `\` continuation and are joined into one `command`. Bare
  lines (extended history off) leave `time` empty.
- **fish — YAML-ish.** Records look like:
  ```yaml
  - cmd: git push origin main
    when: 1690000000
    paths:
      - main
  ```
  Parse `cmd:` into `command` (fish escapes embedded newlines as `\n`) and `when:`
  into `time`; `paths:` is ignored. Use a tolerant line scanner keyed on `- cmd:`
  / `  when:` rather than a strict YAML parser — fish's writer is line-oriented and
  not always spec-clean.
- **PowerShell PSReadLine — plain lines.** `time` empty.

**Key caveat — `time` is usually empty.** bash does not store timestamps unless
`HISTTIMEFORMAT` is set at write time, which on the overwhelming majority of
systems it is not; a default `.bash_history` is bare command lines and `time` is
empty for every entry. PowerShell never records time. Only fish (always) and zsh
with `EXTENDED_HISTORY` reliably populate `time`. Missing `time` is not a parse
failure. Content authors must null-check (`time != empty`) before comparing.

Other considerations: history files can be large (stream rather than slurp where
practical), may contain non-UTF8 bytes (parse leniently), and a per-session
`HISTFILE` override can move the file — only the default locations are covered.

## Resource Schema

```lr
user @defaults("name uid gid") {
  // …existing fields (authorizedkeys, sshkeys, group, loggedIn, ntuserDat)…

  // Parsed ~/.ssh/known_hosts for this user
  knownHosts(home) knownhosts
  // Parsed interactive shell history (bash/zsh/fish/ksh/PowerShell)
  shellHistory() []shellHistory.command
}

// SSH known_hosts file
//
// A single known_hosts file: its path, the file reference, the raw content, and
// the parsed entry list. Select by absolute path
// (e.g. knownhosts(path: "/etc/ssh/ssh_known_hosts")) for the system-wide file,
// or reach it per account through user.knownHosts.
knownhosts {
  []knownhosts.entry(file, content)
  init(path string)
  // Path to the known_hosts file
  path string
  // known_hosts file
  file file
  // known_hosts file content
  content(file) string
}

// A single known_hosts entry
//
// One parsed line from a known_hosts file. `host` holds the raw host field; when
// isHashed is true it is an opaque |1|… token and the original host is not
// recoverable. @cert-authority / @revoked markers are stripped from the front and
// the entry is kept.
private knownhosts.entry @defaults("host type") {
  // Line number of the entry in the file
  line int
  // Host pattern(s); raw hashed token when isHashed is true
  host string
  // Whether the host field is hashed (|1|…); the original host is unrecoverable
  isHashed bool
  // Host key type (e.g., ssh-ed25519, ecdsa-sha2-nistp256, ssh-rsa)
  type string
  // Base64-encoded host key material
  key string
  // Source file
  file file
}

// A single shell history command
//
// One entry from a user's interactive shell history. `time` is populated only
// when the shell records it (fish always; zsh with EXTENDED_HISTORY; bash only
// with HISTTIMEFORMAT) and is empty otherwise — notably for default bash and
// PowerShell. `file` identifies the source history file, and thus the shell.
private shellHistory.command @defaults("command") {
  // Owning account name
  user string
  // Command text
  command string
  // Execution time, when recorded by the shell (else empty)
  time time
  // Source history file path (.bash_history / .zsh_history / fish_history / …)
  file string
}
```

### `knownhosts.entry` fields

| Field | Type | Source |
|-------|------|--------|
| `line` | int | 1-based line number in the file |
| `host` | string | Host field verbatim (hostname/IP/pattern list, or `\|1\|…` when hashed) |
| `isHashed` | bool | True when the host field begins with `\|1\|` |
| `type` | string | Key type token (field 2) |
| `key` | string | Base64 key material (field 3) |
| `file` | file | The `knownhosts` source file resource |

### `shellHistory.command` fields

| Field | Type | Source |
|-------|------|--------|
| `user` | string | The owning `user.name` |
| `command` | string | Command text (extended-format prefix stripped; fish `cmd:`) |
| `time` | time | Epoch from zsh-extended / fish `when:` / bash HISTTIMEFORMAT; empty otherwise |
| `file` | string | Absolute path of the history file the entry came from |

## Transport Compatibility

| Transport | Method |
|-----------|--------|
| Local | `conn.FileSystem()` read of the resolved path(s) |
| SSH | Same, over the SSH filesystem |
| WinRM | Same, with Windows path resolution (`%USERPROFILE%\.ssh`, PSReadLine profile) |
| Container image / mounted disk | Same — pure file reads, no exec needed |

Because there is no command execution, every transport — including offline image
and container scans — behaves identically.

## Implementation

Target files:

- `providers/os/resources/user.go` — add the `knownHosts` and `shellHistory`
  getters next to `authorizedkeys` / `sshkeys`.
- `providers/os/resources/knownhosts.go` — **new**: `init`, `id()`s, `content`,
  and the `list` getter.
- `providers/os/resources/knownhosts/parse.go` — **new**: a pure,
  unit-testable `Parse(io.Reader)`.
- `providers/os/resources/shellhistory.go` — **new**: the `user.shellHistory`
  getter body (path discovery + per-file dispatch) and the
  `shellHistory.command` `id()`.
- `providers/os/resources/shellhistory/parse.go` — **new**: pure per-format
  parsers (`ParseBash`, `ParseZsh`, `ParseFish`, `ParsePlain`).

Mirror the existing `authorizedkeys.go` per-user file parsing. The `knownHosts`
getter resolves `<home>/.ssh/known_hosts` and hands off to a `knownhosts`
resource exactly as `authorizedkeys(home)` does today; the `knownhosts.list`
getter reads through the `file` resource and runs `knownhosts.Parse`, creating one
`knownhosts.entry` per record. The parser is platform-independent — the only
platform branch is Windows path resolution in the getter.

The `shellHistory` getter builds a list of `{path, parser}` sources under `home`
(bash/zsh/fish/ksh), swapping to the single PSReadLine file on the `windows`
family (`conn.Asset().Platform.IsFamily("windows")`), opens each via
`conn.FileSystem()`, skips those that are missing, and emits one
`shellHistory.command` per parsed entry with `time` set from a `*time.Time` (nil →
empty). All per-shell format differences live in the parser functions.

Schema wiring:

1. Edit `providers/os/resources/os.lr` to add the two `user` fields and the three
   new resources above.
2. Run `make providers/build/os` to regenerate `os.lr.go` / `os.lr.json` /
   manifest.
3. Implement the hand-written getters and parsers in the `.go` files above.

## Verification

**Unit tests + fixtures** — `knownhosts/parse_test.go`:

- Cleartext single-host line → `isHashed=false`, correct `type`/`key`.
- Hashed line `|1|salt|hash ssh-ed25519 AAAA…` → `isHashed=true`, host preserved
  verbatim (not reversed).
- `@cert-authority` and `@revoked` marker lines → marker stripped, entry kept.
- Comma-separated host list and `[host]:port` → single entry, host verbatim.
- Comment/blank lines skipped; `line` numbers stay correct across skips.

`shellhistory/parse_test.go`:

- `ParseBash` on a plain `.bash_history` → all entries `time == nil`.
- `ParseBash` on a HISTTIMEFORMAT file (`#<epoch>` markers) → `time` set.
- `ParseZsh` extended (`: 1690000000:0;git status`) → `time` = that epoch,
  `command == "git status"`; multi-line `\` continuation joined.
- `ParseZsh` on a plain (non-extended) file → `time == nil`.
- `ParseFish` on a `- cmd:/when:/paths:` block → `command` + `time` from `when:`;
  escaped `\n` in `cmd` handled.
- PSReadLine plain file → `time == nil`.

**Interactive** (`cnquery run os -c '...'`):

```
cnquery run os -c 'user(name: "root").knownHosts.list { line host type isHashed }'
cnquery run os -c 'knownhosts(path: "/etc/ssh/ssh_known_hosts").list { host type }'
cnquery run os -c 'users.all(knownHosts.all(isHashed))'
cnquery run os -c 'user(name: "deploy").shellHistory { command time file }'
cnquery run os -c 'user(name: "root").shellHistory.where(command == /curl .*\|.*sh/)'
```

**Content checks** (`content/`):

```yaml
# No weak RSA host key is trusted for a privileged account.
mql: user(name: "root").knownHosts.all(type != "ssh-rsa")
```

```yaml
# No plaintext secrets committed to any user's shell history.
# time-independent, since bash time is usually empty.
mql: users.all(shellHistory.none(command == /(?i)(password|secret|token|api[_-]?key)=\S+/))
```
