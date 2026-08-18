# Runtime Portability — Final Impact Assessment (Three Questions)

**Prepared by:** Don (Backend Dev)
**Date:** 2026-08-17T06:15:17-07:00
**For:** Barbara (close-out ruling)

---

## Q1 — Attendance: What Does Declaring It Actually Cost?

### Current wiring — confirmed

`buildApprovalGate()` body in `internal/adapter/wire.go:319`:

```go
func buildApprovalGate(opts WireOptions) governance.ApprovalGate {
    if opts.TTYOutput {
        return internalgovernance.NewTerminalApprovalGate(os.Stdin, os.Stdout)
    }
    return internalgovernance.NewNoOpApprovalGate()
}
```

`TTYOutput` is a plain `bool` field on `WireOptions` (`internal/adapter/options.go:25`). It is NOT auto-detected from terminal state — it is a **hardcoded constant** at every call site:

| Call site | Value set | Source |
|-----------|-----------|--------|
| `cmd/gert/run.go:148` | `TTYOutput: true` | Hardcoded — `gert run` always gets Terminal gate |
| `cmd/gert/serve.go:110` | `TTYOutput: false` | Hardcoded — `gert serve` always gets NoOp |
| `cmd/serve/main.go:83` | `TTYOutput: false` | Hardcoded — standalone serve binary |
| `internal/e2e/helpers_test.go:171` | `TTYOutput: false` | Hardcoded — test harness always non-interactive |
| `internal/perf/fixtures.go:102` | `TTYOutput: false` | Hardcoded — perf fixtures |

**`gert run` hardcodes `TTYOutput: true` regardless of whether an actual TTY is attached.** There is no `isatty()`/`term.IsTerminal()` detection anywhere. A CI pipeline running `gert run` gets the Terminal gate and blocks on stdin — a bug that profile-declared attendance would fix.

### What else `TTYOutput` drives beyond gate selection

`TTYOutput` branches in TWO additional places in `wire.go`:

1. **`buildInputProvider()` (line 305):** `if opts.TTYOutput { promptInput = internalinput.NewPromptProvider(os.Stdin, os.Stdout) }` — whether stdin/stdout are wired as an input provider for collector/choice steps.

2. **`buildPromptProvider()` (line 312):** `if !opts.TTYOutput { return nil }` — whether a `TerminalInputProvider` is constructed for human-in-the-loop prompt steps.

These are NOT the same semantic as "is a human present to approve." A non-interactive CI run may still want structured prompt inputs from env vars (the `ChainProvider` already handles that via `envProvider` regardless of TTYOutput). **This means `TTYOutput` currently conflates three distinct properties: (a) approval attendance, (b) stdin/stdout terminal interactivity for collector steps, and (c) prompt rendering.** Decoupling them is right.

### Recommended shape

Add `Attended *bool` to `WireOptions`. When the profile declares `attended: true` or `attended: false`, set it explicitly. When absent, derive from TTY detection (or keep the legacy `TTYOutput` default). The gate selection becomes:

```
if opts.Attended != nil {
    attended = *opts.Attended  // profile wins
} else {
    attended = opts.TTYOutput  // legacy inference
}
```

The stdin/stdout wiring for prompt/input providers should remain driven by `TTYOutput` (it's about whether physical IO streams are usable, not about approval semantics). These are now separate switches.

**Call sites to update:** `buildApprovalGate()` — 1 function, 3 lines changed. No other logic changes. The `WireOptions` struct gets one field. The profile loader populates it when profile has `attended:` declared. `cmd/gert/run.go` sets `TTYOutput: true` and leaves `Attended` nil (inherits `TTYOutput` inference). CI profile sets `Attended: false` explicitly, regardless of whether `gert run` was invoked with a TTY.

**Effort: 0.5 days.** Field addition, gate selection update, profile loader sets the field. No other structural changes.

---

## Q2 — Test-Context Binding Restriction: Statically Enforceable?

### Is `transport.mode` available on `plan.Tools` at plan time?

Yes. `internalplanner.Plan()` returns `plan.Tools map[string]*schema.ToolDef`, and each `ToolDef.Transport` is a `TransportConfig` struct with `Type schema.Transport` (the `mode` value, already resolved from the YAML `mode:` field via `UnmarshalYAML`'s `Mode → Type` copy). This is populated by the catalog resolution + `RuntimeToolDef()` conversion path, not by any transport runtime. **The transport mode is fully available at plan time without executing anything.** The test-context check lands squarely in the early `gert plan` work.

### Is "must be native" too strict?

Yes — too strict. Checking the repo's own test/conformance posture:

- The **e2e test harness** (`internal/e2e/helpers_test.go`) injects a `mockToolRuntime` that overrides dispatch entirely. Transport mode in the tool YAML is irrelevant — the mock runtime intercepts before any transport runs.
- The **conformance corpus** (`tv-enum.yaml`) uses `transport.mode: stdio` on all 25 fixture tool definitions (the same `kubectl` tool with `allowed-environments: ["real"]`). Transport mode in a conformance fixture is structural metadata, not execution intent.
- **No fixture or test in the Gert repo uses `transport.mode: mcp` (subprocess) in a test context.** Zero matches across test YAML files.

However, the right rule must account for a legitimate use case the SQL team themselves will encounter: **a hermetic local test server** — a deterministic fake MCP subprocess (`fake-icm-mcp`) that responds predictably without hitting a real endpoint. A strict "native only" rule would ban this valid testing pattern. The SQL team's `tests/packages/incident-routing-mock/` is already `transport.mode: native`, which is correct for deterministic runbook-backed mocks. But a team running a local fake subprocess for integration testing of the MCP transport layer itself needs `mode: mcp` to be allowed.

**Proposed corrected rule for `context: test`:**

| Transport mode | Default | Override |
|----------------|---------|----------|
| `native` | ✅ Always allowed | — |
| `mcp` (subprocess) | ❌ Blocked by default | Allowed with explicit profile opt-in: `transport.allow_subprocess_in_test: true` |
| `mcp-http` | ❌ Never allowed in test context | No override — real HTTP endpoints are production |

The opt-in for subprocess keeps the fail-closed default (test = native, deterministic) while not breaking the legitimate "hermetic local fake server" pattern. The key invariant: `mcp-http` is always banned in test context — that's the actual production-endpoint protection the SQL team cares about.

### Interaction with `--package-map`

The check is exactly the right layer. `--package-map` resolves first (determines which `.tool.yaml` backs each toolRef). The profile test-context check reads `plan.Tools` after catalog resolution. If a test-context profile is combined with a `--package-map` pointing at the real production package (which declares `transport.mode: mcp-http`), the check reads `mcp-http` from the resolved `plan.Tools` entry and fires a Tier 0 error: "test context profile does not permit mcp-http transport for tool 'icm'." Exactly the mistake it should catch. The layering makes this work for free.

**Effort: 0.5 days.** The check is a loop over `plan.Tools` in the new `gert plan` command. One function, ~15 lines. Ships with the early `gert plan --profile` work.

---

## Q3 — Interactive `unspecified` Fires the Gate: Migration Hazard

### Is the regression real?

Yes, and it is significant. Here is the actual scope:

- The Gert core repo has 4 real `.tool.yaml` files in `tools/`. `ping`, `nslookup`, and `curl` use the legacy map-form actions (no `- name:` prefix) and have no `classification` or `requires-approval` — they would become `unspecified`.
- The `icm.tool.yaml` in `tools/` has 3 actions via the canonical list form, no `classification`.
- The conformance corpus (`tv-enum.yaml`) has 75 tool actions embedded as fixture strings; 25 of those have `requires-approval: false` explicitly set, the remaining 50 have neither field.
- **The gert-sqllivesite production tools (outside this repo) are entirely unclassified.** Every one of their `icm`, `tsg-recommendation`, and any other tool actions would become `unspecified` on day 1 of classification shipping.

However, there is an important nuance from the current approval wiring: the gate for plain (non-substitution) tool calls in interactive mode currently fires based on **runbook-level governance** (`engine.go:506` — `evalResult.RequiresApproval` from `GovernancePolicy`), not on tool-level `requires-approval`. Tool-level `RequiresApproval` is only checked on the substitution path. So today, a plain `icm.get-incident` call in interactive mode does NOT prompt — there is no gate. Under the new rule, if `classification` is absent and we fire the gate, every plain tool call suddenly prompts. For a runbook with 10 tool steps, that is 10 approval prompts where there were 0 before. **This is a breaking UX regression for every existing interactive runbook.**

### Recommendation: Grandfathering via explicit `requires-approval: false`

The SQL team's principle is right: classification is an opt-in to REDUCED friction. Absence must not grant additional execution rights. But they also said "existing `RequiresApproval` behavior remains authoritative for legacy definitions." These two principles together define the migration path.

**Proposed rule:**

1. `requires-approval: false` explicitly set on a tool action (or inherited from the tool's governance block) acts as a **legacy classification override** equivalent to `classification: read-only` for the purpose of the classification gate. The action is treated as read-only: no prompt, no approval evidence required. This is the grandfathering clause.

2. `requires-approval: true` explicitly set continues to fire the gate as today (unchanged).

3. `classification` absent AND `requires-approval` absent (or default `false` not explicitly set — the ambiguity from `omitempty` on bool) → `unspecified`. In interactive contexts: fires the gate. In test contexts: auto-approve only for native transport (per Q2). In unattended contexts: deny.

4. A profile-level field `legacy_unspecified_policy: allow | prompt` that defaults to `prompt` in new profiles and can be set to `allow` for a migration window. This lets existing deployments opt in to the old behavior for a deprecation period, explicitly, not by default.

**Why grandfathering on `requires-approval: false` is the right anchor:** It is already an explicit author signal. The author of `kubectl.tool.yaml` who wrote `requires-approval: false` was explicitly saying "this tool does not need human approval." Under the proposed scheme, that explicit signal is preserved without requiring tool authors to touch their YAML. The gate fires only for tools where the author made no explicit decision either way.

**The practical migration checklist:**
1. Ship `Classification *string` on `ToolAction`.
2. Ship `legacy_unspecified_policy` on the profile schema with default `prompt` and a clear deprecation notice.
3. Emit a `PKG-W` warning (non-fatal, stderr) for every unclassified action encountered at plan time when the profile is non-test: "action 'icm.get-incident' has no classification; treating as unspecified. Add `classification: read-only` to suppress this warning."
4. Existing deployments that need continuity set `legacy_unspecified_policy: allow` in their profile during the migration window. They see the warnings but don't get prompted.
5. Set a deprecation deadline: profiles that don't set `legacy_unspecified_policy` get `prompt` behavior once `classification` ships, with the explicit opt-out available.

This gives the SQL team their principle (unspecified fires the gate by default in new profiles) while giving existing runbook operators a named, explicit, auditable escape hatch — not a silent reversal of the principle.

**Effort for Q3 plumbing:** `Classification *string` addition: 0.5 days. `legacy_unspecified_policy` field on profile schema + routing in gate: 0.5 days. PKG-W warning at plan time: 0.5 days. **Total: ~1.5 days**, plus documentation of the migration path.


---

