# ADR 032: Mapping FEX findings to compliance framework controls

## Status

Proposed

## Context

Findings we generate should map to the compliance controls they bear on (GDPR,
SOC 2, PCI DSS, CIS, …). This is producer-agnostic: it applies to every finding
that reaches Mondoo Platform through FEX — findings imported from third-party
scanners (SARIF, Burp, DefectDojo, JUnit, ZAP), findings from query packs that
read external sources (AWS Security Hub, Qualys, nuclei), and findings from
first-party detectors.

Today none of these FEX findings participate in cnspec's compliance graph. The
third-party import path (`cnspec upload --format sarif`, and the other converters
in `cnspec/upload/report_conversion/`) makes this concrete: imported findings
land as platform findings and are never attributed to a control.

cnspec already has the model to attribute them (`cnspec` `policy/cnspec_policy.proto`):
a `Framework` declares `controls`; a `FrameworkMap` binds each control by uid to
the checks/queries/policies that satisfy it (`ControlMap`/`ControlRef`); and
authors express that binding ergonomically as `compliance/<framework-uid>:
<control-uid>` tags on a check, which enterprise tooling turns into
`FrameworkMap` objects. The gap is that this machinery is wired for MQL checks
only — not for the checks behind FEX findings.

## Decision

Attribute FEX findings to controls by treating **the check that produced the
finding** as a node in the framework graph, reusing the identity FEX already
carries and the `compliance/*` tag convention cnspec already uses. **No FEX
wire-format change.** Compliance scoring of findings is a separate, harder
problem and is explicitly deferred (see Deferred).

Three parts, none of them a proto change:

### 1. Check identity from existing fields

`FindingExchange` already carries `ref` — "a unique identifier in the original
data source," populated today with the producing check/rule id — and
`source.name`, the producer. Together `(source.name, ref)` identify the check.
The platform models that pair as a check node the framework graph can reference,
the same way it references an MQL check by uid. No new field is required; confirm
with the platform team that `(source, ref)` is a sufficient key before adding
anything.

### 2. Control binding via the `compliance/*` tag convention

The control mapping is a property of the **check**, expressed with the same
`compliance/<framework-uid>: <control-uid>` tags MQL checks use (`false` =
"explicitly no fitting control"). For a first-party check we control, the tags
live with the check definition. For a third-party import that has no cnspec check
definition, a `FrameworkMap` references the external check by its `(source, ref)`
key and binds it to controls. Either way the binding feeds `FrameworkMap`
generation exactly as it does for MQL checks — the mapping reuses cnspec's own
vocabulary, so any producer's checks map to controls the same way native checks
do.

### 3. Platform + content wiring (the actual work)

The work is not in the wire format; it is:

- **Platform:** model an uploaded finding's `(source, ref)` check as a
  control-graph node the framework graph can reference, and resolve/generate the
  `FrameworkMap` entries for these checks so the finding attributes to its
  control(s).
- **Content:** put `compliance/*` tags on first-party checks, and author the
  framework-map entries for the third-party rule ids we care about.

## Deferred: compliance scoring of findings

Attribution answers "which control does this finding relate to." It does **not**
make a control *scored* (`compliant` / `total`). Most finding producers are
presence-only: they emit a finding on failure and stay silent otherwise, so a
control fed only by findings has failures (numerator) but no denominator, and
cannot distinguish **compliant** (check ran, nothing found) from **unassessed**
(no check covers it, or the asset was not scanned).

Solving that means giving the control graph a two-sided signal (which checks ran
and passed), which is a larger change — likely a new per-check/per-asset
assessment stream on the FEX upload, and therefore a real wire-format change
gated on server etl-schema coordination. That is out of scope here and belongs in
its own ADR once mapping (this ADR) has shipped and the product actually needs
findings to drive control scores. This ADR deliberately ships attribution first
so it carries no wire-format change and no etl dependency.

## Consequences

- **No FEX wire change, no etl gate.** Mapping rides existing fields and the
  existing tag convention, so this ADR is not blocked on server-side schema
  coordination (unlike the deferred scoring work).
- **Producer-agnostic.** Third-party imports, query-pack findings, and
  first-party detectors all attribute through the same `(source, ref)` +
  `compliance/*` path.
- **Honors the FEX contract's stance.** Nothing is added to `FindingDetail`;
  compliance stays "handled elsewhere in this contract — frameworks for
  compliance" (ADR 029). No `Compliance { framework, control }` label on the
  finding.
- **Attribution only.** Controls mapped this way show related findings; they are
  not scored until the deferred scoring work lands.

## Alternatives considered

- **`Compliance { framework, control }` field on `FindingDetail`.** Rejected: it
  contradicts the "frameworks for compliance" stance (ADR 029), and a label on a
  presence-only finding can never be scored — it would look like compliance while
  being decoration, on a public wire contract expensive to change twice.

- **A new `CheckRef` message + `check` field on `FindingExchange`.** Not needed
  for mapping: `ref` + `source.name` already identify the check. A dedicated,
  human-readable `check_title` field may be worth adding later for rendering, but
  it is a trivial additive field and not required to attribute a finding to a
  control — so it is deliberately left out of this ADR to keep it wire-neutral.

- **Bundling a `CheckAssessment` scoring stream into this ADR.** Rejected as
  scope creep and the main source of complexity: scoring is separable from
  attribution, needs a wire-format change and etl coordination, and is only
  strictly novel for pure third-party imports. Deferred to its own ADR.

## Prior art

Builds on cnspec's `compliance/<framework>: <control>` check-tag convention
(`cnspec` `content/CLAUDE.md`, `content/mondoo-aws-security.mql.yaml`) and its
`Framework`/`FrameworkMap`/`ControlMap` model (`policy/cnspec_policy.proto`), so
any producer's checks map to controls the same way MQL checks do. Consistent with
ADR 029's treatment of compliance as a framework concern rather than a
finding-detail type.
