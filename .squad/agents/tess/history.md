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

## Current Work (2026-08-17) — Contract Parity Harness + Proof

### Consumer Request Item 1: Reusable Contract-Parity Harness — SHIPPED (Commit a94f35a)

**Status:** COMPLETE — `go build ./pkg/contractparity/... exit 0`, `go test ./pkg/contractparity/... -count=1 -v` 30/30 PASS.

**Deliverable:** `pkg/contractparity` — importable, fixture-agnostic harness.

**Exported API:**
- `Binding{Name, Invoke func, Meta}` — one transport binding
- `ActionMeta{Classification, RequiresApproval, Idempotent *T}` — declared governance fields (all pointer; nil = skip)
- `Report{Violations []Violation}` — structured diff (usable outside tests)
- `CompareOutputs`, `CompareMeta` — pure comparison functions
- `Check` — invokes both bindings, returns combined Report
- `AssertParity` / `RequireParity` — thin `testing.TB` adapters

**Negative controls (11 tests):** each injects a deliberate violation (missing key, extra key, type mismatch, meta mismatch, invoke error, adapter marks-failed) and asserts detection with "NEGATIVE CONTROL FAILED" message when broken. Mirrors `TestCredentialLeak_SweepDetectsIntentionalLeak` rigor.

### Parity Proof: ops-synthetic native vs mcp-http — SHIPPED (Commit 05fd13b)

**Status:** COMPLETE — 4 tests PASS, 0 FAIL.
**File:** `internal/tool/contractparity_proof_test.go` (package `tool`, new file)

**Comparisons made and outcome:**
1. Action A (status-query): native vs mcp-http — **PASS** (identical 5-field shape + classification)
2. Action B match-found: native vs mcp-http — **PASS** (matched+pattern+count)
3. Action B no-match: native vs mcp-http — **PASS** (matched-only, no pattern key)
4. Negative control (divergent stub with extra key): **CORRECTLY DETECTED** — harness reports `[extra_output_key] field="internal_debug_flag"`

**No divergence found:** both bindings are genuinely contract-identical. The no-match shape comparison is particularly important — the `pattern` field is correctly absent on both sides.

**Key finding:** `NativeCLITransport` leaves `Output=nil` (does not parse JSON). The native `Invoke` wrapper must parse `result.Stdout` explicitly to produce `map[string]any`. This is by design — the harness's `Invoke` signature (`func(ctx, args) (map[string]any, error)`) delegates that responsibility to the caller.

## Current Work (2026-08-17) — Runtime Output Contract Enforcement

### Task: Enforce declared returns: for non-substituted tool actions — SHIPPED (Commit ca867a1)

**Status:** COMPLETE — `go build ./... exit 0`, `go test ./... exit 0`, 82 packages.

**Defect closed:** `returns:` / `outputs:` on non-substituted `.tool.yaml` actions (native, mcp-stdio, mcp-http) was never checked at runtime. Only substituted (execute.kind: runbook) actions enforced output contracts. Capture expressions silently resolved to nil; gert reported success.

**Deliverables:**
- `enforceOutputContract()` added to `internal/executor/tool.go`
- Reuses `coerceOutputAny` so both enforcement paths (substituted + non-substituted) agree on type coercion
- `processLevelOutputs` rule: stdout/stderr/exit_code always excluded from enforcement (process envelope, not semantic contract)
- Empty/absent `outputs:` = unconstrained (0 existing tools affected)
- 16 tests in `tool_output_contract_test.go`; mutation controls measured (see learnings)
- Reachability entry: `ToolAction.Outputs (non-substituted enforcement)` → `statusReachable`

**Pre-existing tests modified:** None. All 82 packages passed unchanged.

## Learnings

**Spy testing.TB pattern:** To assert a testing adapter calls t.Error/Fatal, wrap `testing.TB` with a `spyTB` struct that intercepts calls without calling `runtime.Goexit`. Fatal override must NOT forward — letting the test continue allows inspection of `spy.failed`. This is the correct pattern for testing harness adapters.

**Mutation control measurement — PowerShell regex is unreliable for multi-line Go:** PowerShell `-replace` with multi-line patterns fails silently due to CRLF vs LF and greedy matching issues. Use Python `re.sub(..., re.DOTALL)` for reliable multi-line substitutions in mutation tests. ALWAYS measure, never reason.

**makeWorkDir returns a relative path:** `os.MkdirTemp(".", ...)` produces a relative path. If you call `chdirForTest(t, dir)` with that path and then try `filepath.Join(dir, "file")`, the join is relative to the new cwd — wrong. Call `filepath.Abs(dir)` immediately after `makeWorkDir` before any chdir.

**gert's "complete: status=completed" appears even when a step fails:** The engine prints its run summary on stdout/stderr regardless of step outcome. A run can print `complete: status=completed steps=N` and still exit with code 1 (exitFailure) if a step failed with a fatal error. Do not mistake the summary text for the process exit code.

**NativeCLITransport leaves res.Output nil:** The native transport captures stdout as plain text only; it never parses JSON into Output. Any native tool with `outputs:` declared will always fail enforcement. This is correct by design — native tools producing structured output should use mcp transport. The `enforceOutputContract` unconstrained-escape (nil declaredOutputs) handles ALL existing tools because no existing .tool.yaml declares outputs: on non-substituted actions.

**git checkout -- during mutation testing wipes uncommitted working-tree changes:** When applying a mutation with Python and then restoring with `git checkout -- <file>`, the restore targets HEAD — not the pre-mutation working tree state. If the working tree changes were not yet committed, `git checkout --` wipes them. Always commit (or stash) before applying mutations. Future self: apply mutations as additive edits and revert them explicitly, or commit the production change first then mutate a copy.

**GCP-PARSE-001 widening required by runtime enforcement:** Shipping `enforceOutputContract` (ca867a1) without widening the planner check would have made declared outputs enforceable at runtime but uncapturable at plan time — worse than either alone. The planner validation (`stepBindsDeclaredOutput`) must be shipped in the same logical feature set.

**Pure-function + thin-adapter split:** Separate the comparison logic (`CompareOutputs`, `CompareMeta`, `Check`) from the `testing.T` binding (`AssertParity`, `RequireParity`). This makes the core logic testable without `*testing.T` and importable in non-test code.

**CompareMeta nil-skipping:** When a metadata field is nil on either side, skip comparison rather than reporting a violation. The consumer deliberately opts out of asserting fields they don't declare.

**NativeCLITransport leaves Output nil:** The native transport emits JSON to stdout but does not parse it. Any parity proof wrapping a native binding must parse `result.Stdout` explicitly into `map[string]any` before passing to the harness.

**Parity proof placement in `internal/tool`:** When unexported test helpers (like `newOpsSyntheticMCPServer`) are needed, placing the proof in the same package (new file, not editing existing files) is the correct move. The harness's public API is still used exclusively — no harness internals are touched.

## Learnings (historical)

**High-entropy sentinels required for credential sweep:** David's knownToken = "***" is too weak (asterisks appear in truncated error strings). Use TESS_SENTINEL_TOKEN_D42E9B1C pattern.

**Assertion shape: unconditional error required.** After fixing warn-and-continue to fatal error, use if err == nil { t.Fatal } not if err != nil — latter silently passes on regression.

**Pre-staged git index files are a hazard.** Always verify git diff --cached --name-only before commit. Unstage unwanted files with git restore --staged -- <path>.

**Empirically verify runtime behavior.** TTYOutput is hardcoded true in CLI, so TerminalApprovalGate always fires in subprocess (not auto-approve). Vector design must account for actual behavior, not theory.

## Archive

Phase 1 runtime portability design complete (33 rulings, 31 conformance vectors). Phase 1B additions: Items 1–6 coordination pending completion.
