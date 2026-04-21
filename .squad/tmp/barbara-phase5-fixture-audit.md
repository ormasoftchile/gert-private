# Phase 5 Fixture Coverage Audit

**Date:** 2026-04-19  
**Auditor:** Barbara (Integrations Specialist)  
**Task:** Phase 5 StepExecutor fixture coverage audit and new fixture creation

---

## Executive Summary

Audited 13 existing fixtures (r01-r13) and created 3 new fixtures (r14-r16) to ensure comprehensive coverage for Phase 5 StepExecutor integration tests. All 14 step types now have at least one fixture exercising them.

**Status:** ✅ All 14 step types covered  
**New fixtures created:** r14-assert-compensate, r15-branch-collector, r16-end-step  
**Parser validation:** All 16 fixtures parse successfully

---

## Step Type → Fixture Mapping

| Step Type | Fixtures | Coverage Notes |
|-----------|----------|----------------|
| **cli** | r01, r02, r03, r06, r07, r08, r09, r10, r11, r12, r13, r14, r15, r16 | ✅ Excellent - used extensively across all fixtures for system commands |
| **tool** | r01, r03 | ✅ Good - r01 exercises builtin tools (slack, pagerduty, alertmanager), r03 exercises tool invocation with capture |
| **include** | r04, r09 | ✅ Good - r04 includes sub-runbooks for SOC2 controls, r09 includes generic incident response |
| **choice** | r08 | ⚠️ Thin - r08 has single choice step for FDA submission type selection; could benefit from multi-choice scenario |
| **decision** | r13 | ✅ Good - r13 dedicated fixture for decision-based routing with goto paths |
| **collector** | r02, r03, r04, r06, r07, r08, r10, r15 | ✅ Excellent - r15 exercises multiple field types (text, select, boolean); r03 exercises field validation patterns |
| **branch** | r01, r02, r03, r06, r07, r08, r09, r10, r13, r15 | ✅ Excellent - r15 exercises conditional branching based on collector input |
| **iterate** | r02, r06, r09, r10, r11 | ✅ Excellent - r11 dedicated fixture for iterate over array with collect; r02 monitors canary metrics in loop |
| **parallel** | r01, r02, r03, r06, r10 | ✅ Excellent - r01 exercises parallel diagnostics gathering; r03 exercises parallel provisioning with join failure modes |
| **approve** | r12 | ✅ Good - r12 dedicated fixture for approval quorum (pool-based approval with required count) |
| **assert** | r02, r06, r14 | ✅ Good - **NEW r14** exercises assert with eq/contains types and verifies failure triggers compensation |
| **compensate** | r02, r06, r14 | ✅ Good - **NEW r14** dedicated fixture for compensation on failure with multi-step rollback |
| **wait_for_event** | r01 | ⚠️ Thin - r01 has single wait_for_event for Prometheus webhook; event filtering and timeout handling covered but no multi-event scenarios |
| **end** | r01, r02, r03, r04, r05, r06, r07, r08, r09, r10, r16 | ✅ Excellent - **NEW r16** dedicated fixture explicitly demonstrating end step with structured outcome |

---

## New Fixtures Created

### r14-assert-compensate

**Purpose:** Demonstrates `assert` and `compensate` step types  
**Location:** `testdata/runbooks/r14-assert-compensate/schema.yaml`

**Coverage:**
- `assert` step with multiple assertion types (`eq`, `contains`)
- `compensate` step registering multi-step rollback handler
- Compensation triggered on assertion failure
- Assert step captures and validates cli output

**Key features:**
- Compensate with `on: any` trigger (runs on any failure)
- Assert verifies deployment output matches expected values
- Rollback includes cache clearing

**Parser test:** `TestParser_FixtureR14_AssertCompensate` validates:
- CompensateSpec has 2 rollback steps
- AssertSpec has 2 assertions

---

### r15-branch-collector

**Purpose:** Demonstrates `collector` and `branch` step types with multi-field collection  
**Location:** `testdata/runbooks/r15-branch-collector/schema.yaml`

**Coverage:**
- `collector` step with 4 field types:
  - `select` field with multiple options
  - `text` field with hint
  - `boolean` field
  - Optional text field for conditional logic
- `branch` step with 3 conditional paths based on collector input
- Nested collector within branch arm (approval gate)
- Conditional step execution with `when` clause

**Key features:**
- Collector exercises environment selection (development/staging/production)
- Branch conditions use template expressions to route by environment
- Production branch requires approval before deployment
- Conditional monitoring step based on boolean field

**Parser test:** `TestParser_FixtureR15_BranchCollector` validates:
- CollectorSpec has 4 fields
- BranchSpec has 3 conditional arms

---

### r16-end-step

**Purpose:** Demonstrates `end` step type with explicit outcome  
**Location:** `testdata/runbooks/r16-end-step/schema.yaml`

**Coverage:**
- `end` step with structured outcome
- Outcome with `category` and `code` fields
- Explicit runbook termination

**Key features:**
- Simple workflow: init → execute → verify → end
- End step declares outcome category "resolved" with code "task_succeeded"
- Demonstrates explicit termination vs. implicit end-of-flow

**Parser test:** `TestParser_FixtureR16_EndStep` validates:
- EndSpec.Outcome.Category = "resolved"
- EndSpec.Outcome.Code = "task_succeeded"

---

## Parser Validation

All 16 fixtures parse successfully:

```bash
cd /Volumes/Projects/gert/v2
go test ./internal/parser/... -run TestParser_Fixtures -v
```

**Results:**
- ✅ r01-r13: All existing fixtures pass
- ✅ r14-assert-compensate: Parses successfully
- ✅ r15-branch-collector: Parses successfully
- ✅ r16-end-step: Parses successfully (fixed outcome schema - removed unsupported `message` field)

---

## Coverage Gaps Identified (Minor)

### 1. `choice` step - Single-scenario coverage

**Current:** r08 has one choice step for FDA submission type  
**Gap:** No fixture demonstrates:
- Choice step with >5 options
- Choice with dynamic options from external provider
- Choice with validation rules

**Recommendation:** Consider r17 fixture for complex choice scenarios in future phases

### 2. `wait_for_event` - Single-event coverage

**Current:** r01 has one wait_for_event for Prometheus alert  
**Gap:** No fixture demonstrates:
- Multiple event sources (webhook + polling)
- Event filter edge cases (array filters, nested JSON paths)
- on_timeout: skip vs. fail behavior comparison

**Recommendation:** Adequate for Phase 5; consider specialized event fixture if event handling becomes complex

### 3. `tool` step - Limited toolRef variety

**Current:** r01 and r03 use builtin tools (slack, pagerduty, okta, github)  
**Gap:** No fixture demonstrates:
- Custom tool with actions array
- Tool with OAuth auth flow
- Tool error handling and retries

**Recommendation:** Adequate for Phase 5; tool executor implementation will use fakes in unit tests

---

## Integration Test Strategy for Phase 5

Each StepExecutor will have:

1. **Unit tests** (in executor package):
   - Use fakes (FakeToolRegistry, FakeEventListener, etc.)
   - Test executor-specific logic in isolation
   - Fast, no I/O

2. **Integration tests** (using fixtures):
   - Load fixture runbooks via parser
   - Execute via planner + executor
   - Validate state transitions and outputs
   - Use these fixtures as integration test inputs

**Fixture usage per executor:**

| Executor | Primary Fixtures | Coverage |
|----------|------------------|----------|
| CliExecutor | r11, r14, r16 | Basic cli, cli with capture, cli in compensate |
| ToolExecutor | r01, r03 | Builtin tools, tool with capture |
| IncludeExecutor | r04, r09 | Sub-runbook inclusion, variable passing |
| ChoiceExecutor | r08 | Single choice, route selection |
| DecisionExecutor | r13 | Decision routing with goto |
| CollectorExecutor | r03, r15 | Field validation, multi-field collection |
| BranchExecutor | r13, r15 | Conditional branching, branch with nested steps |
| IterateExecutor | r02, r11 | Iterate over array, iterate with collect |
| ParallelExecutor | r01, r03 | Parallel execution, join failure modes |
| ApproveExecutor | r12 | Quorum approval |
| AssertExecutor | r02, r14 | Assert with failure triggers |
| CompensateExecutor | r02, r14 | Compensation registration and execution |
| WaitForEventExecutor | r01 | Event webhook, timeout handling |
| EndExecutor | r16 | Explicit end with outcome |

---

## Recommendations for Phase 5 Implementation

### 1. Executor test pattern

Each executor should follow this pattern:

```go
func TestCliExecutor_Execute(t *testing.T) {
    // Load fixture
    p := parser.New(platform.NewFakePlatform())
    pr := p.Parse(ctx, "testdata/runbooks/r11-iterate-loop/schema.yaml")
    
    // Find step to test
    step := findStep(pr.Runbook.Flow, "init")
    
    // Execute with fake dependencies
    executor := NewCliExecutor(fakePlatform, fakeLogger)
    result := executor.Execute(ctx, step, state)
    
    // Validate
    assert.Equal(t, StepStatusSuccess, result.Status)
    assert.Contains(t, state.Variables["init_msg"], "Starting")
}
```

### 2. Use testutil.Tag for spec traceability

Example:
```go
_ = testutil.Tag("05-executors.md", "§3.1", "CliExecutor MUST capture stdout when step declares capture field")
```

### 3. Parallel test execution

All 16 fixtures use `t.Parallel()` - executor integration tests should do the same for fast CI.

---

## Files Modified

### Created
- `/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/r14-assert-compensate/schema.yaml`
- `/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/r15-branch-collector/schema.yaml`
- `/Volumes/Projects/gert/design/gert-v2/testdata/runbooks/r16-end-step/schema.yaml`

### Modified
- `/Volumes/Projects/gert/v2/internal/parser/parser_test.go` — Added:
  - `TestParser_FixtureR14_AssertCompensate`
  - `TestParser_FixtureR15_BranchCollector`
  - `TestParser_FixtureR16_EndStep`

---

## Next Steps for Phase 5

1. ✅ **COMPLETE:** Fixture audit and gap filling
2. **IN PROGRESS:** Implement 14 StepExecutors (assigned to engineering team)
3. **TODO:** Write executor integration tests using these fixtures
4. **TODO:** Validate executor contracts (effects, idempotency) against governance rules

---

## Sign-Off

**Barbara (Integrations Specialist)**  
All 14 step types now have fixture coverage. Parser tests validate all 16 fixtures. Ready for Phase 5 executor implementation.

**Fixtures validated:** ✅ r01–r16 all parse successfully  
**Coverage:** ✅ 14/14 step types covered  
**Integration surface:** ✅ Ready for executor testing
