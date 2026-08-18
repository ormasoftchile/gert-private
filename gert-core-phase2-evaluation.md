# Phase 2 Architectural Evaluation
## VS Code Authenticated MCP Runtime Binding

**Date:** 2026-08-17
**From:** Barbara (Lead/Architect, Gert Core)
**Ruling:** ACCEPT WITH MODIFICATIONS
**Disposition:** Feasible but sequencing-dependent. Three blocking prerequisites must be satisfied before Phase 2 work begins. The scope as stated is accurate and well-bounded, but it assumes infrastructure that does not exist in either repo today, and it implicitly requires a core-side fix (output contract enforcement) that the consumer has correctly identified but not explicitly scoped as Phase 2 work.

---

## 1. Ruling and Rationale

**ACCEPT WITH MODIFICATIONS.**

The ask is technically sound and architecturally clean. The consumer has, again, done their homework — every scope point maps to a real gap or a real mechanism in the codebase. The transport-chain addition pattern is well-understood after Phase 1B. The executor-kind constraint is correctly identified. The security model (loopback + per-session capability secret) is the right shape.

The modifications are:

1. **Output contract enforcement is prerequisite work, not Phase 2 work.** The consumer's safety rule ("the extension may normalize an MCP result only into the tool's declared output fields; missing, unknown, or type-incompatible output fails closed") and their mutation control ("remove the output contract parity check and prove drift is caught") both assume a runtime check that **does not exist** for non-substituted tools in Gert core. This is the fifth instance of the declared-but-unenforced pattern class. The extension can and should do its own normalization, but the core-side check is independently required: a `vscode-mcp` tool result that silently resolves to nil in a capture expression is the same failure mode we built `TestCLI_ReachabilityGate` to prevent — code that parses correctly and does nothing. This is core work, not extension work, and it must land before the bridge is load-bearing.

2. **Three blocking prerequisites in `gert-vscode` must be resolved first** (§4 below).

3. **The existing `serve` security issue must be fixed before or concurrently with Phase 2**, not after (§6 below).

---

## 2. The Gap That Matters Most: Output Contract Enforcement

**Finding:** For non-substituted tools (native, mcp-stdio, mcp-http, and the proposed vscode-mcp), the `returns:` declaration in a `.tool.yaml` is **informational only**. The output evaluation, coercion, and ENUM-009 checks in `internal/executor/tool.go:executeSubstitution` are gated on `execute.kind: runbook`. For all other tool kinds, a binding that returns different fields causes capture expressions in the consuming runbook to silently resolve to nil or empty. Gert does not detect this.

**Consequence:** The consumer's required mutation control — "remove the output contract parity check and prove drift is caught" — cannot be satisfied today because no such check exists at runtime. Their Phase 1B `pkg/contractparity` harness validates contracts at **test time** against known bindings. That is necessary but not sufficient: a runtime binding that drifts (e.g., an MCP tool adds a field, renames a field, or changes a type) will silently break runbooks without any error.

**Assessment:** This is a **fifth** instance of the declared-but-unenforced class that `TestCLI_ReachabilityGate` was created to catch. The pattern: a schema declares a constraint (here, `returns:`), tests may validate it in isolation, but the runtime execution path never checks it, so violations are silent. The same class that hit `ProfileToolOverride.Endpoint`, `RequiresCapabilities`, `AllowedEnvironments`, and the original transport-mode dead arms.

**Recommendation:** Add output contract enforcement in `internal/executor/tool.go` for ALL non-substituted tool executions, not only `vscode-mcp`. The check should:
- Compare returned keys against `returns:` declared keys
- Reject unknown keys (fail closed, not ignore)
- Reject missing required keys
- Type-check values against declared types where declared
- Be testable via the same mutation-control pattern: remove the check, prove a drift is not caught

This is **core work** (Gert repo). The extension does its own normalization as a defense-in-depth layer, but the authoritative check belongs in the engine. Estimate: 2–3 days, parallelizable with other Phase 2 work but must merge before the bridge is used for anything beyond smoke tests.

---

## 3. Executor-Kind Hazard

**Risk:** If the `vscode-mcp` transport is registered under a new step kind (e.g., `"vscode-tool"` instead of `"tool"`), it bypasses the classification injection at `engine.go:executeStep` lines ~527–543. `ToolClassification` would be nil. For `read-only` actions this is conservative-by-accident (nil classification triggers INDETERMINATE), but it is not correct governance — it is the absence of governance misread as safe governance.

**Ruling:** The `vscode-mcp` binding MUST remain within the existing `"tool"` executor kind. It is a transport variant, not a new executor. The same approval chokepoint, classification injection, and INDETERMINATE semantics must apply.

**Structural enforcement mechanism (required, not advisory):**

1. **Reachability registry extension.** `TestCLI_ReachabilityGate` already maintains a registry of entries that must be reachable. Add a **negative registry**: a test that enumerates all registered executor kinds that invoke tools and asserts each one passes through the `executeStep` classification injection. Concretely: a test that greps `engine.go` for `step.Kind ==` switch arms and asserts that every arm containing a tool invocation call (identified by function signature, not string matching) also contains the classification-injection block. This is fragile to refactoring but catches the exact error mode.

2. **Preferred alternative — structural.** Extract the classification-injection + approval-gate logic from `executeStep` into a named function (e.g., `applyToolGovernance`). This makes omission visible and greppable: any tool invocation path that does not call `applyToolGovernance` is a findable defect. However, this alone is a **convention**, not a compile-time guarantee — a caller can simply not call the function. To make it fail closed: have `applyToolGovernance` return an unexported struct type (e.g., `type governanceProof struct{ ... }` with no exported fields and no public constructor) that the tool invocation function requires as a parameter. Because only `applyToolGovernance` can construct a `governanceProof`, a caller that skips governance cannot obtain one and will not compile. This is a real technique and it does fail closed — the Go type system enforces it, not convention. Pair this with a registry test (mechanism 1) as defense-in-depth, since the structural mechanism protects against bypass within Go code but not against an entirely new executor registered under a different step kind that never calls the tool invocation function at all.

3. **Code review gate (minimum).** If neither structural mechanism ships in Phase 2, add an explicit `// GOVERNANCE: any executor kind that invokes a tool MUST pass through applyToolGovernance` comment and a review checklist item. This is the weakest option and I do not recommend it as the sole control.

---

## 4. Sequencing and Prerequisites

### Blocking Prerequisites (must complete before Phase 2 delivery)

**P0: `gert-vscode` reproducibility remediation.**
The extension repo is unbuildable from a clean checkout. `src/enumInputs.ts`, `src/serverRoot.ts`, and the entire `test/` directory are untracked. Seven tracked files have uncommitted modifications. `extension.ts` imports untracked files, so `tsc` fails on clean HEAD. This is the identical class of failure we just spent Phase 1B remediating in `gert` core, and we made reproducibility a formal release gate with this consumer. We cannot ship Phase 2 features on top of an unreproducible base. **Estimate: 0.5–1 day. Owner: whoever has commit access to `gert-vscode`.**

**P1: Raise `engines.vscode` minimum.**
Current floor is `^1.85.0`. The `vscode.lm.tools` and `vscode.lm.invokeTool()` APIs require a higher minimum (sources indicate 1.90–1.91; exact version must be verified against VS Code release notes before committing). The `@types/vscode` devDependency must match. This is a breaking change for users on older VS Code versions. **Estimate: 0.5 day. Owner: extension.**

**P2: Add extension-host test harness.**
The extension has only `node --test test/*.test.js` today. No `@vscode/test-electron` or `@vscode/test-cli`. Every item of the consumer's required extension test coverage (tool discovery, invocation arguments, result normalization, authorization-unavailable, timeout, cancellation) requires an extension host to exercise `vscode.lm.tools` and `vscode.lm.invokeTool()`. Without this harness, the required automated coverage is structurally impossible. **Estimate: 1–2 days. Owner: extension.**

### Ordered Sequence

| Order | Item | Blocks | Owner | Est. |
|-------|------|--------|-------|------|
| P0 | Extension reproducibility | Everything extension-side | Extension | 0.5–1d |
| P1 | Raise engines.vscode | Extension-host tests, LM API usage | Extension | 0.5d |
| P2 | Extension-host test harness | All required extension test coverage | Extension | 1–2d |
| 1 | Output contract enforcement (core) | Mutation control proof | Core | 2–3d |
| 2 | `vscode-mcp` transport mode (core) | Bridge integration | Core | 2–3d |
| 3 | Loopback bridge server (core) | Extension integration | Core | 3–4d |
| 4 | Extension bridge client + tool resolution | End-to-end proof | Extension | 3–4d |
| 5 | `serve` security fix | Independent but should ship with P2 | Core/Extension | 1d |
| 6 | End-to-end acceptance + mutation controls | Delivery | Joint | 2–3d |

**Parallelization:** P0→P1→P2 is serial (extension). Items 1 and 2 are parallel with each other and with P0–P2. Item 3 can start after Item 2. Item 4 requires P2 and Item 3. Item 5 is independent. Item 6 requires everything.

**Total estimate:** 10–14 days parallelized; 16–20 days serial. The critical path runs through the extension prerequisites.

---

## 5. Ownership Split

### Gert Core Owns

| Deliverable | Notes |
|---|---|
| `vscode-mcp` transport constant + validation | Five-point transport chain (schema, tool, validate, scan, runtime) |
| `runtime.Invoke` arm for `vscode-mcp` | Loopback HTTP call to bridge, context-aware cancellation |
| Output contract enforcement for non-substituted tools | New, prerequisite, all transports benefit |
| Loopback bridge server component | Bind 127.0.0.1 only, per-session capability secret, JSON-RPC framing, deadline/cancellation, version negotiation |
| Request-ID generation and duplicate-request registry | New infrastructure |
| Bridge-version mismatch detection and error | New infrastructure |
| Credential sweep extension for capability secret | Reuse pattern from Phase 1B `auth_credential_leak_test.go` |
| `serve` loopback-only + auth fix | Independent security fix |

### Extension Owns

| Deliverable | Notes |
|---|---|
| Reproducibility remediation (P0) | Commit untracked files, verify clean `tsc` |
| `engines.vscode` raise (P1) | Verify exact minimum for `vscode.lm` APIs |
| Extension-host test harness (P2) | `@vscode/test-electron` or equivalent |
| Bridge client (HTTP to loopback) | Initiate connection, send capability secret, handle version negotiation |
| Tool resolution via `vscode.lm.tools` | Discover registered MCP tools by name |
| Tool invocation via `vscode.lm.invokeTool()` | Map bridge request to VS Code LM API call |
| Result normalization | Map MCP result → tool's declared output fields, fail closed on mismatch |
| Authorization-unavailable detection | Clear error when no valid tool-invocation authorization exists |
| Extension-disconnect signaling | Notify bridge on deactivation/window close |

### Consumer Owns

| Deliverable | Notes |
|---|---|
| `vscode-mcp` bindings for `icm.get-incident` and `tsg-recommendation.recommend` | See conflict note below |
| Verification | Run the unchanged `icm-tsg-router.runbook.yaml` in VS Code with their authenticated ICM MCP tool and confirm end-to-end flow |

**Ownership conflict — requires explicit resolution.** The consumer's Phase 2 scope point 2 asks us to "provide a contract-identical VS Code binding for `icm.get-incident` and `tsg-recommendation.recommend`." But in Phase 1B Rev 3 we ratified an ownership split where the consumer owns their specific action bindings and contract proofs, and Gert core owns only the mechanism and proves it on a synthetic contract. These two positions contradict on this specific item.

**Proposed resolution (consistent with Rev 3):** Core owns the `vscode-mcp` transport mechanism and proves it works using a synthetic contract we own. The consumer authors the two `vscode-mcp` bindings for their own actions (`icm.get-incident` and `tsg-recommendation.recommend`) in their own tool definitions, using the mechanism we provide. This is the same split as Phase 1B: we proved managed-identity + mcp-http on `ops-synthetic`; they prove their exact contract in their repo. This requires their explicit confirmation — added to §8.

**Note:** The consumer explicitly stated the runbook must not change. This is achievable because the transport binding is resolved at runtime from the profile + tool definition, not from the runbook. The runbook says `tool: icm` / `action: get-incident`; the transport mode comes from the tool definition (or profile override), not the runbook step.

---

## 6. Security Issue: `serve` Binds All Interfaces Without Auth

**This exists today, independent of Phase 2.** The extension spawns `gert serve --addr :<port>` with no host prefix and no auth flags. Go's `net.Listen("tcp", ":7778")` binds all interfaces. The extension picks the port by probing `127.0.0.1:0` and talks to `localhost`, which makes it appear loopback-scoped, but the server accepts connections from any network interface.

**Impact:** Any process on the local network can send JSON-RPC commands to the Gert server while the graph preview is open. No authentication is required.

**Recommendation:**
1. Change the flag default in `cmd/gert/serve.go:39` from `":7778"` to `"127.0.0.1:7778"`. The all-interfaces bind originates at the flag default (`addr := fs.String("addr", ":7778", ...)`), not at the `net.Listen` call in `internal/serve/server.go`, which faithfully binds whatever address it receives. Changing the Listen call would be the wrong fix — it would override a caller's explicit choice. Additionally, add validation that rejects or warns when a non-loopback bind address is specified.
2. The extension should pass `--addr 127.0.0.1:<port>` explicitly regardless of the default.
3. Enable auth for all `serve` invocations from the extension, using a per-session random secret generated by the extension and passed via `--auth-token <secret>` (or `--auth-jwt-secret <secret>` for JWT-based auth; the two flags are mutually exclusive, per `cmd/gert/serve.go:42-44`).

**This fix should ship before or with Phase 2, not after.** The Phase 2 bridge has a strict loopback + capability-secret requirement (scope point 7), and shipping that on top of a server that binds all interfaces with no auth would be contradictory.

**Estimate:** 1 day including extension-side changes.

---

## 7. Genuinely New Infrastructure

| Component | Precedent in Phase 1B? | Notes |
|---|---|---|
| Per-session capability secret | **No.** `serve` auth is static token or pre-configured JWT. No per-session random secret concept exists. | Must be cryptographically random (e.g., `crypto/rand` 32 bytes, hex-encoded). Generated by Gert core at bridge startup, communicated to extension via stdout or a startup handshake, never logged. |
| Request-ID registry / duplicate-request idempotency | **No.** JSON-RPC IDs are sequential integers, no dedup, no registry. | New data structure. Bounded (cap + LRU or TTL eviction). Must handle concurrent requests. |
| Bridge versioning + version-mismatch handling | **No.** No versioning concept in `serve` or anywhere else. | Simple: integer version in handshake; hard-fail on mismatch with clear error message including both versions. Do not attempt negotiation in v1. |
| Extension-disconnect semantics | **Partial.** WebSocket in `serve` has connection lifecycle, but no graceful-disconnect protocol. | Bridge must detect TCP close, cancel in-flight requests, and transition to a clean error state. Reuse `context.Context` cancellation pattern from `mcp-http`. |
| Loopback-only enforcement | **Partial.** `serve` binds all interfaces but extension talks to `localhost`. | Enforcement is trivial: `net.Listen("tcp", "127.0.0.1:<port>")`. The harder part is ensuring no configuration path can override this to a non-loopback address. |
| Extension LM API integration | **No.** Zero usage of `vscode.lm` in the extension today. | Entirely new code. Requires understanding VS Code's `LanguageModelTool` interface, `ChatContext`, and the invocation result shape. |

**Note on cancellation coverage:** Context-based cancellation is verified as honoured only for `mcp-http` (via `http.NewRequestWithContext`). For stdio-`mcp` and `native` transports, context is passed but honouring is unverified. This is not central to `vscode-mcp` if the bridge uses loopback HTTP (where cancellation is enforced by the HTTP request context), but it should not be assumed as verified coverage across all transports.

---

## 8. What We Need From the Consumer

**Nothing that violates the boundary we established in Phase 1B Rev 3.** We do not need repo access to `gert-sqllivesite`. We do not need their tool definitions (we have our own synthetic contract and know their two actions are `read-only` stdio-`mcp`).

We need:

1. **Confirmation that the manual acceptance proof can use a mock/fake ICM MCP tool registered in VS Code** rather than requiring their production ICM credentials. If they require production ICM, the proof is gated on their credential provisioning (same blocker as Phase 1B Item 4). If they accept a fake MCP tool that returns the correct typed outputs, we can prove the flow end-to-end without their infrastructure.

2. **The exact `engines.vscode` minimum version they are willing to accept.** Raising from 1.85 to a higher floor (sources indicate 1.90–1.91; exact version must be verified) drops support for ~6 months of VS Code releases. They may have constraints we don't know about.

3. **Confirmation that they own the `vscode-mcp` bindings for their two actions.** Their Phase 2 scope point 2 asks us to "provide" the bindings for `icm.get-incident` and `tsg-recommendation.recommend`. Our ratified Phase 1B Rev 3 ownership split says the consumer owns their specific action bindings and Gert core owns only the mechanism. We propose the Rev 3 split applies here too: core proves the mechanism on a synthetic contract; they author the two `vscode-mcp` bindings in their tool definitions. If they disagree and want us to author the bindings, that is a dependency on their tool definitions that Rev 3 was designed to avoid, and we need to renegotiate the boundary.

---

## 9. Risk Summary

| # | Risk | Severity | Mitigation |
|---|------|----------|------------|
| 1 | Output contract enforcement is new core work not scoped by the consumer | **High** | Scope it explicitly as Phase 2 prerequisite; it benefits all transports, not just `vscode-mcp` |
| 2 | Extension repo is unbuildable from clean checkout | **Blocking** | Remediate before any extension work begins |
| 3 | Extension-host test harness does not exist | **Blocking** | Required for all consumer-mandated extension test coverage |
| 4 | `serve` all-interfaces/no-auth is a live security issue | **High** | Fix independently, ship before or with Phase 2 |
| 5 | `engines.vscode` floor insufficient for LM APIs | **Blocking** | Raise and verify exact minimum |
| 6 | Executor-kind bypass allows governance skip | **Medium** | Structural enforcement (§3 recommendation 2) |
| 7 | Per-session capability secret has no precedent | **Medium** | Clean-room implementation; pattern is well-understood (crypto/rand + HMAC) |
| 8 | Duplicate-request idempotency has no precedent | **Medium** | Bounded registry with TTL; straightforward but must be concurrent-safe |

---

## 10. Summary

Phase 2 is feasible, well-scoped by the consumer, and architecturally sound. The core transport-chain pattern is proven by Phase 1B. The main risks are not in the Phase 2 design itself but in the state of the extension repo (unreproducible, no test harness, wrong VS Code floor) and in a core gap (output contract enforcement) that the consumer's requirements correctly assume exists but doesn't.

The consumer has been right nine times. Their tenth ask — a runtime output contract check — would make it ten for ten. We should build it.

Estimated delivery: 10–14 days parallelized after prerequisites clear. Prerequisites: 3–4 days serial in the extension repo.
