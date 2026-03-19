# MQL SSH Scan Command Reduction Plan

## Goal
Reduce repeated remote commands during `cnspec scan` over SSH without lying about system state, weakening correctness, or introducing broad command-string memoization that could cache volatile results.

This plan is intentionally split into three steps, in recommended execution order:

1. Targeted service resolution and batched systemd collection
2. SSH file/stat de-duplication for shell-backed access paths
3. Scan-scope reuse for stable resource fields, plus observability to prove the win

## Verified findings

### 1) `service("name")` currently hydrates the full service inventory
Verified in repo:
- `providers/os/resources/services.go:initService`
- `providers/os/resources/services.go:list`
- `providers/os/resources/services/systemd.go:List`
- `providers/os/resources/services/manager.go`

Current behavior:
- `service("foo")` creates the `services` resource
- `services.refreshCache(nil)` calls `services.GetList()`
- `services.list()` resolves the service manager and calls `List()`
- `SystemDServiceManager.List()` runs:
  - `systemctl list-unit-files --type service --all`
  - then for every service:
    - `systemctl is-active <name>`
    - `systemctl status <name>`

Implication:
- a single named service lookup is currently O(all services), not O(1)
- the debug log full of `systemctl is-active ...` / `systemctl status ...` is expected from current code

### 2) SSH + sudo switches file access to shell-backed `cat`/`stat` paths
Verified in repo:
- `providers/os/connection/ssh/ssh.go:FileSystem`
- `providers/os/connection/ssh/cat/cat.go`
- `providers/os/connection/ssh/cat/cat_file.go`
- `providers/os/connection/local/statutil/stat.go`

Current behavior:
- when sudo is active, SSH does not use SFTP for reads; it switches to `cat` FS
- `cat.Fs.Open()` and `cat.Fs.Stat()` each create a fresh `statutil` helper
- `statutil.Stat()` performs:
  - `uname -s` for parser detection
  - `test -e <path>`
  - `stat -L <path> -c ...`
- file reads then use `cat <path>`

Implication:
- repeated `uname -s`, `test -e`, `stat -L`, and `cat` lines are a direct consequence of current shell-backed file access
- `uname -s` is noisier than necessary because the stat helper is recreated repeatedly

### 3) PAM/file checks trigger repeated but explainable file work
Verified in repo:
- `providers/os/resources/pam.go`
- `providers/os/resources/utils.go:getSortedPathFiles`
- `providers/os/resources/files.go`
- generated getters in `providers/os/resources/os.lr.go`
- `providers/os/resources/file.go`

Current behavior:
- `pam.conf.files()` checks `/etc/pam.d`
- `getSortedPathFiles()` asks the same `file` resource for both `exists` and `permissions`
- directory enumeration falls back to shell `find -L ...` on SSH because SSH does not advertise `Capability_FindFile`
- file content access separately uses `cat`

Implication:
- the repeated `/etc/pam.d` existence/stat/content calls are not random duplication; the main avoidable waste is the `exists()` vs `stat()` split, while `permissions` / `size` / `user` / `group` are already coalesced once `file.go:stat()` runs

### 4) Caching exists, but the boundaries are too local
Verified in repo:
- field memoization: `providers-sdk/v1/plugin/runtime.go:GetOrCompute`
- resource instance cache: generated resource creation in `providers/os/resources/os.lr.go`
- LLx executor-local cache: `llx/llx.go`
- scan executor in sibling `cnspec` repo: `cnspec/policy/executor/internal/execution_manager.go:executeCodeBundle`
- scan path: `cnspec/policy/scan/local_scanner.go`

Current behavior:
- field values are memoized on a resource instance
- resource instances are reused inside a provider runtime by `resource-name + MqlID`
- LLx step caching is per executor instance
- `cnspec` creates a fresh `llx.NewExecutorV2(...)` per execution query/code bundle

Implication:
- reuse exists inside a resource/runtime, but not as a global “same remote work once per scan” guarantee
- the current scan path still re-enters expensive provider/resource code more often than necessary

### 5) Existing infrastructure already gives us two important design precedents
Verified in repo:
- targeted managers already exist for users/groups:
  - `providers/os/resources/users/manager.go`
  - `providers/os/resources/groups/groups.go`
- ephemeral in-memory recording already exists:
  - `providers/runtime.go:EnableResourcesRecording`
  - `apps/cnspec/cmd/scan.go`
  - `providers-sdk/v1/recording/asset_recording.go`

Implication:
- adding targeted service lookup is aligned with existing manager patterns
- scan-scope reuse should build on recording/resource semantics, not on ad hoc memoization of raw shell command strings

## External research

### systemd / systemctl
Verified from primary docs:
- `systemctl status` is intended for human-readable output, not machine parsing:
  - https://man7.org/linux/man-pages/man1/systemctl.1.html
- `systemctl show` is intended for computer-parsable output and supports `--property=`:
  - https://man7.org/linux/man-pages/man1/systemctl.1.html
- `systemctl show` accepts one or more units, which makes batched property collection viable:
  - https://www.freedesktop.org/software/systemd/man/latest/systemctl.html
- relevant properties such as `ActiveState`, `UnitFileState`, and `Description` are part of the systemd unit model:
  - https://www.freedesktop.org/wiki/Software/systemd/dbus/

Design implication:
- `systemctl status` should not remain the primary parsing surface
- `systemctl show` is the correct primitive for targeted and batched service collection

## Baseline verification performed
Targeted tests were run before writing this plan:

- `go test ./providers/os/resources/services -run 'Test(ParseServiceSystemDUnitFiles|ParseServiceSystemDUnitFilesPhoton|SystemdFS|SystemDExtractDescription)$' -count=1`
- `go test ./providers/os/connection/ssh/cat -run '^TestCatFs$' -count=1`
- `go test ./providers/os/resources -run 'TestResource_(Services|Service|Pam)$' -count=1`

Observed result:
- all passed

These do not prove the new design yet, but they verify the current code paths and give us a safe baseline for refactoring.

## Mandatory end-to-end benchmark
This optimization work must be validated against one fixed end-to-end scan. The target host IP and policy file are mandatory and must not be substituted.

Benchmark command (verified paths):
```
MONDOO_AUTO_UPDATE=false \
PROVIDERS_PATH=/home/syl/.config/mondoo/providers \
/usr/bin/cnspec scan ssh ubuntu@34.204.71.195 \
  -i /home/syl/.ssh/policy-v2-testing-rsa \
  --sudo \
  -f /home/syl/self/development/cnspec-enterprise-policies/worktrees/fix-usb-storage-modprobe-resource/policies/ubuntu-20.04.mql.yaml \
  --auto-update=false \
  --incognito \
  --verbose
```

Execution notes:
- Run from `/home/syl/self/development/cnspec` for consistency.
- The user-suggested repo-local `./tmp/providers` path did not exist during verification, so the corrected benchmark uses the existing provider cache at `/home/syl/.config/mondoo/providers`.
- Before/after timing must use the same binary path, provider path, SSH key, target IP, policy file, and flags.
- Measure with `/usr/bin/time -f 'ELAPSED=%e USER=%U SYS=%S EXIT=%x' ...` so the elapsed wall-clock time is captured alongside the scan result.

Measured baseline before fixes (verified 2026-03-11):
- The command was run with the exact host, policy, key, and flags above.
- Result: the scan did not complete within 1800 seconds of wall-clock time.
- Observed failure mode: the run hit the 30-minute timeout while verbose output was still traversing deep `/snap/core20/...` paths over shell-backed SSH file access.
- Current baseline for planning purposes is therefore `>1800s` rather than a completed full-run elapsed time.

Measured A/B result after rebasing this branch onto `origin/main` (verified 2026-03-12):
- The branch was rebased onto `origin/main`, which includes the upstream snap-package optimization from `mql#6878` (`ed01e239c`).
- Two isolated provider roots were built for an apples-to-apples benchmark:
  - `origin/main` only: `PROVIDERS_PATH=/home/syl/self/temp/providers-origin-main`
  - rebased branch: `PROVIDERS_PATH=/home/syl/self/temp/providers-rebased-branch`
- `origin/main` benchmark result: `ELAPSED=483.59 USER=11.73 SYS=4.54 EXIT=0`.
- Rebasing branch benchmark result: `ELAPSED=349.88 USER=11.16 SYS=4.27 EXIT=0`.
- Measured improvement attributable to this branch's `Reduce SSH service and file scan commands` work on top of main: `133.71s` faster (about `27.65%` lower elapsed time).
- Compared to the original pre-snap-fix timeout baseline, the rebased branch still saves at least `1450.12s` versus `>1800s`.


After-fix measurement requirement:
- After implementing the fixes, rerun the exact same command and record the completed `ELAPSED` value in this plan or the implementation PR.
- Treat completion in under 1800 seconds as the minimum success bar for the current verified baseline.
- Report time saved as a lower bound of `1800s - after_elapsed` until an unfixed full run is captured to completion.
- If the unfixed command is later allowed to finish to completion, replace the lower-bound baseline with the actual full elapsed time and recompute the exact savings.

---

## 1. Targeted service resolution and batched systemd collection

### Problem
The current design makes `service("name")` pay the cost of `services.list`, and `services.list` pays two extra `systemctl` commands per discovered unit.

This is the largest single avoidable source of command amplification.

### Recommended design

#### 1.1 Extend the service manager contract
Change the service abstraction from list-only to list-plus-targeted lookup.

Current interface:
- `providers/os/resources/services/manager.go`
  - `type OSServiceManager interface { Name() string; List() ([]*Service, error) }`

Proposed interface:
- add `Get(name string) (*Service, error)`

Rationale:
- this matches existing patterns already used for users/groups (`User(id)`, `Group(id)`)
- it keeps the decision at the right abstraction boundary: fix the invariant where it is broken, not in the caller

#### 1.2 Change `initService()` to use targeted lookup
Change `providers/os/resources/services.go:initService` so that:
- `service("foo")` resolves the manager directly
- calls `Get("foo")`
- returns a concrete `service` resource from that result
- only falls back to a synthetic “not installed” service when the manager definitively reports absence

Do not call `services.refreshCache(nil)` from `initService()` anymore.
This targeted path also removes the secondary O(N) `refreshCache()` map rebuild that currently happens on every `service("name")` initialization.

Expected result:
- named service checks stop forcing full inventory hydration

#### 1.3 Rework the systemd implementation around `systemctl show`
In `providers/os/resources/services/systemd.go`:

For `Get(name)`:
- use one targeted `systemctl show <unit> --property=Id,LoadState,ActiveState,UnitFileState,Description --value` or equivalent parseable form
- normalize service names by suffixing `.service` only where needed
- derive:
  - `Installed` from `LoadState != not-found`
  - `Running` from `ActiveState == active`
  - `Enabled`, `Masked`, `Static` from `UnitFileState`
  - `Description` from `Description`

For `List()`:
- keep `systemctl list-unit-files --type service --all` as the inventory source
- replace per-unit `is-active` + `status` calls with batched `systemctl show` calls over chunks of units
- parse `Id`, `ActiveState`, `UnitFileState`, `Description`, optionally `LoadState`
- merge the batch properties back into the inventory list

Why chunking matters:
- `systemctl show` supports multiple units, but we should not assume unbounded argv size
- chunk at a fixed safe size, e.g. 100–200 units per invocation

#### 1.4 Keep `services.list` semantics stable
The plan is not to make `services.list` lazy or partial.

It should still return the full inventory of services. The improvement is to compress the remote collection shape from:
- `1 + 2N` commands

to approximately:
- `1 + ceil(N/chunk)` commands

#### 1.5 Add a dedicated parser for `systemctl show`
Do not keep parsing `status` output for the steady-state implementation.

Add dedicated parsing helpers in `systemd.go`, for example:
- `ParseServiceSystemDShow(io.Reader) (map[string]*Service, error)`
- or parser helpers over text blocks keyed by `Id=`
- explicitly handle blank-line-delimited records from multi-unit `systemctl show` output

Keep `SystemDExtractDescription()` only as compatibility glue if another manager still needs it. Otherwise remove it once obsolete.

### Affected files
Primary:
- `providers/os/resources/services/manager.go`
- `providers/os/resources/services.go`
- `providers/os/resources/services/systemd.go`
- `providers/os/resources/services/systemd_test.go`
- `providers/os/resources/services_test.go`

Potential follow-on:
- other managers in `providers/os/resources/services/*.go` to implement `Get(name)` efficiently, or use a default list-based fallback that reuses the existing `FindService()` helper where native lookup is not worth adding

### Risks and design constraints
- `UnitFileState` values are richer than boolean enabled/disabled; map them deliberately
- `systemctl show` output must be parsed as machine-readable key/value, not by string slicing assumptions copied from `status`
- avoid changing `services.list` output shape as part of this refactor
- do not introduce compatibility shims that keep the list-only design reachable

### Verification for step 1
Add or update tests to prove:
- `service('dbus')` no longer requires loading all services
- `List()` still returns the same service attributes for existing fixtures
- `Get(name)` handles:
  - enabled service
  - static service
  - masked service
  - not-found service
  - template-ish names and suffix normalization
- parser tests cover multi-unit `show` output

Recommended tests:
- unit tests in `providers/os/resources/services/systemd_test.go`
- integration-style resource test in `providers/os/resources/services_test.go`
- if needed, extend the mock command fixture to include `systemctl show` outputs and assert the minimum command set required for `service('name')`

Success criteria:
- named service lookup no longer performs service inventory hydration
- full list collection no longer emits two commands per service

---

## 2. SSH file/stat de-duplication for shell-backed access paths

### Problem
With sudo enabled, SSH file access becomes shell-backed. The current design pays repeated stat-detection and metadata costs for file fields that could share the same truth.

The most obvious waste today is repeated `uname -s` from a freshly constructed stat helper, plus repeated `test -e` / `stat -L` work for the same path.

### Recommended design

#### 2.1 Make stat helper lifetime match the SSH filesystem lifetime
Current behavior in `providers/os/connection/ssh/cat/cat.go`:
- `Open()` creates `statutil.New(...)`
- `Stat()` creates `statutil.New(...)`

Change the `cat.Fs` implementation to hold a single reusable stat helper:
- construct it once in `cat.New(...)`
- reuse it in `Open()` and `Stat()`

Expected result:
- `uname -s` becomes once per cat-FS instance, not once per stat call

This is low-risk and should be done even if nothing else in step 2 lands. `statHelper` already caches parser detection per instance; the fix is to make the helper lifetime match the filesystem lifetime.

#### 2.2 Coalesce `exists()` and `stat()` into one metadata load
Current `mqlFile` behavior already coalesces `permissions`, `size`, `user`, and `group` once `stat()` runs.
The remaining avoidable duplication is the `exists()` vs `stat()` boundary:
- `exists()` goes through `afero.Exists()` and shell-backed stat work
- `stat()` goes through `conn.FileInfo()` and shell-backed stat work again

Refactor `providers/os/resources/file.go` and generated access patterns so one metadata load can populate:
- `Exists`
- `Permissions`
- `Size`
- `User`
- `Group`

Design shape:
- introduce one internal helper such as `loadMetadata(path string) (bool, error)` or equivalent
- have it populate `Exists` plus all stat-derived fields in one place
- let `GetExists()` and stat-backed getters share that load when available
- let `GetContent()` avoid a second “does it exist?” probe when metadata already established the answer

Important constraint:
- keep resource semantics truthful; a stat miss must not be converted into a fake empty file

#### 2.3 Fix the PAM helper path to reuse file truth
Current `getSortedPathFiles()` does:
- `GetExists()`
- then `GetPermissions()`

After 2.2, this helper should benefit automatically from shared stat state for the same `file` resource instance.

Do not special-case PAM first. Fix the underlying file invariant.

#### 2.4 Do not replace shell `find` in this step
`files.find` over SSH currently falls back to shell `find -L ...` because SSH does not advertise `Capability_FindFile`.

This plan does **not** recommend replacing that immediately.

Reason:
- for directory enumeration like `/etc/pam.d`, a single remote `find` can still be cheaper than emulating a tree walk with repeated `ls/stat` over the shell backend
- the current biggest waste is repeated stat and file metadata work, not the presence of `find` itself

#### 2.5 Keep “sudo means shell backend” unchanged in step 2
Do not redesign the whole SSH filesystem stack in the same change.

Longer-term, a layered filesystem that prefers SFTP when permissions allow and falls back to privileged reads on access denial may be valuable, but that is a separate design decision with broader risk.

For this plan, focus on making the current shell-backed path much less chatty.

### Affected files
Primary:
- `providers/os/connection/ssh/cat/cat.go`
- `providers/os/connection/local/statutil/stat.go`
- `providers/os/resources/file.go`
- generated getters in `providers/os/resources/os.lr.go`
- `providers/os/resources/utils.go`
- `providers/os/resources/pam.go`

Tests:
- `providers/os/connection/ssh/cat/cat_test.go`
- `providers/os/resources/pam_test.go`
- add targeted tests for `file` resource stat reuse if none exist yet

### Risks and design constraints
- do not invent a general command cache here
- keep stat-derived truth authoritative for one resource instance only
- be careful with not-found versus permission-denied behavior
- avoid broad formatting/regeneration churn in generated resource code unless necessary

### Verification for step 2
Add or update tests to prove:
- repeated `Stat()` / `Open()` over `cat.Fs` does not re-run platform detection each time
- `file.exists` and a stat-derived field can be satisfied from one metadata load
- additional stat-derived fields remain cached after that first load
- `pam.conf` queries continue to behave the same semantically

Recommended additional verification:
- add a command-counting mock wrapper around `RunCommand` in `cat_test.go`
- assert upper bounds for repeated file-field access on the same path

Success criteria:
- obvious duplicate `uname -s` calls disappear
- repeated `exists` + stat-derived file work on the same path stops redoing shell metadata commands

---

## 3. Scan-scope reuse for stable resource fields, plus observability to prove the win

### Problem
Even after steps 1 and 2, policy execution still crosses many LLx executor boundaries. Current caching is strong inside a resource/runtime, but not intentionally scan-global.

At the same time, broad memoization of raw shell commands would be unsafe:
- commands may be time-sensitive
- commands may encode side effects
- the `command` resource is an explicit part of MQL semantics

So the right abstraction is resource/field reuse, not command-string reuse.

### Recommended design

#### 3.1 Reuse the existing recording infrastructure instead of inventing a new cache
Verified existing hooks:
- `providers/runtime.go:EnableResourcesRecording`
- `apps/cnspec/cmd/scan.go` already enables resource recording for `mql.StoreResourcesData`
- `apps/cnspec/cmd/scan.go:RunScan` passes `scan.WithRecording(config.runtime.Recording())`
- `providers/runtime.go` consults recording before provider `GetData`

Design:
- this phase is coordinated cross-repo work: recording hooks live in `mql`, but scan activation and rollout live in sibling `cnspec`
- introduce an ephemeral, in-memory scan recording mode for local scans
- use the existing recording object with `DoRecord: true, DoNotSave: true`
- keep upload/persistence behavior behind the existing feature gates

This gives us scan-scope reuse with existing semantics and much lower implementation risk than a bespoke cache.

#### 3.2 Bound the semantics: stable-resource reuse first
Do **not** turn this on for every resource type blindly.

Recommended first allowlist:
- `service`
- `services`
- `file`
- `users`
- `groups`
- `pam.conf`
- `files.find`

Recommended initial exclusions:
- `command`
- `powershell`
- obviously volatile process/network/time resources

If the current recording hook is too coarse-grained, add a lightweight opt-in/opt-out filter at the provider runtime boundary rather than falling back to raw shell memoization.

#### 3.3 Add scan observability before changing defaults
Before enabling scan-wide reuse broadly, add counters/logging for:
- recording hit / miss by resource type
- total SSH commands executed by provider/connection
- top repeated command patterns
- scan duration deltas for representative policies

Practical implementation options:
- increment counters in `providers/os/connection/ssh/ssh.go:runRawCommand`
- increment counters in recording lookup/add paths in `providers/runtime.go`
- emit a debug summary at end-of-scan under debug logging or a dedicated feature flag

#### 3.4 Make this step explicitly gated at first
Roll out as:
- off by default initially
- enabled by a feature flag or scan option for targeted benchmarking
- enable by default only after command-count and scan-duration evidence

This keeps the semantics defensible and measurable.

### Affected files
This repo (`mql`):
- `providers/runtime.go`
- `providers-sdk/v1/recording/*`
- `providers/os/connection/ssh/ssh.go`

Sibling repo (`cnspec`):
- `apps/cnspec/cmd/scan.go`
- `policy/scan/local_scanner.go`

Potential additions:
- a small metrics/reporting helper for scan command counts
- targeted tests around recording hit behavior for stable resources

### Risks and design constraints
- do not cache arbitrary command resources by default
- do not hide mutable-state bugs behind a global scan cache
- keep the default behavior conservative until command-count measurements prove value
- ensure incognito/local-only scans can still use ephemeral recording without any upload side effects

### Verification for step 3
Add or update tests to prove:
- repeated stable resource-field reads can hit recording instead of provider `GetData`
- excluded volatile resources are not silently cached if we add filtering
- enabling ephemeral recording does not write or upload anything when persistence is disabled

Recommended benchmark / measurement work:
- add a small benchmark or debug harness for a representative Linux policy set heavy on `service(...)` and PAM/file checks
- capture before/after:
  - total SSH commands
  - total `systemctl` commands
  - total `stat`/`cat` commands
  - wall-clock scan time

Success criteria:
- repeated stable resource-field fetches are reused across query executors within one scan
- the improvement is visible in command counts, not just theory

---

## What we should not do

### Do not memoize raw `RunCommand("...")` globally
This is the tempting shortcut, but it is the wrong abstraction.

Problems:
- command outputs may be intentionally volatile
- some commands can have side effects
- the cache key would be shell text, not domain meaning
- correctness would become opaque to callers and maintainers

If we need reuse, it should be attached to resource/field semantics that already define what the system means.

## Recommended implementation order

### Phase A
Implement step 1 first.

Reason:
- it addresses the biggest avoidable multiplier immediately
- it improves both correctness of abstraction and command count
- it is conceptually isolated and easy to verify

### Phase B
Implement step 2 second.

Reason:
- it cleans up the worst repeated SSH file/stat noise
- it is lower-risk than scan-global behavior changes
- it benefits PAM and many other file-heavy policies automatically

### Phase C
Implement step 3 last, gated and measured.

Reason:
- it is the broadest behavior change
- it should build on the more truthful resource boundaries from phases A and B
- it requires coordinated changes across `mql` and sibling `cnspec`
- it needs instrumentation to justify default enablement

## Definition of done
The plan is complete when all three conditions are true:

1. `service("name")` is truly targeted and no longer hydrates the entire service inventory.
2. Repeated `exists` + stat-derived file work over SSH is coalesced enough that the same path does not trigger obvious redundant metadata commands.
3. Scan-scope reuse is measured and bounded by resource semantics, with proof in command-count deltas rather than assumptions.
