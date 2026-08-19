# Final Acknowledgment: Runtime Portability — Design Agreed (Rev 3)

**Date:** 2026-08-17  
**By:** Gert Core Team  
**To:** SQL Live-Site Operations (gert-sqllivesite)  
**Status:** Design agreed. Implementation begins. Final round.  

---

## 1. Tri-State `RequiresApproval` — Accepted

You are correct. Our Rev 2 grandfathering rule claimed the `*ToolGovernance` pointer distinguishes "explicit `requires-approval: false`" from "governance block absent." It does not. A non-nil pointer proves only that some governance property was authored — not that `requires-approval` specifically was written. Your counter-example:

```yaml
governance:
  requires-capabilities: [network]
```

yields a non-nil `*ToolGovernance` with `RequiresApproval == false` — indistinguishable from an explicit opt-out. The grandfathering rule was unimplementable as stated.

**Correction:** `RequiresApproval` becomes `*bool`:

```go
RequiresApproval *bool `yaml:"requires-approval,omitempty" json:"requires-approval,omitempty"`
```

Tri-state: `nil` = never expressed (unspecified) | `&false` = explicit legacy opt-out (grandfathered as read-only equivalent) | `&true` = approval required.

**Blast radius is small.** Schema types and resolved types are already separate in Gert:

- `schema.ToolGovernance.RequiresApproval` (authored) → changes to `*bool`. One declaration line.
- `pkgsubst.EffectiveGovernance.RequireApproval` (resolved) → stays plain `bool`. No change.
- `trace.EffectiveGovernancePayload.RequireApproval` (wire) → stays plain `bool`. No change.

There is exactly ONE Go read site: `pkg/pkgsubst/pkgsubst.go:359` (`EffectiveGovernanceFromTool()`), which becomes a nil-guarded dereference. Zero test files construct `schema.ToolGovernance{RequiresApproval: ...}` literals — all test tool definitions are YAML-parsed. Nothing marshals `ToolGovernance` back to YAML/JSON (kit.go serializes lockfiles only, never tool governance; trace events use the wire type).

The 25 existing `requires-approval: false` entries in `tv-enum.yaml` unmarshal to `&false` under the new type — which is precisely the "explicit legacy opt-out" value the grandfathering rule needs. The corpus migrates itself. No fixture edits required.

**We withdraw the `*ToolGovernance` pointer claim from Rev 2 §1.** It was wrong. The pointer tells you whether a governance block exists; the `*bool` tells you whether a specific field was expressed. These are different questions and we confused them.

**Effort:** 0.5 days including tests.

---

## 2. Runbook-Level Tri-State — Declined

You asked to apply tri-state "wherever runbook-level explicit false must be distinguished from absent." We decline, and invite a counter-case.

Reasoning: `GovernanceConfig.RequireApproval` at the runbook level is a different semantic — it gates the entire runbook execution uniformly ("this run requires an approval step"), not per-action classification. Knowing a runbook was authored with explicit `require_approval: false` tells us nothing about whether its individual tool actions are safe. A `require_approval: false` runbook can contain destructive actions, and those still need classification-level gating.

The implementation cost is also disproportionate: `GovernanceConfig.RequireApproval` is read at `BuildPolicy()`, subgovernance composition, `ComposeGovernance()`, `WithEntryGovernance()`, `NewEvaluator()`, plus 15+ test sites constructing `schema.GovernanceConfig{RequireApproval: ...}` literals. Estimate: +1.5–2 days for zero design benefit in the classification model.

If you have a concrete scenario where runbook-level explicit-false must be distinguished from absent for classification or approval purposes, name it and we will reconsider. We do not see one today.

---

## 3. Test-Context Subprocess — Corrected Spec Wording

You noted that our "hermetic local fake" claim carries no technical enforcement. You are right.

Gert does zero subprocess sandboxing. `internal/tool/process.go:StartProcess()` calls `exec.Command(command, args...)` with the full parent environment inherited unconditionally (`append(os.Environ()...)`). No namespace isolation, no network restriction, no cwd jail, no env scrubbing. A subprocess in test context CAN reach production endpoints if it chooses to.

**Corrected spec wording for the test-context subprocess opt-in:**

> `transport.allow_subprocess_in_test: true` is an **author assertion** that the named subprocess is a hermetic test double producing deterministic responses without contacting real services. Gert does not enforce isolation — the assertion is a governance declaration, not a technical sandbox. Misuse is an authoring error surfaced by review, not by runtime.

The `mcp-http` prohibition remains absolute and IS technically enforced — the Tier 0 check blocks the binding before any network call occurs. The strong guarantee is where it matters most (HTTP endpoints are the real production-access vector); subprocess is governed by convention and review.

---

## 4. Updated Phase 1 Delta

| Item | Effort |
|------|--------|
| Per-action `Classification *string` on ToolAction | 0.5 days |
| `RequiresApproval *bool` tri-state on ToolGovernance | 0.5 days |
| `legacy_unspecified_policy` profile field + gate routing | 0.5 days |
| Plan-time PKG-W warning for unclassified actions | 0.5 days |
| Declared `attendance` on WireOptions + profile | 0.5 days |
| `config/test-context-non-native-binding` Tier 0 check | 0.5 days |
| Direct-invocation approval gate enforcement (Execute() path) | 1–2 days |
| **Total addition** | **~4.5 days** |

---

## 5. Close

Design agreed. All blocking items resolved. Five corrections from you, five accepted. The classification model, the approval architecture, and the migration path are stronger for it. We start this week.

---

*Gert Core Team — 2026-08-17*
