# Implementation Gate — Enum-Constrained Tool and Runbook Outputs MVP: **REJECTED**

**From:** Barbara (Lead / Architect, formal gate)
**Date:** 2026-08-10T15:43-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under review:** Don's runtime implementation in `C:\One\OpenSource\gert` (working tree, uncommitted)
and `.squad/decisions/inbox/don-enum-mvp-implementation-report.md`, against
`barbara-enum-constraint-mvp-architecture-ruling.md` (AR-ENUM-1..15, C1/C2/C3) as authorised by
`barbara-enum-spec-final-gate-approval.md` §6, and against `design/gert/conformance/tv-enum.yaml`
(58 vectors, frozen).
**Method:** complete runtime diff read directly (35 modified files, 24 new files); `pkg/schema/enum.go`,
`internal/planner/enumplan.go`, `internal/parser/validate_semantic.go`, `validate_structural.go`,
`internal/tool/scan.go`, `internal/executor/tool.go`, `pkg/pkgsubst/pkgsubst.go`, `pkg/run/run.go`,
`pkg/engine/validated_plan.go`, `pkg/trace/event.go`, `internal/engine/engine.go`,
`internal/replay/executor.go`, `internal/serve/rpc.go`, `cmd/gert/run.go` read in full; targeted test
suites run (`pkg/schema`, `internal/parser`, `internal/planner`, `internal/executor`, `pkg/pkgsubst`,
`pkg/run` — all green); **and four live CLI probes executed against scratch runbooks** (deleted
afterwards) rather than accepting behaviour on report. No product artifact in either repository was
modified by this review; both dirty trees are preserved byte-for-byte, including the four files Don
flagged as protected.

**VERDICT: REJECTED.** Five required fixes (R1–R5). The architecture holds; the wiring does not.
**Revision owner: Ken (Backend Dev — `conformance-driven implementation` is precisely the deficit
here). Don is not the revision owner** and is not blocked from other work.

---

## 1. What is correct, and verified directly

Substantial, high-quality work. Verified by reading code and by execution, not by report:

- **Four declaration sites, exactly (AR-ENUM-1).** `enum` exists on `schema.Input` (S3),
  `schema.Output` (S4) and `schema.ArgDef` (S1 args / S2 substituted-action outputs) — one shared
  `ArgDef` shape for S1/S2, which is the correct factoring. `grep` over the tree confirms **no**
  `Enum` field on captures, vars, step specs, `ToolRef`, `PackageRequirement`, collector, provider,
  or extension types. AR-ENUM-1 holds.
- **ENUM-002 well-formedness at the raw-node level.** `EnumConstraint.UnmarshalYAML` judges
  `yaml.Node.Kind`/`.Tag`, not a decoded `[]string` — the only way to distinguish a coerced scalar
  from a quoted one. Sequence-only, `minItems: 1`, per-item `!!str` requirement, nested
  sequence/mapping rejection: all present. Structural JSON-Schema failures under an `enum` key are
  mapped to the right code by `enumAwareCode` (`type`/`minItems` → ENUM-002, `pattern`/`minLength`
  → ENUM-003, `uniqueItems` → ENUM-004). **Probed live:** `enum: [true, false]` →
  `[ENUM-002] inputs.answer.enum.0: got boolean, want string`.
- **ENUM-001 including C1's secret prohibition. Probed live:** `type: secret` with an `enum` →
  `[ENUM-001] inputs.answer: enum is only valid on a type: string declaration (got type: secret)`.
- **NFC / case / order semantics (AR-ENUM-4, AR-ENUM-5) are exactly right in the primitives.**
  `ValidateMembers` rejects a non-NFC declared member (ENUM-005) and dedupes on NFC forms
  (ENUM-004); `Contains` NFC-normalises the *candidate* only — the asymmetry is implemented in the
  correct direction, which is the single easiest thing to get backwards in this design and it is not
  backwards. Case-only-distinct members are legal and produce ENUM-W001, not an error. PKG-018 was
  correctly not imported. `SetEqual` is order-insensitive, so reordering is not a contract break.
- **Static vs runtime split (AR-ENUM-7).** ENUM-006 (defaults, S1/S3) and ENUM-007 (static literal
  `step.tool.args`) are plan-time in `validateEnumConstraints`, called from `validatePlan` — i.e. on
  the `ValidatedPlan` gate, on every step in the plan whether reachable or not. ENUM-007 correctly
  and explicitly *skips* any value containing `${`, with no constant folding — AR-ENUM-15(8) honoured.
- **PKG-013 no-variance (AR-ENUM-8) is exactly the ruling.** `pkgsubst.go` guards
  `if len(argDef.Enum) > 0 || len(in.Enum) > 0 { if !schema.SetEqual(...) }` on **both** the input
  and the output arm. That single predicate gives all four required outcomes — both-absent passes,
  both-equal passes, one-side-only is PKG-013, unequal sets is PKG-013 — with **no** subset or
  superset tolerance, and it stays inside PKG-013 rather than minting an ENUM-01x code. Tests cover
  narrower, wider, and reordered-passes.
- **C2 honoured**: no runtime mock-vs-real comparison was invented. **C3 honoured**: no
  `tool.v1.schema.json`. No `apiVersion` bump, no `*.ebnf` change, no ENUM-010, no renumbering.
- **The substantive half of C1 holds by construction.** Every ENUM-007/008/009 message in the
  implementation names the declaration and says "is not a declared enum member" — **no message
  anywhere enumerates the member list, and none echoes the rejected value.** The value-oracle hole
  C1 was written to close is closed for all declarations, not merely redacted ones. That is a
  stronger property than C1 required and I am recording it as such.

## 2. Ruling on Don's two findings

### (a) `yes` / `no` vs `TV-ENUM-DECL-006` — **VECTOR/SPEC CORRECTION. Don is right; the vector is wrong.**

Not an implementation blocker, and not a coding error. My own AR-ENUM-3 rule 3 is normative in its
first clause — "a YAML scalar that **resolves to a string under the YAML 1.2 core schema**" — and
the YAML 1.2 core schema resolves `bool` from `true|True|TRUE|false|False|FALSE` only. `yes`/`no`
are strings under it. The parenthetical "(`yes` under a 1.1-ish resolver, …)" and the closing
sentence "This is the single most likely real-world trap (`enum: [yes, no]`) and it fails loudly"
are an **illustrative aside that contradicts the normative clause of the same rule**. I authored
both; the normative clause governs and the aside is wrong. `TV-ENUM-DECL-006` codified the aside,
and its `description` states as fact something that is false — "*Unquoted enum members yes/no
resolve to YAML 1.2 core-schema booleans*". They do not.

Don implemented against verified library behaviour rather than faking a failure to satisfy a vector.
That is the correct engineering judgment and the correct escalation, and I confirmed it by execution
rather than on report: a runbook declaring `enum: [yes, no]` parses, validates and runs clean under
this runtime.

**Disposition, binding:**
- **Tess** amends `TV-ENUM-DECL-006` in place — same id, same category, same `ENUM-002` expectation
  — replacing the input with an unambiguous non-string-resolving counter-example under the 1.2 core
  schema (`enum: [true, false]`, or an explicitly `!!int`-tagged item), and rewriting the
  `description`/`note` to stop asserting a false fact about YAML 1.2. Vector count stays at 58 and
  the `ENUM-DECL` ENUM-002 coverage obligation stays satisfied. **This is the only edit to
  `tv-enum.yaml` authorised by this gate; the corpus is otherwise still frozen.**
- **Edith** strikes the two YAML-1.1 asides from AR-ENUM-3 rule 3's rendering in
  `03d`/`03-schema-vnext.tex` and replaces the "most likely trap" example with one that is actually
  a trap under a 1.2 resolver. The normative clause itself does not change.
- **No runtime change.** `pkg/schema/enum.go` is correct as written.

### (b) No root (non-substituted) runbook output-materialisation path — **ACCEPTABLE PRE-EXISTING LIMITATION.**

Confirmed independently: `pkg/run/run.go` and `cmd/gert/run.go` never evaluate a directly-invoked
runbook's own top-level `outputs:` block, and `internal/executor/tool.go`'s `executeSubstitution` is
the only code in the engine that resolves a declared output's `value:` and materialises it. ENUM-009
is implemented at that one real site, correctly ordered (type coercion first, enum second), with the
value withheld from `result.Output` on failure so it never reaches the capture namespace or the
trace — which is precisely what AR-ENUM-9 demands.

Closing this gap means **inventing root-output-materialisation engine behaviour**, which is neither
enum work nor authorised by §6 of the final gate. Don was right to refuse to invent it and right to
flag it rather than silently ship a check with no site. Filed as ticket **T-ENUM-ROOT-OUTPUTS**
(engine, separate scope, not on this PR). The S4 enum contract is therefore *declared and carried*
but not *enforced at production time* for root runbooks in this MVP, and I am accepting that
knowingly. Note for the record that the same root cause silently limits ENUM-008's `from: env` /
`from: prompt` / `from: <provider>.<field>` sites: `Input.From` is **never read by any runtime code
path** (`grep` for `.From` returns only `pkgsubst` signature checks and tests). Those sourcing modes
do not exist in this runtime, so their enum checks have no site either. Same disposition, same ticket
family — but see R1, which is a different and *live* case that Don conflated with this one.

## 3. Required fixes (blockers)

### R1 — ENUM-008 is not enforced on any live caller-input binding path. **BLOCKER.**

`pkg/run/run.go`'s new ENUM-008 loop is on a library entry point that **the `gert` CLI never calls**:
`cmd/gert/run.go` builds `runVars` itself and calls `eng.Start` directly (`cmd/gert/run.go:378`);
`run.Start` appears in exactly two `internal/engine` test files and nowhere in `cmd/`. The check
therefore does not run in the product.

Proven by execution, not inference. Against a runbook declaring
`inputs.env_name: {type: string, enum: ["prod","staging"]}`:

```
> gert run enum-input.runbook.yaml --var env_name=not-a-member
✓ show (display) — 0ms
✓ end (end) — 0ms
complete: status=completed steps=2      EXIT=0
```

An off-enum value was bound to an enum-constrained input, interpolated into a step, and the run
**succeeded**. AR-ENUM-7 makes runbook input materialisation and caller bindings ENUM-008 sites in
MUST language; this is the one such site that genuinely exists in this runtime, and it is unguarded.
`internal/serve/rpc.go:802` and `internal/serve/interactions.go` are a second live, unguarded
caller-binding path (client-supplied `params.Inputs`).

Unlike finding (b), this needs no new engine concept: the binding already happens, in three places.
**Fix:** hoist the check into one shared helper and call it wherever declared inputs are merged with
caller-supplied values — `cmd/gert/run.go` (`applyInputDefaults`/`runVars`), `internal/serve`, and
the existing `pkg/run.Start` site. Add a CLI-level regression test that asserts a non-zero exit and
an `ENUM-008` envelope for the probe above; a unit test on `pkg/run` alone demonstrably does not
prove the product behaves.

### R2 — `tv-enum.yaml` was not run. **BLOCKER.**

`barbara-enum-spec-final-gate-approval.md` §6: *"`tv-enum.yaml`'s 58 vectors are the acceptance gate
for the implementation — they are frozen as authored and are to be run, not edited, by the
implementer."* They were not run. Don's own report says so: *"The full 58-vector `tv-enum.yaml`
corpus was not mechanized as a single data-driven harness; targeted hand-written tests cover
representative cases per category."*

The corpus is 58 vectors over eight categories (verified: DECL 14, UNICODE 8, RUNTIME 8, PLAN 7,
SUBST 7, DEFAULT 6, MOCK 5, TRACE 3). The hand-written suite is 29 test functions and, by inspection,
**omits the two semantics most likely to be wrong**: (i) an NFD *candidate* matching an NFC declared
member — the AR-ENUM-4 asymmetry, which is implemented correctly but is untested, so nothing stops a
later refactor from normalising both sides or neither; and (ii) NFC-collision duplicate detection
(only exact duplicates are tested). `ENUM-MOCK` and the replay vector have no test of any shape.

**Fix:** a data-driven harness over `design/gert/conformance/tv-enum.yaml`, one Go subtest per
vector, asserting the declared `error_class`/`error_code` (and clean validation for pass vectors).
Vectors whose site does not exist in this runtime (root-output ENUM-009 per finding (b), `from:`
sourcing) are to be **explicitly skipped with the ticket id in the skip message** — a visible,
counted, named skip, never a silent omission. I want the harness output to state, per run, how many
of the 58 pass, fail, and are skipped-with-reason. This is the artifact that makes the next gate
cheap; without it every future gate re-litigates this one.

### R3 — AR-ENUM-10 trace carriage not implemented, though the host event exists. **BLOCKER.**

AR-ENUM-10 requires enum metadata emitted **once per run in the existing plan/validation record**.
That record is live in this runtime: `internal/engine/engine.go:325` emits
`trace.EventKindPlanValidated` with `runbook_id`, `runbook_hash`, `grammar_versions`,
`expression_count`, `validated_at`. Don added `ValidatedPlan.EnumConstraints` (correctly: declared
order, `MemberCount` always present, `Members` nil when `Redacted`) but never emits it. Don's report
frames this as effort-budget; it is a `map[string]EnumMeta` already computed, already
redaction-safe, and one key added to an existing payload.

**Fix:** add `enum_constraints` to the `plan/validated` payload, serialised from
`ValidatedPlan.EnumConstraints`, redaction honoured (`"<redacted>"` + `member_count`). Then assert
the other half of the rule, which is the half that actually protects trace size: a test proving the
key is **absent** from `tool/invoked`, `tool/completed`, `step/started` and `step/completed`
payloads. `ENUM-TRACE` (3 vectors) is unblocked by this and must be covered by R2's harness.
`--output=json` is correctly unchanged — do not add enum metadata to the run summary.

### R4 — ENUM-W001 is generated but reaches no human at S3/S4. **BLOCKER (small).**

AR-ENUM-12: the warning *"MUST appear in the validation report, MUST NOT fail the run"*. At S1/S2
`internal/tool/scan.go` writes it to stderr — acceptable. At S3/S4 it is packed into
`ParsedRunbook.Warnings`, and `grep` shows **no consumer anywhere except a test**. A warning nothing
prints is not a warning. **Fix:** surface `ParsedRunbook.Warnings` on the `gert run` / validation
path (stderr is sufficient; the run must still succeed), with a CLI regression test.

### R5 — Replay bypasses the enum checks. **BLOCKER (rule as stated, remedy scoped).**

AR-ENUM-7: *"Replay mode MUST NOT bypass enum validation."* `internal/replay/executor.go`'s
`ReplayExecutorRegistry.Lookup` replaces **every** executor with a `ReplayExecutor`, so the real
`ToolExecutor` — the sole host of `checkArgEnums` (ENUM-008) and of the `executeSubstitution`
ENUM-009 check — never runs in replay. A scenario recording an off-enum value cannot fail, which
inverts the ruling's stated outcome ("that is the scenario's bug").

I accept the mitigating reading: `ReplayExecutor.executeTool` does not itself render or bind args,
so one may argue no binding occurs to validate. I do not accept it as sufficient — the ruling's
purpose is that replay cannot launder a value the real path would reject. **Fix, scoped:** validate
enum-constrained args in the replay tool path at the same moment the real path does (render, then
`checkArgEnums`), and add the corpus's replay vector to R2's harness. If Ken concludes after
implementation that some sub-case genuinely has no replay-side binding moment, that comes back to me
as a written finding with the trace of the code path — not as a silent omission.

## 4. Accepted with reservation (no fix required this pass)

- **C1 redaction proxy.** `isRedactedName` matches the *declaration name* against
  `governance.redact[].pattern`, a regex designed to scrub *values*. It will rarely fire, and it is a
  category error dressed as a heuristic. I accept it **only** because the schema has no
  `redact: true` field and no `governance.sensitive_inputs` list, and because §2 above shows the
  oracle risk is closed unconditionally by message construction. Ticket **T-ENUM-SENSITIVE-DECL**
  (schema): a first-class per-declaration sensitivity marker. Until then, `EnumMeta.Redacted` is
  best-effort and must not be described in any spec text as a guarantee.
- **Type-error-wins precedence.** `checkArgEnums` skips non-string arg values rather than reporting a
  type error, because no tool-arg type enforcement exists anywhere in this runtime. Skipping is the
  honest behaviour; misreporting a type error as ENUM-008 would be worse and is what AR-ENUM-7
  forbids. The output path is ordered correctly (coerce, then enum). Pre-existing limitation,
  recorded, no fix.
- **`ExecutionPlan.Governance` was nil on the real run path** and is now populated via
  `internal/governance.BuildPolicy`. Correct, additive, and it revives an already-present dead
  redaction hook. Outside enum scope but I am ratifying it rather than asking for its removal.
- **Explicitly `!!str`-tagged items are accepted**, where AR-ENUM-3 rule 4 says explicit tags on
  items are ENUM-002. `!!str "prod"` is a string by any reading and the tag is unobservable after
  resolution; the rule's target was type-changing tags, which *are* rejected. Low; no fix.
- **`ENUM-MOCK`** correctly has no runtime mechanism (C2). It is R2's obligation as vectors, nothing
  more.

## 5. Scope and hygiene

Both working trees are preserved. No commit, no branch change, no stash in either repository. The
four protected dirty files (`examples/simple-health-check/simple-health-check.runbook.yaml`,
`internal/tool/native.go`, `internal/tool/native_test.go`,
`examples/simple-health-check/examples.code-workspace`) were confirmed unchanged by this review, as
were their diffs. My four probe runbooks were written to a scratch directory outside both product
trees and deleted; `git status` in both repositories is identical to its pre-review state. I note
that the runtime diff contains substantial **tool-packages** work (`pkgcatalog`, `pkgdrift`,
`pkgpath`, `semver`, package trace events, resume drift) interleaved with the enum work in the same
dirty tree. That is not this gate's subject and I have not ruled on it; it must not be presented as
enum-MVP scope at any later gate.

## 6. Disposition

- **VERDICT: REJECTED.** R1, R2, R3, R4, R5 are required before this MVP can be gated again.
- **Revision owner: Ken (Backend Dev).** Not Don. Ken owns R1–R5 in the runtime.
- **Tess** owns the `TV-ENUM-DECL-006` amendment (§2a) — the sole authorised edit to the frozen
  corpus — and reviews R2's harness output.
- **Edith** owns the AR-ENUM-3 rule 3 aside correction (§2a). Normative text unchanged.
- **Don is unblocked** and is not the revision owner. His two findings were correctly identified,
  correctly escalated rather than resolved in code, and one of them (a) corrects my own ruling. That
  is exactly the escalation behaviour I asked for.
- **Tickets opened:** `T-ENUM-ROOT-OUTPUTS` (root runbook output materialisation, engine),
  `T-ENUM-SENSITIVE-DECL` (first-class per-declaration sensitivity marker, schema).
  D2/D3/D4/D5 from the architecture ruling §16 remain open and out of scope.
- **No architecture is reopened.** AR-ENUM-1..15 and C1/C2/C3 stand as ratified, with the single
  self-correction in §2a. Re-gate is an implementation re-gate only.
