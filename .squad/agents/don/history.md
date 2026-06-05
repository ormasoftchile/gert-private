# Don — Project History

## Learnings

### 2026-06-03 — Declaration/Consent Scenario Gap Analysis

**Deliverable produced:** `.squad/decisions/inbox/don-declaration-runtime-gaps.md`

**Gap inventory (10 gaps identified):**

| # | Gap | Priority | New Primitive |
|---|---|---|---|
| G-1 | Signature capture | P0 | `UserInputKind.Signature`, `DeclarationCollectedEvent` |
| G-2 | Identity proofing at declaration moment | P0 | `IdentityProofingLevel` on `UserInputRequest` |
| G-3 | Witness / co-signer flow | P0 | `IWitnessGate`, `WitnessAttestedEvent` |
| G-4 | Presented-document version hash | P0 | `document_hash` + `document_version` on trace event |
| G-5 | Multi-language / locale provenance | P1 | `Locale` (BCP-47) on `DeclarationCollectedEvent` |
| G-6 | On-behalf-of / capacity (subject ≠ declarant) | P1 | `DeclarationPrincipal` record |
| G-7 | Time-bounded validity (expiry) | P1 | `valid_until` field + `IDeclarationRegistry.GetEffectiveStateAsync` |
| G-8 | Revocation (inverse event, no rewrite) | P1 | `DeclarationRevokedEvent` + `IDeclarationRegistry` |
| G-9 | Contextual PII in free-text fields | P2 | `PiiZone` policy flag (no new engine) |
| G-10 | QTSP integration (eIDAS/ESIGN/Ley 19.799) | P2 | `UserInputKind.QualifiedSignature`, `QualifiedSignatureReceivedEvent` |

**What already fits well:**
- JSONL append-only trace is the right substrate for declaration registration — synchronous-before-dispatch invariant directly satisfies "must be registered before process proceeds."
- Service Bus at-most-once prevents duplicate declaration collection on worker restart.
- RE2 redaction handles structured PII (RUT, SSN, IBAN) well — gaps only in free-text narrative PII.
- `IApprovalGate` covers genuine downstream authorizers (underwriters, administrators) but NOT witnesses or co-signers.
- `Confirmation` kind is the closest current primitive to a declaration acknowledgment — insufficient for legal weight without G-1 and G-2.
- `client="web"` RunOptions tagging already fits per-channel audit needs.

**Proposed new trace event types:** `DeclarationCollectedEvent`, `WitnessAttestedEvent`, `DeclarationRevokedEvent`, `DeclarationValidityCheckedEvent`, `QualifiedSignatureReceivedEvent`.

**Proposed new interface:** `IWitnessGate` (parallel to `IApprovalGate`; fires inside `IRunHandle.NextAsync()`).

**Proposed extensions:**
- `UserInputKind.Signature` and `UserInputKind.QualifiedSignature` added to enum.
- `DeclarationContext` added to `UserInputRequest` for declaration-type steps.
- `DeclarationPrincipal` record for on-behalf-of flows.
- `IDeclarationRegistry` for effective-state queries (Active | Expired | Revoked).

**Non-goals confirmed:** GERT is NOT a TSP, does NOT store biometrics, does NOT render legal text, does NOT adjudicate legal validity, does NOT enforce per-jurisdiction compliance rules.

**Chilean legal context:** RE2 patterns for RUT format already expressible. Ley 19.799 (Firma Electrónica) compliance requires QTSP integration (G-10) for advanced e-signature tier — GERT records TSP tokens, does not issue signatures.

---

### 2026-06-03 — C# Governance Parity Design

**Deliverable produced:** `design/web-platform/c-sharp-governance-parity.md`

**Key design decisions made:**

- **Pipeline mapping:** Full Go-to-C# interface mapping documented. `IRunHandle.NextAsync()` is the governance boundary — identical to Go's `RunHandle.Next()`. Every stage (Parser → Planner → Runtime → RunHandle) has a named C# equivalent with method signatures.
- **IApprovalGate stubbed for A6/A8 swap:** Designed so `ServiceBusApprovalGate` (A6: held thread, Service Bus reply queue) can be swapped for `DurableApprovalGate` (A8: `WaitForExternalEvent`) without changing `IRunHandle` or the governance layer. The runbook contract sees only `IApprovalGate`.
- **IStepRunner enforcement via `internal sealed`:** `IStepRunner` and all concrete runners (`CliStepRunner`, `ToolStepRunner`, etc.) are `internal` to `Gert.Runtime.Core`. No `InternalsVisibleTo` grants to adapter assemblies. Bypass is a compile error, not a code review finding. Two Roslyn analyzers (`GERT0001`, `GERT0002`) enforce this and the RE2 namespace restriction at build time.
- **RE2 via `Google.Re2` NuGet:** All redaction evaluation uses `Google.Re2.Regex`, never `System.Text.RegularExpressions`. Patterns containing lookahead/lookbehind/possessive quantifiers will throw at startup (fail-fast). Named group syntax difference (`(?P<name>...)`) flagged as open question OQ-5.
- **Template engine gap:** Go `text/template` has no C# equivalent. Option A (minimal port) is recommended for MVP. Option B (WASM compiled Go evaluator) is the fallback. 10 canonical test vectors defined (TV-TMPL-001..010).
- **Trace synchronous write:** `IRunStore.WriteTraceAsync()` must complete before step dispatch. Call order in `RunHandleImpl.NextAsync()` is: governance check → emit event → **synchronous trace write** → step dispatch. Fire-and-forget is explicitly forbidden.
- **Validation gate:** 10 gates (G-01..G-10) defined. Go `gert replay` on C#-generated traces (G-06) is the strongest parity check. No production promotion with any gate failing. Roslyn analyzers enforce G-08 and G-09 at build time.
- **60 minimum test vectors** across 12 categories (allowlist, denylist, env blocking, redaction, approval gates, template evaluation).

**Open questions filed:** OQ-1 (template strategy), OQ-2 (approval reply queue topology), OQ-3 (canonical Go test harness), OQ-4 (sequence counter strategy), OQ-5 (RE2 named group syntax).

---

### 2026-06-03 — Execution Adapter Pattern Review

**Patterns reviewed:** Queue-triggered worker, HTTP-triggered Container App, Durable execution (Azure Durable Functions), Sidecar/agent model.

**Key governance concerns identified:**

- GERT's governance layer (allowlist/denylist, env var filtering, contract risk assessment, approval gates) fires entirely inside the **Runtime Core** during the `RunHandle.Next()` loop. It cannot be preserved by any pattern that bypasses `Runtime.Start → RunHandle.Next`.
- The JSONL trace file is written **synchronously inside the Runtime Core before each action proceeds**. Any pattern that writes trace events from outside the Go binary (e.g., a Function orchestrator writing its own log) would produce a trace that is NOT the authoritative GERT trace and would fail replay determinism.
- The `RunHandle` interface is **stateful and sequential** (`Next()` must not be called concurrently, re-entrant only per run). It cannot be split across multiple processes or Function invocations.
- Splitting execution into per-step Azure Durable Function activities is a **governance red line**: each activity invocation re-enters the binary cold, losing the in-process governance engine state and potentially allowing steps to execute without the full pre-flight chain.
- Approval gates (`RunHandle.Approve()`) and evidence submission (`RunHandle.SubmitEvidence()`) require a **live, running Go process** that holds the `RunHandle`. Any pattern that doesn't keep the worker alive through the pause window cannot support interactive runbooks without an explicit resume mechanism (checkpoint → reload from RunStore).
- The `client: "web"` field in `run/started` must be set correctly. Forgetting to identify the client means audit trail queries filtering by client will miss web-triggered runs.
- Queue-triggered worker is the strongest match: decoupled submission, worker owns the full Runtime Core, governance fires completely, trace goes to Blob Storage, approval gates handled via reply messages or a dedicated interaction queue.

**2026-06-04:** Brainstorm output merged to decisions.md. Governance red lines formalized and documented. Pattern evaluation matrix recorded. Requires team ratification before implementation. Orchestration log created.

---

### 2026-06-03 — C# Runtime Option Analysis

**Trigger:** User confirmed they are NOT attached to the Go binary for web execution. Open to a C# reimplementation of GERT governance for the Azure backend.

**What this unlocks:**
- Durable Functions Orchestrators can now legitimately implement `RunHandle.Next()` semantics — the orchestrator IS the governance engine, not a bypass of it. This was a red line when Go binary was fixed because each activity invocation would cold-start the binary, losing governance state. In C#, governance fires inside the orchestrator before each activity, which is correct.

---

## [Archived Sessions — Pre-2026-06-04]

**Historical summary:** Don completed 7 major analysis cycles (2026-06-03 through 2026-06-04T02:50:37Z):
1. Declaration/consent gap analysis (10 gaps, 5 new primitives proposed)
2. C# governance parity design (10 validation gates, Roslyn analyzer strategy)
3. Execution adapter pattern review (Red Line architecture decisions)
4. C# runtime option analysis (BackgroundService + Durable orchestration design)
5. Choice gate runtime contract (Interactive waiting patterns)
6. User input gate correction (Generalized IUserInputGate)
7. Interactive waiting patterns cross-review

**Key decisions adopted:** IRunHandle.NextAsync as governance boundary, Google.Re2 for redaction, IUserInputGate interface (not IChoiceGate), Roslyn analyzers GERT0001/GERT0002, 10 validation gates before C# production.

**Active open questions:** OQ-1 through OQ-5 (template strategy, approval queue topology, sequence counter, RE2 named group syntax).

---

## Team Directive — 2026-06-04T17:15:45-07:00

**Leslie uses he/him pronouns.** All team members and the coordinator must refer to Leslie with he/him going forward.

