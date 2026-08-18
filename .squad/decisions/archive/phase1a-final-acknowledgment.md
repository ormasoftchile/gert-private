# Final Acknowledgment: Runtime Portability — Design Agreed (Rev 2)

**Date:** 2026-08-17  
**By:** Gert Core Team  
**To:** SQL Live-Site Operations (gert-sqllivesite)  
**Status:** Design agreed. Implementation begins.  

---

## Accepted

All items accepted. Corrections incorporated below, plus two disclosures from our implementation costing.

---

## 1. Interactive `unspecified` — Corrected

Accepted. Human presence does not guarantee human attention. We adopt your principle as the governing design rule for the entire classification feature:

> **Classification is an opt-in to reduced friction. Absence of classification must never grant additional execution rights.**

Final behavior table:

| Context | `unspecified` behavior |
|---------|----------------------|
| Interactive operator | **Gate fires.** Active confirmation required. |
| CI / headless | Denied unless explicitly permitted by policy. |
| Test (native-only) | Auto-approve. |
| Retry | Never. |
| Late/lost result | INDETERMINATE + halt. |

### Disclosure: migration impact of "interactive unspecified → gate fires"

While costing this change we found a breaking regression we must address together. Today:

- The approval gate on plain (non-substituted) tool calls fires based on runbook-level governance, NOT per-tool `requires-approval`. Tool-level `RequiresApproval` only applies on the substitution path.
- So today, a plain `icm.get-incident` call in interactive mode triggers **zero prompts**.
- Under the new rule, every unclassified tool action in interactive mode would prompt. **A 10-step runbook goes from 0 prompts to 10 prompts.**
- Scope: every tool in gert-sqllivesite is currently unclassified. You would be hit hardest on day 1.

**Mitigation — grandfathered explicit opt-out:**

| Governance state | Behavior |
|------------------|----------|
| `requires-approval: false` **explicitly written** | Treat as legacy equivalent to `classification: read-only`. No prompt. The author explicitly opted out. |
| `requires-approval: true` explicitly written | Gate fires, unchanged. |
| Governance block **absent entirely** (nil pointer) | `unspecified`. Interactive: gate fires. Unattended: deny. |

**Why this is implementable:** `ToolGovernance` is a pointer (`*ToolGovernance`) on the tool definition. A nil pointer = no governance block = author never expressed intent = unspecified. A non-nil pointer with `RequiresApproval: false` = author explicitly wrote a governance block opting out = honor that. This distinction exists in the schema today.

**Migration path:**

1. **New profile field: `legacy_unspecified_policy: allow | prompt`.** Default: `prompt` (enforces the new rule). Existing deployments set `allow` during migration to preserve pre-classification behavior. Explicit, named, auditable escape hatch.
2. **Plan-time warning (PKG-W level):** For each unclassified action when the profile is non-test: `"action 'icm.get-incident' has no classification; treating as unspecified. Add classification: read-only to suppress."` Migration pressure visible without breaking runs.
3. You add `classification: read-only` to your tool actions at your own pace. When complete, remove `legacy_unspecified_policy: allow` from your profile.

This reinforces rather than weakens your principle: absence of a governance block still means unspecified and still means the gate fires. We are only honoring an explicit prior opt-out that an author already wrote down.

---

## 2. Attendance as a Separate Profile Property — Accepted

You are separating execution host (context) from human presence (attendance). Correct — an autonomous agent inside VS Code is unattended despite the host being `vscode-operator`.

**Profile schema:**

```yaml
apiVersion: runtime-profile/v1
id: vscode-autonomous
context: vscode-operator        # execution HOST
attendance: unattended           # approval semantics
approval:
  scope:
    allow_read: true
    allow_mutating: false
    allow_destructive: false
```

`attendance` is a top-level enum: `attended | unattended`. Orthogonal to `context` and `approval.scope`:

- `context` → which tool bindings are valid (AllowedEnvironments).
- `attendance` → which approval gate behavior applies.
- `approval.scope` → which classifications the profile permits at all.

**Disclosure: TTYOutput conflates three properties.** Current Gert hardcodes `TTYOutput: true` unconditionally in `cmd/gert/run.go` — there is no `isatty()` detection. Consequence: `gert run` in a CI pipeline today gets the TerminalApprovalGate and will **block on stdin** if any approval fires. This is a live latent bug, and it is exactly the failure mode your declared-attendance requirement fixes.

Additionally, `TTYOutput` currently drives three distinct things: (a) approval gate selection, (b) physical stdin/stdout availability for collector/choice steps, and (c) terminal-rendered prompts. Profile-declared `attendance` replaces ONLY (a). Physical IO availability (b, c) remains driven by TTY detection because those concern whether stdin/stdout exist, not whether a human is attentive. Implementation: `Attended *bool` on WireOptions; nil inherits legacy TTY inference; profile populates it explicitly when present.

**Edge cases:**

- Profile declares `attended` but no TTY → **Tier 0 failure** (`config/attendance-mismatch`). A profile promising an approver who cannot be reached is a configuration error.
- `attended: false` with `context: cli-operator` → **Legal.** Your autonomous-agent case. Gets unattended rules.

---

## 3. Test Context Must Not Bind to Production — Accepted (revised)

**Tier 0 rule (corrected):**

| Transport mode | In test context |
|---------------|----------------|
| `native` | Always allowed. |
| `mcp` (subprocess) | **Blocked by default.** Allowed with explicit profile opt-in: `transport.allow_subprocess_in_test: true`. |
| `mcp-http` | **Never allowed.** No override. |

Rationale for the subprocess escape hatch: a hermetic local fake MCP server (`fake-icm-mcp` binary producing deterministic responses, no real endpoint) is a legitimate pattern for integration-testing the MCP transport layer itself. Banning subprocess entirely forecloses that. The opt-in is explicit, named, auditable — and HTTP endpoints remain absolutely forbidden.

Error code: `config/test-context-non-native-binding`  
Message (mcp-http case):
```
error: tool "icm" uses transport mode "mcp-http" in a test-context profile.
  Test profiles must not bind to HTTP endpoints.
  fix: use a package-map pointing to your mock package, or change the profile context.
```

Message (subprocess, no opt-in):
```
error: tool "icm" uses transport mode "mcp" (subprocess) in a test-context profile.
  Subprocess transport requires explicit opt-in in test context.
  fix: set transport.allow_subprocess_in_test: true in the profile, or use native transport.
```

This check reads `plan.Tools` (post-resolution, post-package-map), so it ships with the early `gert plan` work at zero additional effort. The `--package-map` interaction validates exactly as designed: test profile + production package-map → reads `mcp-http` → Tier 0 error. Right layer, correct behavior.

The approval rule remains sound: native transport has no real side effects; subprocess fakes under explicit opt-in are by definition hermetic (the author vouched by opting in); HTTP is banned. Auto-approve in test context is safe.

---

## 4. OQ2 — Closed: Subprocess

Confirmed. Extension already uses a Gert subprocess. Phase 0 validates framing and lifecycle:

| Phase 0 deliverable | Owner |
|---------------------|-------|
| Framing protocol document | Gert Core |
| Lifecycle spec (start, stop, crash-recovery, version detection) | Gert Core + extension team |
| Capability-advertisement handshake | Gert Core |
| CancelRequest message type | Gert Core |
| Feasibility validation against current extension | SQL Live-Site Ops |

Library integration only if subprocess proof fails on a blocking criterion.

---

## 5. Final Status

**Design agreed. Gate closed. Implementation starts.**

New Phase 1 items from this round:

| Item | Effort |
|------|--------|
| Per-action `Classification *string` on ToolAction | 0.5 days |
| `legacy_unspecified_policy` profile field + gate routing | 0.5 days |
| Plan-time PKG-W warning for unclassified actions | 0.5 days |
| Declared `attendance` on WireOptions + profile | 0.5 days |
| `config/attendance-mismatch` Tier 0 check | included |
| `config/test-context-non-native-binding` Tier 0 check | 0.5 days |
| Direct-invocation approval gate enforcement (Execute() path) | 1–2 days |

These join the existing Phase 1 scope. Total addition: ~4 days.

You have corrected us four times in this exchange and been right every time — your unspecified correction surfaced the migration hazard we would otherwise have shipped blind. The design is better for it. We start work this week.

---

*Gert Core Team — 2026-08-17*


---

