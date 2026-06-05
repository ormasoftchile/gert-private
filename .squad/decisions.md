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
