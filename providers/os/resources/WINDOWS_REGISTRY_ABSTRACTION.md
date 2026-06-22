<!--
Copyright Mondoo, Inc. 2024, 2026
SPDX-License-Identifier: BUSL-1.1
-->

# When does a Windows check earn a typed resource?

A decision guide for authors (human and AI) deciding whether a Windows
hardening check should be expressed as a raw `registrykey` query or promoted
into a typed resource such as `windows.smb.serverConfiguration`.

## TL;DR

> **Default to a raw `registrykey` query. Promote a check to a typed resource
> only when answering it *correctly* requires logic that a single-path,
> single-value registry read cannot express.** Readability alone is not a
> reason — and when you do build the resource, name the exact registry
> keys/values it maps in its doc-comments so a search by registry path finds
> it.

If you only remember one line: **the resource has to buy correctness, not
just prettier prose.**

## The debate this resolves

We are layering typed resources over Windows policy data. The pull comes from
two directions:

- **Readability.** `windows.smb.serverConfiguration.requireSecuritySignature`
  reads better than a registry path with a DWORD name.
- **Cost & discoverability.** Every resource is a generated Go struct, a
  `.lr` block, one `.lr.versions` entry per field, serialization, and test
  surface. Worse, an AI writing a policy *doesn't know the resource exists* —
  left alone it emits the `registrykey` query straight from the audit text;
  forced onto the schema it burns several iterations locating the resource
  (often reading provider Go to confirm the key) only to land on the same
  assertion at higher token cost.

The trap is a resource that exists **only** to rename a registry value. Take:

> *Ensure 'Server (LanmanServer)' is set to 'Disabled'* →
> `HKLM\SYSTEM\CurrentControlSet\Services\LanmanServer:Start == 4`

```mql
registrykey.property(
  path: 'HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\LanmanServer',
  name: 'Start'
).data == 4
```

That value lives in exactly one place, has no Group Policy override, and means
exactly what it says. Wrapping it as `windows.smb.serverConfiguration.serviceStart == 4`
adds nothing but a second name you "had to know" — and `LanmanServer = the SMB
server service` is the *same* domain knowledge the raw key needed. The
abstraction moved the obscurity, it didn't remove it.

## The gate: does this check need a typed resource?

Run this on every Windows check **before** you reach for a new resource or
field. It mirrors the [Step 1.5 typed-reference gate](../../../CLAUDE.md) — a
manual judgment call, not a linter.

**Start with raw `registrykey`.** Stay there if the audit text resolves to a
single value at a single known path whose literal contents are the answer on
every Windows version you support. Most "set this DWORD to N" checks are here.

**Promote to a typed resource when *any* of these is true** — because each is
logic that a single `registrykey.property(...).data == X` cannot express, and
that every policy author would otherwise have to re-implement (usually
*wrongly*):

| Signal | Why a raw query is wrong/insufficient | Example |
|---|---|---|
| **Precedence across sources** | The effective value depends on which of several keys is set. A query that checks one path silently returns the wrong answer when the setting is applied elsewhere. | GPO `…\Policies\Microsoft\Windows\LanmanServer` wins over service `…\Services\LanmanServer\Parameters`. `windows.smb.serverConfiguration` resolves policy → parameters. |
| **Default semantics** | "Unset" implies a non-trivial effective value, or you must tell an explicit `0` from absent. A raw read of a missing value just errors or is null. | `enablePlainTextPassword` — absent means OS default; CIS wants an explicit/effective disabled. Nullable `bool`/`int` encode this. |
| **Sentinel / enum / unit / bitmask decoding** | The raw DWORD isn't the answer; it needs decoding. | eventlog `0xFFFFFFFF` → "never overwrite"; `maxSizeKB` where policy stores KB but the service stores bytes; security-layer `0/1/2` → meaning. |
| **Aggregation of multiple keys** | One logical verdict is a function of several values (or of child keys you must enumerate). | A posture that is "compliant" only if three related DWORDs across two keys all hold. |
| **Source isn't the registry at all** | `registrykey` literally cannot reach it. These *must* be resources. | `windows.firewall` (`Get-NetFirewallProfile`), `windows.defender` (`Get-MpComputerStatus`), `windows.tpm` (`Get-Tpm`), `windows.auditPolicy` (`auditpol`), secpol. |
| **The value is a typed reference** | A raw ID/SID/path string should resolve to another modeled resource. | A setting that names a service, user, or file → return the typed resource, per the [typed-reference gate](../../../CLAUDE.md). |

If **none** apply, do **not** create the resource or field. The raw query is
shorter, the AI already knows how to write it, and it costs the schema nothing.

### Cohesion exception

The gate decides whether to *create a resource*, not whether every field on it
independently passes. Once a resource exists for a good reason (e.g.
`windows.smb.serverConfiguration` earns its place via signing-precedence
fields), adding an adjacent trivial single-source value to it — like
`serviceStart` — is cheap and improves cohesion. The mistake is standing up a
*whole resource* just to host one trivial value.

## The discoverability rule (mandatory for registry-backed resources)

A resource that passes the gate is only worth it if authors can *find* it.
The original complaint — "the AI doesn't know it exists" — is real, and the
fix is a convention, not an exhortation:

> **Every registry-backed field must name, in its doc-comment, the exact
> registry value(s) it maps.** Then a search of the schema for `LanmanServer`,
> `RequireSecuritySignature`, or `Start` surfaces the resource that covers it.

The SMB resources already do this — *"Maps the RequireSecuritySignature
DWORD; 1 = required. Null when unset."* That single sentence converts "you had
to know" into "you can grep," and it documents the precedence logic that
justified the resource in the first place. Treat it as required, not optional.

A practical corollary for **policy authors / AI**: when the audit text gives
you a registry path, grep the schema for the key name or the DWORD before
writing a raw query. If a resource maps it, use the resource (you get the
precedence/decoding for free). If nothing maps it, the raw `registrykey` query
is the correct, expected answer — don't go hunting for an abstraction that
isn't there.

## "Is a fully dynamic / automatic decider possible?"

Short answer: **no — and you shouldn't try to build one.** The deciding factor
("does correctly answering this need precedence/default/decoding logic?")
depends on Windows semantics and the supported-version matrix, which live in
Microsoft's docs and in maintainers' heads, not in anything machine-readable.
A build-time linter can't infer it.

What *is* buildable, in increasing order of effort, and what we recommend:

1. **The documented gate above (do this).** A deterministic checklist a human
   or AI applies per check — exactly like the existing typed-reference gate.
   Zero infrastructure, immediate payoff. This document *is* that deliverable.

2. **Machine-readable path → resource index (high payoff, low effort).**
   Because rule #1 of the discoverability section forces every registry-backed
   field to name its key, we can mechanically extract a
   `registry-path → resource.field` map from the doc-comments (or a small
   `@maps("HKLM\…:Value")` annotation if we want it structured). That index:
   - lets an author/AI answer "is this path already modeled?" in one lookup
     instead of 3–4 search iterations, killing the token cost Dimitar flagged;
   - powers a *lint* that warns when a policy uses raw `registrykey` for a path
     a resource already covers (with precedence the raw query is getting
     wrong), or flags a new resource field duplicating an existing one.

   This is the "dynamic system" worth investing in — not an auto-decider, but
   an auto-*discoverer* that removes the only real downside of building good
   resources.

3. **Auto-generating raw queries from audit text (skip).** For trivial
   single-value checks the AI already does this reliably from the CIS audit
   section; codifying it as tooling adds maintenance for little gain.

**Recommendation:** adopt the gate (1) now, and treat the path-index (2) as the
follow-up that makes the whole strategy self-correcting. Resist (3) and resist
any "every setting gets a resource" reflex — the schema cost and discoverability
tax are real, and the gate is what keeps them in check.

## Cheat sheet

| Situation | Do |
|---|---|
| Single DWORD, one path, literal value is the answer | Raw `registrykey.property(...).data == X` |
| GPO can override a service-key value | Resource that resolves precedence |
| Unset ≠ disabled; must distinguish explicit `0` | Resource with nullable field |
| Value needs decoding (sentinel/enum/unit/bitmask) | Resource that decodes it |
| Verdict spans multiple keys/children | Resource that aggregates |
| Data comes from a cmdlet / COM / auditpol / secpol | Resource (registry can't reach it) |
| Adjacent trivial value on an already-justified resource | Add the field (cohesion) |
| A whole new resource for one trivial value | Don't — use raw `registrykey` |
