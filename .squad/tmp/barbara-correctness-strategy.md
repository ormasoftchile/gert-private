# Gert v2 Correctness Strategy

**Author:** Barbara (Integrations Specialist)  
**Date:** 2026-04-19  
**Purpose:** Define how spec compliance is verified through testing at every phase of gert v2 development.

---

## 1. Spec as Law

**Core Principle:** Every normative statement in `design/gert-v2/spec/*.md` (all MUST/MUST NOT/SHALL rules) has a corresponding executable test. The spec files are not documentation—they are contracts.

### Enforcement Pattern

Each spec file's normative rules map to tagged test cases:

```go
// spec: 03-schema-vnext.md §4.2.1 — MUST reject invalid step kind
func TestParser_RejectsInvalidStepKind(t *testing.T) {
    input := `apiVersion: runbook/v2
steps:
  - id: test
    kind: unknown_kind`
    
    _, err := parser.Parse(context.Background(), strings.NewReader(input), ParseOptions{})
    require.Error(t, err)
    var validationErr *ValidationError
    require.ErrorAs(t, err, &validationErr)
    assert.Contains(t, validationErr.Message, "invalid step kind")
}
```

**Tag convention:** `// spec: {file} §{section} — {RULE_TEXT}`

This creates a traceable link: spec rule → test → implementation. During Ken's review, he can grep for spec tags and verify coverage.

### Spec Compliance Tracking

A spec coverage tool (Phase 0 deliverable) extracts all MUST/MUST NOT/SHALL statements from spec files and cross-references test tags:

```bash
$ gert dev spec-coverage
Spec Compliance Report
======================
03-schema-vnext.md
  §4.2.1 MUST reject invalid step kind ✅ (engine_test.go:142)
  §4.2.3 MUST validate step ID uniqueness ✅ (parser_test.go:89)
  §5.1.2 MUST preserve apiVersion field ⚠️  (NO TEST FOUND)
  
Coverage: 47/53 rules (89%)
Missing: 6 rules have no tagged tests
```

**Exit criteria for Phase 0:** Spec coverage tool exists and runs in CI as a warning (not blocking). By Phase 13, coverage MUST be ≥95%.

---

## 2. Test Types by Phase

Each phase uses a specific test mix based on what it delivers:

| Phase | Primary Test Types | Secondary Tests |
|-------|-------------------|-----------------|
| **Phase 0: Foundation** | Unit (pure functions), Interface contract | None |
| **Phase 1: Parser** | Unit (table-driven schema validation), Contract (Parser interface) | Golden schema fixtures |
| **Phase 2: Planner** | Unit (import/tool resolution), Contract (Planner interface) | Integration (parser → planner) |
| **Phase 3: Runtime Core** | Contract (Executor interface, event ordering), Unit (state machine) | **Golden trace** (first use) |
| **Phase 4: Governance** | Unit (allowlist/denylist), Contract (governance hooks) | Integration (governance + runtime) |
| **Phase 5: Step Types** | Unit per step type, Contract (StepExecutor interface × 14) | Integration (all step types in corpus) |
| **Phase 6: Tool Runtime** | Contract (stdio/jsonrpc/mcp transports), Unit (capability gate) | Integration (tool invocation E2E) |
| **Phase 7: Extension Host** | Contract (handshake protocol, Ed25519 trust), Unit (manifest validation) | Integration (extension lifecycle) |
| **Phase 8: Input Providers** | Contract (provider/v2 interface), Unit (built-in providers) | Integration (choice/decision/collector) |
| **Phase 9: gert serve** | Contract (exec/v2 RPC, events/v2 WebSocket), Unit (RBAC) | Integration (TLS, WebSocket reconnect) |
| **Phase 10: Adapters** | Contract (adapter uses only public contracts), Integration (TUI + VSCode E2E) | None |
| **Phase 11: Evidence** | Unit (snapshot write/read, HMAC chaining), **Golden trace** (HMAC verification) | Integration (resume, replay) |
| **Phase 12: Observability** | Unit (OTel span emission), Contract (Prometheus metrics) | Integration (`gert verify` golden) |
| **Phase 13: Acceptance** | **Acceptance corpus** (all runbook types), **Golden trace** (v1 corpus) | Performance benchmarks |

### Test Type Definitions

- **Unit:** Pure logic, mocked dependencies, <10ms per test. Table-driven preferred.
- **Contract:** Verifies interface implementations satisfy the spec (ordering guarantees, field presence, error types). Uses conformance helper functions.
- **Integration:** Real components wired together, tempdir I/O, <500ms per test. Build tag `//go:build integration`.
- **Golden trace:** Deterministic trace comparison (normalized timestamps/IDs). Uses `testdata/golden/*.jsonl`.
- **Acceptance corpus:** Full runbook execution with assertions from `testdata/scenarios/{name}/test.yaml`.

---

## 3. Spec Compliance Tests

### Tagging Pattern

Every test that validates a normative spec rule carries a `// spec:` comment linking to the source:

```go
// spec: 05-tool-runtime.md §3.1 — MUST enforce capability gate before tool invocation
func TestToolRuntime_CapabilityGateEnforced(t *testing.T) { ... }

// spec: 07-security-and-trust.md §4.2 — MUST verify Ed25519 signature for remote extensions
func TestExtensionHost_VerifiesSignature(t *testing.T) { ... }

// spec: 12-evidence-tracing-resumption.md §2.3 — MUST emit run/started as first event
func TestRuntime_EmitsRunStartedFirst(t *testing.T) { ... }
```

### Coverage Tool

The `gert dev spec-coverage` tool (written in Phase 0) does:

1. Parse all `spec/*.md` files using CommonMark AST
2. Extract sentences containing MUST/MUST NOT/SHALL (RFC 2119 keywords)
3. Extract `// spec:` tags from all `*_test.go` files
4. Cross-reference: for each spec rule, find matching test tag
5. Report: covered rules (✅), uncovered rules (⚠️), orphaned tags (❌)
6. Exit 0 if coverage ≥95%, exit 1 otherwise (used in CI gate for Phase 13)

**Phase 0 deliverable:** `internal/dev/speccoverage/` package + `gert dev spec-coverage` subcommand.

---

## 4. Golden Traces

### What is a Golden Trace?

A deterministic, normalized JSONL trace file stored in `testdata/golden/{scenario}.jsonl` that represents the expected output for a specific runbook scenario.

**Example:** `testdata/golden/linear-cli-steps.jsonl`

```json
{"event_id":"00000000-0000-0000-0000-000000000001","sequence":0,"kind":"run/started","timestamp":"2026-01-01T00:00:00Z","run_id":"test-run-001","path":null,"payload":{"runbook_id":"linear.runbook.yaml","user":"test","mode":"real","inputs":{},"client":"cli"}}
{"event_id":"00000000-0000-0000-0000-000000000002","sequence":1,"kind":"step/started","timestamp":"2026-01-01T00:00:01Z","run_id":"test-run-001","path":["steps",0],"payload":{"step_id":"echo-hello","step_type":"cli","step_name":"Echo Hello"}}
{"event_id":"00000000-0000-0000-0000-000000000003","sequence":2,"kind":"step/completed","timestamp":"2026-01-01T00:00:01Z","run_id":"test-run-001","path":["steps",0],"payload":{"step_id":"echo-hello","exit_status":0,"duration_ms":5,"captures":{},"evidence":[]}}
{"event_id":"00000000-0000-0000-0000-000000000004","sequence":3,"kind":"run/completed","timestamp":"2026-01-01T00:00:02Z","run_id":"test-run-001","path":null,"payload":{"outcome":"passed","steps_passed":1,"steps_failed":0}}
```

**Normalized fields:**
- `event_id`: Deterministic UUID sequence (`00000000-...`)
- `timestamp`: Fixed base time + offset (preserves relative timing)
- `run_id`: Fixed `test-run-001`
- `duration_ms`: Optional normalization for flaky timing-dependent tests

### Golden Trace Test Pattern

```go
func TestGolden_LinearCLISteps(t *testing.T) {
    runID, err := exec.RunFixture(
        t.TempDir(),
        "testdata/runbooks/linear.runbook.yaml",
        WithInputs(map[string]string{}),
        WithScenario("testdata/scenarios/linear/scenario.yaml"),
    )
    require.NoError(t, err)
    
    actual := testutil.ReadTrace(t, runID)
    normalizedActual := testutil.NormalizeTrace(actual)
    
    golden := testutil.ReadGoldenTrace(t, "linear-cli-steps")
    
    if *update {
        testutil.WriteGoldenTrace(t, "linear-cli-steps", normalizedActual)
    }
    
    assert.Equal(t, golden, normalizedActual, "trace differs from golden")
}
```

**Update workflow:** `go test -update` regenerates golden files. Reviewers inspect the diff before merge.

### HMAC Golden Traces (Phase 11+)

When `GERT_TRACE_KEY` is set, golden traces include the `sig` field:

```json
{"event_id":"...","sequence":0,"kind":"run/started",...,"sig":"a3f8c9d2..."}
```

**HMAC verification test:**

```go
// spec: 12-evidence-tracing-resumption.md §7.2 — MUST verify HMAC chain integrity
func TestHMACChain_GoldenTrace(t *testing.T) {
    key := []byte("test-signing-key")
    trace := testutil.ReadGoldenTrace(t, "hmac-chain")
    
    err := verify.HMACChain(trace, key)
    require.NoError(t, err, "HMAC chain verification failed")
}
```

**Tampering detection test:**

```go
func TestHMACChain_DetectsTampering(t *testing.T) {
    trace := testutil.ReadGoldenTrace(t, "hmac-chain")
    // Modify second event payload
    trace[1].Payload["step_id"] = "tampered"
    
    err := verify.HMACChain(trace, []byte("test-signing-key"))
    require.Error(t, err)
    assert.Contains(t, err.Error(), "HMAC mismatch at sequence 1")
}
```

### When to Use Golden Traces

**Use golden traces when:**
- The scenario is fully deterministic (no real time, no randomness, no external I/O)
- The trace structure itself is what you're validating (event ordering, field presence, HMAC chain)
- Regression detection is critical (any trace change should be reviewed)

**Don't use golden traces when:**
- Scenario has non-deterministic outputs (real timestamps, random IDs, network calls)
- You only care about specific events (use `AssertEvent` helpers instead)
- The test is already covered by contract tests

---

## 5. Phase Gates

Each phase has a **Definition of Done** that includes:

1. **All deliverables complete** (packages, files, interfaces listed in PLAN.md)
2. **Spec compliance checklist passed** (phase-specific subset of spec rules)
3. **Test coverage ≥80%** for new code (enforced by CI)
4. **Ken's architectural review approved** (spec alignment, interface contracts)
5. **Integration tests green** (if applicable to phase)
6. **No regressions** in prior phases' tests

### Phase-Specific Checklists

Each phase's exit criteria include a **spec compliance checklist**. Example for Phase 3 (Runtime Core):

**Phase 3 Exit Checklist:**

- [ ] `run/started` emitted as first event (spec: §12 §2.3)
- [ ] `run/completed` or `run/failed` emitted as final event (spec: §12 §2.4)
- [ ] No events emitted after terminal event (spec: §12 §2.5)
- [ ] Event `sequence` field is monotonically increasing (spec: §12 §1.3)
- [ ] Trace file is crash-safe (atomic append with fsync) (spec: §12 §7.1)
- [ ] Redaction applied before trace write (spec: §11 §3.2)
- [ ] Event bus delivers events in order (spec: §06 §2.1)

These checklists are derived from PLAN.md's "Spec References" column and expanded with specific MUST rules from the referenced spec sections.

### Ken's Review Criteria (Per Phase)

Ken reviews each phase against the spec before approving. He checks:

1. **Interface correctness:** Do Go interfaces match spec contracts? (Method signatures, error types, ordering guarantees)
2. **Dependency direction:** Does the dependency graph follow §02 Architecture rules? (No upward imports)
3. **Completeness:** Are all spec rules for this phase implemented, or are gaps explicitly marked with `// TODO(v2.1)`?
4. **Test coverage:** Do tagged tests exist for all MUST rules in this phase's spec sections?
5. **Error handling:** Are all error paths spec-compliant? (Structured errors, correct error codes from §13)
6. **Normative alignment:** Does the code faithfully implement the spec's intent, or does it introduce silent divergence?

**Ken's approval gates the next phase.** If Ken finds spec drift, the phase is not done.

---

## 6. Brian's TDD Workflow

**Brian is the Go programmer implementing each phase.** His workflow is spec-driven TDD:

### Step-by-Step Loop

1. **Read the spec section** for the current phase (e.g., Phase 3 → read `spec/06-runtime-events.md`, `spec/12-evidence-tracing-resumption.md`)

2. **Extract MUST rules** from the spec. Example:
   - "The runtime MUST emit `run/started` as the first event"
   - "The `sequence` field MUST be monotonically increasing"
   - "No events MUST be emitted after the terminal event"

3. **Write failing tests** for each MUST rule, tagged with `// spec:`:

   ```go
   // spec: 12-evidence-tracing-resumption.md §2.3 — MUST emit run/started first
   func TestRuntime_EmitsRunStartedFirst(t *testing.T) {
       bus := newTestEventBus()
       rt := NewRuntime(...)
       rt.Execute(ctx, runbook, bus)
       
       events := bus.Events()
       require.NotEmpty(t, events)
       assert.Equal(t, "run/started", events[0].Kind)
   }
   ```

4. **Run the tests** → they fail (red)

5. **Implement the minimum code** to make the tests pass:

   ```go
   func (r *Runtime) Execute(ctx context.Context, rb *Runbook, bus EventBus) error {
       // Emit run/started first
       bus.Publish(Event{Kind: "run/started", Sequence: 0, ...})
       
       // Execute steps...
       
       return nil
   }
   ```

6. **Run the tests again** → they pass (green)

7. **Refactor** if needed (simplify, extract helpers, improve naming)

8. **Repeat for next MUST rule** until all rules in the spec section are covered

9. **Run spec coverage tool** to verify all MUST rules have tagged tests:

   ```bash
   $ gert dev spec-coverage --phase 3
   Phase 3 Spec Compliance: 12/12 rules covered (100%)
   ```

10. **Submit for Ken's review** with evidence: tests pass, coverage complete, no spec drift

### Workflow Discipline

- **Never implement without a failing test first** (except trivial getters/setters)
- **Always tag tests with `// spec:` comments** linking to the normative rule
- **Run `gert dev spec-coverage` before submitting** to catch uncovered rules
- **If a spec rule is ambiguous, ask Ken** (don't guess and diverge)

---

## 7. Ken as Spec Reviewer

**Ken (Software Architect) is the spec authority.** Every phase ends with Ken's review before it's considered done.

### What Ken Reviews

1. **Spec alignment**
   - Does the implementation faithfully match the spec's contracts?
   - Are all MUST rules from this phase's spec sections implemented?
   - Are there silent divergences (code does something the spec doesn't mention)?

2. **Interface contracts**
   - Do Go interfaces match the spec's contract definitions?
   - Are method signatures correct? (Parameters, return types, error semantics)
   - Are ordering guarantees preserved? (E.g., events in sequence order)

3. **Dependency graph**
   - Does the code follow §02 Architecture's dependency rules?
   - Are there upward imports violating the layering?
   - Are adapter dependencies leaking into the core?

4. **Test coverage**
   - Does `gert dev spec-coverage --phase N` show ≥95% coverage for this phase?
   - Are there MUST rules with no tagged tests?
   - Are contract tests present for all interfaces?

5. **Error handling**
   - Are structured errors used consistently?
   - Do error codes match §13 Adapter Contracts registry?
   - Are error messages redacted before logging/tracing?

6. **Completeness**
   - Are all deliverables from PLAN.md present? (Packages, files, interfaces)
   - Are deferred features marked with `// TODO(v2.1)` and documented?
   - Is the phase exit checklist 100% complete?

### Ken's Approval Criteria

Ken marks a phase as **APPROVED** only when:

- ✅ All spec rules for this phase have tagged tests
- ✅ All tests pass (unit, contract, integration)
- ✅ No dependency violations detected
- ✅ No ambiguous or divergent implementations
- ✅ Phase exit checklist is 100% complete
- ✅ Ken's spot-check of 3-5 random spec rules shows faithful implementation

**If Ken finds issues:** Phase is marked **NEEDS REVISION**, Brian fixes the gaps, and re-submits.

---

## Summary

The gert v2 Correctness Strategy ensures every line of code is traceable to a spec rule and validated by a test:

1. **Spec as law:** Every MUST rule has a tagged test; spec coverage tool enforces ≥95% by Phase 13.
2. **Test types by phase:** Unit → Contract → Integration → Golden trace → Acceptance corpus, matched to each phase's deliverables.
3. **Spec compliance tests:** All tests carry `// spec: {file} §{section}` tags for traceability.
4. **Golden traces:** Deterministic JSONL comparisons with HMAC verification (Phase 3+). Updated via `go test -update`.
5. **Phase gates:** No phase is done until tests pass, checklist complete, coverage ≥80%, and Ken approves.
6. **Brian's TDD workflow:** Read spec → extract MUST rules → write failing tests → implement → pass → refactor → submit to Ken.
7. **Ken as spec reviewer:** Ken verifies spec alignment, interface correctness, dependency rules, test coverage, and completeness before approving each phase.

**Result:** Spec-compliant, correct-by-construction code with audit-grade traceability from requirements to tests to implementation.
