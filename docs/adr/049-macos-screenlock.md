# ADR: cnspec macOS Screen Lock Resource

**Status:** Proposed
**Date:** 2026-07-20

---

## Context

Requiring a password to wake a Mac from the screen saver or display sleep is one
of the most fundamental endpoint controls: it stops an unattended, unlocked
machine from being trivially accessed. The setting lives in the
`com.apple.screensaver` preference domain via two keys — `askForPassword`
(whether a password is required) and `askForPasswordDelay` (the grace period in
seconds before it is demanded). As prior art, osquery exposes the same domain
through its `screenlock` table.

The `macos` namespace already models posture elements — `macos.filevault`,
`macos.gatekeeper`, `macos.sip`, `macos.firewall`, `macos.softwareupdate` — but
there is no way to assert "the screen locks and a password is required promptly
after it engages." Multiple CIS macOS benchmark controls depend on this: a
password must be required on wake (`askForPassword == true`) and it must be
required immediately or within a small window, historically ≤ 5 seconds
(`askForPasswordDelay` small). Related idle / auto-logout controls
(`loginWindowIdleTime`, `com.apple.autologout AutoLogOutDelay`) live in the same
loginwindow space and are candidates for follow-up fields, but are out of scope
here.

The two settings that matter both live in the **per-user, per-host**
`com.apple.screensaver` domain:

| Key | Type | Meaning |
|-----|------|---------|
| `askForPassword` | bool (0/1) | Prompt for a password when the screen saver / lock screen is dismissed |
| `askForPasswordDelay` | int (seconds) | Grace period before the password is required. `0` = immediately. `2147483647` (`0x7FFFFFFF`) effectively disables the requirement |

This resource is registered under the `macos` namespace and is macOS-only; like
the existing `macos.*` resources it is not resolvable on other platforms.

### Per-user reality and modern-macOS caveats

The resource must not paper over the messy details of this control:

- **It is per-user and per-host.** The values live in the *ByHost* variant of the
  domain — `~/Library/Preferences/ByHost/com.apple.screensaver.<UUID>.plist` —
  not a single system-wide file. A machine with several local accounts can have
  different (or missing) values per user. There is no single host-wide value for
  the setting.
- **Reads are session-sensitive.** On modern macOS the effective screen-saver
  preferences are served by `cfprefsd`, so a plain
  `defaults read com.apple.screensaver askForPassword` run as the wrong user (or
  without the user's session) may return nothing even when a value is set. The
  reliable CLI path is `defaults -currentHost read com.apple.screensaver …` run
  as the target user.
- **MDM configuration profiles override the default.** In fleets this is
  increasingly governed by an MDM configuration profile carrying the
  `com.apple.screensaver` payload (`askForPassword` / `askForPasswordDelay`; on
  recent macOS the delay must be an `integer`, not a `real`). A profile-enforced
  value can differ from — and supersedes — the per-user default.

The resource therefore reports the **per-user default** it can read and
documents that an MDM-enforced value may supersede it. Enumerating the
authoritative profile-enforced value is delegated to the existing
`macos.profiles` / `macos.mdm` resources, which already parse installed
configuration profiles. This keeps `macos.screenlock` focused and avoids
presenting a single number as "the effective policy" when a profile is in play.

## Decision

Attach a lazy getter to the existing `macos` namespace resource and add one new
private element resource:

```lr
macos {
  // …existing: computerName / userPreferences / userHostPreferences /
  //            globalAccountPolicies / systemExtensions…
  // Screen lock (screen saver password) configuration
  screenlock() macos.screenlock          // NEW
}

// macOS screen lock (screen saver password) configuration
//
// Whether waking from the screen saver or display sleep requires a
// password (`askForPassword`) and the grace period in seconds before the
// password is demanded (`askForPasswordDelay`, 0 = immediately). Read from
// the per-user, per-host `com.apple.screensaver` preference domain. On
// managed fleets an MDM configuration profile may enforce different values;
// see `macos.profiles` for the profile-delivered payload. CIS macOS
// benchmarks require a password on wake with a short (ideally immediate)
// delay.
private macos.screenlock @defaults("enabled askForPasswordDelay") {
  // Whether a password is required to dismiss the screen saver / lock screen
  enabled bool
  // Raw askForPassword preference value (alias of enabled)
  askForPassword bool
  // Grace period in seconds before a password is required (0 = immediately)
  askForPasswordDelay int
}
```

`private` matches the sibling element resources (`macos.filevault`,
`macos.firewall.app`): `macos.screenlock` is only reachable through
`macos.screenlock()`, never constructed directly.

### Which user?

Because the setting is per-user, the resource commits to a documented rule
rather than silently picking one account:

- **Default subject: the primary console user** — the account tied to the
  connection, falling back to the first non-system user with a ByHost
  screensaver plist. This is the account that actually sits in front of the
  machine.
- `enabled` is `true` only when that user has `askForPassword == 1`. `enabled`
  and `askForPassword` are aliases so checks can use whichever reads better;
  `askForPasswordDelay` carries the grace period.
- A future `perUser()` breakdown (one entry per local account) is a natural
  extension for fleets that must assert the control on *every* user, but is
  deferred — the CIS single-value check is satisfied by the primary-user shape
  and keeps the default query cheap.

## Data Gathering

Ordered by transport capability; all paths feed the same three fields.

1. **Live host / SSH as the user — `defaults` (preferred).** Read the
   per-current-host domain for the target user:

   ```
   defaults -currentHost read com.apple.screensaver askForPassword
   defaults -currentHost read com.apple.screensaver askForPasswordDelay
   ```

   `-currentHost` is required: the values live in the ByHost domain. Missing keys
   exit non-zero and are treated as "unset" (see semantics below), not as an
   error.

2. **Offline / image / filesystem — plist read (fallback).** Parse the ByHost
   plist directly through the existing `parse.plist` resource:

   ```
   ~/Library/Preferences/ByHost/com.apple.screensaver.<HardwareUUID>.plist
   ```

   The `<HardwareUUID>` suffix (or, on newer macOS, a fixed
   `…/ByHost/com.apple.screensaver.plist`) is globbed per user home. This path
   works on a mounted disk image or container-style scan where no
   `defaults` / `cfprefsd` is available. The `macos` resource already exposes
   `userHostPreferences()`, which enumerates ByHost domains via
   `defaults -currentHost export` — `com.apple.screensaver` can be pulled from
   that map on live hosts, and the raw plist read covers the offline case.

3. **MDM-enforced value — out of band.** Not read here. Documented caveat: if a
   `com.apple.screensaver` configuration profile is installed, the enforced value
   overrides the per-user default and should be confirmed via `macos.profiles`.

### Value semantics (fail closed)

- `askForPassword`: `1`/`true` ⇒ `enabled = true`; `0`/absent ⇒ `false`. Absence
  is treated as **not required** (the insecure default), so a machine that never
  set it does not falsely pass.
- `askForPasswordDelay`: integer seconds. Absence normalizes to a **large
  sentinel** (`math.MaxInt32`) so a "delay ≤ 5" check does **not** pass on
  missing data. `2147483647` is surfaced as-is (effectively disabled).

## Resource Schema

`.lr` block as under **Decision**. Field summary:

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| `enabled` | bool | `askForPassword` key | `true` only when explicitly `1` |
| `askForPassword` | bool | `askForPassword` key | Alias of `enabled` |
| `askForPasswordDelay` | int | `askForPasswordDelay` key | Seconds; `0` = immediate, `2147483647` = disabled, unset ⇒ non-passing sentinel |

`@defaults("enabled askForPasswordDelay")` — a bare `macos.screenlock` in a shell
prints the two decision-relevant fields.

## Transport Compatibility

| Transport | Method |
|-----------|--------|
| Local (macOS) | `defaults -currentHost read com.apple.screensaver …` as the console user |
| SSH | Same `defaults` invocation over the SSH session |
| Filesystem / disk image / mounted volume | `parse.plist` on `…/ByHost/com.apple.screensaver*.plist` per user home |
| MDM-managed value | Not resolved here; cross-check `macos.profiles` (documented caveat) |

## Implementation

**Target file:** `providers/os/resources/macos_screenlock.go` (new), following the
lazy-fetch pattern of `macos_gatekeeper.go` / `macos_filevault.go`. The generated
struct `mqlMacosScreenlock` comes from the `.lr` after `make providers/build/os`.
The attach getter `screenlock()` goes on `mqlMacos`.

Getter signatures (post-generation):

```go
func (m *mqlMacos) screenlock() (*mqlMacosScreenlock, error)      // attach getter on macos

func (m *mqlMacosScreenlock) enabled() (bool, error)
func (m *mqlMacosScreenlock) askForPassword() (bool, error)
func (m *mqlMacosScreenlock) askForPasswordDelay() (int64, error)
```

A single lazy `fetch()` guarded by a mutex populates all three fields (mirroring
`fetchStatus` in `macos_gatekeeper.go`); each field getter calls `fetch()` then
returns its cached value:

```go
type mqlMacosScreenlockInternal struct {
	lock    sync.Mutex
	fetched bool
	ask     bool
	delay   int64
}

func (m *mqlMacosScreenlock) fetch() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.fetched {
		return nil
	}
	conn := m.MqlRuntime.Connection.(shared.Connection)

	// per-current-host domain: values live under ByHost. Missing key => unset.
	if out, err := runDefault(conn, "askForPassword"); err == nil {
		m.ask = strings.TrimSpace(out) == "1"
	}
	if out, err := runDefault(conn, "askForPasswordDelay"); err == nil {
		if v, perr := strconv.ParseInt(strings.TrimSpace(out), 10, 64); perr == nil {
			m.delay = v
		}
	} else {
		m.delay = math.MaxInt32 // absent => non-passing sentinel, not 0
	}
	m.fetched = true
	return nil
}
```

Offline transports substitute a `parse.plist` read of the ByHost plist for the
`defaults` invocation, selected on transport capability. `fetch()` resolves the
primary console user first (the connection's user, falling back to the first
local home with a ByHost screensaver plist); the per-user breakdown is deferred.

**Schema wiring:**

1. Edit `providers/os/resources/os.lr` — add `screenlock() macos.screenlock` to
   `macos {}` and the `private macos.screenlock` block.
2. Run `make providers/build/os` to regenerate `os.lr.go` /
   `os.lr.manifest.yaml`.
3. Add `macos_screenlock.go` with the getters above.

## Verification

- **Unit test** (`macos_screenlock_test.go`) against plist fixtures under
  `providers/os/resources/macos/testdata/`: a ByHost `com.apple.screensaver`
  plist with `askForPassword=1` / `askForPasswordDelay=0` asserting
  `enabled == true` and `askForPasswordDelay == 0`; and a second fixture with the
  key **absent** asserting `enabled == false` and a large (non-passing) delay, so
  the CIS check fails closed. Reuse the mock-provider recording pattern used by
  `macos_gatekeeper_test.go`.
- **Interactive:**

  ```
  cnquery run os -c 'macos.screenlock { askForPassword askForPasswordDelay }'
  ```

- **Content check** (the CIS assertion this resource exists to enable — password
  required and grace period ≤ 5s):

  ```mql
  macos.screenlock.askForPassword == true && macos.screenlock.askForPasswordDelay <= 5
  ```

  For strict "immediately," `askForPasswordDelay == 0`.
