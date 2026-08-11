# Spec Authoring Gate Review — `enum` String Constraint MVP

**From:** Barbara (Lead / Architect, formal authoring gate)
**Date:** 2026-08-10T14:03-07:00
**Requested by:** Cristián Ormazábal Ortega
**Under review:** Edith's uncommitted enum-related diff in `C:\One\OpenSource\gert-private`
(`design/gert/sections/03-schema-vnext.tex`, `03d-parse-time-enforcement.tex`,
`06-tool-runtime.tex`, `07-runtime-events.tex`, `08-security-and-trust.tex`,
`13-evidence-tracing-resumption.tex`, `16-observability-diagnostics.tex`,
`design/gert/schemas/runbook.v1.schema.json`, `design/gert/schemas/README.md`,
`design/gert/schemas/examples/acme-incident-tools/**`,
`design/gert/testdata/runbooks/r23-tool-package/schema.yaml`)
**Against:** `.squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md`
(AR-ENUM-1..15, C1/C2/C3, D1), plus the existing package / substitution / error-catalog /
trace contracts in `06`, `03d`, `07`, `13`, and `design/gert/conformance/vector.schema.json`.
**Method:** normative artifacts read directly and line-by-line against the ruling; schema
definitions extracted and compared field-by-field against the prose that describes them;
LaTeX `\label`/`\ref` closure verified (no dangling references); fixtures read end-to-end
(tool definition → substitute runbook → calling runbook). No product/design artifact was
modified by this review.

**VERDICT: REJECTED.**

**Revision owner: David (Integration Engineer).** Edith is locked out — she authored the
text under review. Tess is locked out by role: she owns `tv-enum.yaml` (the `ENUM-MOCK` /
`ENUM-*` corpus that discharges C2), and an author who both writes the normative prose and
writes the vectors that test it removes the only independent check this workstream has.
David has already served as independent revision owner once on this programme
(`.squad/decisions/archive/david-tool-packages-revision-archived.md`) and the residual work
is reconciliation, not design: five bounded prose/schema contradictions, all of which have a
single already-ratified correct answer stated below. **No design is reopened. Nothing in
AR-ENUM-1..15 is re-litigable.**

This is a rejection of strong work. The substance of the enum specification — the hard
parts: the plan/runtime seam, single-cause reporting, asymmetric NFC, the two orderings,
PKG-013 no-variance in both directions, the C1 oracle rule, the C2 conformance discharge —
is correct, complete, and faithful. The rejection rests entirely on the gate question I
said I would ask at gate 1: *which untouched text did the new text just falsify.* Four of
the five blockers are exactly that.

---

## 1. What I verified as CORRECT (not re-litigable in the revision)

| Ruling item | Finding |
|---|---|
| **AR-ENUM-1** (four sites) | `03` §Input/§Output and `06` §Tool Definition Schema/§Action Substitution each state the four sites and cross-name the other three. Exclusion list (`capture:`, `capture_defaults:`, `vars:`, step fields, collector fields, `toolRefs`, `requires`) present, correctly attributed to ordinary unknown-key rejection, no new code. Provider `fields:` example carries the required "informative, not GERT-validated, no `ENUM-*` raised" sentence (`03:3674`). Collector `options:` is untouched: `02` §Collector Field Validation Contract has no enum edit and is not re-specified in enum terms. |
| **AR-ENUM-2** (string only) | `enum` on non-string, incl. `secret`, is `ENUM-001` in `03`, `06`, `03d`, and both schema `$defs`. "`enum` is a constraint, never a type; no PJVM type-set change" stated verbatim. |
| **AR-ENUM-3** (well-formedness) | All seven clauses present and code-mapped: sequence/`minItems: 1`/non-string-resolving scalar/nested/tagged → `ENUM-002` (with the `yes`/`no` trap named explicitly and "no coercion"); empty/whitespace/edge-whitespace → `ENUM-003`; NFC duplicates → `ENUM-004`; UTF-8/NFC/control/bidi → `ENUM-005` with the exact codepoint ranges. Closure-wide, reachability-independent checking stated in `03d`. |
| **AR-ENUM-4** (Unicode/equality) | Codepoint-wise NFC equality, no folding/collation/confusables/trim. Asymmetric normalisation stated *with* its rationale. Case-only-distinct legal + `ENUM-W001`; PKG-018 correctly **not** imported. |
| **AR-ENUM-5** (two orderings) | Both normative, both defined, correctly assigned. Set-equality identity for PKG-013 stated. No new digest machinery anywhere; `06` §Non-Goals says so explicitly. |
| **AR-ENUM-6** (defaults/absence) | `ENUM-006`; "constrains values, not presence"; optional-absent ≠ violation; `null` never a member. Schema `Input.default` description carries the ENUM-006 linkage. |
| **AR-ENUM-7** (plan vs runtime) | `03d` §Ordering and precedence gives the four-step binding order exactly; type-error-wins single-cause reporting stated in both `03` and `03d`; `${count}` → `"2"` pass documented; no partial evaluation / constant folding, and the runtime-only failure declared a designed outcome, not a defect; no coercion / no nearest-match / no default-fallback. `ENUM-009` production-time gate correctly placed *before* caller boundary, `outputs.<name>` capture, trace, and `--output=json`. Replay no-bypass stated in **three** places (`03`, `03d`, `06` §Replay semantics) and, correctly, also added to `13` §Replay Mode's own bypass list. |
| **AR-ENUM-8** (PKG-013, mocks) | `03d` PKG-013 row extended in place — no `ENUM-010`, no renumbering, `PKG-019` still reserved. `06` §Input contract states no-variance **in both directions** with the narrower *and* wider arguments spelled out; §Output contract mirrors it. C2 discharged as `ENUM-MOCK` conformance obligation in `03d` and `06` §Non-Goals; binding-independence stated as MUST NOT weaken/skip/short-circuit; no lock field, no per-run signature registry, PKG-009 named as the whole integrity story. |
| **AR-ENUM-9** (includes/outputs) | Declaration-local, lexically scoped, never inherited/merged/intersected/unioned; parent+child same-name different-enum explicitly not a conflict; lazy-include materialisation timing with the "no PKG-017-analogous hole" rationale; PKG-027 × ENUM-006 independence stated in `06`. |
| **AR-ENUM-10** (metadata/redaction) | `03d` §ValidatedPlan carries `enum_constraints` in declared order, once per run, with the `"<redacted>"` + `member_count` form; `07` states `tool/invoked`/`tool/completed` carry values only; `13` §Event Type Catalogue states once-per-run + no repetition; `16` states the run summary is unchanged and ENUM errors use the `03d` envelope. C1 is rendered in `08` as normative MUST/MUST NOT bullets, not a note — as I required at handoff. |
| **AR-ENUM-11** (compatibility) | `03:115` paragraph states that *introducing* an enum is narrowing and breaking, with the limiting-case argument. No `apiVersion` bumped anywhere. Absence is never modelled as empty/wildcard/null enum. |
| **AR-ENUM-12** (taxonomy) | One `ENUM-0NN` family in `03d`, ENUM-001..009 with Raised-By/Condition/Remediation exactly as ruled, `ENUM-W001` in a warnings table with "MUST appear in the validation report, does not block". |
| **AR-ENUM-13** (grammars) | `gxl.ebnf` and `gis.ebnf` untouched; the `gcp.ebnf` diff is tool-packages §3.4a and contains no enum production, keyword, or error class. |
| **C3 / AR-ENUM-15** | `tool.v1.schema.json` **not** created. `tool-package.v1`, `project-config.v1`, `package-lock.v1` unchanged. No object/structural validation, no options unification, no labels/i18n, no variance, no inheritance, no case-insensitive matching, no constant folding, no lock members, no normative "did you mean". Scope creep: none found. |
| **Vector-schema compatibility** | `conformance/vector.schema.json` accepts `TV-ENUM-*` ids, the eight `ENUM-*` categories, `error_class: ENUM`, `^ENUM-\d{3}$`, and an `ENUM-W\d{3}` warnings array — and correctly documents that the type-before-enum precedence vector asserts the **type** code, never an `ENUM-*` code. Tess is unblocked on codes. |
| **Fixtures** | `kubectl.tool.yaml` `drain-node` gains `args.strategy` (`enum: ["graceful","force"]`, `default: graceful`); the substitute `runbooks/drain-node.yaml` declares the identical member set with `from: context`; `r23-tool-package/schema.yaml` binds the literal `strategy: "graceful"` and annotates it as the plan-time (ENUM-007) path. One reference package, one reference runbook — exactly the scope I set, no corpus-wide sweep. |

---

## 2. BLOCKERS

### B1 — `03-schema-vnext.tex:268`: the flagship example still authors `from: captures.*`, which the same PR now declares rejected

`03` §Complete Annotated Example still contains:

```yaml
outputs:
  health_status:
    type: string
    from: captures.http_result
```

while the new normative text at `03:589–591` says that form "MUST NOT be authored and is
rejected as an unknown key," and `$defs.Output` is `additionalProperties: false` with no
`from` property. The file now contradicts itself inside 320 lines, and the contradicting
instance is the *annotated example a reader copies from*. D1 was ratified as "fix in the
same PR" precisely so that two shapes for one object would not survive this change; the
declaration site was fixed and the example was not.

**Required fix:** rewrite that example's `outputs:` block to the `value:` GCP-path form.
Grep the whole corpus for `from: captures` before declaring it done — this was the only
survivor, but that is a fact to re-establish after the edit, not to assume.

### B2 — `03-schema-vnext.tex:577`: the Output `type` line still disagrees with the canonical schema

New prose: `type: string | number | boolean | secret | object | array`.
`$defs.Output.type` enum: `string, number, integer, boolean, object, array`.

The line was *edited in this diff* — `object`/`array` were added — so this is not
inherited drift, it is a half-applied reconciliation. It adds `secret`, which the schema
forbids, and drops `integer`, which the schema requires. Edith's own Q2 correctly named
the schema canonical and I ratified that; a canonical-source ruling that leaves the
type vocabulary unreconciled has not been applied.

The `secret` half is not cosmetic: C1 forbids `enum` on `secret`, and this line is now the
only text in the specification implying a `type: secret` **output** exists at all. A reader
reconciling C1 against it gets a rule about a construct the schema cannot express.

**Required fix:** make the line `string | number | integer | boolean | object | array`.
If anyone believes `secret` outputs should exist, that is a new question for me — not an
edit to make while reconciling.

### B3 — `03-schema-vnext.tex` §Output enum paragraph asserts a `default` that `$defs.Output` cannot express

The new text: "If `value:` carries a `default` and the output is otherwise unproduced on a
terminal path (`PKG-027` …), the default MUST itself be a member (`ENUM-006`)."

`$defs.Output` has properties `{type, description, value, enum}` and
`additionalProperties: false`. There is no `default`. Worse, "`value:` carries a
`default`" is not a coherent statement about a field whose value is a GCP path string.

AR-ENUM-6/§9 place the default rule on the **substituted action's declared output**
(S2, tool-side) — and `06` §Output contract states it correctly. The S4 restatement in `03`
invents an unauthorable field.

**Required fix:** delete the `default` clause from the `03` §Output enum paragraph and
cross-reference `06` §Action Substitution §Output contract for the S2 default rule.
Note for the record: `06`'s pre-existing "or declare a default" phrasing on substitute
outputs has the same schema gap; it arrived with the tool-packages workstream, is **not**
enum-caused, and is **not** to be fixed here. Raise it as a separate ticket (D5).

### B4 — `runbook.v1.schema.json` `$defs.Input.enum` asserts `type` may be absent; the same `$defs.Input` requires `type`

New description: "Valid only when `type` is `'string'` (**or absent, since string is the
default type**)". `$defs.Input` has `"required": ["type"]`.

AR-ENUM-2 does say absence is legal at S3 (string is the documented default). The schema
says otherwise, and has since it was hand-authored. That is a genuine ruling-vs-artifact
conflict I did not resolve and did not list in §16 — my miss, not Edith's. Edith's error is
the disposition: my handoff said *"If you find a semantic I did not resolve, stop and write
`.squad/decisions/inbox/edith-*.md`."* Instead the diff asserts, inside the very object that
contradicts it, that the contradiction does not exist.

**Ruling, so the revision is not blocked on me:** the **schema is canonical** — consistent
with the resolution I already ratified for Q2/D1. `type` remains REQUIRED on `Input`.
`Input.required`'s list is unchanged.

**Required fix:** strike "(or absent, since string is the default type)" from the
`$defs.Input.enum` description. In `03` §Input Declarations, keep `string` documented as the
default *type semantic* but state that `runbook.v1.schema.json` requires the `type` key to be
present, so an omitted `type` is a structural schema error, never an `ENUM-001`. This is
also the correct reading of AR-ENUM-2's own last clause ("absence is the existing schema
error, not an enum error") — it simply now applies at S3 as well.

### B5 — `03-schema-vnext.tex:412`: `\paragraph{type field}` used inline, mid-sentence

```latex
when the input resolves to \texttt{type: string} (the default type per
\paragraph{type field} above, whether stated explicitly or omitted).
```

`\paragraph` is a sectioning command. This emits a run-in heading in the middle of a
normative sentence and breaks the paragraph in two. The rendered specification will read as
a corrupted sentence at the single most-read definition in the feature.

**Required fix:** `(the default type per \S\ref{...} / the \emph{type field} paragraph
above ...)` — plain text or a proper `\ref`, never a sectioning command inline. Sweep the
enum diff for any other inline sectioning command while there. Also fold in B4's wording
change in the same edit.

---

## 3. Required lower-severity fixes (do them; not independently blocking)

| # | Item | Fix |
|---|---|---|
| L1 | `03` §Input YAML shape block still lists `type: string \| number \| boolean \| secret` while the bullets below normatively add `array` and `object`. Pre-existing, but the block was edited in this diff (the `enum:` line was added to it). | Align the shape block with the bullet list and with `$defs.Input.type`. |
| L2 | `03d` §Enum Constraint Error Codes does not restate that all `ENUM-*` messages are subject to §10 redaction; `08` and `03` both say it, `03d` — the catalog an implementer reads first — does not. | One sentence after the ENUM table cross-referencing `08` §Secret Resolution and Redaction. |
| L3 | `13` §Event Type Catalogue and `03d` §ValidatedPlan describe the redacted form slightly differently ("replaced with a member count" vs. `"<redacted>"` **plus** a `member_count` integer). | Use `03d`'s exact two-field form in `13` and `03`. One wire shape, stated once, referenced everywhere. |

## 4. Tickets raised, NOT to be fixed in this revision

| # | Item |
|---|---|
| **D5** (new) | `06` §Action Substitution §Output contract permits a substitute output to "declare a default"; `$defs.Output` has no `default` property. Tool-packages-era drift, enum-adjacent only. |
| D2, D3, D4 | Unchanged from the ruling §16: `$defs.Input` missing `pattern`/`example`; `$defs.Input.from` vs. §03's binding forms (note `03:262`'s `from: tracking.id` in the annotated example is an instance of D3 — leave it); the `\S\ref` in a `minted` comment at `03:184`. |

## 5. Instructions to David

1. Fix B1–B5 and L1–L3. Change nothing else. No design is reopened; every blocker above has
   its correct answer stated in this document.
2. Do not create `tool.v1.schema.json`. Do not touch `*.ebnf`. Do not bump any `apiVersion`.
   Do not touch `tool-package.v1`, `project-config.v1`, or `package-lock.v1`.
3. Do not consult Edith. Do not touch runtime code in `C:\One\OpenSource\gert` — this
   workstream is design-only.
4. Re-verify after editing, and state the evidence: no `from: captures` remains in
   `design/gert/`; `$defs.Input`/`$defs.Output` property sets match the prose that describes
   them field-for-field; `\label`/`\ref` closure still clean.
5. Write `.squad/decisions/inbox/david-enum-spec-revision.md` with a blocker-resolution
   matrix. I re-gate on B1–B5 only, plus the same standing question: which untouched text
   did the fix just falsify.

**Tess is unblocked now.** The `ENUM-001..009` / `ENUM-W001` numbering, the eight `ENUM-*`
categories, and the `vector.schema.json` shapes are ratified and will not move under this
revision. Begin `tv-enum.yaml` (minimum 54 vectors, `ENUM-MOCK` mandatory) against the
codes as they stand.
