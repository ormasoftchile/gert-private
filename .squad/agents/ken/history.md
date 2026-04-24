# ken — History

## Core Context (Phases 1–17 Summary)

**Project:** gert — Governed Executable Runbook Engine (v2 redesign)  
**Mission:** Complete redesign applying v1 learnings; establish v2 as specification-driven reference  
**Role:** Architect (design review, correctness strategy, decision authority)

### Foundation

**Team:** Ken (Architect) + Brian (Implementation), Barbara (Integration), Dennis (Research), Leslie (Docs), Raphael (UI)

**Codebase:** 
- `design/gert-v2/` — LaTeX specification document
- `v2/` — Go implementation (parser, planner, engine, adapters)
- `domains/home/` — First official domain kit (gert-domain-home compiler)

### v1 → v2 Key Transition

**v1 Success Factors:**
- Governance (approval gates, allowlists, redaction) beats feature parity
- Evidence capture (append-only JSONL, SHA256, resumption) enables auditability
- Extension isolation (out-of-process .tool.yaml, .provider.yaml) prevents side effects

**v2 Goals:**
1. Retain all governance differentiators
2. Add OpenTelemetry hooks, W3C Trace Context
3. Clarify core/extension boundary via JSON-RPC
4. Support saga/compensation patterns (Temporal-style)
5. Policy-as-code path (OPA integration deferred to v2.1)

### Critical Architectural Decisions (Locked Phases 1–14)

**Execution Model:**
- Flat ExecutionPlan (no DAG); runtime makes branch/iterate decisions dynamically
- Single-run-per-process for v2.0 (concurrency deferred to v2.1)
- Client-driven step advancement (no auto-advance; explicit state machine)

**Governance:**
- Denylist wins over allowlist (deny-by-default for unknown commands)
- 5 primitives: allowlist, denylist, env-block, redaction, approval-gates
- Approval gate model: ordered checkpoints, timeout/escalation, trace audit trail

**Infrastructure:**
- Crash-safe trace: single `write(2)` + `fsync` per DurableEvent
- 14 normative event types; HMAC-SHA256 tamper evidence (optional)
- SSE for run output streaming; JWT bearer auth for `gert serve`
- Evidence types: text, checklist, attachment (SHA256 content-addressed)

### Design Document Sections

**Complete (locked):**
- §02 Architecture: five-component model, data flow, lifecycle, concurrency control
- §11 Governance: primitives table, checkpoint logic, approval UX, audit trail
- §12 Evidence & Tracing: JSONL format, event taxonomy, crash safety
- §08 Testing: scenario discovery, contract tests, golden trace comparison, CI/CD gates

**Pending:**
- Kit-Zero (DRI-specific) → being replaced by domain-agnostic Domain Kit Examples
- Policy-as-code (§10) → deferred to v2.1

### Phase Outcomes

| Phases | Work | Completion |
|--------|------|-----------|
| 1–7 | Governance, planner, parser, runtime core, events | ✅ Locked |
| 8–13 | Layers: input providers, HTTP/WS/SSE, evidence, OTLP, CLI | ✅ Complete |
| 14–17 | Integration testing, server hardening, security audits | ✅ Complete |
| 18+ | Domain kit integration (home, scheduler, mobile app) | 🔄 In progress |

---

## Phase 3 — Domain Home Compiler Integration (2026-04-23 to 2026-04-24)

### Status: ✅ Complete

Brian's integration test deliverables for gert-domain-home compiler are complete and all tests passing.

### Deliverables

1. **Integration Test Suite** — `domains/home/integration_test.go`
   - 3 integration tests (`//go:build integration` tag)
   - Proves compiled YAML is syntactically valid and structurally correct
   - All 7 compiled outputs (5 routines + 2 incident templates) validated
   - Pass rate: 100% ✅

2. **CLI Tool** — `domains/home/cmd/home-validate/main.go`
   - 69 lines
   - Compiles Property to YAML and pretty-prints
   - Enables human-friendly demo and manual validation

3. **Decision Record** — Merged to `.squad/decisions.md`
   - Documented GERT v2 entry points (Parser, Planner, Engine, E2E harness)
   - Explained integration approach rationale
   - Analyzed Go module boundary constraints (internal/ packages)
   - Recorded learnings and patterns

### GERT v2 Integration Points

**Parser Layer (`v2/pkg/parser`):**
- Two-phase validation: JSON Schema structural + semantic cross-field rules
- Entry: `Parser.ParseBytes(ctx, []byte) (*ParsedRunbook, error)`
- Exit: `*ParsedRunbook` wrapping `*schema.Runbook`

**Planner Layer (`v2/pkg/planner`):**
- Resolves includes, performs tool lookup, builds ExecutionPlan
- Entry: `Planner.Plan(ctx, *ParsedRunbook) (*ExecutionPlan, error)`
- Exit: `*engine.ExecutionPlan` (flat list of executable steps)

**Engine Layer (`v2/pkg/engine`):**
- Real runtime with event bus, trace writer, step executor
- Entry: `Engine.Start(ctx, *ExecutionPlan, RunOptions) (Handle, error)`
- Control: `Handle.Next(ctx) (*StepResult, error)` (client-driven)
- Output: `RunState` (Completed/Failed/Waiting/etc)

**E2E Harness Pattern** (`v2/internal/e2e/helpers_test.go`):
- Uses `platform.Real()` (not mocks)
- Wires via `adapter.BuildEngineConfig(ctx, WireOptions)`
- Temp dirs for trace/run artifacts
- Drives engine to completion via `for { Next(); if EOF { break } }`
- Validates trace JSONL events for lifecycle assertions

### Validation Strategy

**What we proved (parser boundary):**
- ✅ Compiled YAML is syntactically valid (no unmarshal errors)
- ✅ Structure is complete (all required fields present, correct types)
- ✅ Round-trip I/O preserves structure (write → read → parse)

**What we did NOT prove (requires internal packages):**
- ⚠️ JSON Schema validation (parser internals)
- ⚠️ Semantic cross-field validation (planner internals)
- ⚠️ Full engine execution (runtime internals)

**Rationale:** Compiler's responsibility is boundary correctness. GERT engine tests validate execution.

### Key Learnings

1. **Three-layer stack architecture** — Parser → Planner → Engine (clean separation)
2. **Go module boundaries strict** — internal/ packages only accessible to parent module; no workaround
3. **Adapter pattern** — Production config wiring via `adapter.BuildEngineConfig()`
4. **Pragmatic testing** — Test the boundary you control; trust downstream validation
5. **Parse-time validation sufficient** — Proves syntactic correctness (90% of value)

### Consequences

**Positive:**
- ✅ Integration tests prove compilation boundary correctness
- ✅ No subprocess complexity (no CLI dependency)
- ✅ Fast tests (no engine overhead)
- ✅ Clear separation of concerns (compiler vs. runtime)

**Gaps (acceptable for now):**
- ⚠️ No full JSON Schema validation (requires parser internals)
- ⚠️ No execution validation (requires engine internals)
- ⚠️ Manual testing still needed for full validation

### Future Work (Deferred)

1. **Full E2E test in v2 repo:** Create `v2/internal/e2e/domain_home_test.go` (can import internals from same module)
2. **CI smoke test:** Shell out to `gert validate` or `gert run --dry-run`
3. **Scheduler integration:** Wire compiler output into scheduler's policy merge

---

## Phase 4 — v2 Runbook Roundtrip Schema Conformance (2025-04-24)

### Status: ✅ Investigation Complete

Attempted to implement a roundtrip test that validates compiler output against the actual v2 schema types (`v2/pkg/schema.Runbook`). Discovered critical schema design constraint that prevents direct YAML unmarshaling.

### Deliverable

**Decision Record:** `.squad/decisions/inbox/ken-phase4-roundtrip.md`

### Key Finding: v2 Schema Inline Spec Design

**Discovery:** The v2 schema uses `yaml:",inline"` tags for step type-specific specs, which creates an incompatibility with standard `yaml.Unmarshal()`.

**Root Cause:**
```go
// v2/pkg/schema/step.go
type Step struct {
    ID    string   `yaml:"id"`
    Type  StepType `yaml:"type"`
    // ...
    CollectorSpec *CollectorSpec `yaml:",inline"`  // ← Inline
    ChoiceSpec    *ChoiceSpec    `yaml:",inline"`  // ← Inline
    DecisionSpec  *DecisionSpec  `yaml:",inline"`  // ← Inline
}

type CollectorSpec struct {
    Prompt string           `yaml:"prompt"`  // ← Conflicts when inlined
    Fields []CollectorField `yaml:"fields"`
}
```

Multiple inline specs have overlapping field names (`prompt`, `variable`, etc.). When `yaml.v3` encounters this, it panics:

```
panic: duplicated key 'prompt' in struct schema.Step
```

### v2 Parser Custom Unmarshal Logic

The v2 parser (`v2/internal/parser/unmarshal.go`) has **custom unmarshal logic** to handle this:

1. Decodes top-level runbook fields (excluding `flow`)
2. Extracts `flow` as raw `yaml.Node`
3. Custom `parseFlowNodes()` type-dispatches steps based on `type` field
4. Constructs `schema.Step` with only the appropriate spec populated

**Critical constraint:** This custom parser is in `v2/internal/parser` — **not accessible** from `domains/home`.

### Importability Assessment

| Package | Status | Reason |
|---------|--------|--------|
| `v2/pkg/schema` | ✅ Importable | Public package, well-structured |
| `v2/pkg/parser` (interface) | ✅ Importable | Public interface defined |
| `v2/internal/parser` (impl) | ❌ Not importable | Internal package |
| Parser factory | ❌ Not available | No public `New()` in `v2/pkg/` |

### Validation Coverage Comparison

| Validation Aspect | Phase 3 (map) | Phase 4 (attempted) | v2 Parser |
|------------------|---------------|---------------------|-----------|
| YAML syntax | ✅ | ✅ | ✅ |
| Structure (fields exist) | ✅ | ✅ | ✅ |
| Type correctness | ⚠️ Weak | ❌ Blocked | ✅ Full |
| JSON Schema rules | ❌ | ❌ | ✅ |
| Semantic validation | ❌ | ❌ | ✅ |

### Decision: Keep Map-Based Validation

**Chosen approach:** Continue using Phase 3 map-based structural validation

**Rationale:**
1. Compiler's responsibility is **boundary correctness** (YAML syntax + structure) ✅
2. GERT engine's responsibility is **execution validation** (parser + planner) ✅
3. No access to internal parser — workaround would violate architecture
4. Map-based validation proves 90% of what the compiler controls

**Consequences:**
- ✅ Integration tests prove compilation works
- ✅ Clear separation of concerns (compiler vs. runtime)
- ✅ Fast tests (no parser/engine overhead)
- ⚠️ Cannot validate full schema conformance at compile time
- ⚠️ Semantic validation requires downstream v2 Parser

### Recommendations

**Immediate (v2.0):**
- Keep Phase 3 map-based validation as primary test boundary
- Document schema constraint in decision record
- Accept validation gap (full validation requires internal parser)

**Future (v2.1+):**
- Option 1: Export Parser factory (`v2/pkg/parser.New()` public constructor)
- Option 2: Create E2E test in v2 repo (`v2/internal/e2e/domain_home_test.go`)
- Option 3: Use CLI validation (`gert validate compiled.yaml`)

### Learnings

1. **v2 schema design trade-off:** Custom parser complexity in exchange for type-safe step handling
2. **Inline spec limitation:** `yaml:",inline"` with overlapping fields requires custom unmarshal
3. **Module boundaries:** `internal/` packages are architecture enforcement — cannot be bypassed
4. **Pragmatic testing:** Test the boundary you control; trust downstream validation
5. **Parser is mandatory:** Cannot work with `schema.Runbook` types directly via `yaml.Unmarshal`

### Files Modified

- `.squad/decisions/inbox/ken-phase4-roundtrip.md` — Complete investigation and decision record

### No Code Changes

Attempted to create `domains/home/roundtrip_test.go` but removed it after discovering the schema constraint. No lasting code changes — existing Phase 3 tests remain unchanged and continue to pass.

---

## Architecture Checkpoints

**Locked (high confidence):**
- Execution model (flat plan, client-driven)
- Governance primitives (denylist-wins, approval gates)
- Evidence model (JSONL, tamper evidence, resumption)
- Event taxonomy (14 types, lifecycle semantics)

**Under refinement:**
- Policy-as-code language (OPA, CEL, custom DSL?)
- Scheduler/delegation integration
- Mobile app architecture

**Deferred to v2.1+:**
- Concurrent runs per process
- Policy-as-code implementation
- Advanced compensation patterns

## Key Files

- `design/gert-v2/main.tex` — Design document (325 pages, 1.4MB)
- `v2/internal/{parser,planner,engine,adapter}` — Core implementation
- `v2/pkg/{parser,planner,engine,schema}` — Public APIs
- `domains/home/pkg/compiler/` — Domain kit compiler
- `.squad/decisions.md` — All decisions (Phases 1–3)
