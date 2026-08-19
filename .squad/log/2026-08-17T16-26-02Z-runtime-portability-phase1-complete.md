# Runtime Portability Phase 1 Complete

**Date:** 2026-08-17  
**Status:** Code-complete, verified exit 0, zero FAIL  

## Phase 1 Manifest Summary

| Agent | Mode | Delivery | Commit(s) |
|-------|------|----------|-----------|
| Don (don-2) | background | Slices 3/6: profile schema, loader, --profile flag; classification validation | 773f332, 7c7402d, 2968006 |
| Tess | background | Slice 7: 25 fixture migrations (allowed-environments→allowed-modes) + TV-CONFORM-GOV-001..013 (13 governance vectors) | 62 PASS / 10 SKIP / 0 FAIL |
| Edith | background | Chapter 17: runtime profiles specification | 48fd214 |
| Ken | background | Slice 4: ProfileApprovalGate, declared attendance, fail-closed; Stream A schema validation | c810b96, ff429d4, 31c0aa0, d7b77c1, b420dda |
| David | background | Slice 5: Tier 0 preflight; Stream C: AzureCLIAuthProvider, token redaction | b982804 |
| Ken (followup) | background | Wiring: Profile into planner + 6 reachability tests | d7b77c1 |
| Barbara | background | Phase 1 reviewer gate: APPROVED-WITH-CONDITIONS; 6 ratifications + 33 architectural rulings | 0 new commits (ruling authority) |

## Execution Summary

- **All streams shipped and integrated.**
- **Barbara's reviewer gate:** APPROVED (all 8 requirements MET). Conditions: clarified approval ≠ classification (orthogonal), monotone-OR composition holds structurally, runbook-level false cannot suppress tool-level true (proven in source).
- **SQL Live-Site counterparty:** Raised Q1–Q3 on governance, received binding answers + evidence; issued 3 counter-positions on classification default, unattended gate, environment vocabulary; all accepted with refinements.
- **Coordinator process errors:** Two noted (grep-only file location misses, erroneous "fabrication" suspicion on existing artifacts); recorded for learning.
- **Deferred defects:** B-18 (static-include governance gap), B-19/B-20 (inert when: fields), B-30 (stdio ensureStarted); cleanup pass recommended post-Phase-1.
- **Known limitations documented:** profileless CI hang hazard (TTYOutput hardcode), AllowedModes not yet enforced (Phase 1 defer), per-tool auth override Phase 3, SubsysAutoShutdown required for proper lifecycle.

## Key Decisions Captured

1. **Approval ≠ Classification:** Two orthogonal dimensions. RequiresApproval gates approval routing only; never implies classification value.
2. **Fail-closed unspecified:** Absent classification must never grant additional execution rights. Interactive: gate fires; unattended: deny.
3. **Vocabulary collision resolved:** `"real"` is RunMode, not context identifier. Recommend AllowedModes field for RunMode enforcement.
4. **Per-tool transport no override:** Profile provides endpoint/auth parameters for declared transport mode; does not switch modes. --package-map selects YAML; profile configures transport.
5. **MCP HTTP host allow-list:** Token attachment restricted to allowed_hosts; replay vector closed via audience-scoped tokens + host restriction (defense-in-depth).

## Quality Metrics

- **Conformance vectors:** 72 total — 62 pass, 10 skip, 0 fail (Slice 7 fixture migration + 13 governance vectors TV-CONFORM-GOV-001..013; Don added TV-CONFORM-GOV-014; GOV-004 recast post-empirical investigation)
- **Build:** `go build ./...` exit 0
- **Test:** `go test ./...` 62 packages, exit 0
- **Defects filed:** 0 new, 3 deferred (pre-existing)
- **Evidence trail:** B-27 trace event wiring verified production-ready; B-24 token redaction surface audit clean

## Phase 2 Preconditions (open items)

- Resolve OQ2: library vs. subprocess for host bridge
- Profile file location/search path (CLI, repo-local, user-config)
- Framing protocol for host bridge IPC (if subprocess chosen)
- Per-tool binding resolver (extends --package-map, not replaces)
