# Edith — Spec Editor

**Role:** Specification Editor

## Session Archive

Detailed history archived in history-archive.md due to size threshold (>= 15360 bytes).

## Recent Activity

### 2026-08-17 — OQ patch: resolved three open questions in Chapter 17

**Session:** Barbara ratified OQ-1/2/3 from the initial Chapter 17 delivery.

**Changes made to `design/gert/sections/17-runtime-profiles.tex`:**

1. **OQ-1 (§17.1 `id` field):** Added format constraint `[a-z0-9][a-z0-9-]*` max 64 chars. Rationale: shell-quoting safety, tab-completion, filename stem. Leading digit permitted. Marked [IMPL].

2. **OQ-2 (§17.1 `tools.<tool-id>` paragraph):** Added auth-inheritance rule (per-tool overrides inherit profile-level auth by default). Phase 3 scoping for per-tool `auth:` block: loader MUST reject it in Phase 1 with clear error. Marked [IMPL — endpoint + inheritance; per-tool auth is SPEC Phase 3].

3. **OQ-3 (`gert plan --show-profiles` zero-match behavior, §17.6):** Exit 0, empty stdout, single-line stderr message specified verbatim. Rationale recorded.

4. **`config/no-binding-for-profile` message format (§17.6 new subsubsection):** Mandatory message format including `note:` line ("this runbook runs correctly in profiles: X, Y") specified precisely. Don is implementing `gert plan` against this text.

5. **Implementation status updates (§17.11):** Profile schema/loader, ProfileApprovalGate + attendance, PLAN-011/012 all updated to [IMPL] with commit references. Attendance tcolorbox and governance-composition tcolorbox updated to reflect current [IMPL] state.

**`.squad/decisions/inbox/edith-profile-spec.md`:** OQs marked resolved; stale pre-ruling content removed.

### 2026-08-17 — Runtime Profiles and Execution Context specification

**Session:** Runtime Portability spec delivery for SQL Live-Site Operations.

**Deliverable:** `design/gert/sections/17-runtime-profiles.tex` — Chapter 17 of the Gert LaTeX design document. Added `\input{sections/17-runtime-profiles}` to `design/gert/main.tex`.

**What was encoded:** All twelve agreed design points from the Rev 4 Final Acknowledgment (2026-08-17): profile document schema, five-value closed context vocabulary, AllowedEnvironments↔context correspondence (answering the counterparty's blocking question), attendance orthogonality, profile flatness, package-map/profile composition, per-action classification governing principle, approval↔classification orthogonality (the most critical section), unspecified-by-context normative table, late/lost result semantics table, runbook-level monotone composition, test-context transport rules (including subprocess-is-assertion-not-sandbox honesty), and tiered preflight (Tier 0 mandatory/offline, Tier 1 local, Tier 2 live).

**Implementation status discipline maintained:** Every normative claim marked [IMPL] or [SPEC]. The external team can read this document and know exactly what is enforced today vs. what is committed for Phase 1.

**Open questions found:** None that required halting. One near-miss: decisions.md does not specify the exact format of `profile-id` (free string vs. slug constraint). Flagged as open question in inbox file; did not invent an answer.

## Learnings

- **2026-08-17:** The `gert-private` spec lives in `design/gert/sections/` as numbered LaTeX chapter files included from `design/gert/main.tex`. New sections must be added to `main.tex` explicitly. File naming convention is `NN-kebab-name.tex`.
- The schema/effective-type split (`pkg/schema.ToolGovernance` vs. `pkg/pkgsubst.EffectiveGovernance`) is critical: the pointer tri-state lives on the schema (authored) type only; the effective type stays `bool`. This distinction matters when writing normative claims about YAML authoring vs. runtime behavior.
- `AllowedEnvironments` is already in the schema but completely unenforced — zero runtime references. Safe to specify against it as a Tier 0 check target.
- The subprocess opt-in sentence "Gert does NOT enforce isolation" must appear verbatim (or equivalent) whenever the subprocess escape hatch is described. An external team relying on isolation for security would be dangerously misled without it.
- `decisions.md` is the authoritative source and contains the full negotiation record. Always read it before writing normative claims; do not reconstruct decisions from memory or from the task prompt alone.

**Session:** Scribe coordination session (barbara, don, david) evaluated SQL Live-Site Operations "Runtime Portability for Gert Runbooks" ask.

**Potential implications for Edith:**
- All three agents independently flagged that ask requires tool schema changes (capability-declaration, action classification) despite §13 non-goal stating "no tool contract schema changes"
- Any spec work on runtime binding resolver will depend on resolving this contradiction first
- Schema implications: additive `contexts:` field on tool-package, additive `classification:` and `idempotent:` fields on tool actions, `allowed_hosts` field on `AuthConfig`
- These are backward-compatible (non-breaking) but represent a decision point

**Recommendation:** Stand by for guidance from Cristiano and Barbara on whether to proceed with the schema expansion.

### 2026-08-15 — Continuing as team specialist

Edith continues in the role of Specification Editor for the GERT project.



## 2026-08-17 — Phase 1 Closure: Runtime Portability Complete

**Status:** COMPLETE — code exit 0, test exit 0, zero FAIL. 33 architectural rulings. All blocks satisfied.

**Key accomplishments:**
- Tri-state RequiresApproval (bool → *bool) + Classification field added
- ProfileApprovalGate + declared attendance implemented
- MCP HTTP transport (Don), auth provider (David), fixture migration (Tess), schema validation (Ken), profile spec (Edith)
- 31 conformance vectors: 31 PASS / 0 SKIP / 0 FAIL
- SQL Live-Site counterparty: 3 counter-positions accepted, refined

**Deferred:** AllowedModes field (RunMode separate from context), per-tool auth override (Phase 3), lifecycle sanity (Phase 3)

**Next phase:** OQ2 (library vs. subprocess) spike; resolver extends --package-map; Phase 2 host bridge with explicit framing protocol