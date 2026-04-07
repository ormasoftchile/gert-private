# Edge Color Fix Verification — Knov E2E Report

**Date:** 2026-04-06  
**Tested by:** Knov (E2E Tester)  
**Commit verified:** `6e6f8dd` — `fix(web): only color edges green when source node was actually visited`

---

## Verdict: ✅ FIX CONFIRMED — DNS-failed edges are GREY

The edge color bug is **fixed and verified in the browser**.

---

## What was tested

**Runbook:** `examples/service-health-branching.runbook.yaml`  
**Input:** `server_name = github.com`  
**Outcome:** RESOLVED (DNS resolved → HTTP OK → Service is healthy)

The "DNS failed" branch was **not taken** during this run — its edges must appear GREY.

---

## Evidence

### SVG edge color breakdown (post-run)

| Color | Count | Meaning |
|-------|-------|---------|
| `#4caf50` (green) | **5** | Taken path: start → resolve_dns → check_http → confirm_healthy → end |
| `#404040` (grey)  | **2** | Not-taken path: DNS-failed branch edges |
| Other | 0 | — |

**5 green + 2 grey. No stray greens on the not-taken branch.**

### Screenshots

- `edge-color-02-run-complete.png` — full UI at RESOLVED outcome  
- `edge-color-03-graph-closeup.png` — workflow map panel closeup showing the bifurcated graph

The graph closeup visually confirms:
- Main path (center/right): bright green nodes and green edges
- DNS-failed branch (left stub): dim nodes with grey edges — **correctly not highlighted**

---

## Spec file

`web/tests/specs/knov-edge-color-verify.spec.ts`  
Retained for future regression coverage. Runs in ~7.5 seconds.

---

## No other regressions observed

The full run completed cleanly. The workflow map, outcome banner (RESOLVED), and runbook instructions panel all rendered correctly. No visual artifacts noted.
