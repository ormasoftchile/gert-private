# barbara

## 2026-08-18 — SUMMARY: Runtime Portability Architecture Complete

**Status:** Phase 1B implementation complete (commit 05fd13b). Five of six items delivered; Item 4 dual-binding proof pending SQL Live-Site's mode confirmation. All 33 architectural rulings published.

**Key outcomes:** Tri-state RequiresApproval + Classification (profile contexts + approval isolation), ProfileApprovalGate implementation, declarative attendance orthogonal to context, fail-closed defaults, explicit breaking-change migration paths.

**Phase 2 scoped:** Accept with modifications. Extension prerequisites (gert-vscode reproducibility, engines.vscode floor, test harness) + core work (output contract enforcement). Estimate 10–14 days parallelized.

**Critical finding (Systemic Bug #13):** Mutation testing on gert-vscode exposed stale build artifacts pattern — 13th bug class in engagement. Binding rule: regenerate artifacts after source mutations. Committed to all agents.

---

## 2026-08-17 — Runtime Portability Negotiation Concluded

**Status:** Design agreed, gate closed, Phase 1 scope finalized.

**Negotiation:** Four-round multi-agent negotiation with SQL Live-Site Operations on runtime portability for Gert runbooks (rounds 2–4; round 1 committed as d355bf5).

**Key technical outcomes:**
- AllowedEnvironments/AllowedModes field separation (profile contexts vs RunMode)
- Approval gate coverage closure (direct-invocation path enforcement added to Phase 1)
- Grandfathering rule for interactive unspecified (legacy requires-approval: false acts as read-only override)
- Declared attendance orthogonal to context (fixes TTYOutput conflation, resolves CI blocking-on-stdin bug)
- Test-context binding restriction at plan time (native-only default, subprocess opt-in, mcp-http never)
- OQ2 closed as subprocess continuation with Phase 0 framing deliverables

**Agents involved:** Barbara (architect), Don (backend verification), David (round 1).

**Design principle:** Fail-closed by default; explicit migration paths for breaking changes; source-grounded verification.

**Impact:** Phase 1 addition ~4 days. All blocking conditions satisfied. Implementation starts this week.

---

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

---

## ARCHIVED — Phase 1B Detailed Entry Log

Detailed entries from 2026-08-17 (Phase 1B planning and execution) archived to history-archive.md:
- Phase 1B Plan Issued (Scope Correction)
- Phase 1B Implementation: Items 1–6 progress
- Phase 1B Rev 1–3 Planning notes
- Phase 1B Completion Note
- Phase 2 Architectural Evaluation

**Key metrics:** 5 of 6 Phase 1B items delivered. Conformance corpus: 72/72 pass (Item 4 pending). External blocker: SQL Live-Site credential provisioning.

---

## 2026-08-18 — SYSTEMIC BUG CLASS #13 BINDING RULE: Stale Build Artifacts

**Alert:** All agents performing verification on this engagement must regenerate build artifacts after every source mutation before trusting test results.

**Finding:** Ken's mutation testing on gert-vscode (commits 93214d0, d173ad7+130b81e, f667f0a) exposed a 13th distinct bug class: tests import from gitignored `out/` directory. A source mutation without recompile yields stale `out/`, making mutation testing evidence unreliable in both directions (false green + false red).

**Binding rule for verification:**
1. After source edit (mutation or revert), regenerate build artifact (e.g., `npm run compile`, `tsc`, `cargo build`, `make`).
2. Run test suite AFTER artifact regeneration.
3. Only then trust the pass/fail result.

**Scope:** All repositories where:
- Tests import from gitignored build output (e.g., `out/`, `dist/`, `build/`, `target/`)
- Build artifact is NOT automatically regenerated before tests (fixed in gert-vscode via pretest hook)
- Mutation testing is used as primary verification

**For architecture review:** Mutation-testing evidence is load-bearing on this engagement. Any claim that a test validates a commit must explicitly state whether the build artifact was regenerated after the source edit. Unmarked or ambiguous evidence should be questioned.