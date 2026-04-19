# Correctness Strategy Gap Analysis
**Author:** Ken (Software Architect)  
**Date:** 2026-04-18  
**Context:** Critical gaps in correctness strategy identified by Cristian

---

## Gap 1: Parallel Step Validation

### What the Problem Actually Is

The spec (§03-schema-vnext.md:968-1003) defines an explicit `parallel` step kind with:
- **Goroutine-per-branch:** each branch executes on its own goroutine concurrently
- **Join semantics:** `wait_for: all | any | majority` with `on_failure: fail | continue`
- **Variable isolation during execution:** branches have isolated variable writes until join
- **Post-join merge:** captured variables from all branches merged with last-writer-wins on conflict
- **Q2 decision:** steps are otherwise sequential — no implicit parallelism

The spec does NOT define how to test that this concurrency is CORRECT. Specifically:

1. **Race conditions:** Two branches reading/writing shared state (e.g., both branches invoking tools that manipulate the same external resource). The spec says variable writes are isolated but doesn't specify how external effects are isolated.

2. **Join ordering:** When `wait_for: all`, does the join occur deterministically? If branch A finishes before branch B, are their events sequenced deterministically in the trace?

3. **Partial failure behavior:** When `on_failure: continue` and branch A fails while branch B succeeds, what is the run's terminal outcome? Which captures are available?

4. **Timeout handling:** The spec doesn't mention per-branch timeouts. What happens if one branch hangs? Does the entire parallel block hang, or is there an implicit timeout?

5. **Event sequencing in trace:** When two branches emit `step/started` events concurrently, the trace file appends them sequentially. What ordering guarantee exists? Is it goroutine schedule-dependent (non-deterministic)?

6. **Nested parallel blocks:** Can a parallel branch contain another parallel step? The spec doesn't forbid it. What happens to the goroutine topology?

### What Test Infrastructure Is Needed

To validate parallel execution correctness, we need:

1. **FakeStepExecutor with controllable timing:**
   - Package: `pkg/testutil`
   - Type: `type FakeStepExecutor interface { Execute(ctx, step) error; Delay(stepID, duration); Signal(stepID) }`
   - Capability: Inject deterministic delays and wait for explicit signals before step completion
   - Use: Control interleaving of branch execution to force specific race scenarios

2. **ConcurrentEventCollector:**
   - Package: `pkg/testutil`
   - Type: `type ConcurrentEventCollector struct { events []Event; mu sync.Mutex }`
   - Capability: Thread-safe event collection from concurrent goroutines with timestamped order capture
   - Use: Assert event ordering properties (e.g., all branch A events before join, or interleaved but sequenced)

3. **DeterministicScheduler (test mode):**
   - Package: `pkg/engine` (feature-flagged test mode)
   - Capability: Replace real goroutine dispatch with a controllable scheduler that advances goroutines in round-robin or explicit order
   - Use: Make parallel execution deterministic for golden trace testing
   - Alternative: Use Go's race detector (`-race`) for race detection but still need deterministic scheduling for reproducible tests

4. **Timeout Injection:**
   - Package: `pkg/testutil`
   - Type: `type TimeoutInjector interface { InjectTimeout(stepID, duration) }`
   - Capability: Simulate a step hanging indefinitely (blocks until test cancels context)
   - Use: Validate timeout behavior and cancellation propagation

5. **External Effect Tracker:**
   - Package: `pkg/testutil`
   - Type: `type EffectTracker struct { effects []Effect; conflicts []Conflict }`
   - Capability: Record external effects (e.g., "branch A wrote file X", "branch B wrote file X") and detect conflicts
   - Use: Validate that parallel branches don't have undetected side-effect conflicts

6. **Parallel Block Test Harness:**
   - Package: `pkg/engine_test`
   - Function: `func TestParallel(t *testing.T, branches []BranchSpec, joinPolicy JoinPolicy) TraceAssertion`
   - Capability: Compose parallel test scenarios declaratively, assert on trace ordering and captures
   - Use: Table-driven tests for all join combinations

### Which Phases Are Affected

From the 13-phase implementation plan (PLAN.md):

- **Phase 3 (Runtime Core & Trace):** Goroutine topology, event sequencing, trace append ordering
- **Phase 5 (Step Types):** `parallel` step execution, join semantics, variable merge
- **Phase 11 (Evidence & Replay):** Replay of parallel blocks (non-deterministic event order?)
- **Phase 13 (Acceptance):** Integration corpus must include parallel scenarios

### What's Missing Today

1. **No goroutine execution model in §02-architecture.md:** The concurrency model (L557-571) describes goroutine topology for a single run but doesn't specify parallel block execution model. Does each branch get its own sub-RunHandle? How do we enforce variable isolation?

2. **No trace event sequencing guarantee for concurrent events:** §06-runtime-events.md guarantees `sequence` field is monotonic within a run but doesn't define ordering semantics when events are emitted from concurrent goroutines. Is it insertion order (mutex-protected append)? Goroutine ID order? Undefined?

3. **No test infrastructure exists yet:** The only test harness mentioned in §08-testing-and-acceptance.md is `exttest.NewHarness` for extensions. No `FakeStepExecutor`, `ConcurrentEventCollector`, or deterministic scheduler.

4. **No parallel block fixture in integration corpus:** §08-testing-and-acceptance.md L186-193 lists required fixtures — no parallel scenario mentioned.

5. **No decision on nested parallel blocks:** Is `parallel` step allowed inside a parallel branch? If yes, what's the goroutine limit? If no, validation must reject it.

6. **No specification of external effect isolation:** Spec says variable writes are isolated until join, but what about tool invocations? If two branches both call `kubectl apply -f deployment.yaml`, who wins? Is this a correctness bug or expected behavior?

---

## Gap 2: Wait-on-Event Testing

### What the Problem Actually Is

The spec (§03-schema-vnext.md:813-851) defines a `wait_for_event` step kind that:
- **Pauses execution** until an external event arrives
- **Four event sources:** `webhook`, `message`, `signal`, `channel`
- **Only works in `gert serve` mode** (requires EventDispatcher, §02-architecture.md:603-637)
- **Timeout behavior:** `on_timeout: fail | continue | branch:<step-id>`
- **Filter and payload validation:** events must match `filter` map and optional JSON Schema

The spec does NOT define how to test this without a real external event source. Specifically:

1. **No fake EventDispatcher:** How do we test webhook delivery without an HTTP server?
2. **No event injection mechanism:** How do we simulate an event arriving at time T during a test?
3. **Timeout race conditions:** How do we deterministically test that timeout fires correctly when event arrives 1ms before vs. 1ms after timeout?
4. **Event ordering:** If two `wait_for_event` steps are in sequence and both wait for the same channel, does the first one consume the event or do both see it?
5. **Cancellation interaction:** What happens if a run is cancelled (SIGINT) while waiting for an event? Does the wait immediately fail or does it check for pending events first?
6. **Filter match failure:** If an event arrives but doesn't match the `filter`, is it dropped silently or logged?

### What Test Infrastructure Is Needed

To validate `wait_for_event` correctness, we need:

1. **FakeEventDispatcher:**
   - Package: `pkg/testutil`
   - Interface: Implements `EventDispatcher` from §02-architecture.md:608-621
   - Capability: Allows test code to call `dispatcher.Inject(eventID, payload, delay)` to schedule event delivery
   - Use: Inject events at controlled times without HTTP/message queue infrastructure

2. **TimeController for wait tests:**
   - Package: `pkg/testutil`
   - Type: `type TimeController struct { now time.Time; timers map[string]<-chan time.Time }`
   - Capability: Replace `time.Now()` and `time.After()` with controllable fake time
   - Use: Advance time to timeout without real wall-clock waits; test timeout vs. event arrival race deterministically

3. **EventArrivalSimulator:**
   - Package: `pkg/testutil`
   - Function: `func SimulateEvent(dispatcher, eventID, payload, arrivalDelay) error`
   - Capability: Schedule an event to arrive N milliseconds from now (using fake time)
   - Use: Test scenarios like "event arrives 100ms before timeout" or "event arrives 100ms after timeout"

4. **Wait-step test harness:**
   - Package: `pkg/engine_test`
   - Function: `func TestWaitForEvent(t *testing.T, source EventSource, filter, timeout, arrivalTime) TraceAssertion`
   - Capability: Declarative wait-step test setup with fake dispatcher and fake time
   - Use: Table-driven tests for all event source × timeout × filter combinations

5. **EventFilter validator:**
   - Package: `pkg/engine` (production code)
   - Function: `func MatchesFilter(payload map[string]any, filter map[string]string) bool`
   - Unit tests: `pkg/engine/filter_test.go` with table-driven cases for exact match, partial match, type mismatch, missing key

6. **Stub implementations for each event source:**
   - `FakeWebhookServer` (in-memory HTTP handler)
   - `FakeMessageQueue` (in-memory channel queue)
   - `FakeSignalHandler` (deliver signals programmatically)
   - `FakeChannel` (just a Go chan with controllable send)

### Which Phases Are Affected

- **Phase 3 (Runtime Core & Trace):** Wait semantics, event arrival, run state `WAITING`
- **Phase 5 (Step Types):** `wait_for_event` step execution
- **Phase 9 (`gert serve`):** EventDispatcher implementation, webhook endpoint, timeout heap
- **Phase 11 (Evidence & Replay):** Replay of wait steps (must inject event at same sequence point)
- **Phase 13 (Acceptance):** Integration corpus needs wait-step scenarios

### What's Missing Today

1. **No fake/stub EventDispatcher:** §02-architecture.md defines the interface but no test double exists. Without it, `wait_for_event` is untestable except in full integration mode.

2. **No specification of event consumption semantics:** If step A waits for `channel:foo` and step B (later in sequence) also waits for `channel:foo`, does step A consume the event (step B blocks forever) or does the channel broadcast (both see it)? Spec is silent.

3. **No specification of filter matching algorithm:** §03-schema-vnext.md says `filter` is "key-value pairs that must match" but doesn't define matching rules. Exact equality? Substring? Regex? Type coercion (JSON number vs. string)?

4. **No timeout enforcement contract:** §02-architecture.md:636 says "min-heap of timeout deadlines; background goroutine fires on earliest deadline" but doesn't specify what "fire" means. Emit `step/failed`? Emit `step/timeout`? Jump to `branch:<step-id>` if `on_timeout: branch:fallback`?

5. **No trace event for event arrival:** §06-runtime-events.md has no `event/received` or `wait/resumed` event. How does the trace distinguish "step is waiting" from "step completed because event arrived"? Audit trail gap.

6. **No wait-step fixture in integration corpus:** §08-testing-and-acceptance.md L186-193 doesn't include wait scenarios.

7. **No decision on cancellation during wait:** If a run is in `WAITING` state and receives `Cancel()`, does it fail immediately or does it gracefully unregister listeners first? Spec is silent.

---

## Gap 3: Multi-OS / Multi-Platform Considerations

### What the Problem Actually Is

The spec assumes gert v2 runs on **Linux, macOS/Darwin (dev), and potentially Windows**. Key OS-sensitive behaviors:

1. **File path separators:** YAML runbooks use `/` paths (Unix convention). Windows uses `\`. Does the parser normalize? Do tool definitions need OS-specific path fields?

2. **Process spawning:** `cli` steps call `exec.Command`. On Windows, `.exe` extension, `cmd.exe` shell, different signal handling (no SIGTERM). Spec doesn't address this.

3. **Signal handling:** `wait_for_event` with `source: signal` (§03-schema-vnext.md:830). Windows has no POSIX signals. What signals are supported? SIGINT, SIGTERM only? What about SIGUSR1?

4. **Temp directories:** Runbooks might use `/tmp` (Unix) vs. `%TEMP%` (Windows). Does gert provide a `{{ .gert.temp }}` variable?

5. **Stdio transport for tools:** Extensions communicate over stdin/stdout (§04-extension-runtime.md). Windows has different pipe buffering and newline conventions (`\r\n` vs. `\n`). Does this break JSON-RPC framing?

6. **File locking for trace append:** §02-architecture.md:455 says trace writes use `O_APPEND` for Unix atomicity. Windows has different file locking. Is append atomic on Windows?

7. **Sandboxing enforcement:** §04-extension-runtime.md:283 says "pledge/unveil on OpenBSD, seccomp on Linux, host-side validation on other platforms." So Windows/macOS have weaker sandboxing. Is this acceptable?

The spec does NOT define:
- Which behaviors are OS-sensitive and need abstraction?
- Which behaviors are Linux-only by design?
- What test matrix covers cross-OS correctness?

### What Test Infrastructure Is Needed

To validate cross-OS correctness, we need:

1. **Platform abstraction layer:**
   - Package: `pkg/platform`
   - Types: `type Platform interface { TempDir() string; PathSeparator() string; NormalizePath(string) string; SupportedSignals() []Signal }`
   - Implementations: `LinuxPlatform`, `DarwinPlatform`, `WindowsPlatform`
   - Use: Centralize OS-specific behavior; mock for unit tests

2. **FakePlatform for unit tests:**
   - Package: `pkg/testutil`
   - Type: `type FakePlatform struct { tempDir string; separator string; signals []Signal }`
   - Capability: Simulate different OS behaviors without running on that OS
   - Use: Test path normalization on Windows without a Windows machine

3. **Cross-platform subprocess executor:**
   - Package: `pkg/engine`
   - Function: `func spawnCommand(platform Platform, cmd []string) (*exec.Cmd, error)`
   - Capability: Wrap `exec.Command` with OS-specific logic (`.exe` suffix on Windows, shell selection)
   - Unit tests: Table-driven tests with FakePlatform for each OS

4. **Trace append atomicity test:**
   - Package: `pkg/store_test`
   - Test: `TestTraceAppendConcurrency` (runs on all OS)
   - Capability: Spin up N goroutines, each appending 1000 events to the same trace file; assert no corruption
   - Use: Validate that `O_APPEND` or equivalent Windows mechanism prevents interleaved writes

5. **Stdio transport newline normalization:**
   - Package: `pkg/transport`
   - Function: `func normalizeNewlines(reader io.Reader) io.Reader` (wraps reader, converts `\r\n` → `\n`)
   - Unit tests: `TestStdioFraming` with CRLF and LF inputs
   - Use: Ensure JSON-RPC framing works on Windows

6. **CI matrix test harness:**
   - GitHub Actions workflow: `.github/workflows/test.yml`
   - Matrix: `os: [ubuntu-latest, macos-latest, windows-latest]`
   - Per-OS conditionals: Skip signal tests on Windows, skip pledge/unveil tests on non-OpenBSD
   - Use: Run unit + integration tests on all three platforms

### Which Phases Are Affected

- **Phase 1 (Parser & Schema):** Path normalization
- **Phase 3 (Runtime Core & Trace):** File append atomicity, signal handling
- **Phase 5 (Step Types):** `cli` step process spawning, `wait_for_event` signal source
- **Phase 6 (Tool Runtime):** Stdio transport newline handling
- **Phase 7 (Extension Host):** Sandboxing enforcement (platform-dependent)
- **Phase 13 (Acceptance):** Cross-OS integration tests

### What's Missing Today

1. **No platform abstraction layer:** Code assumes Unix paths and signals. No `pkg/platform` exists yet.

2. **No decision on Windows support status:** Is Windows Tier 1 (must work), Tier 2 (best-effort), or unsupported? This changes the test matrix cost.

3. **No trace append atomicity guarantee on Windows:** Spec says "Unix atomicity" but doesn't define Windows equivalent. Need to research Windows file append semantics and document decision.

4. **No stdio newline normalization:** JSON-RPC framing might break on Windows if extension writes `\r\n`. No code exists to handle this yet.

5. **No signal source restrictions:** §03-schema-vnext.md:830 allows `source: signal` but doesn't say which signals. Need to define allowed signal set (e.g., SIGINT, SIGTERM only) and reject SIGUSR1 on Windows at validation time.

6. **No temp dir abstraction:** Runbooks can't portably reference temp dirs. Need `{{ .gert.temp }}` or similar.

7. **No path separator normalization in parser:** If a runbook has `include: subfolder\runbook.yaml` (Windows-style), does it work on Linux? Parser needs to normalize.

8. **No CI matrix configured:** No `.github/workflows/` exists yet. When it does, need OS matrix.

---

## Architectural Decisions Required

### Decision 1: Event Sequencing for Parallel Blocks

**Question:** When two parallel branches emit events concurrently, what ordering is guaranteed in the trace?

**Options:**
1. **Goroutine schedule-dependent** (non-deterministic): Events appended in arrival order at trace writer (mutex-protected). Simplest but breaks golden trace testing.
2. **Branch-order deterministic:** Events from branch 0 always appear before branch 1, even if branch 1 finishes first. Requires per-branch event buffering until join.
3. **Timestamped with microsecond precision:** Events reordered by `timestamp` field before trace append. Deterministic but complex.

**Recommendation:** Option 2 (branch-order deterministic). Trade-off: adds complexity (per-branch buffer) but preserves replay determinism and golden trace compatibility.

**Impact:** Phase 3, Phase 5, Phase 11

### Decision 2: Nested Parallel Blocks

**Question:** Is a `parallel` step allowed inside a parallel branch?

**Options:**
1. **Allow:** No validation restriction. Goroutine explosion risk (4 branches × 4 branches = 16 concurrent goroutines).
2. **Forbid:** Semantic validation rejects nested parallel. Clear bound on concurrency (max = branch count).
3. **Limit depth:** Allow 1 level of nesting, forbid deeper.

**Recommendation:** Option 2 (forbid). Rationale: Goroutine explosion risk, no clear use case for nested parallelism in runbooks. Can lift restriction in v2.1 if user demand emerges.

**Impact:** Phase 1 (semantic validation), Phase 5 (parallel execution)

### Decision 3: Event Consumption Semantics for `wait_for_event`

**Question:** If two `wait_for_event` steps wait for the same `channel:foo`, does the first one consume the event?

**Options:**
1. **Consume:** First step consumes event, second step blocks forever (or times out). Simple, matches Go channel semantics.
2. **Broadcast:** Both steps receive the event. Requires EventDispatcher to track all waiting steps per channel.

**Recommendation:** Option 1 (consume). Rationale: Matches Go channel semantics, simpler implementation. If broadcast is needed, use a different `event.id` for each step.

**Impact:** Phase 9 (EventDispatcher)

### Decision 4: Trace Event for Event Arrival

**Question:** Does event arrival emit a trace event?

**Options:**
1. **No new event:** `step/completed` is sufficient (implies event arrived).
2. **Add `event/received`:** Emitted when event arrives, before step resumes. Captures event payload in trace.

**Recommendation:** Option 2 (add `event/received`). Rationale: Audit trail requirement — need to know *when* event arrived and *what* the payload was, not just that the step completed.

**Impact:** Phase 3 (event catalog), Phase 9 (EventDispatcher), §06-runtime-events.md (spec update needed)

### Decision 5: Windows Support Tier

**Question:** Is Windows a Tier 1 platform (must work) or Tier 2 (best-effort)?

**Options:**
1. **Tier 1:** Full CI matrix, all tests must pass on Windows. Blocks release if Windows-specific bug.
2. **Tier 2:** CI advisory (allowed to fail), Windows bugs are P2 not P0.
3. **Unsupported:** Document Linux/macOS only.

**Recommendation:** Option 2 (Tier 2) for v2.0, promote to Tier 1 in v2.1 if adoption warrants. Rationale: Most gert users are on Linux/macOS (DevOps/SRE context). Windows support is valuable but not load-bearing for MVP.

**Impact:** Phase 13 (CI matrix), documentation

### Decision 6: Signal Source Support

**Question:** Which signals are supported for `wait_for_event` with `source: signal`?

**Options:**
1. **POSIX signals only:** SIGINT, SIGTERM, SIGUSR1, SIGUSR2. Validation fails on Windows.
2. **Portable subset:** SIGINT only (works on Windows as Ctrl+C). Reject others at validation time.
3. **OS-specific allow list:** Linux allows full POSIX set, Windows allows SIGINT only, macOS allows POSIX set.

**Recommendation:** Option 3 (OS-specific allow list). Rationale: Balances portability (SIGINT works everywhere) with power-user needs (SIGUSR1 for custom workflows on Linux).

**Impact:** Phase 1 (validation), Phase 5 (wait step), §03-schema-vnext.md (spec update needed)

---

## Test Matrix Summary

### Parallel Block Test Matrix

| Scenario | Branches | Join Policy | Failure Mode | Coverage |
|----------|----------|-------------|--------------|----------|
| Happy path | 2 | all | none | Event ordering, variable merge |
| Partial failure | 3 | all | branch 2 fails | on_failure: fail (run fails) |
| Partial failure continue | 3 | all | branch 2 fails | on_failure: continue (run succeeds, captures from branches 1&3) |
| Majority join | 5 | majority | 2 fail, 3 succeed | Join after 3 completions |
| Timeout in branch | 2 | all | branch 1 hangs | Verify cancellation propagates to hanging branch |
| Variable conflict | 2 | all | both write same var | Last-writer-wins warning emitted |
| Nested parallel (forbidden) | 1 | all | branch contains parallel | Validation rejects |

### Wait-on-Event Test Matrix

| Scenario | Source | Timeout | Event Arrival | Filter | Coverage |
|----------|--------|---------|---------------|--------|----------|
| Happy path | webhook | none | arrives immediately | matches | Step completes, captures payload |
| Timeout before event | webhook | 10s | arrives at 11s | matches | Step fails (on_timeout: fail) |
| Timeout after event | webhook | 10s | arrives at 5s | matches | Step completes |
| Filter mismatch | webhook | 10s | arrives at 5s | doesn't match | Step blocks, times out |
| Filter match | channel | 30s | arrives at 2s | matches | Step completes |
| Cancellation during wait | channel | none | never arrives | matches | Cancel() → step fails |
| Event consumption | channel | none | 1 event, 2 steps | matches | Step 1 consumes, step 2 times out |

### Multi-OS Test Matrix

| Phase | Test | Linux | macOS | Windows | Notes |
|-------|------|-------|-------|---------|-------|
| 1 | Path normalization | ✓ | ✓ | ✓ | Parser converts `\` → `/` |
| 3 | Trace append atomicity | ✓ | ✓ | ✓ | Concurrent goroutines append without corruption |
| 5 | CLI step spawn | ✓ | ✓ | ✓ | `.exe` suffix on Windows, shell selection |
| 5 | Signal wait (SIGINT) | ✓ | ✓ | ✓ | Portable signal |
| 5 | Signal wait (SIGUSR1) | ✓ | ✓ | ✗ | Validation rejects on Windows |
| 6 | Stdio newline handling | ✓ | ✓ | ✓ | CRLF → LF normalization |
| 7 | Sandboxing (seccomp) | ✓ | ✗ | ✗ | Linux-only; fallback to validation on others |
| 13 | Integration corpus | ✓ | ✓ | advisory | Full suite on Linux/macOS, subset on Windows |

---

## Next Actions for Each Phase

### Phase 1 (Parser & Schema)
- Add semantic validation: reject nested `parallel` steps
- Add platform-aware validation: reject unsupported signals on Windows
- Add path normalization: convert Windows `\` to `/` in all path fields

### Phase 3 (Runtime Core & Trace)
- Implement per-branch event buffering for deterministic trace ordering (Decision 1)
- Add `event/received` event to catalog (Decision 4)
- Research Windows file append atomicity, document findings

### Phase 5 (Step Types)
- Implement `parallel` step with goroutine-per-branch and join semantics
- Implement `wait_for_event` step with timeout and filter matching
- Wrap `exec.Command` in platform abstraction for OS-specific spawning

### Phase 6 (Tool Runtime)
- Add stdio newline normalization for Windows CRLF handling

### Phase 7 (Extension Host)
- Document sandboxing limitations on Windows/macOS (host-side validation only)

### Phase 9 (`gert serve`)
- Implement EventDispatcher with consume semantics (Decision 3)
- Implement timeout heap and background goroutine

### Phase 11 (Evidence & Replay)
- Add replay support for parallel blocks (deterministic ordering required)
- Add replay support for wait steps (inject event at same sequence point)

### Phase 13 (Acceptance)
- Build `FakeStepExecutor`, `ConcurrentEventCollector`, `FakeEventDispatcher` in `pkg/testutil`
- Add parallel block fixtures to integration corpus
- Add wait-step fixtures to integration corpus
- Configure CI matrix for Linux/macOS/Windows (Windows advisory)
- Add cross-OS integration tests per matrix above
