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

> **A null value is fine. A null receiver is not.**

- An expression may evaluate to null. `a.b` where `b` is genuinely absent is null,
  and that is a correct, reportable answer.
- Reading *through* a null — using it as the receiver of a further access — is an
  error in strict mode, attributed to the exact link that was null.
- `?` immediately before the access marks that access optional. An optional access
  on a null receiver short-circuits **the remainder of the chain** to null,
  without error.

Concretely, with `b == null`:

| Expression | Non-strict (today, unchanged) | Strict |
|---|---|---|
| `a.b` | null | null |
| `a.b.c` | null | error: `b` is null in `a.b.c` |
| `a.b?.c` | null | null |
| `a["b"]` | null | null |
| `a["b"].c` | null | error |
| `a["b"]?.c` | null | null |
| `a?.b.c` (`a` null) | null | null — `?` short-circuits the whole tail |
| `a?.b.c` (`a` set, `b` null) | null | error — `?` guarded `a`, not `b` |

The last two rows are JavaScript optional-chaining semantics, deliberately: `?`
guards the value to its left and, when it fires, abandons the rest of the chain.
Guarding two links takes two marks (`a?.b?.c`). This is the behavior users already
have in their fingers, and it is the only variant in which `?` stays cheap to
write on deep chains.

**This rule does not rescue the case [#7079][pr7079] reverted, and should not
pretend to.** `dict["domain"]["key"]` with `domain` absent is a null *produced* by
the first read and *dereferenced* by the second, so in strict mode it errors —
exactly as it did between [#6633][pr6633] and [#7079][pr7079]. The difference is
that it now only errors for content that opted in, and `dict["domain"]?["key"]` is
available to say the key is genuinely optional. Carving out "a missing map key is
not itself an error" (below) keeps the terminal read legal, but it cannot make a
null receiver safe without giving up the whole point.

### 2. What counts as a dereference

Strict mode applies to every operation that reads through a receiver:

- **Resource field access** — `x.field`, compiled at `mqlc/mqlc.go:1084-1095`, run
  by `runResourceFunction`.
- **Index access** — `x[k]` on dicts, typed maps, and arrays, plus the bare-word
  sugar `json.params.A.B` that the compiler rewrites into `["A"]["B"]`
  (`mqlc/mqlc.go:1578-1592`).
- **Block binding** — `x { … }`, where a null `x` currently yields an empty block
  result.
- **Implicit array mapping** — `xs.field` over a null `xs`. Note that a *non-null*
  list whose elements produce nulls is unaffected: those are produced values, not
  receivers.
- **Builtin methods on a null receiver** — `x.length`, `x.all(…)`, `x.contains(…)`,
  the comparison helpers. These are the 104 `bind.Value == nil` sites, and strict
  mode replaces their individual folklore with the single rule above.

Two things explicitly stay as they are:

- **A missing map key is not an error.** `a["nope"]` reads successfully and
  produces null; the key set is not statically known and demanding `?` on every
  map read would be unusable. Only `a["nope"].c` errors.
- **Array index out of bounds is already an error** in both modes
  (`llx/builtin_array.go:87-95`). Strict mode does not change it.

### 3. Out of scope for this ADR

Strict mode governs **access chains only**. It does not touch:

- **Three-valued boolean logic.** `null && null` still evaluates truthy, which is
  a genuine false-green source (already flagged in `CLAUDE.md §5`). It is a
  separate change with a much larger blast radius on existing content, and strict
  chains remove a large share of the nulls that feed it. Tracked as a follow-up.
- **Comparison against null.** `nonNilDataOpV2` returning `false` stays.
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

1. **Wire `?` through, no behavior change.** Add `Nullability` to the proto and to
   `Function.checksumV2`; add `IsConditional` to the bracket-accessor AST node
   (`mqlc/parser/parser.go:614-618`, which drops it today, so `a?["b"]` does not
   work); thread the flag into `compileBoundIdentifier` (`mqlc/mqlc.go:1017`) and
   the array/block paths. Emit `OPTIONAL` where the author wrote `?`, `UNSET`
   everywhere else. Runtime honors `OPTIONAL` by short-circuiting the tail — what
   it does today anyway — so this phase is observably a no-op and lands the wire
   change early.
2. **Unify the nil handling.** The 104 `bind.Value == nil` branches each decide for
   themselves; a mode switch cannot sit on top of that. Collapse them into one
   decision point ahead of dispatch, reproducing today's behavior exactly, as a
   pure refactor with no semantic change. This is the prerequisite for step 3 and
   the largest piece of work in the plan.
3. **Implement `REQUIRED` in the VM**, at the decision point step 2 created, plus
   `runResourceFunction`. Non-strict compilations never emit `REQUIRED`, so nothing
   changes for existing content.
4. **Add the strictness knob to `CompilerConfig`** and the tri-state `strict` key
   to `CommonOpts`, with the content-then-config precedence rule. Shell, `mql run`,
   and `mqlx` pick up the config default; cnspec wires the policy declaration.
5. **Add the min-version gate.** Populate and enforce `min_mondoo_version` for
   bundles containing `REQUIRED` chunks.
6. **Lint for an explicit mode.** In v14 every policy must state strict or
   non-strict; the linter fails an unstated one. Pair it with a diagnostic pass
   that reports each unguarded dereference in a policy about to go strict, so
   authors can find the chains that need `?` before flipping.
7. **Later, and separately: streamline.** Reducing the ceremony and eventually
   retiring non-strict is explicitly future work, gated on evidence from real v14
   policy runs rather than on a release date.

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

- **Should `.lr` declare field nullability?** If it did, the compiler could reject
  `?` on a non-nullable field and demand it on a nullable one, turning strict mode
  from a runtime rule into a static check. That is a much larger schema project
  (~3,500 `StateIsNull` sites would need auditing) and should not gate this ADR,
  but strict mode is the thing that would make it pay off.
- **Is there a guard for value production, not just dereference?** `xs[9]` out of
  bounds and a future "missing map key is an error" mode would both want a
  postfix form. No syntax is proposed here; `?` deliberately guards only the
  dereference to its left.
- **What does `?` mean inside a block?** `a { b.c }` binds `a` to `_`. If `a` is
  null the block does not run at all today; strict mode should error, but whether
  `a? { … }` is the right spelling for the guard needs a call.
- **Recordings.** A recording captured in non-strict mode contains nulls that a
  strict replay would now error on. Replay of old recordings against strict
  bundles needs a defined answer.

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
