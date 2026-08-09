# Orchestration Log — barbara-gate-review-2 (Architecture Gate #2)

**Timestamp:** 2026-08-09T15:40-07:00  
**Agent:** Barbara (Lead/Architect)  
**Session:** GERT Tool Packages MVP — Gate Review #2 (Don's R1–R15 revision against AR-TP-1..10)  
**Task Type:** Architecture review (sync, gate decision)

## Session Summary

Verification gate for Don's revision of all 15 blocking items (R1–R15) from Gate #1. All fifteen issues re-derived from artifacts and validated independently. R1–R15 are confirmed genuinely resolved. Two new blocking items identified (S1, S2) — narrow documentation/mechanical sweep required; architecture direction approved and locked. No re-litigation of ruled items.

**Verdict:** REJECTED — narrow blocking editorial sweep (S1/S2) required; all architecture resolution verified complete

## Validation Method

Each of R1–R15 re-derived manually from the actual artifacts (grammar files, schema files, spec text, fixture examples), not accepted on revision author's word. Independent validation runs:
- `runbook.v1.schema.json` vs. `drain-node.yaml` → 0 errors
- `python design/gert/scripts/verify_corpus.py` → 280/280
- Full `\label`/`\ref` closure across `sections/*.tex` → 293 labels, 223 refs, 0 unresolved

## R1–R15: Verified Resolved

All 15 items confirmed resolved. Sampling (see full matrix in full entry at `.squad/decisions/inbox/barbara-tool-packages-gate-review-2.md`):

| # | Verification | Status |
|---|---|---|
| R1 | §06 §Tool Definition Schema now sole normative source for `.tool.yaml` structure; `actions:` always list; all 12 shipped `testdata/tools/*.tool.yaml` use list form | ✅ |
| R3 | New `LocalOutputs` production in `gcp.ebnf` §3.4a, added to `LocalCapture`, restricted to `execute.kind: runbook` steps | ✅ |
| R5 | New Table `tab:tool-path-bases` with 7 rows naming resolution base vs. containment root for all path kinds | ✅ |
| R11 | §06 §Replay semantics + §13 bullet: substitution replays like `invoke`, nested steps matched by `argv` | ✅ |

## Two New Blocking Items Identified (S1, S2)

### S1 — §03 `\subsection{toolRefs}` teaches deleted rules

**Location:** `03-schema-vnext.tex` lines ~170–188

**Issue:** The schema chapter (where readers look first for document shape) still instructs:
1. Author `alias: kubectl-wrapper` — this is now `PKG-021`, a hard parse-gate error
2. Use default discovery path `tools/foo.tool.yaml` — this is DELETED; the ratified model (AR-TP-2) has no such rule

**Fix Required:** Rewrite §03 `\subsection{toolRefs}` to:
- Remove `alias` example and paragraph (replace with one-line note that it's `PKG-021`)
- Remove default-discovery sentence
- Cross-reference §06 §Runbook Tool Bindings and §Tiers / §Name Resolution for the ratified tier model
- Do NOT restate tier rules locally; point at §06

### S2 — Three `.tool.yaml` listings still use mapping-shaped `actions:`

**Locations:**
1. `03-schema-vnext.tex:3175` — `actions: { get-pods: ... }` format
2. `03-schema-vnext.tex:3296` — `actions: { check: ... }` format
3. `08-security-and-trust.tex:320` — `actions: { deploy: ... sensitive_inputs ... }` format

**Issue:** §06 now states normatively that `actions:` is *always a list*, *never a mapping*. Three listings contradict this. §08 case preserves `sensitive_inputs` placement (only normative statement of where it lives).

**Fix Required:** Convert all three to canonical list form (`- name: <action>`), preserving all fields including `sensitive_inputs`.

## Non-Blocking Sweep Approved (Mechanical)

Four optional cleanups approved as mechanically safe:
1. Stale ruling citations `inbox/` → `archive/` (15 occurrences across 14 files)
2. Package name drift `com.acme.` → `acme.incident-tools` (30 occurrences, consistency)
3. `LocalStructured` → `LocalOutputs` §3.4a mis-citation (1 occurrence)
4. r23 dry-run narrative accuracy (1 section)

Plus: one clarifying sentence confirming `StepCapture` gains no `step.{id}.outputs.*` form (MVP scope).

## No Re-litigation of R1–R15

Gate 2 decision: R1–R15 are accepted as genuinely resolved and locked. No further iteration on R1–R15 mechanics. S1/S2 are purely documentation sweep; no design decisions needed. The correct answers are already written down in §06 and §08.

## Decision Status

**Status:** Revision required; limited to S1/S2 only  
**Blockers:** S1 (schema chapter) and S2 (mapping-shaped examples) teach contradictory guidance to spec readers  
**Non-blockers:** Optional cleanups (1–4 above) are approved but not required  
**Follow-up:** Ken (Backend Dev) assigned as independent reviser for S1/S2; Edith and Don locked out per directive; scope explicitly limited to §1 (S1/S2) and Ken's discretion on optional §2 cleanups

## Process Note

This gate is materially different from Gate 1: not a re-opening of architecture or ruling, but verification that implementation is correct and a mechanical documentation sweep to close a spec-reader-facing guidance contradiction. AR-TP-1..10 remain fully authoritative and closed.

## Notes

- All R1–R15 re-derivation work documented in this gate entry; high-confidence acceptance signal
- S1/S2 are editorial/consistency issues, not semantic defects in the architecture or implementation
- The spec goes to user (Cristián) next; reader opening schema chapter first must not be told to author keys that parse gate rejects
- Gate #3 will be diff-only check of S1/S2 changes + independent schema validation re-run
