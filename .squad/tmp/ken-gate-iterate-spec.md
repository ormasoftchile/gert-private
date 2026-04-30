# Architecture Specification: Gate and Concurrent Iterate Features

**Author:** Ken (Software Architect)  
**Date:** 2026-04-28  
**Status:** SPECIFICATION READY FOR IMPLEMENTATION  
**Implementor:** Brian  

---

## Executive Summary

This spec defines the precise architecture for two v2 features:

1. **`gate: stop_if:` on `type: include` steps** — Conditional termination when child runbooks resolve with specific outcome codes
2. **`concurrency:` on `iterate:` nodes** — Parallel worker-pool execution of iteration loops

Both features are validated patterns from gert-for-reference (the legacy prototype) and must maintain the same semantics while integrating cleanly with v2's execution model, trace event contract, and governance primitives.

---

## Feature 1: `gate: stop_if:` on `type: include` Steps

### 1.1 Overview

A **gate** is a conditional termination pattern that allows a parent runbook to "absorb" specific child outcomes without propagating them as failures. When an included child runbook terminates with an outcome code matching the gate's `stop_if` list, the parent stops execution cleanly rather than failing.

**Use case:** A parent runbook delegates escalation logic to a child. If the child resolves the issue (`category: "resolved"`), the parent should stop without further steps. If the child escalates (`category: "escalated"`), the parent should continue to its own escalation steps.

### 1.2 Schema Extension

**Add to `IncludeConfig` (pkg/schema/steps.go:33):**

```go
type IncludeConfig struct {
    Runbook string            `yaml:"runbook"        json:"runbook"`
    With    map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
    When    string            `yaml:"when,omitempty" json:"when,omitempty"`
    Gate    *GateSpec         `yaml:"gate,omitempty" json:"gate,omitempty"` // NEW
}

// GateSpec configures conditional parent termination based on child outcome.
type GateSpec struct {
    StopIf []string `yaml:"stop_if" json:"stop_if"` // List of outcome codes to absorb
}
```

**YAML Example:**

```yaml
- step:
    id: delegate_to_connectivity
    type: include
    include:
      runbook: connectivity_test
      with:
        service_name: "{{ .service_name }}"
      gate:
        stop_if: [resolved, dns_fixed]
```

### 1.3 Execution Semantics

#### 1.3.1 Normal Flow (No Gate)

1. Parent runs `type: include` step
2. Child runbook executes to completion
3. Child terminates with `type: end` step declaring `outcome: { category: "X", code: "Y" }`
4. Parent step completes successfully
5. Parent continues to next step

#### 1.3.2 Gate Trigger Flow

**Preconditions:**
- `gate.stop_if` is non-nil and non-empty
- Child runbook terminates normally (not error/timeout)
- Child's final outcome code is in `gate.stop_if` list

**When gate triggers:**

1. **Step completes successfully** — The include step returns `StepStatusCompleted` (not failed)
2. **Remaining steps are skipped** — The parent's step sequence terminates immediately
3. **Parent run terminates** — The parent run enters terminal state with the **child's outcome** propagated verbatim
4. **Trace event emitted** — `step.completed` event includes `gate_triggered: true` field
5. **Run event emitted** — `run.completed` event records the child's outcome as the parent's final outcome

**Outcome Propagation:**
- The parent run's final outcome is **exactly the child's outcome** (category + code)
- This allows grandparent gates to evaluate against the original child outcome, not a synthetic code

#### 1.3.3 Gate Does Not Trigger

**When:**
- Child's outcome code is **not** in `stop_if` list
- Execution continues to next step normally
- No special events emitted

#### 1.3.4 Error Cases (Gate Bypass)

**Gate is NOT consulted when:**

1. **Child run fails** (step execution error, panic, tool crash)
   - Propagates as normal step failure
   - Parent's `continue_on_fail` or `retry` logic applies
   
2. **Child run times out** (exceeds step timeout)
   - Propagates as timeout failure
   - Parent's timeout handler applies

3. **Child run is cancelled** (context cancellation)
   - Propagates as cancellation
   - No gate evaluation

**The gate ONLY evaluates successful child terminations with explicit outcome codes.**

#### 1.3.5 Nested Gates

**Scenario:** Parent has gate, grandparent has gate.

```
grandparent.yaml (gate: [critical_resolved])
  └─ include parent.yaml (gate: [minor_resolved])
       └─ include child.yaml → terminates with "critical_resolved"
```

**Behavior:**
1. Child terminates with `outcome: { code: "critical_resolved" }`
2. Parent's gate checks: "critical_resolved" NOT in `[minor_resolved]` → gate does NOT trigger
3. Parent's include step completes successfully
4. Parent continues to next step OR terminates with child's outcome
5. Grandparent's gate checks: "critical_resolved" IN `[critical_resolved]` → gate TRIGGERS
6. Grandparent terminates with child's outcome

**Rule:** Each level evaluates its own gate independently. Outcomes propagate up the chain until a gate absorbs them or the root is reached.

### 1.4 Trace Event Contract

#### 1.4.1 Step Completion Event

When a gate triggers, the `step.completed` event MUST include:

```json
{
  "type": "step.completed",
  "step_id": "delegate_to_connectivity",
  "status": "completed",
  "gate_triggered": true,
  "child_outcome": {
    "category": "resolved",
    "code": "dns_fixed"
  },
  "timestamp": "2026-04-28T10:15:30Z"
}
```

**New field:** `gate_triggered: bool` (omitted or false when gate does not trigger)

#### 1.4.2 Run Completion Event

When a parent run terminates due to gate trigger:

```json
{
  "type": "run.completed",
  "run_id": "run-abc123",
  "status": "completed",
  "outcome": {
    "category": "resolved",
    "code": "dns_fixed"
  },
  "terminated_by_gate": true,
  "timestamp": "2026-04-28T10:15:31Z"
}
```

**New field:** `terminated_by_gate: bool` (distinguishes gate termination from normal end-of-flow)

### 1.5 Implementation Checklist

**Parser (v2/pkg/parser):**
- [ ] Parse `gate:` block on `type: include` steps
- [ ] Validate `stop_if` is array of strings
- [ ] Reject empty `stop_if` array (must have at least one code)

**Planner (v2/internal/planner):**
- [ ] No changes required (gate is runtime concern, not planning concern)

**Executor (v2/internal/executor/include.go):**
- [ ] Currently a no-op stub — MUST be rewritten to:
  - [ ] Invoke child runbook via engine
  - [ ] Capture child's final outcome
  - [ ] Evaluate gate condition
  - [ ] Signal parent termination when gate triggers
- [ ] Return special error/signal to indicate "gate triggered, stop parent"

**Engine (v2/pkg/engine):**
- [ ] Detect gate-triggered signal from include executor
- [ ] Skip remaining steps
- [ ] Transition run to terminal state
- [ ] Emit trace events with `gate_triggered` and `terminated_by_gate` fields

**Schema (pkg/schema):**
- [ ] Add `GateSpec` type
- [ ] Add `Gate *GateSpec` field to `IncludeConfig`

**Tests:**
- [ ] Unit test: gate triggers on matching outcome
- [ ] Unit test: gate does not trigger on non-matching outcome
- [ ] Unit test: gate bypass on child error
- [ ] Integration test: nested gates (parent + grandparent)
- [ ] Trace test: verify `gate_triggered` and `terminated_by_gate` events

---

## Feature 2: `concurrency:` on `iterate:` Nodes

### 2.1 Overview

A **concurrent iterate** node executes loop iterations in parallel using a worker pool of size N. This enables high-throughput batch operations (e.g., restarting 100 services with 10 concurrent workers) while maintaining the same semantics as sequential iteration for variable capture, collect aggregation, and until conditions.

**Use case:** Restart 50 services in parallel, collect success/failure status per service, aggregate into a summary list.

### 2.2 Schema Extension

**Add to `IterateNode` (pkg/schema/steps.go:148):**

```go
type IterateNode struct {
    ID          string            `yaml:"id"                     json:"id"`
    Over        string            `yaml:"over,omitempty"         json:"over,omitempty"`
    As          string            `yaml:"as,omitempty"           json:"as,omitempty"`
    Max         int               `yaml:"max,omitempty"          json:"max,omitempty"`
    Until       string            `yaml:"until,omitempty"        json:"until,omitempty"`
    Collect     map[string]string `yaml:"collect,omitempty"      json:"collect,omitempty"`
    Concurrency int               `yaml:"concurrency,omitempty"  json:"concurrency,omitempty"` // NEW
    Steps       []FlowNode        `yaml:"steps"                  json:"steps"`
}
```

**Semantics:**
- `Concurrency == 0` or `Concurrency == 1` → Sequential execution (current behavior, no change)
- `Concurrency > 1` → Worker pool of size N

**YAML Example:**

```yaml
- iterate:
    id: restart_services
    over: $.service_list
    as: svc
    concurrency: 10
    collect:
      results: "{{ .svc }}: {{ .status }}"
    steps:
      - step:
          id: restart
          type: cli
          cli:
            command: systemctl restart {{ .svc }}
```

### 2.3 Worker Pool Execution Model

#### 2.3.1 Sequential Baseline (Concurrency ≤ 1)

Current behavior (already implemented in `iterate.go`):

```
for i, item := range items:
    workingVars[loopVar] = item
    workingVars["iteration"] = i + 1
    results = runner(ctx, steps, workingVars)
    aggregate results into collect map
    check until condition
    check max iterations
```

#### 2.3.2 Concurrent Model (Concurrency > 1)

**Worker Pool Pattern:**

1. Create buffered channel `workQueue` with capacity = `len(items)`
2. Spawn N goroutines (workers) that consume from `workQueue`
3. Each worker:
   - Receives `(index, item)` tuple
   - Creates isolated variable scope: `iterVars := copyVars(parentVars)`
   - Sets `iterVars[loopVar] = item` and `iterVars["iteration"] = index + 1`
   - Executes `runner(ctx, steps, iterVars)`
   - Sends result to `resultsChannel`
4. Main goroutine:
   - Pushes all items to `workQueue`
   - Receives results from `resultsChannel`
   - Aggregates collect variables (thread-safe)
   - Waits for all workers to complete
5. Once all workers finish, evaluate final state

**Pseudo-code:**

```go
workQueue := make(chan workItem, len(items))
resultsChannel := make(chan iterationResult, len(items))
var wg sync.WaitGroup

// Spawn workers
for w := 0; w < spec.Concurrency; w++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for work := range workQueue {
            iterVars := copyVars(vars)
            iterVars[loopVar] = work.item
            iterVars["iteration"] = work.index + 1
            results, err := e.runner(ctx, spec.Steps, iterVars)
            resultsChannel <- iterationResult{index: work.index, results: results, err: err}
        }
    }()
}

// Enqueue work
for i, item := range items {
    workQueue <- workItem{index: i + 1, item: item}
}
close(workQueue)

// Collect results
go func() {
    wg.Wait()
    close(resultsChannel)
}()

var mu sync.Mutex
collected := map[string][]any{} // collect key → list of values
allVars := map[string]any{}
status := engine.StepStatusCompleted

for result := range resultsChannel {
    mu.Lock()
    // Aggregate captured variables
    for k, v := range result.vars {
        allVars[k] = v
    }
    // Aggregate collect expressions
    for key, val := range result.collected {
        collected[key] = append(collected[key], val)
    }
    // Track failure
    if result.status == engine.StepStatusFailed {
        status = engine.StepStatusFailed
    }
    mu.Unlock()
}

// Final result
finalResult := newResult(step, status)
finalResult.Vars = allVars
for k, v := range collected {
    finalResult.Vars[k] = v // list of all collected values
}
```

### 2.4 Error Semantics

**Decision:** **Fail-fast with cancellation**

When any worker encounters an error:

1. **Cancel context** — Signal all other workers to stop
2. **Wait for in-flight workers** — Allow graceful shutdown (workers check `ctx.Done()`)
3. **Report first error** — Return the first error encountered (deterministic via result ordering)
4. **Partial results discarded** — Do NOT aggregate collect variables from incomplete runs

**Rationale:**
- **Fail-fast** aligns with GERT's governance model — errors should halt execution immediately
- **Partial results are unreliable** — If 3 of 10 workers fail, the remaining 7 may not represent valid state
- **Collect semantics preserved** — Collect variables are only meaningful when ALL iterations succeed

**Alternative rejected:** Collect-all-errors mode would require complex error aggregation and ambiguous collect semantics (partial vs. full results).

### 2.5 `collect:` Thread Safety

**Current sequential behavior:**

```go
for i, item := range items {
    // ... execute iteration ...
    for key, tmpl := range spec.Collect {
        val := evaluateTemplate(tmpl, workingVars)
        collected[key] = val // ❌ OVERWRITES previous value
    }
}
```

**Problem:** `collected[key] = val` overwrites the previous iteration's value.  
**Expected:** `collected[key]` should be a **list** of all per-iteration values.

**Corrected sequential behavior:**

```go
collected := map[string][]any{} // key → list of values
for i, item := range items {
    for key, tmpl := range spec.Collect {
        val := evaluateTemplate(tmpl, workingVars)
        collected[key] = append(collected[key], val)
    }
}
// Final: result.Vars[key] = collected[key] (list)
```

**Concurrent behavior with mutex:**

```go
var mu sync.Mutex
collected := map[string][]any{}

// Inside worker goroutine:
for key, tmpl := range spec.Collect {
    val := evaluateTemplate(tmpl, iterVars)
    mu.Lock()
    collected[key] = append(collected[key], val)
    mu.Unlock()
}
```

**Ordering guarantee:** Results are aggregated in the order they are **received**, not the order they are **started**. This is acceptable because collect lists are inherently unordered (workers finish non-deterministically).

**If order matters:** Users should include the iteration index in the collected value:

```yaml
collect:
  results: "{{ .iteration }}: {{ .status }}"
```

### 2.6 `until:` with Concurrency

**Sequential behavior:**

```go
for i, item := range items {
    // execute iteration
    if evalCondition(spec.Until, workingVars) {
        break // stop iterating
    }
}
```

**Concurrent behavior:**

**Option A:** Ignore `until` in concurrent mode (simplest, recommended)
- `until` is meaningless when workers run in parallel (no ordering)
- Parser should **reject** runbooks with `until` AND `concurrency > 1`
- Error message: "until condition is not supported with concurrent iteration"

**Option B:** Poll `until` after each worker completes (complex, deferred)
- Check `until` after receiving each result
- If true, cancel remaining workers
- Requires ordering assumptions (which worker's vars to evaluate?)

**Decision:** **Option A — Reject `until` + `concurrency` at parse time**

**Rationale:**
- `until` implies sequential short-circuit logic
- Concurrent workers have no ordering — "stop after first success" is ambiguous
- Clean error is better than ambiguous runtime behavior

### 2.7 Variable Isolation

**Challenge:** Prevent captured variables from colliding across iterations.

**Sequential behavior:**

```go
workingVars := copyVars(vars) // single shared scope
for i, item := range items {
    workingVars[loopVar] = item // overwrite each iteration
    results := runner(ctx, steps, workingVars)
    for k, v := range results.Vars {
        workingVars[k] = v // captured vars accumulate
    }
}
```

**Concurrent behavior:**

```go
// Each worker gets isolated scope
for work := range workQueue {
    iterVars := copyVars(parentVars) // fresh copy per iteration
    iterVars[loopVar] = work.item
    iterVars["iteration"] = work.index
    results := runner(ctx, steps, iterVars)
    // DO NOT merge iterVars back into parentVars
    // Only aggregate collect expressions and final vars
}
```

**Rules:**
1. **Loop variable (`as: svc`)** — Fresh copy per iteration, no collision risk
2. **Captured variables (`capture: { status: "$.exitcode" }`)** — Aggregated into final result, last-writer-wins (acceptable for status variables)
3. **Collect expressions** — Aggregated into lists (thread-safe with mutex)

**Parent scope isolation:** Workers cannot modify the parent's variable scope (only their isolated copy).

### 2.8 Trace Event Contract

#### 2.8.1 Sequential Events (Current)

```json
{ "type": "iterate.started", "iterate_id": "restart_services" }
{ "type": "step.started", "step_id": "restart", "iteration": 1 }
{ "type": "step.completed", "step_id": "restart", "iteration": 1 }
{ "type": "step.started", "step_id": "restart", "iteration": 2 }
{ "type": "step.completed", "step_id": "restart", "iteration": 2 }
{ "type": "iterate.completed", "iterate_id": "restart_services", "iterations": 2 }
```

#### 2.8.2 Concurrent Events (New)

**Iteration events are interleaved (non-deterministic order):**

```json
{ "type": "iterate.started", "iterate_id": "restart_services", "concurrency": 10 }
{ "type": "step.started", "step_id": "restart", "iteration": 1, "worker_id": 0 }
{ "type": "step.started", "step_id": "restart", "iteration": 2, "worker_id": 1 }
{ "type": "step.completed", "step_id": "restart", "iteration": 2, "worker_id": 1 }
{ "type": "step.started", "step_id": "restart", "iteration": 3, "worker_id": 1 }
{ "type": "step.completed", "step_id": "restart", "iteration": 1, "worker_id": 0 }
{ "type": "step.completed", "step_id": "restart", "iteration": 3, "worker_id": 1 }
{ "type": "iterate.completed", "iterate_id": "restart_services", "iterations": 3 }
```

**New fields:**
- `concurrency: int` on `iterate.started` event (omitted if sequential)
- `worker_id: int` on per-iteration events (0-indexed worker number, omitted if sequential)

**Ordering:** Events are emitted in the order workers complete, NOT the order iterations started.

### 2.9 Implementation Checklist

**Parser (v2/pkg/parser):**
- [ ] Parse `concurrency:` field on `iterate:` nodes
- [ ] Validate `concurrency >= 0`
- [ ] Reject `until` + `concurrency > 1` combination

**Schema (pkg/schema):**
- [ ] Add `Concurrency int` field to `IterateNode`

**Executor (v2/internal/executor/iterate.go):**
- [ ] Detect `spec.Concurrency > 1` and branch to concurrent path
- [ ] Implement worker pool pattern (goroutines + channels)
- [ ] Add mutex-protected collect aggregation
- [ ] Emit `worker_id` in per-iteration events
- [ ] Implement fail-fast cancellation on error
- [ ] Preserve sequential behavior when `concurrency <= 1`

**Engine (v2/pkg/engine):**
- [ ] No changes required (iterate executor handles concurrency)

**Tests:**
- [ ] Unit test: sequential execution (concurrency = 0)
- [ ] Unit test: sequential execution (concurrency = 1)
- [ ] Unit test: concurrent execution (concurrency = 5)
- [ ] Unit test: collect aggregates into lists (concurrent)
- [ ] Unit test: fail-fast on first error (concurrent)
- [ ] Unit test: reject `until` + `concurrency` at parse time
- [ ] Integration test: 100 iterations with concurrency = 10
- [ ] Trace test: verify `worker_id` and `concurrency` fields in events
- [ ] Race detector: run all concurrent tests with `-race` flag

---

## Cross-Feature Concerns

### 3.1 Gate + Concurrent Iterate Interaction

**Scenario:** Parent has concurrent iterate → each iteration includes child runbook with gate

```yaml
- iterate:
    id: check_hosts
    over: $.hosts
    concurrency: 5
    steps:
      - step:
          id: diagnose
          type: include
          include:
            runbook: host_diagnostic
            gate:
              stop_if: [healthy]
```

**Behavior:**
- Each worker iteration is independent
- If child terminates with "healthy", that **iteration's include step** completes successfully
- The iteration does NOT terminate the parent iterate (gate only affects the iteration scope)
- Gate termination at the parent run level requires the gate to be on a top-level include, NOT inside an iterate

**Rule:** Gates operate at the **run scope**, not the **iteration scope**. A gate inside an iterate terminates the iteration, not the parent run.

### 3.2 Governance Interaction

**Approval gates on concurrent iterate:**

```yaml
- iterate:
    id: deploy_services
    concurrency: 3
    steps:
      - step:
          type: approve
          approvals:
            roles: [release_manager]
```

**Behavior:**
- Each iteration requires separate approval (3 concurrent approval requests)
- Approval gate blocks the iteration, not the entire iterate node
- If any approval is denied, that iteration fails (triggers fail-fast cancellation)

**Recommendation:** Place approval gate **before** iterate, not inside.

---

## Acceptance Criteria

### Feature 1: Gate

- [ ] Parser rejects `stop_if: []` (empty list)
- [ ] Gate triggers when child outcome matches `stop_if` code
- [ ] Gate does NOT trigger when child outcome does not match
- [ ] Gate bypassed when child fails (error, timeout, cancellation)
- [ ] Parent run terminates with child's outcome when gate triggers
- [ ] Trace events include `gate_triggered` and `terminated_by_gate` fields
- [ ] Nested gates (parent + grandparent) evaluate independently
- [ ] Integration test: 3-level nesting (grandparent, parent, child)

### Feature 2: Concurrency

- [ ] Sequential behavior unchanged when `concurrency <= 1`
- [ ] Worker pool spawns N goroutines when `concurrency = N`
- [ ] Collect variables aggregate into lists (not overwritten)
- [ ] Fail-fast cancels remaining workers on first error
- [ ] Parser rejects `until` + `concurrency > 1`
- [ ] Trace events include `worker_id` and `concurrency` fields
- [ ] Race detector passes on all concurrent tests
- [ ] Integration test: 100 iterations with concurrency = 10 completes successfully

---

## Open Questions (Deferred to Implementation)

1. **Gate on other step types?** (e.g., `type: tool` with outcome codes)
   - **Answer:** Deferred to v2.1. Include-only for v2.0.

2. **Gate with multiple child outcomes?** (e.g., child calls multiple includes)
   - **Answer:** Gate only evaluates the **final** outcome of the child run.

3. **Concurrency rate limiting?** (e.g., `concurrency: 10, max_per_second: 2`)
   - **Answer:** Deferred to v2.1. Worker pool only for v2.0.

4. **Ordered collect with concurrency?** (e.g., preserve iteration order in results)
   - **Answer:** Not supported in v2.0. Users should embed iteration index in collect expressions if order matters.

---

## References

- **Legacy prototype:** `gert-for-reference/examples/incident-runbook.yaml` (gate usage)
- **Legacy prototype:** `gert-for-reference/examples/service-restart.yaml` (concurrency usage)
- **v2 schema:** `pkg/schema/steps.go`
- **v2 executor:** `internal/executor/iterate.go` (sequential baseline)
- **v2 events:** `pkg/schema/event.go` (trace event types)

---

## Revision History

| Date       | Author | Change                          |
|------------|--------|---------------------------------|
| 2026-04-28 | Ken    | Initial specification (v1.0)    |
