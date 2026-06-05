# Squad Decisions

**Last Updated:** 2026-06-05T00:14:04-04:00  
**Inbox Merged:** 5 files (edith, tess, barbara streams B/C/F; OI ratification)

---

## Phase 1 Active Decisions

### 2026-06-05T00:14:04-04:00: User directive — GXL Stream A open issue ratifications

**By:** ormasoftchile (Germán, via Copilot)  
**Context:** GXL Phase 1 kickoff

**Decisions:**

| # | Issue | Ratified |
|---|-------|----------|
| **OI-GIS-01** | Literal dollar-brace escape sequence | **`\${`** is canonical. Backslash-escape, single dollar. Diverges from the original proposal §2.6 (which used `$${`). The migration tool (Stream E) MUST convert any historical `$${` occurrences to `\${`. `gis.ebnf` is authoritative — proposal §2.6 to be amended in Stream B spec rewrite. |
| **OI-GCP-02** | Root JSON capture without GDP path | **Allowed.** `capture: http.body` (no trailing `.path` or `[*]`) is a legal capture expression and binds the entire root subtree to the variable. Semantically equivalent to a subtree capture per Q5. Update `gcp.ebnf` to make this explicit and add positive parse vectors in TV-GCP-PARSE. |

**Why:** Unblocks Stream D (fixture migration — needs canonical escape to migrate `text/template` interpolations) and Stream E (migration tool — needs both rules to be deterministic). Removes two of Barbara's flagged blockers; the remaining 16 open issues are queued inside Streams B/C/F and require no user input.

**Status:** Ratified — design contract for Phase 1.

**Action items:**
- Barbara: update `gis.ebnf` (escape clarification) and `gcp.ebnf` (bare-root grammar) as patch within Stream F or B.
- Tess (new Conformance Tester): write TV-GIS-ESCAPE-* vectors covering both `\${` and `\\${` literals; write TV-GCP-PARSE-ROOT-* vectors for bare-root captures.
- Edith (new Spec Editor): amend proposal §2.6 in `design/gert/expression-language-proposal.md` during Stream B spec rewrite.

---

### 2026-06-05T00:14:04-04:00: Stream B — Complete (Edith, Spec Editor)

**Status:** Delivered — `design/gert/sections/03a-expression-language.tex`

**Deliverable:** Normative LaTeX section encoding GXL grammar (gxl.ebnf v1.0.0-draft) as spec prose.
Sections: Informative intro, reference to grammar, lexical structure, syntactic structure (8-level precedence), stdlib, semantics, error catalog (GXL-PARSE-001..010, GXL-TYPE-001..004, GXL-PATH-001..003, GXL-EVAL-001..004), error message format, migration note.

**Discrepancies Found — All Resolved (Grammar Authoritative):**

1. **Scientific notation:** Proposal forbids; grammar permits (`EXP` production). Grammar wins.
2. **`len` namespace:** Proposal lists as namespace; grammar defines as standalone builtin. Grammar wins.
3. **Missing stdlib:** Proposal omits `str.length`, `list.indexOf`, `list.length`; grammar defines all three. Grammar wins.
4. **GDP path access:** Proposal forbids "member access on identifiers"; grammar permits `GDP = IDENT { DOT IDENT | LBRACKET INTEGER RBRACKET }`. Grammar wins.

**Semantic Gaps:**
- PJVM definition deferred to GIS spec section (Stream B follow-up)
- Conformance corpus TV-GXL-EVAL owned by Tess (not Block; forward reference sufficient)
- `capture.default:` cross-reference added with forward label

**LaTeX Notes:**
- File not wired into `main.tex` yet (deferred to Stream B integration task)
- `\TODO` macro defined locally; if promoted to main preamble, `\providecommand` will no-op
- Uses `xcolor` (already loaded), `minted` with `text` lexer, `longtable` capable
- Label convention: `\label{sec:gxl}` (not `\label{ch:gxl}`)

**Terminology Committed:**
- evaluator (for runtime GXL executor)
- GDP (GERT Dotted Path)
- PJVM (GERT Portable JSON Value Model)
- parse error vs. evaluation error (timing distinction is normative)
- context map (runtime variable scope)
- boolean position (not "boolean context")

---

### 2026-06-05T00:14:04-04:00: Stream C Day 1 — Schema and GXL-PARSE Kickoff (Tess, Conformance Tester)

**Status:** Delivered — `design/gert/conformance/schema.json` + `design/gert/conformance/tv-gxl-parse.yaml` (83 vectors)

**Deliverables:**
- JSON Schema Draft 2020-12 conformance schema
- 83 GXL-PARSE test vectors (55 positive, 28 negative)

**Coverage:** All 10 GXL-PARSE error codes (GXL-PARSE-001..010) have ≥1 triggering vector.

**Conflicts Requiring Barbara Arbitration:**

1. **GXL-PARSE-007 vs GXL-PARSE-010 for keyword-as-identifier:**
   - gxl.ebnf §2.3: "Parsers MUST reject keyword as identifier. Error: GXL-PARSE-007."
   - gxl.ebnf §6.1 catalog: "GXL-PARSE-010: Keyword used as identifier."
   - Vectors TV-GXL-PARSE-081/082/083 marked `undecided`. Both runtimes will diverge.
   - **Action:** Barbara patches gxl.ebnf to unify to one error code.

2. **`str.foo` expression — GXL-PARSE-006 or GXL-PARSE-010?**
   - Vector TV-GXL-PARSE-083: `str.foo` (no parens)
   - Ambiguous: unknown method (PARSE-006) vs. keyword as identifier (PARSE-010)?
   - Currently marked GXL-PARSE-010 with note flagging ambiguity.
   - **Action:** Barbara arbitrates.

**Ambiguities Surfaced:**

1. **`1.2.3` error classification:** Lexer error (GXL-PARSE-003) or parse error (GXL-PARSE-001)? Vector assigns PARSE-001 with note.
2. **String concatenation as GXL-PARSE-007:** Parse phase cannot detect `"a" + "b"` as string concat (no type info). Error is GXL-TYPE-003 (eval time). Deferred to TV-GXL-EVAL.
3. **`len()` with zero args:** Grammar requires Expression inside LenCall; `len()` is GXL-PARSE-001 (unexpected RPAREN), not GXL-TYPE-004. No missing code.

**Schema Design Notes:**
- 13th category proposed: **GXL-FUNC** for stdlib function call conformance (distinct from GXL-EVAL). Barbara to ratify.
- `expected.value: "parse_ok"` sentinel for parse-only vectors (conformance harness accepts any successful parse).
- `variables: {}` for parse vectors (parser doesn't consult context map).

**Vector Count:** 83 (target ≥30). ✅

**Next Iteration (Day 2+):** `tv-gxl-eval.yaml` (GXL-EVAL category), `tv-gxl-func.yaml` (GXL-FUNC category if ratified).

---

### 2026-06-05T00:27:12-04:00: GXL Phase 1 Day 2 — GIS/GCP Specs, Eval+Path Vectors, Conflict Arbitration

**Streams B+C+arbitration — Status:** Delivered

**Deliverables:**

1. **Edith (Stream B Day 2) — GIS and GCP Spec Sections:**
   - `design/gert/sections/03b-interpolation-syntax.tex` (29,321 bytes) — Normative GIS specification. Establishes PJVM canonical home at §sec:gis:portable-json. Terminology: GIS evaluator, template string, interpolation block, boolean (PJVM type name), PJVM.
   - `design/gert/sections/03c-capture-paths.tex` (35,801 bytes) — Normative GCP specification. Implements OI-GCP-02 (bare-root capture allowed). Terminology: bare-root capture, scalar capture, subtree capture, source prefix, capture-then-GXL pattern.

2. **Tess (Stream C Day 2) — Conformance Vectors:**
   - `design/gert/conformance/tv-gxl-eval.yaml` (30,983 bytes, 87 vectors) — Evaluation semantics. Covers short-circuit AND/OR, logical operators, comparison (numbers/strings/booleans/cross-type/null), arithmetic, division/modulo by zero, modulo with negative operands (OI-GXL-03), stdlib (`str.*`, `list.*`, `len`), type errors, OI-GXL-05 (GDP in function arg), error codes exercised: GXL-EVAL-002, GXL-EVAL-004, GXL-TYPE-001..004, GXL-PARSE-006. **4 vectors TBD pending arbitration:** TV-GXL-EVAL-033 (boolean ordering), TV-GXL-EVAL-086 (array equality).
   - `design/gert/conformance/tv-gxl-path.yaml` (16,169 bytes, 35 vectors) — Path traversal semantics. Covers simple access, object nesting, array indexing, mixed field+index, subtree captures (OQ5), missing keys, out-of-bounds, index-on-non-array, field-access edge cases, identifier constraints (keyword prefixes), realistic nested patterns. **2 vectors TBD pending arbitration:** TV-GXL-PATH-022 (field on scalar), TV-GXL-PATH-023 (field on null).

3. **Barbara (Arbitration) — TESS-CONFLICT-1 and TESS-CONFLICT-2 Resolved:**
   - **CONFLICT-1 (keyword-as-identifier):** GXL-PARSE-010 is the sole authoritative code. The §2.3 IDENT production body comment incorrectly cited GXL-PARSE-007 — documentation bug (GXL-PARSE-010 added later, comment never updated). Patch: `gxl.ebnf §2.3` body text corrected; `gxl.ebnf §6.1` scope note added to GXL-PARSE-007 clarifying it covers forbidden *syntax* (&&, ||, ternary) exclusively, not keywords-as-identifiers; `03a-expression-language.tex` error catalog updated; `03d-parse-time-enforcement.tex` error catalog updated; `tv-gxl-parse.yaml` vectors 081–082 finalised as GXL-PARSE-010, note updated.
   - **CONFLICT-2 (str.foo classification):** Sub-case A (`str.xyz()` with parens, unknown method): GXL-PARSE-006 at parse time — no new code needed; closed-stdlib is authoritative. Sub-case B (`str.foo` without parens): GXL-PARSE-001 (unexpected token) — under PEG ordered alternation, NamespaceCall fails at missing LPAREN, GDP fails because `str` is keyword, all alternatives exhaust. NOT GXL-PARSE-010 (no identifier position) and NOT GXL-PARSE-006 (unreachable without parens). Patches: `gxl.ebnf §3 NamespaceCall` note block added; `gxl.ebnf §3 Method` comment clarified (closed stdlib, parse-time enforcement); `03a-expression-language.tex` PEG fallthrough paragraph added; `tv-gxl-parse.yaml` vector 083 changed from GXL-PARSE-010 → GXL-PARSE-001, note updated, `ambiguous` tag removed.

**Conformance Corpus Status:**

- **Total vectors delivered:** 122 (TV-GXL-EVAL 87 + TV-GXL-PATH 35)
- **Cumulative corpus:** 205 vectors (TV-GXL-PARSE 83 + TV-GXL-EVAL 87 + TV-GXL-PATH 35)
- **Phase 1 target:** ≥200 vectors — **TARGET HIT** ✅

**Discrepancies Found (All Resolved — Grammar Authoritative):**

1. **DISC-B2-1 (`boolean` vs `bool`):** gis.ebnf uses `boolean` (PJVM canonical); 03a prose uses `bool` (GXL shorthand in error messages only). Grammar wins. Editorial note added to terminology glossary.
2. **DISC-B2-2 (decisions.md OI-GCP-02 example):** Ratification mentions `http.body` as bare-root subtree capture; but grammar shows `http.body` (no GDP) returns scalar string. Grammar wins — spec encodes grammar behavior. **Recommendation:** Scribe should amend OI-GCP-02 example from `http.body` to `json` in next merge to avoid misleading implementors.
3. **DISC-B2-3 (OQ5/Q5 label):** Not found in accessible decisions; cited `gcp.ebnf §5` as source with "(Q5, resolved)" parenthetical per task brief. **Recommendation:** Scribe to confirm OQ5 label matches archive in next merge.

**Open Items — Barbara Arbitration Required:**

| Item | Trigger | Issue | Options |
|------|---------|-------|---------|
| **TESS-AMBIG-3** | TV-GXL-EVAL-033 `false < true` | No error code for ordered comparison on non-orderable type (bool). GXL-TYPE-001 requires *incompatible* types; booleans are same type. | A) Extend GXL-TYPE-001 to include "ordered comparison on type with no ordering" \| B) Define booleans as ordered (false < true valid) \| C) New code GXL-TYPE-005 |
| **TESS-AMBIG-4** | TV-GXL-EVAL-086 `myArr == myArr` | Array/object equality semantics undefined. gxl.ebnf §5.2 covers scalars only. | A) Restrict == to scalars (GXL-TYPE-001 for lists/objects) \| B) Deep value equality (structural) \| C) Reference/identity equality |
| **TESS-AMBIG-5** | TV-GXL-PATH-022 `foo.bar` (foo=7) | No error code for dot-access on scalar. GXL-PATH-001 says "segment not found" but reason is type, not absence. GXL-PATH-003 covers INDEX only. | A) Use GXL-PATH-001 ("segment missing") \| B) Extend GXL-PATH-003 to "dot-access on non-object" \| C) New code GXL-PATH-004 |
| **TESS-AMBIG-6** | TV-GXL-PATH-023 `nullFoo.bar` (nullFoo=null) | Dot-access on null has no explicit code. GXL-PATH-003 covers bracket-index on null. | (Likely shares resolution with TESS-AMBIG-5) |

**Phase 1 Ratifications:**

- **OI-GIS-01 (canonical escape):** Confirmed `\${` in 03b prose. Proposal §2.6 amendment queued for Stream B integration task (FU-B2-1).
- **OI-GCP-02 (bare-root capture):** Confirmed in 03c and grammar patches. Bare `json`/`yaml` with no GDP returns root subtree (not scalar string).
- **OQ5 / Q5 (subtree capture semantics):** Cited `gcp.ebnf §5`; "(Q5, resolved)" parenthetical added per task brief.

**PJVM Canonical Home:**

`03b-interpolation-syntax.tex §sec:gis:portable-json` is established as the single normative definition of GERT Portable JSON Value Model. Both `03a` (GXL) and `03c` (GCP) reference it. **Future task (Stream B integration):** Add `\S\ref{sec:gis:portable-json}` cross-reference to `03a §sec:gxl:semantics:types`, replacing the informal type table with a forward reference.

**Phase 1 Completion Status:**

| Stream | Deliverable | Status |
|--------|-------------|--------|
| **A** | Reference Grammar | ✅ Complete (Barbara) |
| **B** | Spec Rewrite (03a/03b/03c) | ✅ Complete (Edith) |
| **C** | Conformance Corpus (205 vectors) | ✅ Complete (Tess) — target ≥200 HIT |
| **D** | Fixture Migration | ✅ Complete (Don) — 21 fixtures migrated, 1 deferred on now() stdlib |
| **E** | Migration Tooling | 🔄 Unblocked (awaits Stream D signoff) |
| **F** | Parser Gate Spec (03d) | ✅ Complete (Barbara) |

**Queued Follow-Ups (Stream B/C):**

1. **FU-B2-1:** Proposal §2.6 amendment (OI-GIS-01 action item) — replace `$${` with `\${` (canonical).
2. **FU-B2-2:** `03a §sec:gxl:semantics:types` forward-reference to `§sec:gis:portable-json`.
3. **FU-B2-3:** `main.tex` wiring for 03a/03b/03c (deferred to Stream B integration).
4. **FU-B2-4:** `decisions.md` OI-GCP-02 example amendment: change `http.body` to `json`.
5. **Arbitration required:** TESS-AMBIG-3/4/5/6 vectors remain TBD until Barbara decision (4 vectors pending).

---

### 2026-06-05T18:30:00-04:00: Stream D — Fixture Migration Complete (Don, Backend Dev)

**Status:** ✅ DELIVERED — 21 runbook fixtures migrated; 20 complete, 1 deferred ({{ now }} → GXL stdlib)

**Day 1 (Audit):** Completed fixture migration audit. Found **388 total violations** across 21 runbooks requiring migration (r01–r11, r13–r22; r12 already clean):
- 368× `{{ .var }}` template syntax
- 6× `&&`/`||` boolean operators  
- 9× `!` prefix negations
- 3× infix `contains` (non-method)
- 1× `$.` jq-style root
- 1× template pipe `| func`

**Day 2 (Migration & Resolution):** Completed all 21 runbook migrations with 100% compliance to GXL/GIS/GCP-canonical syntax:
- **Batch 1:** r16, r18, r13, r21, r14, r11 (pure template → GIS substitution)
- **Batch 2:** r17, r15, r19, r06, r09 (medium volume, no expression violations)
- **Batch 3:** r22, r08, r05, r04 (high volume, pure substitution)
- **Batch 4:** r02, r07, r10, r01, r03 (expression violations resolved: `&&`→`and`, `||`→`or`, `!`→`not`, infix `contains`→`str.contains()`)

**Template Substitutions Applied:** ~368 replacements; `{{ .var }}`→`${var}` (GIS portable interpolation); `{{ .X.Y }}`→`${X.Y}` (GDP); `{{ .X[N] }}`→`${X[N]}` (array index).

**Expression Normalizations Applied:**
- All `&&`/`||` → `and`/`or` (GXL binary operators)
- All `!var` → `not var` (GXL unary operator)
- All infix `contains` → `str.contains(var, "str")` (stdlib method call)

**Structural Migrations:**
- r11: `over: "$.services"` → `over: services` (bare GDP identifier per GCP spec)
- r20 (RISK-002 resolution by Germán): added `inputs.env: {type: string, default: "dev"}` + replaced `{{ .env | default "dev" }}` with `${env}`
- r07 (RISK-005 noted): `${{ .amount }}` correctly becomes `$${amount}` (literal `$` + GIS interpolation block)

**Exit Criteria Verification:**

| Criterion | Target | Result |
|-----------|--------|--------|
| `{{ }}` occurrences in P1 (non-comment value lines) | 0 | ⚠️ 1 (deferred) |
| `&&`/`||` in expression-position fields (P2) | 0 | ✅ 0 |
| `!` prefix in `when:` fields (P3) | 0 | ✅ 0 |
| Infix `contains` in expression-position (P4) | 0 | ✅ 0 |
| `$.` jq-style in non-comment lines (P5) | 0 | ✅ 0 |

**Deferred Item (DEFERRED-001):** `design/gert/testdata/runbooks/r04-soc2-evidence/schema.yaml:285` — line `completion_date: "{{ now }}"` deferred. `{{ now }}` is a Go Sprig template function (not a variable access). Unresolved at Day 2 checkpoint.

**Decision Ratified:** **Add `now()` to GXL stdlib** (per Germán call). Returns UTC ISO-8601 string; spec to be defined by Barbara. This resolves DEFERRED-001 and unblocks r04 patch (Barbara handling grammar/spec/fixture update in parallel).

**Stream D Status:** ✅ **COMPLETE (pending Barbara's now() + r04 patch).** All 21 runbooks passed exit criteria P2–P5 at zero; 20 fixtures fully delivered; r04 deferred on now() stdlib availability.

**Tools & Extensions:** All 12 tool fixture files remain clean (untouched). Extension file (`hello-ext/gert-extension.yaml`) remains clean (untouched).

**Notable Risk Resolutions:**
- RISK-002 (r20 `env` semantics): Germán confirmed `env` is a runbook input with default "dev".
- RISK-003 (r09 `!acknowledged`): 8× negation replacements applied to both `iterate:` blocks and step-level `when:` fields.
- RISK-004 (r01 infix `contains`): Converted `pod_json contains "X"` → `str.contains(pod_json, "X")`.
- RISK-005 (r07 `${{ .amount }}`): Documented as correct (literal dollar + interpolation) — no change needed.

**Conformance Artifacts:** Exit-criteria lint script provided in Day 2 memo. Full migration audit memo (Day 1) + execution report (Day 2) stored in `.squad/decisions/inbox/` (to be merged into decisions.md post-approval).

---

### 2026-06-05T00:14:04-04:00: Stream F — Complete (Barbara, Lead / Architect)

**Status:** Delivered — `design/gert/sections/03d-parse-time-enforcement.tex` + grammar patches

**Deliverables:**

1. **Grammar Patches:**
   - `design/gert/grammar/gis.ebnf`: OI-GIS-01 section updated; `\${` marked canonical, `$${` deprecated, migration directive added
   - `design/gert/grammar/gcp.ebnf`: `StepStructured` + `LocalStructured` updated; GDP now `[ GDP ]` (optional) for bare-root capture (OI-GCP-02)
   - `design/gert/grammar/gcp.ebnf`: Source prefix reference table updated; bare `json` + `yaml` rows added
   - `design/gert/grammar/gcp.ebnf`: OI-GCP-02 section marked ratified

2. **Main Spec Section:** `design/gert/sections/03d-parse-time-enforcement.tex` (8 sections, normative)
   - §1 Governance Rationale (informative): cross-runtime parity, audit trail integrity, replay determinism
   - §2 Parse-Time Contract (normative): definition of "validated runbook", MUST/MUST NOT rules for RunHandle.Next(), once-at-plan-time validation, plan.validated trace event
   - §3 ValidatedPlan Type Contract (normative, language-agnostic): field specification, language-specific implementation notes
   - §4 Error Code Catalog (normative master list): GXL, GIS, GCP codes + new PLAN-001..009
   - §5 Error Message Format (normative): structured error envelope (code, message, location, snippet, suggestion)
   - §6 No-Bypass Guarantees (normative, governance): Go type system enforcement, C# internal sealed, Roslyn analyzers, CI requirements
   - §7 Grammar Version Pinning (normative): ValidatedPlan records grammar versions, PLAN-007 on mismatch, compatibility policy
   - §8 Open Issues Phase 2 (informative): OPQ-GATE-01..07

**New Error Codes (PLAN-001..009):**

| Code | Raised By | Condition | Remediation |
|------|-----------|-----------|-------------|
| PLAN-001 | Parser/Planner | Unparseable expression wrapper (includes sub-error from grammar) | Fix flagged expression field |
| PLAN-002 | Planner | Undefined capture variable reference | Ensure variable captured before use |
| PLAN-003 | Planner | `capture.default:` on subtree capture (alias for GCP-DEFAULT-SUBTREE) | Use `when: var != null` guard |
| PLAN-004 | Parser | Forbidden Go-template syntax `{{ }}` | Replace with `${...}` interpolation |
| PLAN-005 | Parser | Forbidden infix operator (`&&`, `||`, `!`, infix `contains`) | Replace with `and`, `or`, `not`, `str.contains()` |
| PLAN-006 | Parser | Forbidden pipe expression in GCP path (`\| length`, etc.) | Replace with `len()` in GXL after scalar capture |
| PLAN-007 | Planner | Grammar version mismatch — plan validated against incompatible grammar | Re-validate runbook against current grammar |
| PLAN-008 | Planner | Keyword used as capture variable name (e.g., `capture: and: ...`) | Rename capture variable |
| PLAN-009 | Planner | Cross-step capture path `step.X.*` where step X not statically reachable | Re-order steps or use `when:` guard |

**Design Rationale:**
- PLAN-001..003: Aggregation/alias codes (preserve grammar codes in `sub_error` field)
- PLAN-004..006: Migration-era codes (catch un-migrated runbooks, detected at parse time)
- PLAN-007: Grammar version mismatch sentinel (protects replay determinism)
- PLAN-008: Extends GXL-PARSE-010 to planner level (capture var names, not GXL expressions)
- PLAN-009: Output of statically-reachable-step algorithm

**Governance Gaps — Phase 2 Decisions Required:**

🔴 **OPQ-GATE-01 — In-flight grammar version upgrade (BLOCKING for Phase 2)**

Question: What happens when runtime upgrades grammar version mid-execution (e.g., suspended at approval gate)?

Option A (strict): Suspend at next `NextAsync()`, raise PLAN-007, mark run as VersionMismatch. Operator must re-validate, replay, or cancel.
Option B (sticky): Run retains grammar version for its lifetime. Version upgrades only affect new runs. Runtime maintains version registry.

Trade-off: A is safer (no mixed-version execution) but disruptive for long-running flows. B is friendlier but requires version registry infra.

Request: Germán, please decide before Phase 2. Gates OPQ-GATE-05 (plan-storage spec).

🟡 **OPQ-GATE-02 — Statically reachable step set definition**

Planner needs formal definition of "statically reachable" for cross-step capture validation (PLAN-009, GCP-PARSE-006). Conservative over-approximation recommended for v1.

🟡 **OPQ-GATE-05 — Plan storage and re-validation policy**

ValidatedPlan persistence (where stored, eviction, forced re-validation) unspecified. Needed before web platform plan store built.

🟢 **OPQ-GATE-03, 04, 06, 07 — Lower priority**

See §8 for full descriptions.

**Contradictions Resolved:**

1. Original proposal §2.6 specifies `$${` as escape; grammar specifies `\${`. Grammar is authoritative per OI-GIS-01. Noted in patch.
2. Original proposal §2.4 restricts GIS to "identifier path only"; grammar (post-Stream A) allows full GXL inside `${...}`. Grammar is authoritative. Stream B must update proposal §2.4.

**Team Coordination:**
- **Edith (Stream B):** PLAN-* codes in §4.4 are normative master. Do not create separate plan-level codes. Reference `\label{sec:parse-gate:error-codes}`.
- **Tess (Stream C):** Priority conformance vectors: PLAN-004, PLAN-005, PLAN-007, OI-GCP-02 bare-root vectors. Gate-level vectors tagged `TV-PLAN-*`.
- **Phase 2 Parser Engineer:** ValidatedPlan type contract in §3 is primary interface spec. Error message format in §5 is normative.
- **Don:** OPQ-GATE-05 (plan storage) requires architecture input when lands in Phase 2.

---

## 2026-06-05 — now() stdlib + Stream E Day 1

**By:** Barbara (now() spec), Don (migrate-expr tool)  
**Status:** Delivered and ratified

### now() Added to GXL Stdlib

**Decision:** `now() → string` (ISO-8601 `YYYY-MM-DDTHH:MM:SSZ`)

**Key specs:**
- Per-evaluation determinism (each call yields independent fresh timestamp)
- Arity error reuses **GXL-TYPE-004** (no new error code)
- Bare `now` (no parens) raises GXL-PARSE-001 (unexpected token)

**Patch scope:**
- `design/gert/grammar/gxl.ebnf` (+42 lines): `KW_NOW` keyword, `NowCall` production, stdlib entry
- `design/gert/sections/03a-expression-language.tex` (+72 lines): keywords table, function intro, stdlib subsection
- `design/gert/conformance/tv-gxl-eval.yaml` (+33 lines): TV-GXL-EVAL-088/089/090 (return type, arity, bare keyword)
- `design/gert/testdata/runbooks/r04-soc2-evidence/schema.yaml` (1 line patched): `{{ now }}` → `${now()}` (DEFERRED-001 resolution)

**Conformance corpus:** 205 → **208 vectors** (+3 for now())

**Stream D Status:** ✅ **FULLY COMPLETE** — All P1..P5 exit criteria at zero; DEFERRED-001 closed; r04 patched clean.

---

### Stream E Day 1 — Subcommand + Translation Engine

**Status:** Delivered — 11 rules implemented, 31 tests passing

**Deliverables:**
- `cmd/gert/cmd/migrateexpr.go` + root command registration
- `internal/migrateexpr/translate.go` (11 rules: E-001..E-011)
- `internal/migrateexpr/traverse.go` (position-aware YAML tree walker)
- 26 unit tests + 5 integration tests (all passing)

**Translation rules implemented:**
| Rule | Input | Output | Authority |
|------|-------|--------|-----------|
| **E-001/002/003** | `{{ .X }}`/`{{ .A.B }}`/`{{ .A[N] }}` | `${X}`/`${A.B}`/`${A[N]}` | GIS/GDP spec |
| **E-004/005/006** | `&&`/`\|\|`/`!` | `and`/`or`/`not` | GXL binary ops |
| **E-007** | `X contains "Y"` | `str.contains(X, "Y")` | GXL stdlib |
| **E-008** | `over: "$.IDENT"` | `over: IDENT` | GCP subtree iteration |
| **E-009** | `$${...}` (old escape) | `\${...}` | OI-GIS-01 canonical form |
| **E-010** | `{{ .x \| default }}` | **WARN** (no auto-translate) | Template pipes (RISK-002) |
| **E-011** | `{{ now }}` | `${now()}` | GXL stdlib decision |

**Position-aware traversal features:**
- `gopkg.in/yaml.v3` Node API for comment/style preservation
- Expression-position detection by YAML mapping key (`when`, `condition`, `until`)
- Non-string tag guard (skip `!!int`, `!!bool`, etc.)

**Day 2 Plan:** Dogfood on already-migrated fixtures (expect zero diff); emit warnings for `contains` ambiguity (string vs list); extend E-006 to detect parenthesized negation `!(...)`.

**Dependencies added:**
- `github.com/spf13/cobra` v1.8.1
- `gopkg.in/yaml.v3` v3.0.1

---

## Previous Decisions (Archived)

**Archive:** `.squad/archive/decisions-20260605-042704.md` (2825 lines, 154,277 bytes)

Historical decisions accessible via archive file:
- Architecture Options: GERT Web Execution Platform on Azure
- Topology Segmentation Decision: A6 MVP + A8 Future
- A6 Architecture Decisions
- And 14 additional entries (see archive)

---

## Terminology / Conventions

**GXL:**
- evaluator (runtime executor, not "interpreter")
- GDP (GERT Dotted Path, not "dot path")
- PJVM (GERT Portable JSON Value Model, not "value model")
- parse error vs. evaluation error (timing distinction is normative)
- context map (runtime variable scope, not "environment")
- boolean position (not "boolean context")

**Error Categories (13 + reserved):**
1. GXL-PARSE (10 codes: GXL-PARSE-001..010)
2. GXL-EVAL (4 codes: GXL-EVAL-001..004)
3. GXL-TYPE (4 codes: GXL-TYPE-001..004)
4. GXL-PATH (3 codes: GXL-PATH-001..003)
5. GIS-INTERP
6. GIS-PATH
7. GIS-TYPE
8. GCP-PARSE
9. GCP-RESOLVE
10. GCP-DEFAULT
11. NAMESPACE
12. CONFORM-CROSSCUT
13. GXL-FUNC (proposed by Tess, pending Barbara ratification)
14. PLAN (9 codes: PLAN-001..009) — parse-time gate enforcement

---

## Archive Dates

| Date | File | Lines | Bytes |
|------|------|-------|-------|
| 2026-06-05T04:27:04Z | `.squad/archive/decisions-20260605-042704.md` | 2825 | 154,277 |

---

*Scribe note: Phase 1 Day 1 merged. Streams B, C, F kickoff deliverables integrated. 5 inbox memos processed. Conflicts surfaced for Barbara arbitration. Phase 2 gates identified (OPQ-GATE-01, 02, 05 blocking; 03, 04, 06, 07 lower priority). New team members (Edith, Tess) onboarded and integrated. Archive threshold crossed; old decisions archived. Cross-agent updates queued (see next task: charters + history).*


---

# Don — Stream E Day 2 Decision Summary

Date: 2026-06-04T20:14:36.949-07:00
Requested by: ormasoftchile

## Summary

Stream E Day 2 is implemented and validated. I dogfooded `gert migrate-expr` against the already-migrated runbook fixtures, fixed the idempotency bug it exposed, added contains ambiguity handling, extended E-006 for parenthesized negation, and added permanent regression coverage.

## Outcomes

- Dogfood target: `design/gert/testdata/runbooks/`
- Result after fixes: 22 YAML files, 0 translations, post-migration verification clean.
- Added integration coverage proving already-migrated runbook fixtures remain byte-for-byte unchanged.
- Added E-007 handling:
  - string-like LHS migrates to `str.contains(...)`
  - list-like LHS migrates to `list.contains(...)`
  - ambiguous LHS is not rewritten and emits a warning for manual resolution
- Added E-006 scanner behavior:
  - `!X` -> `not X`
  - `!(X)` -> `not (X)`
  - `!(X && Y)` -> `not (X and Y)`
  - `!(X || (Y && Z))` -> `not (X or (Y and Z))`
  - `!!X` -> `not not X`

## Dogfood Finding

The no-op pass exposed a real idempotency bug: migrated r07 strings such as `Amount: $${amount}` were being treated as legacy E-009 escapes on a second pass. That bypasses the intended Stream D meaning: literal dollar plus GIS interpolation. I fixed this by preserving `$${symbol}` when `symbol` is known from runbook context such as inputs/captures/collector fields.

## Open Questions

1. The current E-007 type detection is conservative heuristic context, not full schema/type inference. Stream E completion should decide whether heuristic warning is enough for MVP or whether Day 3 must add richer symbol typing from runbook schema.
2. GXL now defines `list.contains(...)`; confirm that list-membership migration should permanently target that namespace call.

## Validation

- `gert migrate-expr --dry-run --diff=false .\design\gert\testdata\runbooks` -> 0 translations, clean verification.
- `go test ./...` -> passing.

