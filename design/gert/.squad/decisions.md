# Squad Decisions Log

## 2026-04-28: TUI Interface Gaps — Architectural Decision & Implementation

**Session**: gert-tui runner — 3 critical interface gaps blocking TUI implementation  
**Decision**: ✅ OPTION B APPROVED & IMPLEMENTED  
**Consensus**: Ken (Architect) + Brian (Implementer)

### Problem Statement

Gert-tui TUI runner requires 3 interface additions to gert v2 engine:
1. **EventKindStepOutput** — Real-time step output streaming
2. **EventKindRunFailed** — Distinguish run failure from completion
3. **RunState.Plan** — Expose ExecutionPlan for TUI step-list panel

Existing engine model incomplete; TUI cannot render live output, failure state, or step structure.

### Options Considered

**Option A**: Polling-based state machine
- Client polls engine.GetRunState() on timer
- Engine maintains complete state history
- TUI renders from full state snapshot
- ❌ Rejected: Polling overhead, no change events, state machine complexity

**Option B**: Event-based architecture + field addition (✅ CHOSEN)
- Engine emits StepOutput/RunFailed events as they occur
- RunState carries Plan field for TUI consumption
- TUI wires to event stream + reads RunState.Plan
- ✅ Efficient, simple, aligned with gert v2 design

### Decision Rationale

1. **Efficiency**: Event-driven avoids polling; TUI updates only on state change
2. **Simplicity**: No state machine; events are domain facts already in gert v2
3. **Alignment**: Event-stream architecture core to gert v2 tracing/logging
4. **Extensibility**: New event kinds added incrementally
5. **Backward Compatible**: Plan field is additive; existing consumers unaffected

### Implementation

**Implementer**: Brian (claude-sonnet-4.5)  
**Files Modified**: 3
- `pkg/trace/event.go` — EventKindStepOutput, EventKindRunFailed
- `pkg/engine/run.go` — RunState.Plan field
- `internal/engine/engine.go` — Event emission + Plan capture

**Build**: ✅ Passes  
**Tests**: ✅ All pass; no regressions  
**Quality**: Code reviewed; approved

### Commits

- f960d2f: feat: add step/output, run/failed event kinds and Plan to RunState
- 824d70f: docs: document gert-tui interface gaps implementation

### Action Items

- ✅ Implement all 3 gaps (Brian)
- ✅ Build & test verification
- ✅ Merge to main
- 📋 Wire TUI panels to events (next phase)

---

## 2026-04-28: Delegation Spec — Final Approval

**Session**: gert-domain-home delegation spec — Rounds 4-6 fix/review cycle  
**Decision**: ✅ APPROVED FOR MERGE  
**Consensus**: 5/5 reviewers (Ken, Ada, James, Barbara, John)

### Resolution

After 6 review rounds and 2 fix cycles:
- API schema now canonical (snake_case, field names aligned across schema/API/docs)
- Kotlin SDK corrected: STEP_COMPLETED constant, payload direct cast
- Required fields properly marked (delegateContact in POST/PUT)
- assigns transformation documented (flat API arrays ↔ YAML discriminated union)
- casa-santiago unit/type fields aligned
- All blocking issues resolved; no remaining concerns

### Commits

- f83cff2: ken-apply-fixes-4 (10 initial fixes)
- 777d19f, 80b6344: james-sdk-fix (SDK runtime fixes)
- d36106b: ken-apply-fixes-5 (schema alignment, Kotlin corrections)
- 7e1373c: review6 round (inline S09 label fix)

### Action Items

- ✅ Merge into main
- ✅ Deploy updated schema and API docs
- ✅ Release Kotlin SDK with corrected constants/payload handling
- ✅ Update integration guides

