# Testing and Acceptance

This section specifies the complete testing strategy for gert v2: the test framework, unit and contract test patterns, integration test strategy, the `gert test` feature, regression testing against the v1 runbook corpus, performance acceptance criteria, the extension test harness, and CI/CD gate requirements. All acceptance criteria must be satisfied before a v2 release is considered shippable.

---

## Test Framework

### Standard Library and testify

All gert v2 tests are written in Go using the standard `testing` package as the test runner and `github.com/stretchr/testify` for assertions:

- `testify/assert` — non-fatal assertions; test continues on failure.
- `testify/require` — fatal assertions; test halts on failure.
- `testify/mock` — interface mocking for unit tests (engine, event bus, trace writer).

Table-driven tests are the preferred pattern for all logic with well-defined input/output pairs (e.g., expression evaluation, schema validation, governance rule matching):

```go
func TestGovernanceAllowlist(t *testing.T) {
    cases := []struct {
        name    string
        cmd     []string
        allowed bool
    }{
        {"exact match",    []string{"kubectl", "get", "pods"}, true},
        {"partial match",  []string{"kubectl", "delete", "ns"}, false},
        {"empty allowlist", []string{"echo", "hi"}, false},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            result := allowlist.Check(tc.cmd)
            assert.Equal(t, tc.allowed, result)
        })
    }
}
```

### Test Packages

Each Go package ships a co-located `_test.go` file (white-box tests) and optionally an integration test file (gated by build tag):

```
pkg/engine/engine.go
pkg/engine/engine_test.go           // white-box unit tests
pkg/engine/engine_integration_test.go  // build tag: //go:build integration
```

---

## The `gert test` Feature

### Scenario Replay Testing

`gert test <runbook.yaml>` discovers and executes all scenario tests for a runbook. A scenario test consists of:

- **`inputs.yaml`** — resolved input values for the run.
- **`scenario.yaml`** (optional) — pre-recorded command responses and manual evidence.
- **`test.yaml`** — assertions over the resulting trace and captures.

Scenario directories are located by convention:

```
{runbook-dir}/scenarios/{runbook-name}/{scenario-name}/
  inputs.yaml
  scenario.yaml   # optional; omit for runbooks with no CLI/tool steps
  test.yaml
```

The test runner executes the runbook in replay mode using the scenario file as the command/evidence fixture, then evaluates assertions from `test.yaml`.

### Test Assertions (`test.yaml`)

```yaml
assertions:
  outcome: passed           # "passed" | "failed" | "skipped"
  steps_passed: 5
  steps_failed: 0

  captures:
    db_host:
      equals: "prod-db.example.com"
    migration_status:
      contains: "complete"

  trace:
    - type: step/completed
      step_id: run-migration
    - type: governance/blocked
      step_id: drop-table
      data.rule: denylist
```

- `trace` assertions match events in order of occurrence.
- `data.*` dotted paths dereference into the event's `data` object.

### v1 Scenario Compatibility

v1 scenario files (`commands`/`evidence` structure) are forward-compatible with v2 without modification. The v2 parser accepts both the v1 format and the extended v2 format (which adds the `assertions` block). Runbook authors can migrate incrementally by adding `test.yaml` alongside existing v1 scenarios.

---

## Contract Tests

Contract tests verify the interface boundaries between v2's five components. Each contract test lives in the *consumer* package and uses a conformance helper that validates any implementation satisfying the interface.

### Core Runtime Contract

The runtime `Executor` interface MUST satisfy:
- Emits a `run/started` event before executing any step.
- Emits a `step/started` before and `step/completed`/`step/failed` after each step.
- Emits `run/completed` or `run/failed` as the final event.
- Does not emit events after the terminal event.
- Does not call any adapter method directly (events only).

```go
// ConformanceTest validates any Runtime implementation.
func ConformanceTest(t *testing.T, rt Runtime) {
    t.Helper()
    bus := newTestEventBus()
    err := rt.Execute(context.Background(), fixtureRunbook, bus)
    require.NoError(t, err)
    events := bus.Events()
    require.Equal(t, "run/started",   events[0].Type)
    require.Equal(t, "run/completed", events[len(events)-1].Type)
    assertNoEventsAfterTerminal(t, events)
}
```

### Schema Contract Tests

- Every `apiVersion: runbook/v2` document in `testdata/` MUST pass structural validation without errors.
- Every invalid fixture in `testdata/invalid/` MUST fail validation with a structured error (message + path).
- Structural and semantic validation are tested independently: a structurally valid document MAY fail semantic validation (e.g., referencing an undefined capture).

### Extension Manifest and Handshake Tests

- An extension process that responds to the initialisation handshake within the configured timeout is accepted.
- An extension process that fails to respond is terminated and its capabilities are not registered.
- Capability grants in the manifest are enforced: an extension without `cap:schema.register` MUST NOT register schema.
- A manifest with missing required fields is rejected at load time with a structured error.

### Tool Invocation Envelope Tests

- Tool invocations carry the correct `run_id` and `step_id` correlation fields.
- Tool outputs are captured and available as step captures after the step completes.
- A tool that exits with non-zero triggers `step/failed` with `error_type: tool_error`.
- Timeout cancellation: a tool that exceeds its timeout receives a `context.DeadlineExceeded` and emits `step/failed` with `error_type: timeout`.

### Event Bus Determinism Tests

- Events delivered to the event bus maintain the `seq` ordering guaranteed by `RunStore.WriteTrace`.
- Slow adapter consumers MUST NOT cause the engine's execution loop to block (bounded channel or dropping policy required).
- Event replay (`ReadTraceSince`) returns events in ascending `seq` order.

---

## Integration Test Strategy

### End-to-End Fixture Runs

Integration tests spin up the gert runtime against fixture runbooks in `testdata/runbooks/` and assert on the resulting trace files. Gated by the `integration` build tag and run in CI separately from unit tests.

```go
//go:build integration

func TestE2E_BranchingRunbook(t *testing.T) {
    dir := t.TempDir()
    runID, err := exec.RunFixture(dir, "testdata/runbooks/branch.runbook.yaml",
        WithInputs(map[string]string{"env": "prod"}),
        WithScenario("testdata/scenarios/branch-prod.yaml"),
    )
    require.NoError(t, err)
    trace := readTrace(t, dir, runID)
    assertEvent(t, trace, "step/completed", map[string]any{
        "data.step_id": "deploy-prod",
    })
    assertNoEvent(t, trace, "step/completed", map[string]any{
        "data.step_id": "deploy-staging",
    })
}
```

### Fixture Runbook Corpus

The integration test corpus MUST include fixtures covering:
- Linear runbook (all step types: cli, manual, tool, invoke).
- Branching runbook (both branch arms; governance-blocked branch).
- Iterate runbook (list mode, convergence mode, parallel concurrency).
- Nested invoke (parent + two child runbooks).
- Retry policy (transient failure followed by success).
- Resumption (crash after step 2, resume, verify trace continuity).
- Governance: allowlist violation, denylist violation, approval gate.

### Trace Assertion Helpers

Shared test package `pkg/testutil`:

```go
package testutil

// ReadTrace reads and parses all events from a run's trace.jsonl.
func ReadTrace(t *testing.T, runDir, runID string) []engine.DurableEvent

// AssertEvent asserts that at least one event matches type and data predicates.
func AssertEvent(t *testing.T, events []engine.DurableEvent,
    eventType string, predicates map[string]any)

// AssertEventOrder asserts events appear in the given type order.
func AssertEventOrder(t *testing.T, events []engine.DurableEvent,
    types ...string)

// AssertNoEvent asserts no event matches the given type and predicates.
func AssertNoEvent(t *testing.T, events []engine.DurableEvent,
    eventType string, predicates map[string]any)
```

---

## Regression Testing

### v1 Runbook Corpus as Fixtures

The v1 runbook corpus serves as a regression suite for v2 compatibility. The regression test runner:

1. Reads each v1 runbook from the corpus.
2. Runs `gert validate` (v2 schema) and asserts it passes (or passes after migration).
3. For runbooks with associated v1 scenario files, runs `gert test` and asserts the outcome matches the v1 golden output.
4. Fails the build if any previously passing corpus runbook regresses.

Migration compatibility is measured as the percentage of v1 corpus runbooks that pass v2 `gert validate` without modification. The v2 launch target is ≥95%.

### Golden Trace Comparisons

For deterministic scenarios, the expected trace is stored as a golden file. The golden comparison test:

1. Runs the scenario.
2. Normalises timestamps and run IDs (replaces with fixed values).
3. Compares the normalised JSON-lines output to `testdata/golden/<scenario>.jsonl`.
4. Fails if any event differs. Golden files are updated by running with `-update` flag.

---

## Performance Acceptance Criteria

The following latency budgets MUST pass on a reference machine (4-core, 8 GB RAM, SSD):

| Operation | p50 target | p99 target |
|---|---|---|
| `gert validate` (single runbook, <50 steps) | <50 ms | <200 ms |
| Step startup (cli step) | <10 ms | <50 ms |
| Event delivery to adapter (local channel) | <1 ms | <5 ms |
| Trace event write + fsync | <5 ms | <20 ms |
| Snapshot write (100 steps history) | <20 ms | <80 ms |
| `gert test` (10-step replay scenario) | <500 ms | <1 s |
| Tool invocation round-trip (stdio, local process) | <50 ms | <200 ms |

Performance benchmarks are written using Go's `testing.B` and run in CI weekly (not on every PR). A regression of >2× on any metric blocks the nightly build.

---

## Extension Test Harness

### Overview

Extension developers can test their extension logic without spinning up the full gert runtime. The `pkg/exttest` package provides a test harness:

```go
import "github.com/ormasoftchile/gert/v2/pkg/exttest"

func TestMyExtension(t *testing.T) {
    h := exttest.NewHarness(t,
        exttest.WithManifest("testdata/my-ext.manifest.yaml"),
        exttest.WithCapabilities("cap:schema.register", "cap:tool.register"),
    )
    defer h.Close()

    // Start the extension process
    h.Start("./my-extension")

    // Assert it registered expected tools
    tools := h.RegisteredTools()
    require.Contains(t, tools, "my-tool")

    // Invoke a tool and assert the response
    resp := h.InvokeTool("my-tool", map[string]any{"arg1": "value"})
    assert.Equal(t, 0, resp.ExitCode)
    assert.Contains(t, resp.Stdout, "expected output")
}
```

### Harness Capabilities

The test harness provides:
- A mock extension host that implements the full handshake protocol.
- Assertion helpers for registered tools, schema registrations, and event subscriptions.
- A `FakeRunContext` that provides a realistic run context without executing a runbook.
- Automatic process cleanup on test completion.
- Configurable capability grants (to test capability enforcement).

### Mocking the Extension Transport

For pure unit tests of extension logic without spawning a subprocess, the transport layer is mockable:

```go
// InProcessTransport runs the extension handler in-process for unit testing.
type InProcessTransport struct {
    Handler ExtensionHandler
}
```

---

## Acceptance Criteria

The following MUST be satisfied before a v2 release is considered shippable:

1. Core runtime has no direct adapter dependencies — verified by the `no_import` contract test.
2. Extensions register schema and tools through declared contracts — verified by the extension handshake conformance suite.
3. Capability violations fail fast with structured errors — verified by the governance contract test for each capability type.
4. Adapters integrate using the same stable core contracts — verified by running each adapter's integration test against a mock core.
5. Every step type (`cli`, `manual`, `tool`, `invoke`, `branch`, `iterate`) has ≥1 end-to-end integration test covering the happy path and at least one failure path.
6. The v1 runbook corpus passes `gert validate` at ≥95% without modification.
7. All performance targets in the Performance Acceptance Criteria section pass on the reference machine.
8. `gert test` successfully replays all scenario fixtures in `testdata/scenarios/`.

---

## CI/CD Integration

### PR Gate (every pull request)

All five gates MUST pass for a PR to be mergeable:

1. **Unit tests**: `go test ./...` (no build tags). Target: <60 s.
2. **Schema validation**: validate all YAML fixtures in `testdata/` against v2 JSON Schema. Target: <10 s.
3. **Contract tests**: `go test -run Contract ./...`. Target: <30 s.
4. **Linter**: `golangci-lint run` with the project's `.golangci.yaml` config.
5. **Vet**: `go vet ./...`.

### Integration Gate (merge to main)

1. **Integration tests**: `go test -tags=integration ./...`. Target: <5 min.
2. **Regression corpus**: `gert test` against v1 corpus. Target: <2 min.
3. **Golden trace comparison**: normalised trace comparison for all deterministic fixtures.

### Nightly Gate

1. **Performance benchmarks**: `go test -bench=. -benchtime=10s ./...` — compare against stored baseline; fail on >2× regression.
2. **Race detector**: `go test -race ./...`.
3. **Extension conformance suite**: run all built-in extensions through the conformance harness.
