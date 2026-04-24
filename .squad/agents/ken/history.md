# ken — History

## Core Context

This section summarizes foundational work (2026-04-18 to 2026-04-21 Phase 14) to reduce file size while preserving critical decisions.

### Project Foundation

**Project:** gert — Governed Executable Runbook Engine (v2 redesign)  
**Mission:** Complete redesign applying v1 learnings; establish v2 as specification-driven reference  
**Team DRI:** Ken (Architect) with Brian (Implementation Lead), Barbara (Integration), Dennis (Research)  
**Codebase Root:** `design/gert-v2/` (LaTeX), `v2/` (Go implementation)

### v1 → v2 Migration

**v1 Key Features:**
- Execute YAML runbooks with real-time governance
- Governors: approval gates, allowlists, command redaction
- Evidence: append-only JSONL, SHA256, resumption
- Adapters: VS Code ext, TUI, JSON-RPC server
- Extensions: .tool.yaml and .provider.yaml with out-of-process runtime

**v2 Goals:**
1. Retain all v1 differentiators (governance > features)
2. Add traceability layer (OpenTelemetry hooks, W3C Trace Context)
3. Clarify core / extension boundary via JSON-RPC
4. Support saga/compensation pattern for complex workflows
5. Policy-as-code path (OPA integration for v2.1)

**Critical Decisions Made (Phases 1–14):**
- Flat ExecutionPlan (no DAG); runtime makes branch/iterate decisions
- Single-run-per-process for v2.0; concurrent-runs deferred to v2.1
- Step advancement always client-driven; no auto-advance
- Denylist wins over allowlist; deny-by-default for unknown commands
- Five governance primitives (allowlist, denylist, env-block, redaction, approval-gates)
- Approval gate model: ordered checkpoint evaluation, timeout/escalation, trace audit trail
- SSE for run output streaming; JWT bearer auth for gert serve

**Architecture Phases Completed:**
- **Phase 1:** 6 decisions locked (governance, extension isolation, event model)
- **Phase 2:** Planner design (DAG → ExecutionPlan lowering)
- **Phase 3:** Runtime core architecture (step loop, trace, checkpoints)
- **Phases 8–13:** Layers built (input providers, HTTP/WS/SSE, evidence, OTLP, CLI polish)
- **Phases 14–17:** Integration testing, server hardening, security fixes

### Design Document Status

**Sections Authored/Reviewed:**
- §02 Architecture: Complete (five-component model, data flow, lifecycle, concurrency)
- §11 Governance: Complete (primitives table, checkpoints, approval UX, audit trail)
- §06 Events: Implemented in code; design needs catch-up (already have 50+ event types)
- §04 Extensions: Defined JSON-RPC contract; capability taxonomy done

**Sections Pending:**
- Kit-Zero (DRI-specific) → Being replaced by domain-agnostic Domain Kit Examples
- Policy-as-code → Deferred to v2.1

### Key Learnings

1. **Governance is v2's core differentiator** — not features. Every architectural decision (step advancement, RPC contract, extension interface, event schema) must support auditability.
2. **Design must follow code** — When implementation discovers better patterns, update design immediately. Current design lags code by ~50 decision entries.
3. **Specification must be executable** — Testing against spec rules (not just happy-path) catches drift early.
4. **Single-run-per-process simplifies correctness** — Concurrency bugs are expensive; defer scalability to v2.1.
5. **Extensions require strong contracts** — JSON-RPC over stdio, deny-by-default, explicit capability manifests prevent surprise interactions.

### Active Team Members

- **Ken** (Architect): Phase review, design decisions, correctness strategy
- **Brian** (Implementation): Phase design and execution, spec compliance
- **Barbara** (Integration): Cross-phase coordination, E2E test design
- **Dennis** (Research): Industry patterns (saga, OTLP, governance-as-code)
- **Leslie** (Documentation): Design doc authorship, domain kit guides
- **Raphael** (UI): VS Code extension, visual correctness

---

## Recent Work (Phases 14–18)

| NBI-14-03 | IN SCOPE | Correctness bug — iterate/branch sub-steps lack scoped variables |
| NBI-14-02 | IN SCOPE | Completes E2E coverage for all step types |
| NBI-14-01 | DEFERRED | Low priority; suite runs in 3-5s; no CI pressure |
| NBI-12-03 | CLOSED (WONT_FIX) | Four deferrals; current impl is correct/bounded; negligible benefit |

### Part B — New Work Considered

- **run.list RPC wiring** → Deferred (requires auth layer not implemented)
- **gert serve hardening** → Deferred (internal/dev-only; no external exposure planned)
- **gert dry-run audit** → Deferred (lower priority than correctness fix)

### Key Decisions

| ID | Decision |
|----|----------|
| D-15-01 | Skip Depth > 0 steps in engine main loop (fixes variable scoping) |
| D-15-02 | Mock tool runtime in E2E harness |
| D-15-03 | Close NBI-12-03 as WONT_FIX |
| D-15-04 | Defer run.list RPC wiring |

### Deliverables

**Modified files:** 5 (engine.go, engine_test.go, helpers_test.go, e2e_test.go, iterate-runbook.yaml)
**New files:** 1 (tool-runbook.yaml)
**Estimated effort:** 3-4 Brian-days

### NBI Items for Phase 16+

- NBI-15-01: E2E test parallelization
- NBI-15-02: run.list RPC → DirRunStore wiring
- NBI-15-03: gert serve hardening
- NBI-15-04: gert dry-run completeness audit

**Output:** `.squad/tmp/ken-phase15-design.md`, `.squad/decisions/inbox/ken-phase15-design.md`

---

## 2026-04-21 — Phase 15 Review: Iterate/Branch Scoping Fix & Tool E2E

**Action:** Architectural review of Brian's Phase 15 implementation  
**Verdict:** APPROVED (9/10)

**Scope reviewed:**
- Part A: `Depth > 0` skip logic in engine.go, new unit tests
- Part B: mockToolRuntime, WithToolDef(), TestE2E_ToolStep

**Key findings:**

1. **Correctness:** `Depth > 0` skip loop is clean and correct. Sub-steps remain in plan for observability; engine executes only Depth=0 steps. No edge cases where legitimate outer steps could be skipped.

2. **Architecture:** This is the right fix, not a workaround. Depth field was designed for this purpose. No future debt.

3. **Test quality:** Both new unit tests are non-vacuous. TestE2E_ToolStep cannot pass vacuously due to three-gate validation (tool def, runtime, status).

4. **Pre-existing flake:** TestSSE_ConnectReceivesEvents confirmed Phase 9 origin via git blame. NBI-15-05 opened.

**Deviation assessments (all accepted):**
- DEV-15-01: `{{.item}}` is correct Go template syntax
- DEV-15-02: EXEMPLARY — CancelMidRun correctly adapted for new behavior
- DEV-15-03: Unconditional mockToolRuntime is simpler and harmless
- DEV-15-04: SSE flake is pre-existing

**Phase 16 NBI items queued:**
- NBI-15-01: E2E test parallelization
- NBI-15-02: run.list RPC → DirRunStore wiring
- NBI-15-03: gert serve hardening
- NBI-15-04: gert dry-run completeness audit
- NBI-15-05: Fix SSE test timing flake

**Output:** `.squad/decisions/inbox/ken-phase15-review.md`

---

## Phase 15 Review — 2026-04-21

**Status:** APPROVED 9/10

Reviewed Brian's Phase 15 implementation:

**Part A (NBI-14-03) — Iterate/Branch Scoping Fix:**
- Depth > 0 skip logic correctly placed in engine.Next()
- Sub-steps remain in plan (preserving trace visibility)
- Parent executors invoke sub-steps via SubStepRunner with correct scoped variables
- TestEngine_SkipsSubStepsAtDepth and TestEngine_IterateSubStepVars prove correctness
- Architecture assessment: correct fix, no future debt

**Part B (NBI-14-02) — Tool Step E2E Coverage:**
- mockToolRuntime injection and WithToolDef() harness method well-designed
- TestE2E_ToolStep uses tool-runbook.yaml testdata; non-vacuous (requires WithToolDef + mockToolRuntime)
- 11 E2E tests total passing

**Deviations accepted:**
- DEV-15-01: {{.item}} syntax clarification (Go template syntax, not pseudocode)
- DEV-15-02: TestE2E_CancelMidRun adaptation (exemplary correctness)
- DEV-15-03: Unconditional mockToolRuntime injection (simpler, harmless)
- DEV-15-04: SSE flake pre-existing (Phase 9 origin, tracked NBI-15-05)

**Validation gate:** PASSED (go build, go vet, go test ./... -race -count=3 all green)

**Deduction:** -1 for TestSSE_ConnectReceivesEvents flake, though pre-existing.

**NBI items for Phase 16:** NBI-15-01 through NBI-15-05

*Decisions entry: Phase 15 review appended to decisions.md*

---

## 2026-04-21 — Phase 16 Design: run.list RPC Wiring & Server Hardening

**Action:** Design Phase 16 scope and architecture
**Requestor:** Cristian

### Context
- Phase 15 (Iterate/Branch Scoping Fix & Tool E2E) sealed and approved (9/10)
- Five NBI carry-forwards: NBI-15-01 (SSE flake), NBI-15-02 (run.list wiring), NBI-15-03 (serve hardening), NBI-15-04 (E2E parallelization), NBI-15-05 (schema docs)
- NBI-15-02 is highest value — connects CLI/serve layers, unblocks real client usage

### Part A — NBI Carry-Forwards

| ID | Disposition | Rationale |
|----|-------------|-----------|
| NBI-15-02 | IN SCOPE (ANCHOR) | Highest value — wires run.list RPC to DirRunStore |
| NBI-15-01 | ANNOTATED | t.Skip added to stop CI noise; proper fix tracked as NBI-16-01 |
| NBI-15-03 | PARTIAL | CORS + bearer token only; full auth deferred |
| NBI-15-04 | DEFERRED | E2E parallelization low priority |
| NBI-15-05 | DEFERRED | Schema docs follow implementation |

### Part B — New Work

1. **run.get RPC** — fetch single run state by ID (natural companion to run.list)
2. **CORS middleware** — allow cross-origin requests from configured origins
3. **Bearer token middleware** — simple shared secret for dev/internal use

### Key Decisions

| ID | Decision |
|----|----------|
| D-16-01 | Registry authoritative for active runs (live Handle > stale snapshot) |
| D-16-02 | Silent store fallback on error (graceful degradation) |
| D-16-03 | Skip SSE flake test with t.Skip (temporary; NBI-16-01 tracks proper fix) |
| D-16-04 | Bearer token is development-only auth (not production security) |
| D-16-05 | Add run.get RPC alongside run.list wiring |

### Deliverables

**Modified files:** 6 (rpc.go, rpc_test.go, sse_test.go, middleware.go, server.go, config.go)
**New files:** 0
**New tests:** ~14
**Estimated effort:** 2-3 Brian-days

### NBI Items for Phase 17+

- NBI-16-01: Proper SSE test synchronization (fix t.Skip'd test)
- NBI-16-02: E2E test parallelization (carry-forward)
- NBI-16-03: Full auth layer (OAuth2/OIDC/API keys)
- NBI-16-04: Rate limiting for gert serve
- NBI-16-05: run.list RPC schema documentation

**Output:** `.squad/tmp/ken-phase16-design.md`, `.squad/decisions/inbox/ken-phase16-design.md`

---

## 2026-04-21 — Phase 16 Review: run.list RPC Wiring & Server Hardening

**Action:** Architectural review of Brian's Phase 16 implementation
**Verdict:** APPROVED (8/10)

**Scope reviewed:**
- Part A: run.list merge (registry + store), run.get wiring
- Part B: CORS middleware, bearer auth middleware, SSE t.Skip, gert serve CLI

**Key findings:**

1. **Correctness:** run.list correctly merges registry and store with deduplication (activeIDs map). run.get returns rpcRunNotFound when not in registry AND store lookup fails. /health is correctly exempted from auth.

2. **Security (NBI-16-08 opened):** Bearer token comparison uses direct string `!=` instead of `subtle.ConstantTimeCompare`. Low practical impact (dev auth only per D-16-04), but establishes wrong pattern for Phase 17 production auth. Fix is trivial — track in NBI-16-08.

3. **Architecture:** Middleware stack order is correct (CORS before Auth). Duck-typed ListRuns via anonymous interface is idiomatic Go for optional capabilities. Store injection via wire.go → EngineConfig.Store is correct.

4. **Test quality:** All 14 new tests are meaningful and pass. Coverage includes: merge logic, deduplication, fallback, 404 path, CORS preflight, bearer validation, /health exemption.

**Deviation assessments:**

| ID | Assessment |
|----|------------|
| D-16-IMPL-01 | ACCEPT — Duck-typed ListRuns is correct adaptation to minimal interface |
| D-16-IMPL-02 | ACCEPT — CompletedAt optional in API; NBI-16-07 tracks schema enhancement |
| D-16-IMPL-03 | ACCEPT — Single middleware.go follows existing pattern |
| D-16-IMPL-04 | ACCEPT — Single --cors-origin sufficient; NBI-16-06 tracks multi-origin |

**NBI items opened:**
- NBI-16-08: Use subtle.ConstantTimeCompare for bearer token validation

**Output:** `.squad/decisions/inbox/ken-phase16-review.md`

---

## Phase 16 Review — APPROVED (8/10)

**Date:** 2026-04-21  
**Action:** Architectural review of Brian's Phase 16 implementation

**Verdict:** APPROVED (8/10)

**Scope:** Part A (run.list/run.get RPC wiring), Part B (CORS + bearer auth middleware, gert serve CLI)

**Key findings:**
- ✅ Correctness: run.list merge logic sound, run.get 404 path correct, /health exemption working
- ⚠️ Security (NBI-16-08): Bearer token uses string != instead of subtle.ConstantTimeCompare (low impact, must fix in Phase 17)
- ✅ Architecture: Middleware stack order correct (CORS before Auth), duck-typed ListRuns is idiomatic Go
- ✅ Test quality: 14 new tests meaningful and comprehensive

**Deviations accepted:** D-16-IMPL-01 through -04 (duck-typed ListRuns, optional CompletedAt, single middleware.go, single --cors-origin)

**Blocking issues:** None.

**Review document:** `.squad/decisions/inbox/ken-phase16-review.md`

---

## 2026-04-21 — Phase 17 Design: Security Hardening & SSE Stabilization

**Action:** Design Phase 17 scope and architecture
**Requestor:** Cristian

### Context
- Phase 16 sealed and approved (8/10)
- Five NBI carry-forwards: NBI-16-01 (timing-safe auth), NBI-16-02 (auth hardening), NBI-16-03 (E2E parallelization), NBI-16-04 (run.list docs), NBI-15-01 (SSE flake)
- NBI-16-01 is security-critical — must be Phase 17 anchor

### Part A — NBI Carry-Forwards

| ID | Disposition | Rationale |
|----|-------------|-----------|
| NBI-16-01 | IN SCOPE (ANCHOR) | Security-critical: `subtle.ConstantTimeCompare` |
| NBI-16-04 | IN SCOPE | Trivial: godoc for run.list response schema |
| NBI-16-02 | PARTIAL | Token expiry only; full OAuth2/OIDC deferred |
| NBI-15-01 | IN SCOPE | Fix SSE flake properly (remove t.Skip) |
| NBI-16-03 | DEFERRED | E2E parallelization low priority |

### Part B — New Work

1. **Token expiry support** — optional `--auth-token-expiry` flag for simple expiry validation
2. **SSE flake fix** — `WaitForSubscriber` synchronization primitive to replace t.Skip

### Key Decisions

| ID | Decision |
|----|----------|
| D-17-01 | Constant-time token comparison is mandatory |
| D-17-02 | Token expiry is opt-in (zero = opaque secret) |
| D-17-03 | SSE synchronization via WaitForSubscriber (not sleep) |
| D-17-04 | run.delete deferred to Phase 19 (post-auth) |

### Deliverables

**Modified files:** 7 (middleware.go, middleware_test.go, rpc.go, serve.go, serve.go, events.go, sse_test.go)
**New files:** 0
**New tests:** ~6
**Estimated effort:** 1.5-2 Brian-days

### NBI Items for Phase 18+

- NBI-17-01: Full auth hardening (OAuth2/OIDC, API key rotation)
- NBI-17-02: E2E test parallelization (carry-forward)
- NBI-17-03: run.delete RPC (CRUD completion)
- NBI-17-04: Rate limiting for gert serve

**Output:** `.squad/tmp/ken-phase17-design.md`, `.squad/decisions/inbox/ken-phase17-design.md`

---

## 2026-04-21 — Phase 17 Preflight Coordination

**Action:** Coordinate preflight review with Barbara for Phase 17 kickoff  
**Requestor:** Cristian  
**Trigger:** Phase 16 sealed (5f40eae)

### Preflight Status
✅ Barbara confirmed all green: go build, go vet, go test -race, git status, git log

### Baseline Approved
- Commit: ab6f554 (feat(v2): Phase 16 complete)
- Build: Clean, <1s
- Tests: 57 packages passing, race detector clean
- Known flake (TestSSE_ConnectReceivesEvents): Correctly skipped per NBI-15-01

### Phase 17 Ready to Launch
- NBI-16-01 (timing-safe auth) is design anchor ✅
- NBI-16-02 (token expiry) partial scope ✅
- NBI-15-01 (SSE fix) scoped ✅
- Design decisions locked in decision inbox

**Output:** Design document approved and ready for Brian's implementation.

**Status:** Phase 17 kickoff green light — Brian ready to proceed.


---

## 2026-04-21 — Phase 17 Review

**Task:** Review Brian's Phase 17 implementation (Security Hardening & SSE Stabilization)

**Verdict:** APPROVED (9/10)

**Key findings:**
- `subtle.ConstantTimeCompare` correctly applied — timing attack vector closed
- JWT expiry validation sound: checks `exp` and `iat` claims correctly
- `WaitForSubscriber` eliminates SSE timing flake deterministically
- Brian's deviation to use standard JWT format is a sensible interoperability improvement over the custom base64-JSON design

**Tests:** 6 new tests added, all pass under `-race -count=3`. No `t.Skip` remaining.

**NBI items generated:**
- NBI-17-01: Full auth hardening (JWT signatures, key rotation)
- NBI-17-05: Apply WaitForSubscriber to WS timing flake

**Review written to:** `.squad/decisions/inbox/ken-phase17-review.md`

## 2026-04-21 — Phase 17 Review (APPROVED 9/10)

**Deliverable:** Brian's Phase 17 implementation (Security Hardening & SSE Stabilization)

**Review Focus:**
- `subtle.ConstantTimeCompare` application for timing-safe bearer token validation
- JWT expiry validation (`exp` and `iat` claims) via --auth-token-expiry flag
- SSE timing flake elimination via `WaitForSubscriber` synchronization
- Test coverage (6 new tests, deterministic under `-race -count=3`)

**Key Findings:**
- ✅ Timing attack vector closed (ConstantTimeCompare line 154 of middleware.go)
- ✅ JWT expiry logic sound (handles expired, too-old, malformed tokens correctly)
- ✅ Plain tokens bypass expiry for backward compatibility
- ✅ WaitForSubscriber deterministically eliminates SSE race
- ✅ All 156 tests pass, no `t.Skip` remaining

**Deviation Decision: APPROVED**
- JWT format (3-segment base64url) instead of custom base64-JSON
- Rationale: Cristian's explicit task spec + standard format + interoperability
- Impact: Token format is now industry-standard JWT

**Pre-existing Issue Noted:**
- TestWS_RunCompleted_ReceivesTerminal has same timing race (outside Phase 17 scope)
- Recommend Phase 18 follow-up (NBI-17-05)

**NBI Items for Phase 18:**
- NBI-17-01: Full auth hardening (JWT signatures, key rotation)
- NBI-17-02: E2E test parallelization (carry-forward)
- NBI-17-03: run.delete RPC
- NBI-17-04: Rate limiting
- NBI-17-05: WebSocket timing synchronization

**Validation:** `go test ./... -race -count=3` → 156 tests ✅

**Status:** Phase 17 APPROVED, sealed, ready for merge

---

## 2026-04-21 — Phase 18 Design

**Task:** Design Phase 18 (JWT signature verification & CRUD completion)  
**Requestor:** Cristian

### Problem Statement

Phase 17 added JWT expiry validation (`exp` and `iat` claims) but **does not verify the signature**. This is a complete authentication bypass — any attacker can forge a JWT with valid claims and authenticate to `gert serve`.

### Phase 18 Scope

**Part A (Anchor) — NBI-17-01: JWT Signature Verification**
- New flag: `--auth-jwt-secret <base64-encoded-secret>`
- HMAC-SHA256 signature verification (HS256 only)
- Mutual exclusivity: `--auth-token` XOR `--auth-jwt-secret`
- Fail-secure: Reject JWT-like tokens if no jwtSecret configured
- Minimum 32-byte (256-bit) secret required

**Part B — NBI-17-03: run.delete RPC**
- Complete CRUD loop (list + get + delete)
- Safety invariant: Running runs cannot be deleted
- Maps to existing `DirRunStore.DeleteRun()`

**Part C — NBI-17-05: WebSocket Timing Fix**
- Apply `WaitForSubscriber` to `TestWS_RunCompleted_ReceivesTerminal`
- Same pattern as SSE fix from Phase 17

### Deferred Items

| ID | Rationale |
|----|-----------|
| NBI-17-02 | Token rotation — short expiry sufficient, OAuth2 better solution |
| NBI-16-03 | E2E parallelization — low priority, needs shared state audit |
| NBI-17-04 | Rate limiting — lower priority than auth bypass fix |

### Key Decisions

| ID | Decision |
|----|----------|
| D-18-01 | HS256 only (no RSA/ECDSA) |
| D-18-02 | `--auth-token` and `--auth-jwt-secret` mutually exclusive |
| D-18-03 | Fail-secure: JWT-like tokens rejected in plain token mode |
| D-18-04 | Minimum 32-byte secret |
| D-18-05 | run.delete refuses running runs |

### Deliverables

- Modified: 7 files (middleware.go, middleware_test.go, rpc.go, rpc_test.go, ws_test.go, serve.go pkg, serve.go cmd)
- New tests: ~8
- Estimated effort: 2-2.5 Brian-days

### NBI Items for Phase 19+

- NBI-18-01: OAuth2/OIDC token validation
- NBI-18-02: Rate limiting (carry-forward)
- NBI-18-03: E2E parallelization (carry-forward)

---

## 2024-04-24 — gert-domain-home v0 Package Design

**Task:** Design Go package structure for gert-domain-home v0 (home automation domain kit)  
**Requestor:** Cristian  
**Context:** First consumer-domain kit validating GERT v2 compilation model

### Design Approach

**Module placement:** Separate Go module at `domains/home/` (not inside `v2/` package)  
**Rationale:** Domain kits are compilation layers over GERT primitives, not core runtime. Separate module enforces dependency direction (kit depends ON v2, never reverse) and enables future extraction to independent repository.

**Package structure (4 core packages):**

1. **pkg/model/** — Pure domain types (Property, Zone, Routine, Incident, Delegation)  
   - No GERT types — this is household vocabulary only
   - 7 files: property.go, zone.go, asset.go, routine.go, incident.go, delegation.go, executor.go

2. **pkg/loader/** — YAML DSL parser (`.home.yaml` → model types)  
   - Validates zone ID uniqueness, cadence rules (simple only in v0)
   - Rejects v1-only features (seasonal cadence, consumables)
   - 6 files: loader.go (interface), property.go, routines.go, assets.go, validation.go, loader_test.go

3. **pkg/compiler/** — Domain model → GERT execution graph  
   - Routine → timer-backed run + human task  
   - Incident → ad-hoc run + repair sub-run (4 sequential steps)  
   - Cadence → timer interval (simple only: every_n_days * 86400 seconds)  
   - 6 files: compiler.go (interface), routine.go, incident.go, cadence.go, evidence.go, compiler_test.go

4. **pkg/delegation/** — Away mode + delegate routing  
   - DelegationPolicy: evaluate time window + scoped tasks filter  
   - TaskFilter: delegate-visible projection (filter by executor + dates)  
   - 4 files: policy.go, routing.go, projection.go, delegation_test.go

**CLI tool:** `cmd/home-validate/` — validates `.home.yaml` files (proves loader works)

**Testdata:** `testdata/` — golden files (property-simple.yaml, property-full.yaml, property-invalid-zone.yaml)

### Key Interfaces

```go
// PropertyLoader parses .home.yaml into model.Property
type PropertyLoader interface {
    Load(ctx context.Context, r io.Reader) (*model.Property, error)
}

// DomainCompiler compiles Property → GERT ExecutionPlan
type DomainCompiler interface {
    CompileProperty(ctx context.Context, p *model.Property) (*engine.ExecutionPlan, error)
    CompileIncident(ctx context.Context, inc *model.Incident) (*engine.ExecutionPlan, error)
}

// DelegationPolicy evaluates task routing (owner vs delegate)
type DelegationPolicy interface {
    ShouldDelegate(ctx context.Context, d *model.Delegation, taskName string, now time.Time) bool
    GetDelegateExecutorID(ctx context.Context, d *model.Delegation, taskName string, now time.Time) string
}
```

### GERT v2 Dependencies

**Only `pkg/compiler/` imports GERT types:**
- `github.com/ormasoftchile/gert/v2/pkg/engine` (ExecutionPlan, Run, Step)
- `github.com/ormasoftchile/gert/v2/pkg/schema` (HumanTaskStep, TimerStep)
- `github.com/ormasoftchile/gert/v2/pkg/evidence` (Evidence primitive)

**Architectural invariant:** Model, loader, delegation packages operate in pure domain vocabulary — no GERT leakage.

### go.work Integration

Added `./domains/home` to workspace:
```
use (
    .
    ./ext/debug
    ./ext/diagram
    ./ext/mcp
    ./ext/render
    ./ext/serve
    ./ext/tui
    ./v2
    ./domains/home   # ← NEW
)
```

### v0 Implementation Order

**Phase 1 (Week 1):** Model + Loader  
- Scaffold `domains/home/` with go.mod  
- Implement `pkg/model/` (all domain types)  
- Implement `pkg/loader/` (YAML parser + validation)  
- Build `cmd/home-validate/` CLI  
- Parse `testdata/property-simple.yaml` successfully  

**Phase 2 (Week 2):** Compiler  
- Implement `pkg/compiler/` (routine → run + timer, incident → repair sub-run)  
- Wire GERT v2 dependencies (engine, schema, evidence)  
- Test: compile simple routine → verify ExecutionPlan structure  

**Phase 3 (Week 3):** Delegation  
- Implement `pkg/delegation/` (time-bounded routing, projection filter)  
- Test: delegation active → tasks route to delegate; expired → route to owner  

**Phase 4 (Week 4):** Integration (hand off to Barbara)  
- E2E test: load property → compile → execute with v2 runtime  
- Verify: timer wakes → human task created → evidence attached → timer resets  
- Documentation: README.md, godoc comments  

### Open Design Questions

Flagged for Brian to verify before Phase 2:

1. **Timer reset support** — Does `v2/pkg/schema.TimerStep` support dynamic next_wakeup recalculation on completion? (Required for cadence rules)  
2. **Sub-run support** — Does `v2/pkg/engine.ExecutionPlan` support parent-child run relationships? (Required for incident → repair sub-run)  
3. **Evidence API** — How does evidence attach to a human task step? (At step level or run level?)  

### Output

**Design document:** `.squad/tmp/ken-home-package-design.md` (22KB, 10 sections)  
**Decision record:** `.squad/decisions/inbox/ken-home-pkg-layout.md` (architectural decision)  

### Strategic Value

This design validates the **domain kit compilation model** — thin authoring DSL (YAML routines) compiles to GERT primitives with zero runtime reimplementation. If successful, future kits (manufacturing, deployment, compliance) follow the same pattern.

**Success criteria:** Barbara's integration test (Phase 4) executes a routine from `property-full.yaml` end-to-end.
- NBI-18-04: Token revocation/blocklist (if needed)

**Output:** `.squad/tmp/ken-phase18-design.md`, `.squad/decisions/inbox/ken-phase18-design.md`

**Status:** Phase 18 design complete, ready for preflight and implementation

---

### 2026-04-21 — Phase 18 Review (Ken as Reviewer)

**Task:** Review Brian's Phase 18 implementation covering:
- Part A: JWT signature verification (security fix for NBI-17-01)
- Part B: run.delete RPC
- Part C: WebSocket timing flake fix

**Review findings:**

**Part A (Security):**
- ✅ Signature verification occurs BEFORE claims validation
- ✅ `hmac.Equal` used for constant-time comparison
- ✅ Fail-secure: JWT-lookalikes rejected in plain bearer mode
- ✅ Mutual exclusivity enforced at startup
- ✅ 32-byte minimum for HMAC-SHA256 secret
- ✅ HS256-only algorithm enforcement
- ✅ alg:none attack blocked

**Part B:**
- ✅ Safety invariant: checks both in-memory and persisted state
- ✅ 4 tests covering success, running guard, not found, missing param

**Part C:**
- ✅ WaitForSubscriber applied correctly

**Deviations accepted:**
1. Error code -32020 instead of -32001 (conflict avoidance)
2. `validateJWTExpiry` not renamed (trivial)

**Verdict:** APPROVED

**Output:** `.squad/decisions/inbox/ken-phase18-review.md`

**Status:** Phase 18 approved, ready for Scribe commit

---

### 2026-04-21 — Phase 19 Design

**Task:** Design Phase 19 (rate limiting, E2E parallelization, token rotation decision)  
**Requestor:** Cristian

### NBI Carry-forwards Addressed

| NBI ID | Item | Disposition |
|--------|------|-------------|
| NBI-17-02 | Token rotation/revocation | **WONT_FIX** |
| NBI-16-03 | E2E test parallelization | **IN_SCOPE** (Part B) |
| NBI-17-04 | Rate limiting | **IN_SCOPE** (Part A) |

### NBI-17-02 Decision: Token Rotation — WONT_FIX

**Analysis:** An in-memory token blocklist (~100 LOC) was considered. Decision to close as WONT_FIX based on:

1. **Marginal security benefit:** With 5-15 min expiry, blocklist only helps if operators detect compromise faster than tokens expire (rare)
2. **Operational complexity:** In-memory clears on restart; persistent requires shared state for distributed deployments
3. **Better alternatives:** Short expiry + secret rotation; external identity providers for critical use cases
4. **Architectural principle:** `gert serve` is a thin adapter, not an identity provider

**Recommendation for production:** Deploy behind identity-aware proxy (OAuth2 Proxy, Keycloak) if fine-grained revocation is needed.

### Phase 19 Scope

**Part A — Rate Limiting (1.5 days)**
- Flag: `--rate-limit N` (requests/second/IP, default 0 = disabled)
- Algorithm: Token bucket via `golang.org/x/time/rate`, burst = 2×limit
- Scope: `/rpc`, `/ws`, `/events`; `/health` exempt
- Memory cap: 10k IPs with LRU eviction, 5-min TTL cleanup
- Middleware position: after CORS, before auth
- ~200 LOC + 7 tests

**Part B — E2E Parallelization (0.5 days)**
- Analysis confirmed E2EHarness is isolation-safe
- `t.TempDir()` creates per-test dirs, no shared state
- No port allocation in E2E tests (in-process engine, not HTTP)
- Action: Add `t.Parallel()` to all 12 tests
- Expected 3-4× speedup

### Key Decisions

| ID | Decision |
|----|----------|
| D-19-01 | Token revocation via blocklist is WONT_FIX; short expiry + secret rotation is sufficient |
| D-19-02 | Rate limiting uses per-IP token bucket with X-Forwarded-For trust |
| D-19-03 | E2E tests are parallelization-safe; no harness changes needed |

### Deliverables

- Modified: 5 files (middleware.go, middleware_test.go, serve.go pkg, serve.go cmd, e2e_test.go)
- New tests: 7 (rate limiting)
- Estimated effort: 2 Brian-days

**Output:** `.squad/tmp/ken-phase19-design.md`, `.squad/decisions/inbox/ken-phase19-design.md`

**Status:** Phase 19 design complete, ready for preflight and implementation

---

### 2026-04-21 — Phase 19 Review (Ken as Reviewer)

**Task:** Review Brian's Phase 19 implementation covering:
- Part A: Rate limiting (NBI-17-04)
- Part B: E2E test parallelization (NBI-16-03)

### Part A — Rate Limiting Review

**Concurrency Analysis:**
- ✅ `sync.Mutex` protects map on all reads/writes
- ⚠️ `cleanupLoop` is fire-and-forget (acceptable for long-lived server)
- ✅ `evictOldest` called while lock is held
- ✅ O(n) eviction acceptable at 10k cap
- ✅ `extractIP` handles malformed XFF safely
- ✅ `/health` exempt from rate limiting
- ✅ Middleware order: CORS → RateLimit → Auth
- ✅ `burst = 2*limit` correctly set

**Test Coverage:** 7 tests covering disabled, below limit, exceeds limit, burst, health exempt, per-IP, and XFF extraction.

### Part B — E2E Parallelization Review

- ✅ `t.Parallel()` as first statement in all 11 tests
- ✅ No shared mutable global state
- ✅ `t.TempDir()` used (not `os.MkdirTemp`)
- ✅ Race detector clean

**Deviations Accepted:**
1. Separate `ratelimit.go` file (better organization)
2. 11 tests not 12 (design count was approximate)

**Verification:**
```
go test ./... -race -count=1 -timeout=180s
# 43 packages pass, no races
```

**Verdict:** APPROVED ✅

**Output:** `.squad/decisions/inbox/ken-phase19-review.md`

**Status:** Phase 19 approved, sealed, ready for Scribe commit

### 2026-04-21 — Phase 20 Design

**Scope:** Consolidation phase completing gert serve CRUD contract

**Budget:** 1.5 Brian-days

**Phase 20 Items:**

| Part | NBI | Item | Priority |
|------|-----|------|----------|
| A (ANCHOR) | NBI-17-03 | `run.delete` RPC | Medium |
| B | NBI-16-05 | API schema documentation | Low |
| C | NBI-17-05 | WebSocket timing flake fix | Low |

**NBI Carry-Forwards (Phase 21):**
- NBI-16-06: `--cors-origin` repeatable flag (Low)
- NBI-16-07: `CompletedAt` for persisted runs (Low)
- NBI-18-01: OAuth2/OIDC external token validation (Medium, v2.1)

**Key Decisions:**
- D-20-01: `run.delete` requires terminal state (no deleting active runs)
- D-20-02: Delete returns error for missing run (not idempotent)
- D-20-03: Export API response types as Go types with godoc

**Design Document:** `.squad/tmp/ken-phase20-design.md`

**Decision Record:** `.squad/decisions/inbox/ken-phase20-design.md`

**Note:** Phase 20 completes the v2.0 MVP feature set for `gert serve`. All remaining NBI items are low-priority enhancements or v2.1 targets.

---

### 2026-04-21 — Phase 20 Re-scope

**Trigger:** The original Phase 20 design (written the same day) contained Parts A and C that were
already shipped in Phase 18 (commit 24d863e). The design was written from the NBI queue without
first reading the actual codebase.

**Why the original design was stale:**

- **Part A (`run.delete` RPC):** `handleRunDelete` is live at `v2/internal/serve/rpc.go:635`.
  Dispatch is at line 105 (`case "run.delete":`). Error constant `rpcRunDeleteRunning = -32020` is
  at line 37. The safety invariant (can't delete running runs) is enforced at lines 649–662. Four
  tests exist in `rpc_test.go`. The NBI item NBI-17-03 is fully closed.

- **Part C (WebSocket timing flake fix):** `WaitForSubscriber` is applied at
  `v2/internal/serve/ws_test.go:78` in `TestWS_RunCompleted_ReceivesTerminal`. NBI-17-05 is fully
  closed.

**NBI items confirmed closed (pre-Phase 20):**

| NBI | Item | Evidence |
|-----|------|----------|
| NBI-17-03 | run.delete RPC | rpc.go:635 |
| NBI-17-05 | WS timing flake | ws_test.go:78 |

**Corrected Phase 20 scope:**

| Part | NBI | Item |
|------|-----|------|
| A (ANCHOR) | NBI-16-05 + NBI-16-07 | `completedAt` surfacing + `run.get` godoc + fixtures |
| B | NBI-16-06 | Multiple CORS origins — `--cors-origin` repeatable flag |
| C | — | `--trust-proxy-headers` security flag (XFF trust opt-in) |

**Key finding on Part A:** `engine.RunState.CompletedAt` IS serialised to disk by `SaveState`. The
gap is purely in the serve layer: `handleRunGet` (persisted path) and `handleRunList` (both active
and persisted items) never read or return the field. Three response-building sites in `rpc.go` need
a one-liner fix each.

**Key finding on Part C:** The rate-limiter's `extractIP` unconditionally trusts
`X-Forwarded-For`, allowing any client to forge their IP and bypass rate limiting. Phase 19
decision D-19-02 mentioned XFF trust but did not require opt-in. Default-secure behaviour (trust
only with `--trust-proxy-headers`) is the correct fix.

**Lesson recorded:** Always read the actual codebase before writing a phase design. NBI items can
be closed by implementation without the design being updated.

**Output:** `.squad/tmp/ken-phase20-design.md` (overwritten), `.squad/decisions/inbox/ken-phase20-design.md` (overwritten)

---

## Learnings

### 2026-04-22 — Domain Kit Traceability: Reverse Mapping as a Mandatory Contract

**Context:** Domain Kit compilers lower kit-specific abstractions (e.g., Vacation Kit's `DayFlavor`, `MealSlot`, `ActivityPool`) into core GERT YAML using only primitives. After lowering, kit-level semantic information is lost — trace events carry only core step IDs. Without reverse mapping, operators cannot correlate runtime events back to kit concepts, projections become impossible, and debugging requires manual trace-through of lowered runbooks.

**Solution:** Adopted a three-layer reverse mapping strategy as the canonical GERT Domain Kit contract:

1. **Layer 1 — Step ID Naming Convention (MANDATORY, zero cost):**  
   All Domain Kit compilers MUST produce deterministic, structured step IDs:  
   `{kit-prefix}.{concept-kind}.{concept-name}.{sub-element}`  
   Example: `vacation.day.2.afternoon-activity.weather-branch`  
   Projection tools parse step IDs by splitting on `.` to recover kit provenance. Works TODAY in GERT v2.0 without schema changes.

2. **Layer 2 — Compiler-Emitted Source Map (RECOMMENDED):**  
   Kit compiler MUST emit a `{runbook-name}.sourcemap.yaml` sidecar file containing:
   - `version`, `kit`, `runbook`, `lowered_at` metadata
   - `entries` map: `{step-id → {kind, name, source_file, source_line, ...}}`  
   Enables rich projections, operator dashboards, and post-mortem analysis with source file references. Standard best practice for production kits (~150 LOC implementation).

3. **Layer 3 — Schema Extension (OPTIONAL, requires GERT v2.1+):**  
   Add `Meta map[string]string` field to core `Step` struct. Compiler populates `meta` fields on every generated step during lowering. Runtime propagates `meta` unchanged into trace events. Enables self-describing traces (no external source map required), streaming projections, and multi-kit composition. Not available in v2.0 — proposed for v2.1.

**Kit Traceability Contract (5 Rules):**
1. Step IDs MUST follow structured naming convention
2. Compiler MUST emit source map sidecar for every lowered runbook
3. Source map MUST be deterministic (same kit source → same source map)
4. Source map MUST version the kit (`version` and `kit` fields required)
5. Compiler SHOULD populate `meta` fields if Layer 3 available in target GERT version

**Impact:**
- Without traceability: Domain Kits are write-only abstractions — unusable in production (debugging takes 10x longer, projections impossible, audit trails opaque)
- With traceability: Operator dashboards can project domain-level views from core trace events, debugging correlates step IDs to kit concepts in seconds, audit trails are semantically meaningful

**Design implications for GERT core:** Minimal. Layers 1 and 2 are compiler concerns with no runtime dependency. Layer 3 (proposed for v2.1) requires a 1-line schema change (`Meta map[string]string`) and 1-line trace writer update. No new core concepts required — the Clean Kernel Principle is preserved.

**Lesson:** Reverse mapping is not a "nice-to-have" for Domain Kits — it is the **price of admission for production**. This is the same lesson learned by every compiler ecosystem (TypeScript, SASS, Terraform) that lowers high-level abstractions to lower-level runtimes. The Vacation Kit validates this as a general Domain Kit pattern applicable to all future kits.

**Decision record:** `.squad/decisions/inbox/ken-traceability-reverse-mapping.md`  
**Documentation:** Vacation Domain Kit v0, §4.10 (canonical GERT answer, concrete Vacation Kit illustration)

---

### 2026-04-20 — DRI Reference Audit and Decoupling Decision

**Context:** User requested a full audit of DRI-domain references across the three main design files
to decouple gert from domain-specific assumptions. DRI (Directly Responsible Individual) is an
SRE/ops pattern with roles like incident-commander, change-manager, etc. Gert must be domain-agnostic.

**Audit scope:** Three design files totaling 4,153 lines:
- `04-domain-kit-model.tex` (492 lines) — Domain Kit architecture
- `03-schema-vnext.tex` (3,117 lines) — Schema specification and examples
- `12-governance-policy.tex` (544 lines) — Governance primitives and approval gates

**Key findings:**

1. **67 DRI-domain references** identified across all three files
2. **Kit-Zero section (lines 415-444 of 04-domain-kit-model.tex)** explicitly frames the DRI/ops
   model as "gert's first Domain Kit" and uses it to ground the abstract Kit concept
3. **All governance examples** use DRI-specific role names: DRI, change-manager, incident-commander,
   incident-responder
4. **All schema examples** use DRI-domain workflows: incident response, deployment, rollback, triage,
   change management
5. **Domain Kit manifest example** (`gert.ops`) is DRI-specific with step types: change-request,
   rollback, triage

**Architectural issue:** This violates gert's stated design principle: "gert core is domain-agnostic."
The coupling creates confusion: is gert a DRI/ops tool or a general-purpose engine?

**Decision made:** Remove ALL DRI-domain vocabulary from gert v2 core design.

**Solution approach:**

1. **Extract Kit-Zero section** → new PDF `dri-kit-manual.pdf` (DRI operations domain kit)
2. **Replace with generic forward reference** to multiple domain kit examples (workflow, compliance, migration)
3. **Establish generic role vocabulary** for all examples:
   - `approver` (replaces: DRI, change-manager)
   - `reviewer` (replaces: change-manager in multi-party scenarios)
   - `responder` (replaces: incident-commander, incident-responder)
   - `owner` (replaces: DRI as owner)
   - `operator` (replaces: SRE, ops, admin)
4. **Establish generic workflow vocabulary**:
   - incident → request, workflow event
   - deploy/deployment → operation, provision
   - rollback → cleanup, compensate
   - triage → select-priority, classify
   - change-request → approval-request
   - mitigation → procedure
5. **Generalize manifest example** from `gert.ops` to `gert.workflow`

**Deliverables produced:**

1. **`.squad/tmp/ken-dri-audit.md`** — Comprehensive audit with:
   - Section A: 51 specific line-by-line changes with exact current/proposed text
   - Section B: Content inventory for new PDFs (what moves to dri-kit-manual vs domain-kit-guide)
   - Section C: Five architectural notes on narrative changes, forward references, and structural impact
   - Statistics: 67 instances, 51 edits + 1 section extraction

2. **`.squad/decisions/inbox/ken-dri-decoupling.md`** — Architectural decision record:
   - Decision: Remove DRI as Kit-Zero from gert core
   - Principle: Gert core contains zero domain-specific vocabulary in examples or framing
   - Generic role names table (5 roles defined)
   - Generic workflow vocabulary table (8 terms mapped)
   - Implementation plan (5 phases)
   - Alternatives considered (4 rejected options with rationale)

**Key principle established:**

> "Gert core contains zero domain-specific vocabulary in examples, framing, or implementation.
> Domain semantics live exclusively in separately-distributed Domain Kits."

**Architectural insight:** The governance MODEL (allowlists, approval gates, redaction, RBAC) is
already domain-agnostic by design. The coupling exists ONLY in the EXAMPLES. This means the fix is
primarily documentation/presentation, not a structural change to the architecture.

**Implementation strategy:** 6-phase rollout coordinated with team:
1. Extract Kit-Zero to dri-kit-manual.pdf (Leslie)
2. Update 04-domain-kit-model.tex (Ken → Leslie)
3. Update 12-governance-policy.tex (Ken → Leslie)
4. Update 03-schema-vnext.tex (John + Ken → Leslie)
5. Final audit (Ken)
6. Review and sign-off (Barbara, John, Leslie, Brian)

**Lesson:** When establishing architectural principles, audit for violations across ALL design
artifacts, not just implementation code. Documentation examples can contradict stated principles
and create user confusion about the system's identity.


### 2026-04-21 — DRI Domain Kit Manual Complete

**What was written:**

All 10 chapters of the **DRI Domain Kit Manual** (`design/dri-kit-manual/`) are now complete with substantive, authoritative content. This manual documents the `gert.ops` Domain Kit, which implements the DRI (Directly Responsible Individual) accountability model for operations runbooks.

**Chapter breakdown:**
- **00-introduction.tex** (144 lines) — Manual scope, audience, prerequisites, DRI definition, how gert.ops implements DRI, document structure
- **01-dri-concepts.tex** (285 lines) — DRI model, role taxonomy (6 roles: dri, approver, change-manager, incident-commander, responder, observer), role assignment, escalation chains, SLA concepts, DRI transfer mechanics
- **02-schema-reference.tex** (426 lines) — gert.ops Kit schema, apiVersion suffix, 5 step types (ops.cli, ops.manual, ops.approval, ops.change-request, ops.incident), metadata fields (meta.roles, meta.change-id, meta.sla), 3 evidence types with complete field definitions and YAML examples
- **03-authoring-guide.tex** (413 lines) — Step-by-step guide to writing a gert.ops runbook, declaring roles, using approval gates, change-request wrappers, evidence capture, best practices, complete deploy runbook example
- **04-governance.tex** (322 lines) — How gert.ops wires into core governance hooks, role-based gating (requires_role/any/all), allowlist enforcement, env-var blocking, redaction, audit trail events, complete governance configuration example
- **05-evidence-tracing.tex** (258 lines) — Evidence model, record structure, run directory layout, timeline projection, SHA256 verification, compliance bundles, CLI commands for reading evidence
- **06-change-request-workflow.tex** (309 lines) — End-to-end change request walkthrough, pre-execution approval, execution with evidence capture, post-execution closure, rollback scenario, complete database migration example
- **07-incident-response-workflow.tex** (342 lines) — Incident lifecycle (detect→declare→triage→mitigate→resolve→review), incident declaration, IC takeover, triage steps, escalation, complete service degradation incident runbook
- **08-migration.tex** (300 lines) — Migrating v1 DRI runbooks to v2 + gert.ops, core migration reference, gert.ops-specific patterns, 4 migration recipes (manual, approval, deploy, incident), migration validator, 15-item checklist
- **09-reference.tex** (279 lines) — Complete field reference tables for all gert.ops types, role semantics table, environment variables, error codes

**Total:** 3,078 lines of authoritative LaTeX content across 10 chapters.

**Content sources:**
- `.squad/tmp/ken-dri-audit.md` section B "MOVE TO dri-kit-manual" — extracted DRI-specific content from gert-v2 design
- `design/gert-v2/sections/12-governance-policy.tex` — governance primitives and policy evaluation model
- `design/gert-v2/sections/04-domain-kit-model.tex` — Kit model context and lowering/compilation patterns

**Writing approach:**
- Each chapter is self-contained and practical (ops engineers can jump to any chapter)
- Extensive YAML examples using LaTeX minted environments
- Complete end-to-end workflows with annotated examples
- Field reference tables using LaTeX tabular environments
- Authoritative tone appropriate for production runbook authors

**Build system:**
- Uses same MastersThesis.cls and Makefile pattern as gert-v2 design doc
- Scaffold was pre-created by Leslie; Ken filled in all content
- PDF build requires LaTeX (tectonic/latexmk/pdflatex) - not run in this session but structure is correct

**Next steps:**
- Leslie can now build the PDF when needed
- The manual can be distributed separately from gert-v2 design doc
- DRI-specific examples can be removed from gert-v2 core chapters (per decoupling plan in ken-dri-audit.md)

---

## Docs Refactor Session Completion (2026-04-21)

**Scribe Orchestration:** Final session wrap-up by Scribe (2026-04-21T16:09:18Z)

**Team Accomplishments:**
- ken-docs-audit: Identified 67 DRI vocabulary instances in gert-v2 sections
- leslie-scaffold: Bootstrapped LaTeX project structures for both kit guides
- leslie-refactor-core: Applied 51 refactoring changes to generalize core sections
- leslie-kit-guide: Wrote Domain Kit Development Guide (9 chapters, ~2,884 lines)
- ken-dri-manual: Wrote DRI Kit Manual (10 chapters, ~3,078 lines)

**Ken + Leslie Collaboration:**
1. Ken's DRI audit informed Leslie's refactoring scope
2. Leslie's scaffold template was reused by Ken for DRI manual
3. Leslie's domain-agnostic examples influenced Ken's DRI-specific chapter examples
4. Ken reviewed Leslie's refactored sections for consistency with decoupling decision

**Decisions Merged:**
- ken-dri-decoupling: Approved (major) — remove DRI from gert core
- leslie-kit-guide-complete: Approved — guide complete
- ken-dri-manual-complete: Approved — manual complete
- leslie-refactor-complete: Approved — all 67 instances addressed

**Pending:** Ken's cross-consistency review (ken-review) to validate all three gert-v2 sections against new domain kit frameworks.

---

## 2026-04-22 — Vacation Domain Kit Prototype Design

**Task:** Write sections 1, 2, 4, 5, 8 of the Vacation Domain Kit prototype design  
**Requestor:** Cristian

### Key Decisions

| ID | Decision |
|----|----------|
| VK-01 | Vacation Kit is a Domain Kit (compiler + schemas), NOT an extension. Companion tools (token-gen, ledger, suggest, weather) are separate extension tools per §4 boundary. |
| VK-02 | Stay Template → parent run, Day → child sub-run via iterate + invoke. Sub-run isolation protects stay-level state from day-level failures. |
| VK-03 | Credits are GERT runtime state variables, not a dedicated store. Ledger tool enforces non-negativity. Local-first, no external infrastructure. |
| VK-04 | QR guest pass is a self-contained signed token (HMAC-SHA256). Issued/revoked by tool steps. Reuses GERT JWT infrastructure from Phases 17-18. |
| VK-05 | Suggestions are non-blocking human tasks with timeout. Guest's "What now?" does not gate day progression. |
| VK-06 | Weather replanning: event → state variable → branch → replanning child sub-run. Exercises full GERT event pipeline. |
| VK-07 | MVP scope: 2 weeks, ~2,850 LOC. Week 1: foundation (stay + QR + timeline). Week 2: intelligence (suggestions + override + credits + weather). |
| VK-08 | Operator overrides use GERT approval gates directly. No custom override mechanism. |

### GERT Primitive Mappings Established

- Stay Run → parent run
- Day → child sub-run (iterate + invoke)
- Slot transitions → timer tool steps (`gert.timer`)
- QR pass → tool-issued signed token + policy scopes
- Credits → runtime state variables (`credits.<category>.balance`)
- Suggestions → tool step (compute) + manual step with timeout (present)
- Weather override → event → state change → branch → replanning sub-run
- Operator override → approval-gated manual step with role verification
- Activities → manual or tool steps selected via branch conditions
- Evidence → all mutations emit trace events in `trace.jsonl`

### Deliverables

- `.squad/tmp/ken-vacation-kit.md` — Full design (sections 1, 2, 4, 5, 8)
- `.squad/decisions/inbox/ken-vacation-kit.md` — 8 architectural decisions + deferred decisions

---

### 2026-04-22 — Gap Remediation: Step.Meta and Kit Certification Formal Decisions

**Context:** Code review identified two design gaps that needed formal decision tracking:
1. Step.Meta v2.1 proposal (Kit Traceability Layer 3) was fully designed in `.squad/tmp/vacation-domain-kit-v0.md` but lacked a formal decision record
2. Kit certification process was referenced in §4.10 and decision VK-01 but had no defined scope or owner

**Actions taken:**
- Gap 1 filed: Step.Meta v2.1 decision is now formally tracked in `.squad/decisions/inbox/ken-step-meta-v21-proposal.md`
- Gap 3 filed: Kit certification is now a named v2.1 design item with scope defined in `.squad/decisions/inbox/ken-kit-certification-tracking.md`
- Both decisions are in the inbox for Scribe to merge

**Impact:** Prevents design work from being lost between v2.0 ship and v2.1 planning. Kit certification scope is now explicit, reducing risk of colliding step ID prefixes in multi-kit scenarios.

---

## 2026-04-21 — gert-domain-home v0 Architecture

**Action:** Authored architectural specification for gert-domain-home v0  
**Requestor:** Cristian  
**Output:** `.squad/tmp/ken-home-domain-spec.md`

### Context

First non-enterprise GERT domain kit. Validates domain kit compilation model: authoring DSL → GERT core primitives, no runtime reimplementation.

### Domain Purpose

Home management kit for household maintenance. Three core scenarios:
1. **Recurring maintenance** — pool care, lawn mowing, plant watering with seasonal cadence
2. **Reactive incidents** — broken hinge, leak, pump failure → diagnose → buy → fix → verify
3. **Delegation / away mode** — owner travels, delegate gets simplified task view, time-bounded authority

### Architectural Decisions

**1. Domain Kit Compilation Model**  
Home concepts (routine, incident, delegation, zone, asset) compile to GERT primitives:
- routine → timer-backed durable run with human task + evidence
- cadence → timer policy (simple interval in v0, seasonal rules in v1 with OPA)
- incident → ad-hoc run (no timer, user-triggered)
- repair run → sub-run with 4 sequential human task steps
- delegation → time-bounded policy (routes tasks to delegate during absence window)
- away mode → projection (filters property run graph to delegate-scoped tasks)
- evidence → GERT evidence primitive (photo/note attachment)

**2. Mobile-First UX**  
Calm, practical mobile app. No dashboards, no KPIs. Today tab answers "what do I do today?"  
Four tabs: Today (landing), Property (zones/assets), Tasks (repair runs), History (evidence gallery)

**3. Delegation Model**  
Temporary authority transfer with:
- Date-bounded activation (start_date to end_date)
- Scoped task filter (delegate sees ONLY assigned tasks)
- Simplified executor view (Today tab only, no property config access)
- Evidence accountability (owner reviews completion photos on return)

**4. v0 Scope**  
Minimum useful set:
- 2–3 routines with simple cadence (no seasonal rules)
- 1 repair run template (4 steps: diagnose, buy, fix, verify)
- Delegation with simplified delegate view
- Today tab only (no Property/Tasks/History in v0)
- Photo + note evidence (no GPS)
- Manual YAML authoring (no mobile routine editor)

Deferred to v1:
- Seasonal cadence rules (requires OPA, GERT v2.1)
- AI hints (weather-aware suggestions)
- Consumable tracking (auto-routine creation)
- Full mobile app tabs (Property/Tasks/History)
- Reminder notifications (requires timer + notification policy)

### Why This Validates GERT v2

Tests 7 core GERT primitives:
1. **Recurring orchestration** — simplest timer-backed run use case
2. **Long-running runs** — repair runs span days/weeks, validates resumability
3. **Projections** — Today tab and away mode both rely on fast filtering (<100ms)
4. **Delegation** — time-bounded policy with automatic expiration
5. **Timers** — seasonal cadence (v1) tests variable-interval policies
6. **Reactive incidents** — ad-hoc runs with no predefined schedule
7. **Domain kits** — thin DSL compiling to GERT core, zero runtime reimplementation

### Success Criteria

v0 proves:
- Domain kit compilation works (routine.yaml → valid ExecutionPlan)
- Timer-backed runs work (no drift, correct cadence reset)
- Delegation works (time-bounded routing, auto-expiration, evidence review)
- Evidence capture works (photo/note persist, retrievable)
- Reactive incidents work (user-reported → multi-step repair run)
- Mobile app usable by non-technical users (<1 min per task)

Validation: 14-day daily use by 2 non-technical users, track completion rate (target >90%), evidence rate (>70%), bugs (<5 blocking), NPS (7+/10).

### Key Learnings

1. **Home domain is NOT a toy** — it's the hardest GERT validation. Daily use surfaces timer drift, projection staleness, policy bugs immediately. Enterprise workflows (monthly approvals) hide these bugs for months.

2. **Delegation is a critical primitive** — time-bounded role transfer with scoped permissions is a general pattern (not just home). Validates GERT policy model for time-windowed routing.

3. **Projections need <100ms performance** — mobile apps demand fast refresh. If projection query takes >500ms at 100 tasks, Today tab UX fails. This is a hard GERT requirement surfaced by Home domain.

4. **Evidence is proof, not documentation** — photo of mowed lawn is legal-grade proof task was done. Evidence primitive must be immutable, SHA256-hashed, append-only. Home domain makes this concrete (not just enterprise audit trail).

5. **Calm UX requires smart projections** — "what to do today" is a projection, not a raw query. GERT must support projection-as-first-class-primitive, not just "query run state and filter client-side."

6. **v0 scope discipline is critical** — deferring seasonal rules, AI hints, consumable tracking keeps v0 buildable in 6 weeks. Trying to build all features at once would delay validation by 6 months.

### Architectural Risks Identified

**Risk 1: Timer drift**  
If timer reset uses `last_wakeup + interval` instead of `last_completion + interval`, routines drift over time (7-day cadence becomes 8 days after 10 iterations). GERT must use completion-time-based reset.

**Risk 2: Projection staleness**  
If Today tab projection is cached and not invalidated on task status change, user marks task complete but it still appears in list. GERT needs SSE or polling-based projection updates.

**Risk 3: Delegation policy composition**  
If multiple delegations overlap (two delegates for same period), which policy wins? v0 assumes no overlap, but GERT must define precedence (last-created wins? highest-priority wins? error on conflict?).

**Risk 4: Evidence upload failures**  
If photo upload fails (network down), does task completion roll back? Or task completes but evidence queued? v0 assumes always-online (rolls back on failure). v1 needs offline queue.

**Risk 5: Seasonal rule complexity**  
Seasonal cadence requires GERT to pass runtime context (current date, timezone) to policy evaluator. This is NOT designed in GERT v2.0. OPA integration required for v1.

### Decisions Recorded

- **D-HOME-01:** Adopt gert-domain-home as first official consumer domain kit
- **D-HOME-02:** Home kit compiles to GERT primitives, no runtime reimplementation
- **D-HOME-03:** v0 scope limited to simple cadence, 1 repair run, delegation, Today tab only
- **D-HOME-04:** Seasonal rules, AI hints, consumable tracking deferred to v1 (GERT v2.1 dependency)
- **D-HOME-05:** Mobile app targets non-technical users (no YAML editing, no runbook jargon)

**Output files:**
- `.squad/tmp/ken-home-domain-spec.md` (62KB, 10 sections, comprehensive architecture)
- `.squad/decisions/inbox/ken-home-domain-kit.md` (decision record, to be reviewed)


---

## 2026-04-24 — Compiler Interface Contract Design (gert-domain-home)

**Action:** Design the compiler interface contract for gert-domain-home → GERT v2 core  
**Requestor:** Cristian  
**Scope:** Compiler boundary, output strategy, function signatures, go.mod dependencies

### Task Overview

Designed the compiler package for `domains/home/pkg/compiler/` that transforms domain model types into GERT v2 runbook YAML documents. The compiler is the bridge between domain-specific models (`Property`, `Routine`, `IncidentTemplate`, `Delegation`) and GERT core execution primitives.

### Key Decisions

**D-HOME-06: Compiler produces YAML, not Go structs (Option B)**

**Rationale:**
- Clean boundary: domain module is independent of GERT v2 Go types
- Resilient to GERT schema changes (no recompilation required)
- Idiomatic testing via golden file comparison
- GERT's parser is designed for YAML ingestion
- Prevents tight coupling between domain and v2 internals

**Alternatives rejected:**
- Option A (Go structs): Heavy dependency, tight coupling, breaking changes painful
- Option C (IR): No intermediate representation exists in GERT v2

**D-HOME-07: No v2 dependency in domains/home/go.mod**

The `domains/home` module only requires `gopkg.in/yaml.v3` for YAML marshaling. Components that consume the compiler AND invoke GERT (e.g., scheduler daemon) add the v2 dependency themselves.

**D-HOME-08: Compiler functions return (string, error)**

Four core functions:
```go
CompileRoutine(ctx, prop, routine) (string, error)
CompileIncidentTemplate(ctx, prop, template) (string, error)
CompileDelegation(ctx, prop, delegation) (string, error)
CompileProperty(ctx, prop) (*Catalog, error)
```

Output is YAML string ready for `parser.ParseBytes()`.

### GERT v2 Core Inventory (Verified)

Audited v2/pkg to identify available types:

**Available:**
- `schema.Runbook` — Top-level runbook document
- `schema.Step` + 14 step types (cli, tool, collector, choice, decision, branch, iterate, parallel, approve, assert, compensate, wait_for_event, end)
- `engine.ExecutionPlan` — Planner output, engine input
- `engine.RunHandle` — Runtime control surface
- `parser.Parser` — YAML ingestion layer
- `planner.Planner` — Include resolution, tool lookup
- `governance.GovernancePolicy` — Policy enforcement interface

**Not available (domain must implement):**
- Timer/cron scheduling
- Delegation/permission tracking
- Notification system
- Inventory/stock tracking

### Compilation Examples Designed

**Routine → Runbook:**
- `kind: composable` (reusable workflows)
- Single `collector` step with evidence capture
- Scheduling metadata in `metadata.*` (consumed by scheduler, not GERT runtime)

**Incident Template → Runbook:**
- `kind: mitigation` (reactive workflows)
- Domain `human_task` → GERT `collector` step
- Domain `decision` → GERT `decision` step with `goto` routing
- Domain `depends_on` → GERT `when` conditionals

**Delegation → Governance Policy:**
- NOT a runbook — governance config overlay
- `permissions.can_*` → `governance.rules[].action` (allow/deny/require-approval)
- Active window checked by scheduler before merging policy
- `metadata.*` holds contact info for notifications

### Validation Strategy

1. **Compiler unit tests:** Golden file comparison + structural validation
2. **Integration tests:** Compiled YAML → parser.ParseBytes() acceptance
3. **E2E tests:** Compile → Parse → Plan → Execute (dry-run mode)

### Next Steps

| Step | Owner | Deliverable |
|------|-------|-------------|
| 1. Review contract | Ken | Approve or request changes |
| 2. Implement CompileRoutine | Brian | `pkg/compiler/routine.go` + tests |
| 3. Implement CompileIncidentTemplate | Brian | `pkg/compiler/incident.go` + tests |
| 4. Implement CompileDelegation | Brian | `pkg/compiler/delegation.go` + tests |
| 5. Implement CompileProperty | Brian | `pkg/compiler/property.go` + tests |
| 6. Integration test module | Barbara | `tests/integration/` with v2 dep |
| 7. E2E test | Barbara | Full pipeline test |

### Open Questions

1. Should compiler embed `$schema` URL? (schema.go doesn't declare constant)
2. Should GERT reserve `metadata.domain` namespace for domain compilers?
3. Should GERT docs include canonical evidence type mapping table?
4. How should scheduler merge delegation policy into runbook governance?
5. What format for `Catalog` output? (JSON manifest, YAML, directory structure?)

**Output files:**
- `.squad/tmp/ken-compiler-contract.md` (30KB, comprehensive design)
- `.squad/decisions/inbox/ken-compiler-output-strategy.md` (decision record)

