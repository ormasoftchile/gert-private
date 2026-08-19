# Phase 2 — VS Code Authenticated MCP Runtime Binding
## Implementation Status Report (Gert Core + Gert VS Code)

**Date:** 2026-08-18 (Rev 2 — incorporates consumer blocker remediation)
**Status:** Implemented and pushed. Two consumer blockers resolved; two contract
inputs still required from SQL Live-Site Operations (§10.5).
**Repos:** `ormasoftchile/gert` @ `d1f4352` · `ormasoftchile/gert-vscode` @ `c25d147`

---

## 1. Headline

Phase 2 is code-complete and **reproducibly shipped**. Both repositories build and
test green from a clean checkout containing only tracked files, and both are pushed.

This report also discloses two defects we found in our own work during this phase
(§5, §6), consistent with the disclosure discipline established in Phase 1B §6.

---

## 2. What shipped

### 2.1 Gert core — `vscode-mcp` transport

A new, explicitly named transport. **The meaning of `mcp` is unchanged**, per scope item 1.

Wired through all five mandatory chain points, so it is genuinely reachable rather
than a parsed-but-dead sink:

| # | Layer | File |
|---|-------|------|
| 1 | Schema constant | `pkg/schema/tool.go` |
| 2 | Runtime constant | `pkg/tool/tool.go` |
| 3 | Transport validation | `internal/tool/validate_transport.go` |
| 4 | Scan/mapping | `internal/tool/scan.go` (`mapTransport`) |
| 5 | Invocation | `internal/tool/runtime.go` (`Invoke`) |

Registered in the reachability gate (`cmd/gert/reachability_registry_test.go`) with a
probe verified non-vacuous: deleting the `mapTransport` case makes the probe fail.

### 2.2 The loopback bridge — `vscode-mcp-bridge/v1`

Versioned, correlated, loopback-only. Request carries only logical tool/action, typed
args, request ID, deadline, and capability proof. Response carries only a typed result
or a coded, redacted error.

Gert-side client (`internal/tool/vscode_mcp_bridge.go`) covers success, typed-output
mismatch, timeout, cancellation, duplicate request, malformed response, capability
rejection, version mismatch, disconnect, and a redaction sweep.

Extension-side server (`src/mcpBridge.ts`) binds `127.0.0.1` only, resolves the tool
through `vscode.lm.tools`, and invokes via `vscode.lm.invokeTool()`.

**Authorization never leaves VS Code.** No bearer token is sent to, stored by, or
reachable from the Gert process, its run state, traces, results, or error text.

### 2.3 Output-contract enforcement (prerequisite we discovered we lacked)

Recon found that declared `returns:` were **not enforced at runtime** for
non-substituted tools — `internal/executor/tool.go` copied `res.Output` verbatim and
never consulted the declaration. Only substitution actions validated anything.

Phase 2's "fail closed on missing/unknown/type-incompatible output" requirement is
unimplementable on that foundation, so we fixed it first (`ca867a1`), plus the
matching planner change (`4e67655`) that lets declared outputs actually be *captured*
from `native` / `mcp` / `mcp-http` / `vscode-mcp` bindings. Without the second commit,
declared outputs would have been enforceable but uncapturable — which is worse than
either alone, and is precisely the shape of your router runbook's five typed
`get-incident` captures.

### 2.4 Extension prerequisites

- `engines.vscode` floor raised to **1.95.0**. Confirmed by inspecting `@types/vscode`:
  `lm.tools` / `lm.invokeTool` are absent in 1.90–1.94 and present in 1.95.0.
- Extension-host test harness (`@vscode/test-cli`).
- Extension repo reproducibility remediated (it had the same untracked-source disease).

---

## 3. Evidence

All figures below are from a **detached worktree**, which materialises only tracked
files. Working-tree results are not evidence and are not reported here.

### `ormasoftchile/gert` @ `30b6fe0`

| Check | Result |
|---|---|
| `git status --porcelain` | empty |
| `go build ./...` | exit 0 |
| `go test ./...` | exit 0 |
| Packages | 63 ok · 23 no-test · **0 FAIL** |
| Full-suite repeat runs | 2/2 green |

### `ormasoftchile/gert-vscode` @ `caa6778`

| Check | Result |
|---|---|
| `git status --porcelain` | empty |
| `npm ci` | exit 0 |
| `npm run compile` | exit 0 |
| `npm test` | exit 0 — **58/58 pass, 0 fail** |

### Mutation controls — all MEASURED, not reasoned

| Control | Mutation applied | Observed |
|---|---|---|
| Capability check | `checkCapability` → always `true` | rejection test **fails**, exit 1 |
| Normalizer | return raw MCP result unchecked | **3 tests fail**, exit 1 |
| Idempotency | bypass in-flight/completed caches | **fails**: `actual: 2, expected: 1` |
| Fail-closed token | remove absent-token check | **fails**: `expected error…got nil` |
| GCP-PARSE-001 | `stepBindsDeclaredOutput` → `true` | rejection test **fails**, exit 1 |
| `vscode-mcp` reachability | remove `mapTransport` case | probe **fails** |

---

## 4. Design decisions worth your review

**Capability secret is minted by the extension, not by Gert.** Our first draft had
Gert generate it with `crypto/rand` and send it as proof. That authenticates nothing —
the server would have to accept any value, so any local process could call the bridge.
Corrected: the extension mints a 32-byte secret per session, provisions it to the Gert
child process via `GERT_VSCODE_BRIDGE_TOKEN`, and compares with a timing-safe
comparison. **Gert fails closed when the variable is absent** and performs zero HTTP
requests in that case.

**The normalizer rejects unknown extra fields** rather than dropping them. An unknown
field could carry credential-shaped material; refusing to forward it is the safer
default. This is enforced and tested, not conventional.

**Approval and classification semantics are untouched.** The approval chokepoint is in
`internal/engine/engine.go:executeStep`, *above* the executor layer, so a new transport
arm inside `runtime.Invoke` structurally cannot bypass it.

---

## 5. Disclosure — sixth occurrence of the reachability/reproducibility bug class

During this phase a commit landed that used `ExcludeTestTools` **without committing its
definition**. HEAD did not build from a clean checkout. It was caught by detached-worktree
verification, not by any agent's own testing, and fixed forward in `06b0a83`.

This is the sixth instance of the same class we reported to you in Phase 1B §6. The
`TestCLI_ReachabilityGate` merge gate catches parsed-but-unreachable *fields*; it does
not catch *uncommitted definitions*. Detached-worktree verification remains the only
control that catches this, and we now treat it as the sole acceptable evidence.

## 6. Disclosure — a live test defect that broke the clean-checkout gate

`TestSSE_FilterByRunID` passed in isolation and passed for its whole package, but
**reproducibly failed under `go test ./...`** (2/2). The test broadcast events
immediately after the SSE connection returned headers, while the handler registers its
subscriber only *after* writing headers — so both events were dropped and the reader
timed out. Parallel package execution widened the window.

This mattered because your Phase 1B Rev 3 acceptance criterion is literally
"`go test ./...` passing from that clean checkout". A load-sensitive test failure
fails that gate regardless of product correctness.

Fixed in `30b6fe0` by applying the `WaitForSubscriber` guard already used by the
sibling test in the same file, by `runstate_sse_test.go`, and by `ws_test.go`.
Verified 2/2 green afterwards under the same full-suite conditions that reproduced it.

---

## 7. Known limitations — stated plainly, not papered over

1. **Registered MCP tool names are unverified against a live session.** The
   logical→registered mapping lives in one exported place (`TOOL_ACTION_REGISTRY` in
   `src/mcpBridge.ts`). If your ICM MCP tool registers under a different name than we
   guessed, it is a one-line registry change with no other code impact. **We need you
   to confirm the actual registered names.**

2. **The mapping is static, not derived from your tool YAML.** For v1 the declared
   output field set is declared in the registry rather than read from the Gert tool
   definition. This is adequate for two tools and explicitly bounded; deriving it is a
   follow-up.

3. **`toolInvocationToken: undefined`.** Gert is not a chat participant. Invocation
   executes, but chat-UI attribution may not display. We believe this is acceptable for
   the runbook use case — flagging it in case you disagree.

4. **Result content-part shape.** We handle `.value` and `.text`. A different shape
   fails visibly with `result_parse_error` rather than silently degrading.

5. **`go vet` reports two pre-existing issues** in `delegations_test.go:85` and
   `preview_e2e_test.go:253` (`using r3/sseResp before checking for errors`). These
   predate Phase 2 and are untouched. `go build` and `go test` are unaffected.

6. **No end-to-end run against a real authenticated ICM MCP tool has occurred.** All
   coverage uses a faked `vscode.lm`. Per the Phase 1B §7 precedent, live ICM validation
   is a separate integration gate under your control.

---

## 8. What we need from SQL Live-Site Operations

1. **The registered MCP tool names** for `icm.get-incident` and
   `tsg-recommendation.recommend` as they appear in `vscode.lm.tools` in a real session.
2. **Confirmation of the ownership split.** Phase 1B Rev 3 assigned you ownership of
   these two logical actions and their bindings, but Phase 2 scope item 2 asks *us* to
   provide contract-identical VS Code bindings for them. These conflict. We have built
   the mechanism and left the two consumer bindings to you, consistent with Rev 3 —
   please confirm that is what you intend.
3. **A manual acceptance slot.** The required proof in your §"Required proof" is a
   human-in-the-loop VS Code run against your authenticated tool. We cannot execute it
   from here.

---

## 9. Commit index

### `gert`
| SHA | Content |
|---|---|
| `d19f933` | scope profileless non-interactive fail-fast to `RunModeReal` |
| `a6b1106` | `serve`: loopback default; refuse non-loopback without auth |
| `a35c0ae` | `vscode-mcp` transport runtime |
| `3cd262e` | `vscode-mcp` reachability registration + probe |
| `9c39638` | remove unused imports in reachability probes |
| `ca867a1` | enforce declared `returns:` at runtime for non-substituted actions |
| `06b0a83` | commit `ExcludeTestTools` definition (fixes broken HEAD — see §5) |
| `4e67655` | widen GCP-PARSE-001 to allow declared non-substituted output capture |
| `8b1e0f5` | invert capability-secret direction; read token from env; fail closed |
| `30b6fe0` | fix subscriber-registration race in `TestSSE_FilterByRunID` (see §6) |

### `gert-vscode`
| SHA | Content |
|---|---|
| `e08446b` … `4186f21` | reproducibility remediation (5 commits) |
| `5409241` | `engines.vscode` 1.95.0 floor + test dependencies |
| `d1e4cc1` | extension-host test harness |
| `b79cb58` / `ffba7c6` / `8668fc1` | profile-injection attempt and its forward-revert |
| `215cd87` | `vscode-mcp-bridge/v1` loopback HTTP server (`McpBridge`) + 29 tests |
| `caa6778` | wire `McpBridge` into activation and `ServerManager` env injection |

> **Note on §3 and §7:** those sections record evidence and limitations as of Rev 1
> (`30b6fe0` / `caa6778`). §7 items 1 and 2 are superseded by the Rev 2 remediation in
> §10.2 — the static mapping and guessed names are resolved. Current evidence is §10.6.

## 10. Rev 2 — remediation of the two reported blockers

You verified the delivered commits and reported two blockers. Both are accepted as
correct, both are fixed, and fixing them surfaced a third defect you had not yet hit.

### 10.1 Blocker 1 — wrong TSG output contract (ACCEPTED, FIXED)

You are right, and the impact was total rather than partial: our normalizer is
fail-closed, so with the wrong field names **every** `recommend` invocation would have
failed, not merely edge cases.

The contract is now exactly as you specified:

| Field | Required |
|---|---|
| `recommendation_status` | yes |
| `suggested_tsg_id` | no |
| `suggested_tsg_title` | no |

### 10.2 Blocker 2 — guessed registered tool names (ACCEPTED, ROOT-CAUSED)

We did not fix this by substituting better guesses, because the guessing was the
defect. `registeredName` lived in a TypeScript constant, which meant every name could
only ever be a guess and every correction required a code change and an extension
release.

The mapping is now **declarative**:

- Core: a new optional `vscode_tool` field on `transport` (default) and on each action
  (override), resolved by `schema.ResolveVSCodeToolName` — action, then transport, then
  logical name. Validated to be meaningful only under `vscode-mcp`; an empty value is
  rejected; and it is registered in the reachability gate. Mutation measured: removing
  the production read flips the probe from `exitValidation (2)` to `exitRuntime (1)`.
- Extension: `TOOL_ACTION_REGISTRY` is deleted. `src/toolDefinitionRegistry.ts` builds
  the registry at runtime by parsing workspace `.tool.yaml` files with a real YAML
  parser and filtering to `transport.mode: vscode-mcp`.
- A `gert.mcpBridge.toolNameOverrides` setting overrides the derived name, so a
  mismatch discovered in a live session is corrected **without a code change or a
  release** — precisely the situation you hit.
- Tool-not-found diagnostics now name the logical action, the registered name tried,
  and the names actually present in `vscode.lm.tools`.

### 10.3 Third defect — `no-suggestion` was impossible to represent (FOUND BY US)

You did not report this, and it would have blocked your required proof even after the
field names were corrected.

Our normalizer treated every declared output as mandatory. In the `no-suggestion`
outcome there is no `suggested_tsg_id` and no `suggested_tsg_title`, so normalization
would have failed closed and **half of your required two-outcome proof was
unreachable**.

Fixed by honouring `required: false`, with the strict behaviour deliberately preserved
elsewhere: absent-and-optional is accepted; present-but-null is still an error;
present-but-wrong-type is still an error; unknown extra fields still fail closed.
Regression test asserts both outcomes normalize — `suggested` with all three fields,
`no-suggestion` with only `recommendation_status`.

### 10.4 Blocker 3 — consumer bindings not delivered (ACCEPTED, DELIVERED)

Delivered at `gert-private/deliverables/phase2/`:
- `icm.vscode-mcp.tool.yaml`
- `tsg-recommendation.vscode-mcp.tool.yaml`

Both parse through the real loader. Go tests assert the exact declared output names
and their required/optional flags, so a contract edit breaks a test.

We have not re-litigated the Phase 1B Rev 3 ownership question; you ruled, and we
complied.

### 10.5 What we still need from you — two contract inputs

1. **The registered tool names.** `icm-get-incident` and `tsg-recommendation-recommend`
   remain placeholders. They are now one-line YAML edits (or a settings override)
   rather than code changes, but they are still unverified guesses.

2. **The argument contract for `tsg-recommendation.recommend`.** The delivered binding
   declares no `args`, because you have never specified them. Your router must feed
   incident context into `recommend`, so as written the runbook cannot pass anything.
   We deliberately did not invent argument names — having just removed exactly that
   class of guessing from the codebase, reintroducing it here would be indefensible.
   `icm.get-incident` declares `incident_id: string (required)`; please confirm or
   correct that too.

### 10.6 Rev 2 evidence

All from detached worktrees.

| Repo | HEAD | Result |
|---|---|---|
| `gert` | `d1f4352` | 0 untracked · build 0 · test 0 · **63 ok, 0 FAIL** |
| `gert-vscode` | `c25d147` | 0 untracked · `npm ci` 0 · compile 0 · **83/83 pass** |

Measured mutation controls this round:

| Control | Observed |
|---|---|
| `vscode_tool` reachability read removed | probe **fails** — `exitValidation(2)` → `exitRuntime(1)` |
| Optional outputs made strict again | **2 fail** — incl. the `no-suggestion` regression test |
| Settings override ignored | **2 fail** |
| Registered-name precedence forced to fallback | precedence tests **fail** |

### 10.7 Rev 2 limitations, stated plainly

- The byte-identity check between the deliverable YAML and its in-repo test copy
  **skips when `gert-private` is absent**, so it effectively does not run in CI. The
  contract assertions themselves run unconditionally against the in-repo copy, so the
  contract *is* gated; only cross-copy drift is not.
- The registry is built from the **first workspace folder**. Multi-root workspaces are
  not yet handled.
- If a workspace tool definition does not declare `transport.mode: vscode-mcp`, it is
  simply not registered. This surfaces at invocation time via the improved
  tool-not-found diagnostics rather than at startup.
- Still no end-to-end run against a real authenticated ICM MCP tool. Per the Phase 1B
  §7 precedent, that remains your integration gate.

---

## 11. Commit index — Rev 2 additions

| Repo | SHA | Content |
|---|---|---|
| `gert` | `d1f4352` | `vscode_tool` binding on transport + action; validation; reachability probe; `Outputs` doc-comment correction; deliverable contract tests |
| `gert-vscode` | `27c6ec7` | derive registry from `.tool.yaml`; honour optional outputs; settings override; diagnostics |
| `gert-vscode` | `c25d147` | document `gert.mcpBridge.toolNameOverrides` |

