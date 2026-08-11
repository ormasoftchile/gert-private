# Re-Gate Review (Gate 2) — `enum` String Constraint MVP

**From:** Barbara (Lead / Architect, formal gate)
**Date:** 2026-08-10T14:41-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under review:** David's revision of the uncommitted enum diff
(`.squad/decisions/inbox/david-enum-spec-revision.md`), files
`design/gert/sections/03-schema-vnext.tex`, `03d-parse-time-enforcement.tex`,
`13-evidence-tracing-resumption.tex`, `design/gert/schemas/runbook.v1.schema.json`,
read together with the rest of the enum diff (`06`, `07`, `08`, `16`, fixtures,
`conformance/tv-enum.yaml`).
**Against:** `.squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md`
(AR-ENUM-1..15, C1/C2/C3, D1) and
`.squad/decisions/inbox/barbara-enum-spec-authoring-gate-review.md` (B1–B5, L1–L3).
**Method:** exact working-tree diff read line-by-line; `$defs.Input`/`$defs.Output`
extracted programmatically and compared field-for-field against the prose describing
them; corpus verifier run; `from: captures` and inline-sectioning sweeps re-run
independently of David's report; `tv-enum.yaml` re-counted and its `value:`/`default:`
fixture forms cross-checked against the schema and the reference package.
No product/design artifact was modified by this review.

**VERDICT: REJECTED.**

**Revision owner: Don (Backend Dev / runtime-contracts).** Edith is locked out (author);
David is locked out (author of the revision under review). Don is the correct third
independent pair of eyes: the single remaining blocker is a runtime-contract statement
about how a runbook-level `outputs.<name>.value` resolves, which is exactly his surface
(`gcp.ebnf` §3.4a, GIS vs GCP evaluation), and he has not authored any of this text.
**No design is reopened. Nothing in AR-ENUM-1..15 is re-litigable.** The fix is one line.

---

## 1. Verified RESOLVED — B1–B5, L1–L3

| # | Finding |
|---|---|
| **B1** | `03:268` annotated example now reads `value: "captures.http_result"`. Independent sweep of all of `design/gert/` for `from:\s*captures` returns exactly 3 hits: the two intentional rejection-statement passages at `03:596/598` and the `$defs.Output.value` drift-history description. **Zero authored instances remain.** Resolved as specified. (But see B6 — the *stale* form is gone; a *second* shape for the same field is not.) |
| **B2** | `03:584` now `string \| number \| integer \| boolean \| object \| array`; `$defs.Output.type.enum` is `["string","number","integer","boolean","object","array"]`. Exact match. `secret` gone, `integer` restored. Resolved. |
| **B3** | The unauthorable-`default` claim is deleted. Replacement text states `$defs.Output` has no `default` field, therefore no default-membership obligation attaches at S4, and cross-references `06` §Action Substitution §Output contract for the S2 rule — which I re-read and which states ENUM-006 and PKG-027 as independent, both-fire checks. Correctly grounded: this is the vacuity that follows from the schema-canonical ruling I issued in B3/Q2, not a re-reading of AR-ENUM-6. D5 untouched, as instructed. Resolved. |
| **B4** | "(or absent, since string is the default type)" struck from `$defs.Input.enum.description`; replaced with the REQUIRED-`type` statement. `$defs.Input.required == ["type"]`, unchanged. `03` §type field (`:381`) and §enum (`:415`) both now state `type` is REQUIRED and that omission is a structural schema error, never `ENUM-001`, while retaining `string` as the documented default type semantic. Exactly the ruling. Resolved. |
| **B5** | `\label{para:type-field}` added on its own line after the standalone `\paragraph{type field}` heading; the inline invocation is now `\S\ref{para:type-field}`. Independent sweep for a `\paragraph{` preceded by any non-newline character across `design/gert/sections/`: **zero hits**. `para:enum-constraint` label defined once, referenced from `03:608`, `03:3690`, `06:314`, `13` §Event Type Catalogue — closure clean. Resolved. |
| **L1** | Input shape block now `string \| number \| boolean \| secret \| array \| object`; `$defs.Input.type.enum` is identical, element-for-element and in the same order. Resolved. |
| **L2** | `03d` §Enum Constraint Error Codes carries the redaction cross-reference sentence after the table, naming `08` §Secret Resolution and Redaction and the `"<redacted>"` + `member_count` form. Resolved. |
| **L3** | One wire shape everywhere: `03d` §ValidatedPlan (table row + prose), `03` §enum bullet, and `13` §Event Type Catalogue all now say literal `"<redacted>"` **plus** a `member_count` integer. Resolved. |

## 2. Verified — no AR-ENUM drift introduced by the revision

Re-checked against the ruling, not against David's summary: four sites and the exclusion
list (AR-ENUM-1); string-only incl. `secret` → ENUM-001, "constraint never a type", PJVM
untouched (AR-ENUM-2); all seven well-formedness clauses with their codes, incl. the
`yes`/`no` trap and "no coercion" (AR-ENUM-3); codepoint NFC equality, asymmetric
normalisation with rationale, case-only-distinct legal + ENUM-W001, PKG-018 not imported
(AR-ENUM-4); both orderings normative, SET-equality identity, no new digest machinery
(AR-ENUM-5); ENUM-006, "constrains values not presence", `null` never a member
(AR-ENUM-6); the four-step binding order, type-error-wins single-cause reporting,
`${count}`→`"2"` pass, no constant folding, runtime-only failure a designed outcome,
ENUM-009 gate placed before caller boundary/capture/trace/JSON (AR-ENUM-7); PKG-013
extended in place with no-variance stated in both directions, C2 discharged as the
`ENUM-MOCK` vector class with binding-independence as MUST NOT weaken/skip/short-circuit,
PKG-009 as the whole integrity story (AR-ENUM-8); declaration-local lexical scoping,
parent/child same-name different-enum not a conflict, lazy-include timing (AR-ENUM-9);
`enum_constraints` in the ValidatedPlan in declared order, once per run, never repeated in
`tool/invoked`/`tool/completed`/`step/*`, run summary unchanged, C1 redaction normative
(AR-ENUM-10); introducing an enum is narrowing, no `apiVersion` bump, absence never
modelled as empty/wildcard/null (AR-ENUM-11); one ENUM-0NN family, ENUM-001..009 +
ENUM-W001, no ENUM-010, no renumbering, PKG-019 still reserved (AR-ENUM-12); grammars
untouched — the `gcp.ebnf` diff is tool-packages §3.4a only, and no enum production,
keyword, or error class exists in any `.ebnf` (AR-ENUM-13); C3 held — no
`tool.v1.schema.json`, and `tool-package.v1`/`project-config.v1`/`package-lock.v1`
untouched (AR-ENUM-15). **No semantic drift found. No scope creep found.**

Metadata / redaction / error-envelope consistency: `03d` §ValidatedPlan, `03d` §Enum
Constraint Error Codes, `03` §enum, `08`, `13`, `16` now describe one wire shape and one
error envelope. Consistent.

**Tess's corpus:** `tv-enum.yaml` unmodified, **58 vectors** (ENUM-DECL 14, ENUM-UNICODE 8,
ENUM-RUNTIME 8, ENUM-PLAN 7, ENUM-SUBST 7, ENUM-DEFAULT 6, ENUM-MOCK 5, ENUM-TRACE 3) —
above the 54 minimum, `ENUM-MOCK` present. `python design/gert/scripts/verify_corpus.py`
→ **OK 423/423**. Her `TV-ENUM-DEFAULT-003` (S2) / `-006` (S4, documentation-only) shaping
is now *correct by design* rather than by workaround, and her `note` text remains true.
No corpus edit is required by this revision, and none was made. Her fixtures use the bare
GCP-path form for `Output.value` (`value: "step.drain.json.drained"`), consistent with the
schema and the reference package — and inconsistent only with B6 below.

## 3. BLOCKER

### B6 — `03-schema-vnext.tex:586`: the Output shape block illustrates `value:` as a GIS template, contradicting the schema, the corrected example, the reference fixture, and the vectors

```yaml
    value: "${captured_value}"      # GCP path, resolved at runbook completion
```

The comment says GCP path; the value shown is a GIS interpolation. Four artifacts say
otherwise:

- `$defs.Output.value`: "GCP path resolved against the final execution context. **Evaluated
  as GCP at runtime**."
- `03:268`, the annotated example David just corrected under B1: `value: "captures.http_result"`.
- `schemas/examples/acme-incident-tools/runbooks/drain-node.yaml`, the reference package,
  which states in-line that `value:` is "**never a literal and never a GIS `${...}`
  template**" and writes `value: "step.drain.json.drained"`.
- Tess's vectors, uniformly bare GCP paths.

This is not cosmetic and it is not out of scope for this gate. It is (a) the *same*
pathology D1 and B1 were ratified to eliminate — two shapes for one field surviving the
change that touched it, 320 lines apart in the same file — now relocated from the annotated
example into the **canonical shape block**, which is the higher-traffic surface of the two;
and (b) enum-material: `ENUM-009` is defined on the resolution of `outputs.<name>.value` at
production time, and this line is the only text in the specification that would teach an
implementer that S4 resolution runs GIS rather than GCP. AR-ENUM-9 fixes the object
explicitly: "the resolved value of `outputs.<name>.value` (**the GCP path form** in
`runbook.v1.schema.json`)".

The line was authored in this diff (it replaced the `from: captures.<captureName>` line),
so it is squarely inside the change under review; and David's revision note asserts the
prose "now match[es] the corresponding `03-schema-vnext.tex` prose lines verbatim" and
answers the standing gate question with "None found". Both statements are false for this
field. The `from: captures` grep was the right check for the stale key; it could not and
did not catch a wrong *value* form under the right key.

**Required fix (one line, no design decision):** change the shape block to a bare GCP path
placeholder consistent with `$defs.Output.value`, `03:268`, and the reference fixture — e.g.
`value: "captures.<captureName>"` or `value: "step.<id>.json.<key>"` — keeping the
`# GCP path, resolved at runbook completion` comment, which is correct. Then sweep every
`value:` under an `outputs:` block in `design/gert/` (sections, schemas, examples, testdata,
conformance) and confirm no other GIS-templated instance exists at that site. Do not touch
`value:` at any other site (`evidence[].value`, collector `options[].value`) — those are
different fields and are correct as written.

## 4. Standing gate question — which untouched text did the fix falsify?

Nothing. B1/B2/B4/L1 move text toward the canonical schema; B3 removes a claim; B5 is
typographic; L2/L3 restate an already-normative rule in one shape. B6 is a defect the
revision failed to *catch*, not one it introduced.

## 5. Instructions to Don

1. Fix **B6 only**. Change nothing else. Do not touch `*.ebnf`, do not create
   `tool.v1.schema.json`, do not bump any `apiVersion`, do not modify `tv-enum.yaml`,
   do not touch runtime code in `C:\One\OpenSource\gert` — this workstream is design-only.
2. Do not consult Edith or David.
3. Re-verify and state the evidence: the outputs-`value:` sweep above; `verify_corpus.py`
   still `OK 423/423`; `\label`/`\ref` closure still clean.
4. Write `.squad/decisions/inbox/don-enum-spec-b6-fix.md` with the evidence and an answer
   to the standing question.
5. B1–B5 and L1–L3 are **closed** and are not to be revisited. D2–D5 remain open tickets
   and are explicitly out of scope.

**Tess remains unblocked.** No code, category, vector-schema shape, or S4 default
disposition moves under this revision; her 58 vectors stand as authored.

**Conformance status:** corpus green (423/423, 58 enum vectors incl. `ENUM-MOCK`), but the
enum design is **NOT** yet cleared for implementation in `C:\One\OpenSource\gert` — B6 must
land and re-gate first.
