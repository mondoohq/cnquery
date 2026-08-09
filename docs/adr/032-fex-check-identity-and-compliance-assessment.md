# ADR 032: FEX check identity and assessment for compliance-framework mapping

## Status

Proposed

## Context

Findings we generate should map to the compliance controls they bear on (GDPR,
SOC 2, PCI DSS, CIS, …). This is not specific to any one producer: it applies to
every finding that reaches Mondoo Platform through FEX — findings imported from
third-party scanners (SARIF, Burp, DefectDojo, JUnit, ZAP), findings from query
packs that read external sources (AWS Security Hub, Qualys, nuclei), and findings
from first-party detectors. A finding whose control mapping is legible to the
people who commission compliance work is far more useful than one that only names
a CWE; today none of these FEX findings participate in the compliance graph at
all.

The requirement is that this mapping be expressed against **cnspec's existing
compliance-framework model**, not a per-producer tagging scheme, so a control's
status is computed the same way regardless of which producer supplied the
evidence.

### How findings and compliance are modeled today

- Producers upload findings to Mondoo Platform as `fex.FindingExchange`
  (`providers-sdk/v1/upstream/fex/fex.proto`). A finding carries `id`, `ref`
  (the producer-local check/rule id), `mrn` (assigned by the server), `summary`,
  `details` (`severity`, `confidence`, `references` for CWE/OWASP, a free-form
  `properties` map), `affects`, `evidences`, and `remediations`. The SARIF/Burp/
  DefectDojo/JUnit/ZAP importers (`cnspec/upload/report_conversion/`) and every
  query pack that emits FEX produce this same shape.

- cnspec's compliance model (`cnspec` `policy/cnspec_policy.proto`) is a
  **scoring graph**: a `Framework` declares `controls` (uid + title); a
  separate `FrameworkMap` (with a `framework_owner`) binds each control by uid
  to the `checks` / `queries` / `policies` / `controls` that satisfy it
  (`ControlMap` / `ControlRef`). At resolution the engine flattens this into
  `check -> control -> framework`, producing a `ControlScore` with
  `compliant`, `failed`, `total`, and `completion`. Authors express the binding
  ergonomically as `compliance/<framework-uid>: <control-uid>` tags on the
  check; enterprise tooling turns those tags into `FrameworkMap` objects.

### Two gaps sit between these systems

1. **FEX has no check identity.** A `FindingExchange` names its *source*
   (`source.name`) and a producer-local `ref`, but nothing that resolves to a
   **check** node in the compliance graph. `FrameworkMap`/`ControlRef` can only
   point at checks/queries/policies by uid/mrn, so there is no handle for a
   framework map to attribute a finding to a control. The third-party import path
   (`cnspec upload --format sarif`, and the other converters) confirms this:
   imported findings land as platform findings and are **not wired into the
   compliance-control graph** at all.

2. **FEX is failures-only, so a control has no denominator.** Most finding
   producers are *presence-only*: a scanner emits a finding when something is
   wrong and stays silent otherwise. It never asserts "this check passed." A
   control fed only by findings can compute `failed` but not `compliant` or
   `total`, and cannot distinguish **compliant** (the check ran, nothing found)
   from **unassessed** (no check covers the control, or that asset was not
   scanned). Collapsing "clean" and "unassessed" into the same silence reports
   false compliance — which is why cnspec's scoring model is deliberately
   two-sided (MQL checks pass or fail; presence-only findings need the equivalent
   signal to join the graph honestly).

### What the FEX contract already says about compliance

`fex.proto`'s field and enum names mirror a **server-internal etl schema
byte-for-byte** (see the file header and ADR 029), so additions are wire-safe
but only become *useful* once the server etl schema learns them — additions are
therefore gated on server-side coordination. ADR 029's prior-art review is also
explicit that compliance is intentionally kept **out** of the finding-detail
types: SCC's `Compliance` detail type has no FEX analogue because compliance is
"handled elsewhere in this contract — frameworks for compliance." A naive
`Compliance { framework, control }` label on `FindingDetail` would cut against
that stance and, per gap 2, would only ever be an unscoreable label.

## Decision

Make the **check that produced a finding** a first-class node in the
compliance-framework graph, and extend FEX with the minimum needed for that — a
**check reference** on findings and a **check-assessment** result stream for the
denominator — rather than adding a compliance-control field to findings.
Compliance stays where the contract puts it (the framework graph); FEX gains only
the identity and the verdict signal that let findings participate in it. This is
producer-agnostic: it works identically for imported third-party findings, query-
pack findings, and first-party detectors.

Four coordinated pieces:

### 1. `CheckRef` on `FindingExchange` (FEX / mql)

Add a typed reference identifying the check that produced the finding, so the
framework graph can attach it exactly as it attaches an MQL check. New message +
one additive field on `FindingExchange` (field numbers continue the message's
existing sequence; `evidences=20`, `remediations=21`, so `check` takes the next
free number, **22**):

```proto
// CheckRef identifies the check that produced a finding, so the finding can be
// attributed to a control in the compliance-framework graph. The check is the
// producer's unit of detection - an MQL query, a code-scan rule, or an imported
// scanner's rule id; `uid` is that id, `mrn` is assigned on upload, `title` is
// its human-readable name.
message CheckRef {
  // Optional. Mondoo Resource Name of the check. Read-only; assigned by Mondoo.
  string mrn = 1;
  // Required. Producer-local unique id of the check (e.g. the rule/query id).
  string uid = 2;
  // Optional. Human-readable check name.
  string title = 3;
}

message FindingExchange {
  // ... existing fields 1-21 ...

  // Optional. The check (MQL query, scan rule, imported scanner rule) that
  // produced this finding. Lets the compliance-framework graph attribute the
  // finding to a control.
  CheckRef check = 22;
}
```

`ref` (field 2) stays the raw producer id; `check.uid` is the same value
promoted to a first-class, resolvable check identity. Old consumers ignore the
unknown field.

### 2. `CheckAssessment` in `FindingsUploadRequest` (FEX / mql) — the denominator

Add a per-check, per-asset verdict stream to the upload envelope so a control
gets `total` and `compliant`, not just `failed`. Findings remain the FAIL
detail; a check that ran clean emits a PASS assessment with no finding.

```proto
message CheckAssessment {
  // Required. The check that was assessed.
  CheckRef check = 1;
  // Required. The component/asset the check was assessed against.
  Component assessed = 2;

  enum Result {
    RESULT_UNSPECIFIED = 0;
    RESULT_PASS = 1;   // check ran, nothing found -> control evidence: compliant
    RESULT_FAIL = 2;   // check ran, finding(s) raised -> see FindingExchange
    RESULT_ERROR = 3;  // check could not complete
    RESULT_SKIP = 4;   // check not applicable to this asset
  }
  Result result = 3;

  // Optional. Ids of the FindingExchange entries backing a FAIL result.
  repeated string finding_ids = 4;
}
```

`FindingsUploadRequest` gains `repeated CheckAssessment assessments`. This is
the load-bearing half: it is what turns presence-only findings into a
two-sided, scoreable check result. A producer that already reports pass/fail
(an MQL policy) can emit assessments directly; a presence-only producer emits a
PASS for each compliance-relevant check that ran without a finding.

### 3. `compliance/<framework-uid>: <control-uid>` on the producing check (content)

The control binding is a property of the **check**, not the finding, and requires
**no proto change**. A producer's checks carry the same
`compliance/<framework-uid>: <control-uid>` metadata cnspec MQL checks use. The
value `false` means "explicitly no fitting control," matching cnspec semantics.
These tags feed `FrameworkMap` generation identically to MQL checks — the mapping
reuses cnspec's own vocabulary verbatim, so any producer's checks map to controls
the same way native checks do.

### 4. Producers emit check identity + assessments (per producer)

Each producer's FEX upload path populates `FindingExchange.check` from its check
id/title, and emits a `CheckAssessment` per compliance-relevant check that ran
against an asset (PASS when the check found nothing, FAIL with `finding_ids` when
it did). The `compliance/*` tags travel with the check into the framework graph;
they are not copied onto every finding.

## Consequences

- **We get** honest, scoreable compliance for every producer: a control mapped to
  these checks can report `X of N compliant`, and "clean" is distinguishable from
  "unassessed." Findings attribute to framework controls through cnspec's real
  model, not a per-producer label, so the mapping is diffable and reproducible
  like the rest of the compliance graph.

- **Server coordination is required (why this is Proposed, not Accepted).** Per
  ADR 029, the etl schema must learn `check` and `CheckAssessment` before they
  surface; producers can emit them ahead of that and the server ignores them
  until it catches up. The names here must be reconciled with the etl schema's
  naming before merge (the byte-for-byte-mirror constraint).

- **Platform control-graph wiring is a dependency, not part of this ADR.** The
  platform must (a) model an uploaded check as a check node the framework graph
  can reference, and (b) consume `CheckAssessment` into `ControlScore`. Until
  then the FEX additions are inert but harmless.

- **Generated Go** (`fex.pb.go`, `fex_vtproto.pb.go`) is regenerated via the
  existing `go:generate` directive on `fex.go`; no hand-written changes.
  `protolint` must still pass under the file's existing disables.

- **`ref` vs `check.uid` overlap** is intentional and documented: `ref` stays
  the untyped producer id for backward compatibility; `check` is the typed,
  graph-resolvable identity.

## Alternatives considered

- **`Compliance { framework, control }` field on `FindingDetail`.** Rejected.
  It contradicts the contract's "frameworks for compliance" stance (ADR 029
  prior art), and — because FEX is failures-only — it yields a control label
  that can never be scored (no denominator; cannot tell clean from unassessed).
  It would look like compliance while being decoration, and FEX is a public
  wire contract expensive to change twice.

- **Reference producer checks from a `FrameworkMap` `ControlRef` directly, no FEX
  change.** Rejected as insufficient alone. `ControlRef` points at
  checks/queries/policies by uid; with no check identity on the finding and no
  assessment stream, the framework graph still has nothing to attach a finding
  to and no pass signal. The `compliance/*` tags (piece 3) do ride this path —
  but they need pieces 1-2 to be scoreable.

- **Model findings as `Evidence` on cnspec `manual`/`evidence` controls.** Kept
  as a complement, not the primary path. `Control.manual` / `Control.evidence`
  fit controls a check only *corroborates* rather than decides (a partial or
  advisory signal) — attach findings as evidence rather than forcing a
  PASS/FAIL. For checks that are genuinely two-sided, the assessment path
  (pieces 1-2) gives real scoring and is preferred.

- **Carry assessment counts only, no per-check identity.** Rejected: without
  `CheckRef`, assessments cannot be attributed to specific controls, and a
  finding cannot be tied back to the check that scored the control.

## Prior art

Follows ADR 029 (network detail types for FEX Evidence): additive, wire-safe
FEX fields gated on server etl alignment, field numbers continuing the existing
sequence. Mirrors cnspec's own `compliance/<framework>: <control>` check-tag
convention (`cnspec` `content/CLAUDE.md`, `content/mondoo-aws-security.mql.yaml`)
so any producer's checks map to controls the same way MQL checks do.
