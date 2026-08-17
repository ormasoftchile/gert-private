# Tess — History Summary

**Role:** Conformance engineer, test infrastructure specialist

## Current Work (2026-08-17)

### Phase 1B Item 3: INDETERMINATE Semantics — SHIPPED (Commit 22ad3e7)

**Status:** COMPLETE — go build ./... exit 0, go test ./... exit 0.

**Deliverables:**
- StepStatusIndeterminate and RunStatusIndeterminate added
- IndeterminateRecord struct with 7 evidence fields (pointer on StepResult per Rule B)
- AcknowledgeIndeterminate bool on RunOptions; sentinel errors for halt/resume guard
- Idempotent *bool added to ToolAction
- Classification-aware timeout branching:
  - read-only flows to failRun
  - mutating/destructive/unspecified flows to haltIndeterminate
  - Same logic on step-result-level transport loss
- Resume guard: blocked with ErrIndeterminateAcknowledgmentRequired when needed
- 30 test vectors (INDET-001 through INDET-014) all PASS

**Orthogonality constraint:** RequiresApproval never consulted in timeout routing — vectors INDET-012 prove this.

**EndpointHost invariant:** Parsed hostname only. Test sweeps for token-leak with high-entropy sentinel (non-vacuous).

### Phase 1A Work Summary

- MCP HTTP Stream D complete: 31 adversarial tests PASS
- Token-leak sweep verified clean across all surfaces
- Conformance corpus design (71 vectors total: 60 pass, 11 skipped with reasons)
- Dynamic Runbook Includes feature (2026-08-15) feature-complete and approved

## Key Design Decisions

**Ratified Rule B:** IndeterminateRecord pointer is unambiguous signal — nil Output is legitimately returned by tools, but nil pointer means "completion unknown."

**Classification enforcement:** Structural, not approval-routed. Function requiresIndeterminate takes only classification pointer; approval state is never passed. Any signature change breaks INDET-012 regression guard.

**Orchestration step verification:** The actual entrypoint is internal/engine/engine.go, not pkg/engine/engine.go (public interface only). Always grep before trusting line-number citations.

**Transport-loss dual paths:** Some executors return (nil, context.DeadlineExceeded) while others return (&StepResult{Status: Failed}, nil). INDETERMINATE logic must intercept both paths.

## Learnings

**High-entropy sentinels required for credential sweep:** David's knownToken = "***" is too weak (asterisks appear in truncated error strings). Use TESS_SENTINEL_TOKEN_D42E9B1C pattern.

**Assertion shape: unconditional error required.** After fixing warn-and-continue to fatal error, use if err == nil { t.Fatal } not if err != nil — latter silently passes on regression.

**Pre-staged git index files are a hazard.** Always verify git diff --cached --name-only before commit. Unstage unwanted files with git restore --staged -- <path>.

**Empirically verify runtime behavior.** TTYOutput is hardcoded true in CLI, so TerminalApprovalGate always fires in subprocess (not auto-approve). Vector design must account for actual behavior, not theory.

## Archive

Phase 1 runtime portability design complete (33 rulings, 31 conformance vectors). Phase 1B additions: Items 1–6 coordination pending completion.
