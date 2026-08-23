# ADR 043: MQL strict mode — explicit optionality in access chains

## Status

Proposed

## Context

MQL absorbs nulls. `sshd.config.params.DOES_NOT_EXIST.lines.map(downcase)`
evaluates to null: the missing key produces null, and every subsequent link
faithfully passes that null along to the end. The value is not wrong, but the
result carries no indication that `DOES_NOT_EXIST` was the link that broke, and a
chain that resolved cleanly to null looks identical to one that never resolved at
all.

That is convenient in an exploratory shell and dangerous in a policy. A security
check is a claim about a system; "I could not read the thing I was claiming about"
and "the thing is configured correctly" must not produce the same result, and
today they frequently do.

### What happens today: nulls propagate, silently

Null propagates all the way down a chain, and so does an error. That is the part
worth being precise about, because the two are not symmetric in the way that
matters.

Take `sshd.config.params.DOES_NOT_EXIST.lines.map(downcase)`. The missing key
reads as null (`llx/builtin_map.go:32-35`), `.lines` on a null string yields null,
`.map(downcase)` on a null array yields null, and the query returns null. Nothing
along the way objects, and nothing in the result identifies `DOES_NOT_EXIST` as
the link that broke. The same holds for resource fields
(`llx/builtin.go:825-830`, which stores `NilData` and returns no error), dict
access (`llx/builtin_map.go:374-379`), and array index (`llx/builtin_array.go:61-64`).

An error propagates too, but it propagates *as itself*: `runFunction` re-types the
error to the chunk's output type, caches it, and returns it
(`llx/llx.go:792-810`), and `triggerChainError` walks the rest of the chain firing
callbacks (`llx/llx.go:974-1009`). The user sees which step failed.

So the asymmetry is not propagation. It is **attribution**: a propagated error
names its origin, a propagated null does not. And providers convert the first
into the second at scale. The AWS provider calls `Is400AccessDeniedError` in 771
places, each turning a permission failure into a null field, and there are ~3,500
explicit `StateIsNull` sites across all providers. A scan run without the
permissions to read something produces the same null as a scan run against a
system where the thing is genuinely absent, and a policy cannot tell them apart.

The second problem is that the treatment of a null receiver is decided per
builtin. There are 104 `bind.Value == nil` branches across `llx/`, each written
for its own call site, and they do not agree: `dataOpV2` propagates null
(`llx/builtin_simple.go:162-167`), `nonNilDataOpV2` returns `false` on the
reasoning that "a missing config param is not <= 4"
(`llx/builtin_simple.go:172-180`), and `arrayAllV2` returns a null bool
(`llx/builtin_array.go:341-350`). Each choice is defensible where it sits. There
is no rule above them that an author can rely on.

### `?` already means this, and we already shipped it once

This is not a new idea being introduced from scratch. [#6633][pr6633] (Feb 2026,
"optional chaining: a.b.c vs a?.b?.c") built it: `?` marks a link optional, an
unmarked link is required, and dereferencing null without the mark is an error.
That PR shipped the whole front half and it is all still in the tree:

- **Syntax and AST.** `parseOperand` tracks `isConditional` across the `?`/`.`
  loop and stamps it onto the following call (`mqlc/parser/parser.go:526-563`);
  `Call.IsConditional` is part of the AST (`mqlc/parser/parser.go:137`). `a?.b`,
  `a['b']?.c`, and `a.b ?. c` all parse today.
- **Bytecode.** The compiler emits a distinct function id, `[]?` instead of `[]`,
  for a marked access (`mqlc/mqlc.go:1298-1302`, `mqlc/mqlc.go:1578-1585`), so the
  author's intent survives compilation and lands in the chunk checksum.
- **Runtime, and even the diagnostics.** `_dictGetIndex` / `_mapGetIndex` took an
  `isConditional` argument and raised `cannot access field <k>, parent element is
  null` when it was false.

What [#6633][pr6633] did **not** cover is resource field access. It only ever
touched `llx/builtin_map.go`; `runResourceFunction` was never part of it, so
`?` has always been a dict/map-only feature.

Then [#7079][pr7079] (Mar 2026, "propagate null through dict and map bracket
access") took the runtime half back out. It made `[]` behave like `[]?`, deleted
the `isConditional` parameter as dead, and collapsed both ids onto one handler
(`llx/builtin.go:547-548,732-733`). The null-propagation comments now sitting in
`dictGetIndex` and `mapGetIndex` are that revert's rationale. Its stated reason:
`dict["domain"]["key"]` was erroring with `cannot access field key, parent element
is null` whenever `domain` was absent, and that broke real queries.

**[#7079][pr7079] deferred the behavior; it did not abandon it.** The semantics
were never the problem. The problem was ordering: strictness was live before `?`
was established in the field, so content had no backwards-compatible way to say
"this link is legitimately optional" that older clients would also accept.
Deferring the runtime half let the operator settle first, on the reasoning that
once `?` is understood everywhere, content can adopt it freely and strict mode
can be turned on without stranding anyone.

That is the position we are now in. `?` has parsed and compiled to a distinct
chunk id since Feb 2026, so the operator is established and the deferred half is
what this ADR resumes. The remaining pieces are narrow: a mode to switch on,
resource field access wired to it, and an opt-in that lets each piece of content
choose.

[pr6633]: https://github.com/mondoohq/cnquery/pull/6633
[pr7079]: https://github.com/mondoohq/cnquery/pull/7079
## Decision

Introduce **strict mode**: a compile-time mode in which *producing* null is legal
and *dereferencing* null is an error, and in which `?` is how an author declares a
dereference optional.

### 1. The rule

> **Every link in a chain must resolve. `?` after a link waives that link.**

A link **fails to resolve** when:

- its receiver is null — whatever made it null, and
- for a map or dict, the key it names is absent.

A link **resolves to null** when the lookup succeeded but the value is null. That
is a legal outcome, and it is the whole of the field/map distinction:

- **A field always resolves.** A field is declared, so asking for it is never a
  claim that might be false; the type says what comes back, and if that is null
  the field resolved to null. Reading it is fine, reading *through* it is not.
- **A map key lookup may fail to resolve.** Naming a key is a claim that the key
  exists, and only the author knows whether they meant it. In strict mode an
  absent key is an error **at the lookup**; in non-strict mode it is null.

That asymmetry is the point, and it is what makes `params.PermitRootLogn == "no"`
behave correctly: the typo is a claim about a key that is not there, so the lookup
errors and the comparison never runs. A genuinely null field compared the same way
still yields `false`, because a null field is a value the provider actually
reported, not a mistake in the query.

**`?` sits after the link it guards, and pulls double duty.** It waives both
failure modes of that link *and* short-circuits the chain if the link yields null,
so the author never has to know which of the two they are guarding against:

| Situation | Non-strict | Strict | Guard |
|---|---|---|---|
| `m.k`, key absent (terminal) | null | error at `.k` | `m.k?` |
| `m.k == "no"`, key absent | false | error at `.k` | `m.k? == "no"` |
| `m.k.c`, key absent | null | error at `.k` | `m.k?.c` |
| `m.k.c`, key present, value null | null | error at `.c` | `m.k?.c` |
| `a.f`, field null (terminal) | null | null | — |
| `a.f == "no"`, field null | false | false | — |
| `a.f.c`, field null | null | error at `.c` | `a.f?.c` |
| `a.f.c`, `a` null | null | error at `.f` | `a?.f.c` |

Rows three and four are the double duty: one guard, two causes, same position. The
mark always attaches to the link on its left — `a?.f` guards `a`, `m.k?` guards the
`k` lookup — which is also how `a?.b` already reads in JavaScript. Guarding two
links takes two marks (`a?.f?.c`), and when a guard fires it abandons the rest of
the chain, so `a?.f.c` yields null rather than continuing on to fail at `.c`.

Note `m.k? == "no"` in row two: a trailing `?` with no following link is a valid
and necessary spelling, and it is the escape hatch for a key the author knows may
be absent. The parser does not support it today — `parseOperand` consumes the `?`
and then drops it when the next token is not `.` (`mqlc/parser/parser.go:528-536`),
and `parseOperation` swallows a `?` in operator position outright (`:730-732`).

**Relationship to [#6633][pr6633].** That PR treated an absent key as producing
null and raised the error one link later, at the dereference
(`cannot access field "b", parent element is null`). This ADR moves the error to
the lookup, which is what catches the terminal typo — [#6633][pr6633]'s placement
cannot, because a terminal lookup has no following link to raise it. Guard
placement is unchanged from [#6633][pr6633]: `params.d?.b` still works, and now
`params.d?` does too.

### 2. What counts as a dereference

The rule applies to every operation that reads through a receiver, not just field
and key access:

- **Resource field access** — `x.field`, compiled at `mqlc/mqlc.go:1084-1095`, run
  by `runResourceFunction`.
- **Block binding** — `x { … }`. A null `x` yields null without running the block
  today (`llx/llx.go:594-596`); in strict mode it errors, and `x? { … }` is the
  guard. Nothing about blocks is special here — the block-open is the dereference,
  the same way `.field` is.
- **Implicit array mapping** — `xs.field` over a null `xs`. A *non-null* list whose
  elements produce nulls is unaffected: those are produced values, not receivers.
- **Builtin methods on a null receiver** — `x.length`, `x.all(…)`, `x.contains(…)`,
  the comparison helpers. These are the 104 `bind.Value == nil` sites, and strict
  mode answers them with the rule above instead of per-site judgment.

**Arrays are left inconsistent, deliberately.** An out-of-range `xs[9]` errors at
the lookup already, in both modes (`llx/builtin_array.go:87-95`). Under §1 an
absent map key now does the same — but only in strict mode. So the two converge
under strict and stay split under non-strict, where a missing key is null and a
missing index is an error. Arrays are effectively always strict on this point, and
`?` has no defined effect on an index. Whether they should soften to match maps is
genuinely unclear (the current behavior may well be the more useful one) and is
**not being settled in this iteration**. Recorded so the inconsistency is a known
position rather than an oversight.

### 3. Out of scope for this ADR

Strict mode governs **access chains only**. It does not touch:

- **Three-valued boolean logic.** `null && null` still evaluates truthy, which is
  a genuine false-green source (already flagged in `CLAUDE.md §5`). It is a
  separate change with a much larger blast radius on existing content, and strict
  chains remove a large share of the nulls that feed it. Tracked as a follow-up.
- **Comparison against a null *field*.** `nonNilDataOpV2` returning `false` stays.
  `a.f == "no"` with a null `f` is still silently false, and under §1 that is
  correct rather than a gap: the field resolved, the provider reported null, and
  nothing in the query was wrong. The mistyped-*key* version of this,
  `params.PermitRootLogn == "no"`, is caught — see §1. Whether a null field should
  also make a comparison loud is the deferred question, and it belongs with the
  boolean-logic work above, not here.
- **Provider-side null production.** Whether `Is400AccessDeniedError` *should*
  yield null instead of an error is a provider question. Strict mode changes what
  happens downstream of that null, not the decision to emit it — and by making the
  downstream consequence loud, it puts real pressure on the 771 AWS call sites to
  be revisited. That is intended, and it is the largest piece of follow-on work.

### 4. Strictness is baked into the bytecode, per dereference

Strictness must travel with the compiled code, not with the executing client. A
`CodeBundle` is compiled once, checksummed, cached, shipped upstream, and replayed
against recordings; a mode that lived only in the runtime's feature set would let
the same bundle mean two different things on two nodes.

Add a tri-state to the `Function` message (`llx/llx.proto:36-42`), which is
already the node that names its receiver via `binding`:

```proto
message Function {
  string type = 1;
  repeated Primitive args = 3;
  uint64 binding = 4;
  // How this call treats a null binding.
  //   NULLABILITY_UNSET    - legacy/non-strict: null binding yields null
  //   NULLABILITY_REQUIRED - strict: null binding is an error
  //   NULLABILITY_OPTIONAL - author wrote `?`: null binding short-circuits
  Nullability nullability = 5;
}
```

Three consequences follow, and each is load-bearing:

- **`Function.checksumV2` must include it** (`llx/chunk.go:111-127`). It currently
  hashes only type, binding, and args — so without this, `a.b.c` and `a.b?.c`
  compile to byte-different, checksum-identical bytecode, and the two collide in
  the code cache and in any upstream store keyed on chunk checksums. This is the
  single easiest way to get this feature wrong.
- **The bytecode becomes self-describing.** There is no ambient "am I strict"
  question at runtime; each chunk states its own contract. Strictness is uniform
  within a policy (§6), so a bundle-level bit would be semantically sufficient
  today — but `?` is inherently per-chunk and has to enter the checksum regardless,
  and chunk checksums are what results are keyed on, so the mode has to reach that
  level anyway. Given that, carrying `REQUIRED` beside `OPTIONAL` is a small delta
  that removes an ambient-state class of bug and leaves the per-query door (§6)
  costless if it is ever opened.
- **`REQUIRED` must be explicit, not inferred from absence.** Because both modes
  persist (§6), "no marker" genuinely means non-strict rather than
  not-yet-migrated, so a single `optional` bit could not distinguish a strict
  dereference from a lenient one. This is what forces a tri-state; if strict-only
  were the near-term end state, one bit would do.
- **`[]?` stops being the encoding and becomes an alias.** New compilations emit
  `[]` with `nullability = OPTIONAL`. The `[]?` id stays registered
  (`llx/builtin.go:548,733`) permanently, because bundles carrying it already
  exist in the wild.

`CodeBundle` may carry an informational `strict` bit for reporters and
decompilation, excluded from all checksums. The semantic truth is per-chunk.

### 5. Old executors are the dangerous direction

An executor built before this field ignores it and runs everything leniently. A
strict bundle then returns null exactly where it was supposed to error — a silent
downgrade **in the permissive direction**, which can turn a strict check green on
an old node. Everywhere else in this design a mismatch should fail soft; here it
must not.

`?` is safe on this axis, which is the point of having waited: any client that
understands `[]?` predates the strict rollout, so optional-marked content already
runs correctly on the installed base. `REQUIRED` is the new information, and it is
the direction that fails silently.

`CodeBundle.min_mondoo_version` exists on the wire (`llx/llx.proto:117`) but
**nothing in this repository sets or reads it** — it is currently a cnspec/upstream
concern. Strict compilation is the first thing in mql that genuinely needs it. The
gate hangs off [ADR 040](040-cross-version-type-migration.md)'s min-MQL-version
axis: a bundle with any `NULLABILITY_REQUIRED` chunk declares a v14 floor, and an
executor below it refuses the bundle rather than running it leniently. Shipping
strict mode without that check is not acceptable.

### 6. The switch: per-content opt-in, config default underneath

Strict mode is **opted into by the content**, not by the client. A policy declares
whether it is strict, because a policy's outcome has to be predictable regardless
of which operator runs it and with what config. That declaration lives in the
cnspec bundle schema; mql's job is the compile-time knob it drives
(`mqlc.CompilerConfig`) and the precedence rule.

**The policy is the unit, and the only unit.** A declaration applies to the entire
policy, uniformly. Nothing below it — a query, a check, a variant — can override
it; inside a strict policy the only escape is `?` on the individual link, which is
the whole reason the operator exists. Per-query granularity is a door left open,
not a plan: it would be reached for only under real pressure, because it trades
away the property the declaration is there to provide. A reader should be able to
answer "is this policy strict?" without reading its queries.

**Query packs need no separate rule.** They live in cnspec and run on the policy
framework, so they inherit the declaration, the precedence, and the lint without a
second mechanism.

Resolution is a two-level fallback:

1. **The content's own declaration** — a policy that says strict, or explicitly
   says non-strict, gets what it asked for.
2. **The config default** — a tri-state `strict` key in `CommonOpts`
   (`cli/config/config.go:262-303`), used whenever the content carries no setting.
   It must be `*bool`, not `bool`, so "unset" stays distinguishable from "false";
   `Features []string` sitting two fields above is the wrong shape here precisely
   because a feature flag cannot express "explicitly off".

Deliberately **not** a `features.yaml` flag. That mechanism is global and
client-side: flipping it reinterprets every query the client holds, including
third-party policy content the operator did not author and cannot fix. That is the
shape of the thing [#7079][pr7079] deferred, and reaching for it again would
reintroduce the same problem under a new name. The compiler already takes its
configuration from `CompilerConfig` (`mqlc/mqlc.go:89-111`), which is per-compile,
so per-content strictness needs no new plumbing — only a field that is not
`Features`.

**v14 requires every policy to state its mode.** Not "defaults to non-strict" —
*stated*, one way or the other, enforced by a linter. An unstated policy is one
whose result depends on the operator's config file, which is exactly the
unpredictability this design exists to remove. The config default covers ad-hoc
surfaces (shell, `mql run`, `mqlx` embeddings) where there is no content to carry
a declaration; it is not a fallback that policies are allowed to rely on.

**Both modes persist for now.** Strict-only is the intended end state but is some
way off and needs considerably more verification first. Two consequences follow
directly: non-strict compilation stays a first-class path rather than a
deprecation, and the bytecode must carry `REQUIRED` explicitly rather than
inferring it from absence — see §4, where this is what forces the tri-state over
a single `optional` bit.

## Phased plan

1. **Fix the parser so every `?` position survives.** Three gaps, all of them
   spellings §1 needs:
   - `m?["k"]` and `x? { … }` — the loop resets `isConditional` on any token that
     is not `?` or `.` (`mqlc/parser/parser.go:528-531`), and the `[` and `{` cases
     never read it (`:596-618`, `:620`), so both parse and silently drop the mark.
   - `m.k?` trailing, including `m.k? == "no"` — `parseOperand` consumes the `?`
     and discards it when no `.` follows, and `parseOperation` swallows a `?` in
     operator position (`:730-732`). This is the escape hatch for a knowingly
     optional key, so it is not optional itself.
   - The mark binds to the link on its **left**, while the parser records it on the
     call to its right (`Call.IsConditional`). Either shift it at parse time or
     shift it in the compiler, but pick one and write it down.
2. **Add `Nullability` to the proto and to `Function.checksumV2`**, and thread the
   flag into `compileBoundIdentifier` (`mqlc/mqlc.go:1017`), the accessor paths, and
   the block path. Emit `OPTIONAL` where the author marked, `UNSET` everywhere else.
   No `REQUIRED` yet, so this is observably a no-op and lands the wire change early.
3. **Unify the nil handling.** The 104 `bind.Value == nil` branches each decide for
   themselves; a mode switch cannot sit on top of that. Collapse them into one
   decision point ahead of dispatch, reproducing today's behavior exactly, as a
   pure refactor with no semantic change. Prerequisite for step 4 and the largest
   piece of work in the plan.
4. **Implement `REQUIRED` in the VM.** Two checks, because §1 has two failure
   modes: a null receiver at the decision point step 3 created (plus
   `runResourceFunction`), and an absent key inside `mapGetIndex`/`dictGetIndex` —
   which is where [#7079][pr7079] removed exactly this branch, so the shape is
   already known. Non-strict compilations never emit `REQUIRED`, so nothing changes
   for existing content.
5. **Make the short-circuit real.** This is the piece with genuine implementation
   risk and it should not be waved past. Today an `OPTIONAL` link needs no
   machinery, because a null simply propagates step by step and lands as null. Once
   `REQUIRED` exists, that stops working: in `a?.f.c` a null `a` must skip `.f` and
   `.c` rather than let the `REQUIRED` `.f` error — JavaScript throws here only when
   the guard is absent, and so must we. Marking the whole tail `OPTIONAL` at compile
   time is the tempting shortcut and it is wrong: it would also swallow a null `f`
   in `a?.f.c`, which must still error. So the executor has to short-circuit for
   real, delivering null to the entrypoint instead of walking `e.calls`. Blocks and
   array mapping are where "the rest of the chain" gets hard to define; settle that
   before committing to the approach.
6. **Add the strictness knob to `CompilerConfig`** and the tri-state `strict` key
   to `CommonOpts`, with the content-then-config precedence rule. Shell, `mql run`,
   and `mqlx` pick up the config default; cnspec wires the policy declaration.
7. **Add the min-version gate.** Populate and enforce `min_mondoo_version` for
   bundles containing `REQUIRED` chunks.
8. **Lint for an explicit mode.** In v14 every policy must state strict or
   non-strict; the linter fails an unstated one. Pair it with a diagnostic pass
   that reports each unguarded dereference in a policy about to go strict, so
   authors can find the chains that need `?` before flipping.
9. **Later, and separately: streamline.** Reducing the ceremony and eventually
   retiring non-strict is explicitly future work, gated on evidence from real v14
   policy runs rather than on a release date.

## Caveat: replaying non-strict recordings

A recording stores **resource fields**, keyed by resource, id, and field name,
holding the whole `RawData` value (`providers-sdk/v1/recording/recording.go:405-443`
and `:514-529`). That shape decides how faithfully strict mode can replay, and it
treats the two causes of a null receiver (§1) differently:

- **Map keys replay correctly.** A key is not a recorded unit; the map or dict is
  recorded whole, as one field value. An absent key is therefore genuinely absent
  from the recording, and a strict replay errors at the lookup exactly as a live
  strict run would. The concern that a recording would have "invented" a null for a
  missing key does not apply — there was never a per-key entry to invent one in.
- **Field nulls lose their provenance, and this is the caveat.** A field recorded
  as null could have been legitimately null, or a provider call that failed and
  degraded to null (`Is400AccessDeniedError` and the ~3,500 `StateIsNull` sites).
  The recording keeps the value and drops the reason, so a strict replay cannot
  tell the two apart and treats both as a null receiver. Note that live execution
  cannot tell them apart either — the collapse happens in the provider, not in the
  recording — so replay is no worse than the original run. It is documented here
  because it looks like a recording defect and is not one.

The practical consequence is narrower than it first appears: a recording captured
non-strict and replayed strict will **error where the capture succeeded**. The
underlying data is identical; only the interpretation changed. Anyone diffing a
strict replay against its non-strict capture should expect that, and it is not a
regression.

One genuine gap remains, and it is not strict-specific: a recording only contains
fields that were actually *requested* during capture, and a non-strict run
short-circuits. `a.f.c` with `f` null never requests `c`, so `c` is absent from the
recording, and a later query that needs it finds nothing. That is ordinary
recording incompleteness, which strict mode neither causes nor fixes.

## Consequences

**Good:**

- A broken chain names the link that broke, instead of producing a null that
  looks like a finding.
- Under-permissioned scans stop resembling clean ones. This is the single largest
  behavioral win and the single largest migration cost — the same 771 AWS
  access-denied sites are responsible for both.
- The 104 per-builtin nil branches answer to one rule, in one place, instead of
  each site deciding for itself.
- `?` becomes documentation: a marked chain says "this is legitimately absent
  sometimes," which is information the schema does not carry today.

**Costs and risks:**

- **A mistyped key becomes an error instead of a silent `false`.** This is the
  case strict mode most needs to catch, and it is only catchable because the
  lookup errors rather than the dereference (§1).
- **A policy that opts in breaks loudly**, including the dict chains
  [#7079][pr7079] deferred. That is what opting in means, and it is why the
  diagnostic pass in step 6 exists: an author should see every unguarded
  dereference in their policy before they flip it, not after.
- **The v14 lint is itself a migration cost.** Every existing policy has to be
  touched to state a mode, including the large majority that will state
  non-strict and change in no other way. Unavoidable if outcomes are to be
  predictable, but it is churn across the whole content estate.
- **Two live modes is a permanent tax until it is not.** Every nil-handling
  decision, test, and diagnostic now has two correct answers, and the VM carries
  both paths. Accepted deliberately, with strict-only as the eventual exit.
- **The checksum change is a compatibility event.** Bundles recompile and
  re-checksum. It rides the same machinery ADR 040 exists to manage, but it is not
  free.
- **`?` noise on genuinely-sparse schemas.** Some resources return null for most
  fields most of the time, and their queries will grow a lot of question marks.
  The alternative is those queries silently reporting nothing, so this is the
  right trade — but it will be the loudest complaint.
- **Nullability is not in the schema.** The compiler cannot tell "this field is
  never null" from "this field is usually null", so it cannot warn about a
  redundant `?` or a definitely-missing one. See open questions.

## Open questions

- **Should `.lr` declare field nullability?** §1 rests on "a field should hold a
  valid value of its declared type, and null is not one unless the type says so" —
  but no type in the schema says so today, so the rule is a convention the compiler
  cannot check. If `.lr` carried it, the compiler could reject a pointless `?` and
  demand a missing one, turning strict mode from a runtime rule into a static
  check. Large schema project (~3,500 `StateIsNull` sites to audit); should not
  gate this ADR, but strict mode is what would make it pay off.
- **Do arrays ever converge with maps?** An out-of-range index errors in both
  modes today, so arrays are already strict on a point where maps are about to
  become strict only on opt-in (§2). Softening arrays to match would need a guard
  spelling for the index itself, and it is not obvious the current behavior is even
  wrong. Left open, explicitly not addressed in this iteration.

## Alternatives considered

- **Insert an explicit `$notNull` guard chunk between links.** Needs no proto
  change, participates in checksums automatically, and fails *loudly* on old
  executors ("unknown function") instead of silently downgrading — a genuinely
  better failure mode. Rejected on size: it roughly doubles the chunk count for
  access chains, and every chunk is a ref, a cache entry, and a checksum.
- **Encode strictness in the chunk id, as `[]?` already does.** Works for
  accessors and breaks for fields, because `chunk.Id` *is* the field name — it
  indexes `resource.Fields[chunk.Id]` (`llx/builtin.go:856`) and is passed to
  `WatchAndUpdate` (`llx/builtin.go:850`). Suffixing it would require stripping at
  every consumer. Rejected.
- **A single bundle-level `strict` bit, with no per-chunk marks.** Smaller, but it
  makes chunk checksums identical across modes for every unmarked dereference, so
  two semantics share one key. Rejected for the same reason the checksum has to
  include nullability at all.
- **A pure runtime flag, no wire change.** Cheapest to build and wrong: the same
  compiled, cached, upstream-shipped bundle would mean different things on
  different nodes, and recordings would replay under whatever mode the replayer
  happened to have. Rejected.
- **A `features.yaml` flag (`MQLStrictMode`), the ladder MassQueries and PiperCode
  took.** The obvious move, and the wrong one here: a feature flag is global,
  client-side, and two-state, so it cannot express "this policy is explicitly
  non-strict" and it reinterprets third-party content the operator cannot fix.
  Rejected in favor of a content declaration with a tri-state config default; see
  §6.
- **Extend [#6633][pr6633] to resource fields with no opt-in.** The same ordering
  mistake [#7079][pr7079] deferred, over a far larger surface: dict chains are a
  fraction of what resource fields are. Rejected.
- **Declare strictness per query or per check instead of per policy.** Finer
  grained, and it would let a large policy adopt strictness a check at a time.
  Rejected for now: it makes "is this policy strict?" a question you answer by
  reading every query in it, which is the predictability the declaration exists to
  provide. The door stays open, but it needs a forcing case, not a convenience
  argument. `?` already covers the in-policy escape.
- **Skip non-strict entirely and go strict-only in v14.** Tempting, since
  strict-only is the intended end state. Rejected as premature: the behavior needs
  far more verification against real policy runs first, and there is no way back
  from it if the verification turns up something. Both modes ship; strict-only is a
  later decision made with evidence.
- **Settle the 104 nil branches one at a time instead.** This is the current
  approach, and it produced the current state. Each site can only reason about its
  own call, so the branches cannot converge on a rule an author could rely on, and
  a change to any one of them is indistinguishable from a change to the language.
  Rejected.
