# don

Current role: Implementation engineer.

Session: Dynamic Runbook Includes (2026-08-15)
- Feature complete and approved
- Ready for merge

Session: Streamable HTTP MCP Transport (2026-08-16) — updated after Ken's recon
- Implemented `MCPHTTPTransport` in `internal/tool/mcp_http.go`
- SSE parser correlates by expectedID (not just "first non-nil id") — skips notifications and stale responses
- Session ID lifecycle: capture on init, echo on subsequent; 404→re-init-once; 401→invalidate-and-retry
- Runtime wiring via `TokenGate` (allowed-hosts check + audit trace); proven by `TestMCPHTTPTransport_RuntimeWiring`
- Extracted shared types to `mcp_types.go` (Result as json.RawMessage)
- Fixed Ken's import cycle: `auth_gate.go` → `executor.EmitterFromContext` (nonexistent) → created `emit_ctx.go`
- initialize() explicitly checks resp.Error before setting initialized=true (not replicating stdio defect)
- 11 tests pass; `internal/conformance` + `pkg/pjvm` now pass (import cycle was the root cause)

Key learnings:
- Always build first before implementing — Streams A/C had partially landed
- SSE correlation must happen at parse time, not post-hoc
- Import cycles in teammate's files in my package territory are mine to fix

Session: Streamable HTTP MCP Transport — follow-up round (2026-08-16)
- Deleted `emit_ctx.go` (dead code): Ken had already landed `trace.EmitterFromContext` in `auth_gate.go`; my import-cycle fix created a parallel context key that nothing seeded in production
- B-27 audit trail is now live: engine seeds `trace.WithEventEmitter`; gate reads it correctly
- MCP-013 (B-30) redirect blocking: `NewMCPHTTPTransport` sets `httpClient.CheckRedirect` when `gate != nil`; unauthenticated transports follow redirects freely
- End-to-end wiring now proven at two levels: `TestMCPHTTPTransport_RuntimeWiring` (runtime dispatch) + new `TestMCPHTTPTransport_SchemaRuntimeWiring` (schema → mapTransport → runtime)
- Ken's MCP-012 fatal behavior (disallowed host in AttachToken) was already in the file — my earlier read caught an intermediate state
- Tess's DEF-011 skip can be removed (CheckRedirect is now installed)
- go build ✅; go test ./... ✅; go vet — only two pre-existing internal/serve warnings

Detailed history: .squad/decisions/inbox/don-mcp-http-streamb.md

Detailed history: .squad/agents/don/history-archive.md

## 2026-08-16 — Team Orchestration Session: Runtime Portability Evaluation

**Session:** Scribe coordination session with barbara, don, david  
**Task:** Evaluate and document SQL Live-Site Operations "Runtime Portability for Gert Runbooks" implementation request

**Don's session outputs:**
- Ground-truth verification against source code completed
- All claims validated except run-gert.ps1 (not in Gert core)
- Key findings shared with Barbara and David for cross-agent analysis

**Team consensus findings:**
- Schema non-goal (§13) contradicts ask's own capability-declaration requirements (all agents flagged independently)
- Phase 2 estimate relies on OQ-2 answer; currently meaningless (Don + Barbara aligned)
- Lowest-cost preflight improvement: wire existing `ToolGovernance.AllowedEnvironments` into planner's tool-resolution path (Don's specific recommendation)

**Deliverables merged to decisions.md.**

## Learnings

Session: Runtime Portability Final Impact Assessment (2026-08-17T06:15:17-07:00)

**Q1 — Attendance:**
- `TTYOutput` is HARDCODED at every call site (not TTY-detected). `gert run` hardcodes `true`; `gert serve` hardcodes `false`. No isatty() anywhere.
- `TTYOutput` drives THREE things: (a) approval gate selection, (b) stdin/stdout prompt provider, (c) terminal input provider. These semantics must be decoupled.
- `Attended *bool` on WireOptions is the right shape. Profile populates it when declared; `TTYOutput` is the fallback default. Change is 0.5 days, one function.
- Approval gate selection in `buildApprovalGate()` at `wire.go:319`.

**Q2 — Test-context binding:**
- `plan.Tools` has `.Transport.Type` populated at plan time — checkable from `gert plan` without executing anything.
- "must be native" is too strict: hermetic local fake subprocess is a valid test pattern. Correct rule: native always allowed; `mcp` (subprocess) allowed with explicit profile opt-in; `mcp-http` NEVER allowed in test context.
- Check catches the package-map + production-package mistake naturally (package-map resolves first, profile check reads plan.Tools after).
- Ships with early `gert plan --profile` work. 0.5 days.

**Q3 — Unspecified fires gate in interactive:**
- Real regression: `gert run` hardcodes TTYOutput=true, today no gate fires on plain tool calls. Classification shipping with unspecified=prompt would break every existing interactive runbook.
- `requires-approval: false` explicitly set acts as legacy classification override → treated as read-only, no prompt.
- `legacy_unspecified_policy: allow | prompt` profile field provides explicit opt-out during migration window (default: prompt for new profiles).
- PKG-W warning for every unclassified action encountered at plan time when running non-test.
- ~1.5 days plumbing.



Session: Runtime Portability Impact Assessment — Counter-Positions (2026-08-16T16:52:05-07:00)

**A — Vocabulary collision:**
- `AllowedEnvironments` has EXACTLY ONE value in use across the entire repo: `"real"` (25 occurrences in `tv-enum.yaml`). `RequiresCapabilities` has ZERO values — completely blank.
- `"real"` is a RunMode discriminator (`engine.RunModeReal`), NOT a deployment context. It was used to mean "don't dry-run this tool." It is orthogonal to the proposed profile contexts.
- The SQL team's profile contexts (`cli-operator`, `headless-server`, etc.) are a DIFFERENT vocabulary axis. We need two fields: `AllowedEnvironments` for profile contexts, `AllowedModes` for RunMode. The 25 fixture occurrences in `tv-enum.yaml` need a one-line update. No runtime breakage (field is unenforced today).

**B — Classification field:**
- `classification` does not exist anywhere. Must add `Classification *string` on `ToolAction` (NOT `ToolGovernance` — it's per-action, not per-tool). Use pointer to distinguish nil (unspecified) from `""` (invalid).
- CRITICAL FINDING: `RequiresApproval` is only enforced on the substitution path (`executeSubstitution`) today. Plain tool invocations (`Execute()` → `runtime.Invoke()`) have no approval gate at all. This means a `classification: destructive` plain tool call has zero gating today. The Phase 1 gate MUST add a check to the non-substitution `Execute()` path.

**C — Approval gate:**
- `buildApprovalGate()` in `internal/adapter/wire.go` is a binary: TTY → Terminal gate, no-TTY → NoOp gate. NoOp always approves ("noop-gate" approver).
- `ApprovalRecord` has only: `Approver`, `ApprovedAt`, `Token`. Missing: `PolicyID`, `RunID`, `StepID`, `RetryCount`, `Classification`. Extension is backward-compatible (new fields with omitempty, value type not interface).
- Phase 1 minimal gate (fail-closed deny): 1–2 days. Full evidence gate (Phase 3): 1 week.

**D — gert plan early delivery:**
- No `gert plan` command exists today (`main.go` has no `plan` case). `gert preview` does not plan.
- BUT: `internalplanner.Plan()` already returns `plan.Tools map[string]*schema.ToolDef` — the complete resolved tool set. A `gert plan --profile` command that reports binding compatibility can be built in 2–3 days on top of existing `Plan()` output WITHOUT building the binding resolver first. Label it "compatibility report."

**E — --profile / --package-map composition:**
- `--package-map` operates at YAML-selection layer (which file backs a toolRef). Profile binding resolver operates at transport-config layer (which config to use for an already-loaded tool). Natural layering — no deep conflict.
- Precedence rule: `--package-map` wins YAML selection; profile wins transport config. Resolver should operate on post-catalog `plan.Tools` map.
- The one mismatch case (package-map redirects to mock, profile requires mcp-http) is detected correctly as a plan-time binding incompatibility — which is the right behavior.

Deliverable: `.squad/decisions/inbox/don-runtime-portability-impact-assessment.md`



Session: Runtime Portability Ground Truth (2026-08-16T15:59:48-07:00)
- Verified all 13 claims in §3.2/§5/§8 of the SQL Live-Site Operations ask against live Gert source
- One false claim: `run-gert.ps1` does not exist in Gert core (zero .ps1 files in the tree)
- All "genuinely absent" claims confirmed absent: no runtime binding resolver, no profile concept, no managed/workload identity providers, no VS Code host bridge
- Key surprise: `ToolGovernance.AllowedEnvironments` and `RequiresCapabilities` are declared in `pkg/schema/tool.go` but **zero Go code reads or enforces them** — ghost fields
- Tool-not-found is a PLAN-TIME failure (PLAN-010) via `planner.go:resolveTool()`, not a runtime failure — Gert already fails closed before the first step when a tool name/action is unresolvable
- `AzureCLIAuthProvider.Token(ctx)` already propagates deadlines correctly via `exec.CommandContext` — §10.3 question is answered as YES today, no Phase 1 work needed for this specific point
- `DefaultToolRuntime.persistent` cache (transport-by-name) would need to be context/profile-scoped for the binding resolver to work correctly — non-trivial
- `OverlayRegistry` (`internal/tool/overlay_registry.go`) and `--package-map` are the existing precedent for runtime tool substitution; the binding resolver should build on this pattern
- Transport layer (`MCPHTTPTransport`) has no `request_id` field and no idempotency/late-result logic — §10.4 is real new scope
- Phase 1 estimate (4–6 weeks) is optimistic; Phase 2 VS Code estimate is wildly optimistic (misses extension-side work)
- Deliverable written to: `.squad/decisions/inbox/don-runtime-portability-ground-truth.md`

## 2026-08-16T02:33:36Z — MCP HTTP Transport Feature — Team Session Orchestration

**Feature:** Native Streamable HTTP MCP transport (transport.mode: mcp-http)
**Outcome:** ✅ COMPLETE — All requirements met, final review gate APPROVED (unconditional)

- Completed assigned stream work
- Collaborated on binding architecture contract
- Participated in team orchestration
- All deliverables verified and tested
- Ready for production merge


---

## 2026-08-17 — Runtime Portability Negotiation Concluded

**Status:** Design agreed, gate closed, Phase 1 scope finalized.

**Negotiation:** Four-round multi-agent negotiation with SQL Live-Site Operations on runtime portability for Gert runbooks (rounds 2–4; round 1 committed as d355bf5).

**Key technical outcomes:**
- AllowedEnvironments/AllowedModes field separation (profile contexts vs RunMode)
- Approval gate coverage closure (direct-invocation path enforcement added to Phase 1)
- Grandfathering rule for interactive unspecified (legacy equires-approval: false acts as read-only override)
- Declared attendance orthogonal to context (fixes TTYOutput conflation, resolves CI blocking-on-stdin bug)
- Test-context binding restriction at plan time (native-only default, subprocess opt-in, mcp-http never)
- OQ2 closed as subprocess continuation with Phase 0 framing deliverables

**Agents involved:** Barbara (architect), Don (backend verification), David (round 1).

**Design principle:** Fail-closed by default; explicit migration paths for breaking changes; source-grounded verification.

**Impact:** Phase 1 addition ~4 days. All blocking conditions satisfied. Implementation starts this week.
