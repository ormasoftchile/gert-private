# Session Log: Runtime Portability for Gert Runbooks

**Date:** 2026-08-17T13:45:00Z  
**Duration:** Rounds 1–4 (spanning 2026-08-15 to 2026-08-17)  
**Participants:** Gert Core Team (Barbara, Don, David); SQL Live-Site Operations (gert-sqllivesite)  
**Status:** Negotiation concluded, design agreed  

## Overview

Four-round negotiation on runtime portability architecture for Gert runbooks to support SQL Live-Site Operations' deployment requirement: same runbook, different tool bindings (profiles) based on execution context (cli-operator, ci, headless-server, vscode-operator, test).

## Key Technical Outcomes

### 1. AllowedEnvironments / AllowedModes Separation

**Problem:** Existing `allowed-environments: ["real"]` in conformance fixtures is a RunMode discriminator (meaning "don't dry-run"), not a deployment-context identifier. SQL team's profile `context` field is orthogonal — a deployment-context axis. Overloading one field destroys dry-run safety intent.

**Solution:** Two-field separation:
- `AllowedEnvironments []string` → profile-context allowlist (values: `cli-operator`, `vscode-operator`, `ci`, `headless-server`, `test`)
- `AllowedModes []string` → RunMode allowlist (values: `real`, `dry-run`, `replay`)

**Impact:** Fixture migration in tv-enum.yaml (25 occurrences from `allowed-environments: ["real"]` to `allowed-modes: ["real"]`); zero runtime breakage (fields unenforced today).

### 2. Approval Gate Coverage and Classification

**Problem:** Approval gate currently enforced only on substitution path; direct-invocation path (`Execute()` → `runtime.Invoke()`) has zero enforcement. Shipping ProfileApprovalGate without closing direct-invocation gap gives false safety.

**Solution:** Phase 1 adds classification-aware approval check to direct-invocation path. New `Classification *string` field on `ToolAction` (per-action, not per-tool). Four values: `read-only | mutating | destructive | unspecified` (nil = unspecified).

**Impact:** All tool invocations (substituted and direct) pass through gate after Phase 1.

### 3. Interactive Unspecified — Grandfathering

**Problem:** New rule "unspecified fires gate in interactive contexts" causes regression: 10-step runbook goes 0 → 10 approval prompts. Scope: all existing unclassified tools in gert-sqllivesite hit on day 1.

**Solution:** Grandfathering via explicit `requires-approval: false` acting as legacy classification override (equivalent to `classification: read-only`). Profile-level `legacy_unspecified_policy: allow | prompt` field enables migration window. New profiles default to `prompt` (enforces new rule); existing deployments opt into `allow` during migration, see PKG-W warnings, add classifications at their pace.

**Impact:** Day-1 safety (new profiles enforce unspecified gate) without day-1 breakage (existing opt-outs honored).

### 4. Attendance as Separate Profile Property

**Problem:** `TTYOutput` currently conflates three properties: (a) approval gate selection, (b) stdin/stdout physical availability, (c) terminal-rendered prompts. Current hardcoded `TTYOutput: true` in `gert run` blocks on stdin in CI pipeline (latent bug).

**Solution:** Add `Attended *bool` to WireOptions. Profile declares `attendance: attended | unattended` explicitly; gate selection reads profile first, falls back to TTY inference if profile is silent. Physical IO availability remains driven by TTY detection (correct axis).

**Impact:** CI profiles explicitly declare `attended: false`, gate auto-approves without blocking on stdin. VS Code autonomous agents declare `context: vscode-operator, attendance: unattended`, gates deny mutations correctly.

### 5. Test-Context Binding Restriction

**Problem:** Which tools are allowed in test context? HTTP endpoints are dangerous; subprocess mocks may be legitimate for transport-layer testing.

**Solution:** Tier 0 plan-time check (transport.mode available on plan.Tools):
- `native` → always allowed
- `mcp` (subprocess) → blocked by default, allowed with `transport.allow_subprocess_in_test: true` opt-in
- `mcp-http` → never allowed in test context (no override)

**Impact:** Fail-safe default; legitimate hermetic test servers can opt in; real HTTP bindings absolutely forbidden in test. Check ships with `gert plan --profile` command (0.5 days).

### 6. OQ2 — Subprocess Framing

**Status:** Closed (not as library integration, as subprocess continuation).

**Deliverables for Phase 0:**
- Framing protocol document (Gert Core)
- Lifecycle spec: start, stop, crash-recovery, version detection (Gert Core + extension team)
- Capability-advertisement handshake (Gert Core)
- CancelRequest message type (Gert Core)
- Feasibility validation against current extension (SQL Live-Site Ops)

Library integration only if subprocess proof fails on blocking criterion.

## Design Principles Confirmed

- **Fail-closed default:** Absence of declaration means conservative behavior; no authority granted by silence.
- **Composition over replacement:** `--profile` and `--package-map` compose; profile does not rewrite resolved tool definitions.
- **Explicit migration paths:** Breaking changes include named, auditable escape hatches (e.g., `legacy_unspecified_policy`).
- **Source-grounded verification:** All claims verified against actual codebase; ghost fields identified and wired; coverage gaps disclosed early.

## Phase 1 Scope Additions

From four rounds of negotiation:
- Per-action `Classification *string` field: 0.5 days
- `legacy_unspecified_policy` profile field + gate routing: 0.5 days
- Plan-time PKG-W warnings for unclassified actions: 0.5 days
- Declared `attendance` on WireOptions + profile: 0.5 days
- Test-context binding check (Tier 0): 0.5 days
- Direct-invocation approval gate enforcement: 1–2 days
- `gert plan` profile compatibility report: 2–3 days
- AllowedModes field + fixture migration: included

**Total addition:** ~4 days to existing Phase 1 scope.

## Negotiation Dynamics

The SQL team corrected Gert core four times during the exchange and was right every time:
1. Unspecified classification default (not read-only) — correct
2. Fail-closed unattended approval — correct
3. AllowedEnvironments vocabulary collision — correct
4. Environment-vocabulary ruling revision after contradiction — correct

The design benefited from iterative challenge. No adversarial tension; all refinements were technical and grounded in source code. Final design is sound.

## Next Steps

- Confirm SQL team's context vocabulary matches adopted vocabulary (cli-operator, vscode-operator, ci, headless-server, test)
- Start Phase 0 immediately: profile spec, OQ2 framing, managed-identity feasibility
- Phase 1 starts this week: ~6–8 weeks to delivery
