# barbara

## Critical Lessons (must re-read every spawn)

- **Mutation proofs must target the production wiring, not a pure helper.** This mistake has happened twice on this engagement: a test mutated a standalone helper function and declared the behavior covered, but the call site in the production path was never exercised. Mutation evidence is only load-bearing when the mutated path is the one actually exercised by the running system.
- **Never cite a repo "precedent" without grepping for it first.** SQL Live-Site cited "Petals" as a proven pattern; the name appeared in documentation but no implementation was found in the actual codebase. Citing an unverified precedent wastes a round-trip and can justify a wrong design. Always grep before claiming a pattern exists.
- **`npm test` against stale `out/` invalidates mutation testing in both directions.** See Systemic Bug Class #13 below.

## Team Updates

**2026-08-19T17:32:28-07:00 — serve/requires package-map contract ruling:** Issued Barbara ruling in `.squad/decisions/inbox/barbara-serve-requires-contract-ruling.md`. Verdict: current `serve` rejection of package-map `requires:` is deliberate implementation guard, but durable dialect split is a contract/spec defect. `tool-paths:` cannot preserve `^1.0.0` package constraints; Don's conversion is a temporary live-fire unblock only. Long-term target: serve parity with run/plan package catalog resolution, or a formally distinct serve-only schema with conformance vectors until parity lands.

**2026-08-19T17:03:22-07:00 — Ledger archival gate ruling:** Issued Barbara ruling to archive completed effort bodies, not dated headings. Safe blocks: lines 29–1338, 1339–2309, and 2512–3350. Keep active VS Code MCP bridge/live ICM execution material live, especially lines 2310–2511 and 3351–4560. Governing rule: effort-completion is authoritative; size thresholds trigger review/stop-merge, not automatic archival.

**2026-08-19T16:13:34Z — From Scribe:** Ken's `/probe-token` removal completed successfully in gert-vscode. Probe was dangerous because all 4 diagnostic attempts were unconditional; VS Code's MCP client exhausted its restart budget after 4 consecutive failures and disabled the session. The measurement was complete (T1 ≡ T2/T3/T4 failure profile) — probe has no further diagnostic value. All decisions merged to decisions.md with full binding rules for future MCP work.

---

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
---

## 2026-08-19 — Systemic Bug Class #16: Chat Command with No Handler Body (5th Occurrence)

**Defect:** The /run chat command was declared in package.json but had no handler implementation in src/extension.ts. The manifest entry remained in place, so VS Code routed all @gert /run requests to the extension, but the handler body fell through isArmCommand() to "Unknown command" for every invocation since the Petals lifecycle port (commit 742e368).

**Why it shipped:** All 169 unit tests passed before this discovery. None exercise the extension.ts handler body, which requires a live VS Code host to invoke. The handler absence was therefore invisible to the test suite.

**Discovery:** Ken's root-cause analysis during the ICM MCP blocker investigation (2026-08-19).

**Impact:** The single most important code path (@gert /run) was non-functional for every user since the Petals port. This is the **5th occurrence** of the vacuity defect class on this engagement — a pattern indicating systematic test-harness/coverage gaps in the extension layer.

**Lesson for future designs:** Chat commands declared in package.json require end-to-end tests that actually invoke the VS Code host, not just unit tests of the handler function in isolation. Manifest declarations are not a substitute for handler-path coverage.


## 2026-08-19 — serve/requires contract ruling

Barbara ruled that `gert serve` rejecting package-map `requires:` is a deliberate implementation guard, but the durable `run`/`plan`/`serve` dialect split is a contract and spec defect. `tool-paths:` cannot preserve package version constraints, identity, provenance, export surface, or digest semantics; the tactical serve map is an emergency unblock only.

Required design follow-up: specify serve package-map semantics in `gert-private`, preferably parity with run/plan per-run catalog resolution; otherwise formally name and constrain a serve-only restricted dialect and add schema/conformance vectors. This also reinforces the engagement's false-green/vacuity pattern: success must prove the load-bearing path, not just a nearby startup condition.
