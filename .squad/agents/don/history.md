## 2026-08-17 — Runtime Portability Negotiation Concluded

**Status:** Design agreed, gate closed, Phase 1 scope finalized.

**Negotiation:** Four-round multi-agent negotiation with SQL Live-Site Operations on runtime portability for Gert runbooks (rounds 2–4; round 1 committed as d355bf5).

**Key technical outcomes:**
- AllowedEnvironments/AllowedModes field separation (profile contexts vs RunMode)
- Approval gate coverage closure (direct-invocation path enforcement added to Phase 1)
- Grandfathering rule for interactive unspecified (legacy 
equires-approval: false acts as read-only override)
- Declared attendance orthogonal to context (fixes TTYOutput conflation, resolves CI blocking-on-stdin bug)
- Test-context binding restriction at plan time (native-only default, subprocess opt-in, mcp-http never)
- OQ2 closed as subprocess continuation with Phase 0 framing deliverables

**Agents involved:** Barbara (architect), Don (backend verification), David (round 1).

**Design principle:** Fail-closed by default; explicit migration paths for breaking changes; source-grounded verification.

**Impact:** Phase 1 addition ~4 days. All blocking conditions satisfied. Implementation starts this week.


## Learnings

Session: SQL Live-Site Condition Verification (2026-08-17T06:56:00-07:00)

- GovernanceEvaluator is never assigned in any production EngineConfig construction (wire.go:149, run.go:390). The governance pre-flight at engine.go:456 is behind a nil guard that is always nil. Plain tool steps have NO approval gate firing today — not from runbook-level, not from tool-level. My earlier round-2 description was describing the intended code path, not the currently-wired one.
- Plan.Governance (built from runbook governance by planner.go:110) is used ONLY for post-execution redaction (engine.go:641), not for pre-flight approval.
- Substitution path (pkgsubst.go:276) uses strict OR for RequireApproval: eff.RequireApproval || subRequireApproval. Runbook-level false cannot suppress tool-level true. Composition is monotone-increasing.
- No retry or idempotency logic anywhere keys off RequiresApproval. The two concepts are orthogonal.
- Phase 1 constraint: when wiring GovernanceEvaluator for plain steps, must pass both runbook and per-tool configs to BuildPolicy(configs...). Naively passing only runbook config would make tool-level requires-approval: true invisible. Existing variadic API supports this — it is a call-site change only.

Session: Slice 1 — Governance Schema Types (2026-08-17T07:05:00-07:00)

- Implemented three additive schema changes: RequiresApproval bool→*bool, Classification *string on ToolAction, AllowedModes []string on ToolGovernance.
- Assessment findings were accurate. Exactly one Go read site for schema.ToolGovernance.RequiresApproval: EffectiveGovernanceFromTool() at pkgsubst.go:359. Updated to nil-guard deref. No other read sites in production code.
- pkg/pkgsubst/pkgsubst.go was part of the 138 uncommitted pre-existing files — staged only our edit but git tracked the whole file as a new commit object. Expected and harmless.
- tv-enum.yaml's 25 requires-approval: false entries: confirmed empirically via conformance tests. They unmarshal to *bool pointing to false (&false = explicit opt-out). No fixture changes needed.
- Classification validation follows the existing ParseToolFile → helper pattern from validateActionEnums(). New validateActionClassifications() in scan.go, called inline after validateActionEnums(). Matches the convention exactly.
- Case (b) test (governance present, no requires-approval → nil) is the critical counterparty-caught defect. Covered and passing.
- go build ./... and go test ./... both pass clean. Zero failures across entire suite including internal/conformance.
- ORTHOGONALITY: no code derives Classification from RequiresApproval or vice versa. Enforced in comments and confirmed by absence of any coupling in the codebase.
