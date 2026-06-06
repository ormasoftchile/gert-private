# Ken — History

## Seed Context (2026-06-05)

- **Project:** GERT — Governed Executable Runbook Technology
- **Owner / User:** ormasoftchile (Germán)
- **Squad home:** `gert-private` (this repo). The squad is single and serves all repos. Runtime work happens in `ormasoftchile/gert`, but squad memory stays here.
- **My role:** Second Backend Dev, paired with Don
- **Why I was hired:** OQ-M5 of the Runtime Migration Plan (ratified 2026-06-05, commit 541d194) called for a pair to run Streams E (GIS) and F (GCP) in parallel with Don's critical path on Streams A–D, G, H. Target wallclock: 10–15 days vs. 12–18 solo.

## Project Snapshot at Hire

- **Tech Stack:** Go (runtime), Azure (Functions, Service Bus, Container Apps, Static Web Apps, Entra ID), TypeScript (web/extensions)
- **Phase 1 of GXL/GIS/GCP design:** complete in `gert-private`. 208 conformance vectors, full spec sections 03a–03d, parse-gate, three EBNF grammars, fixture migration done, `gert migrate-expr` tool scaffolded.
- **Phase 2 sketch:** Don wrote a working GXL evaluator + parser + stdlib (commits `97ce48b..5c550c0`) in this repo. Stripped per scope correction; designated as cherry-pick source for the runtime migration (OQ-M2).
- **My first assignment (likely):** Phase A of the Runtime Migration in `ormasoftchile/gert` — pair with Don on PJVM + Clock + harness + DRIFT-DETECTION-001 (vector sync script + `make verify-vectors` CI gate). Then split: Don takes Streams B–D (critical path), I take Streams E and F in parallel.

## Key Decisions to Respect

Read `.squad/decisions.md` in full at first spawn. Highlights:
- **OQ-M1:** Vendored vectors + DRIFT-DETECTION-001 sync infrastructure (NEW Phase A scope item)
- **OQ-M2:** Cherry-pick sketch `97ce48b..5c550c0`, treat as unreviewed
- **OQ-M3:** Build tag `//go:build gxl` for the entire migration window
- **OQ-M4:** Hard cutover at Phase H, CHANGELOG only (pre-1.0)
- **OQ-M5:** Pair model — Ken (me) + Don, both in this squad, both working in gert repo
- **Single-squad directive (2026-06-05):** Never create a parallel squad in the runtime repo. Squad memory lives in `gert-private`.

## Learnings

### 2026-06-05 — gert repo conventions (from Phase A / DRIFT-DETECTION-001)

- **No Makefile existed** in `ormasoftchile/gert` — created from scratch. Pattern: `.PHONY` targets, `help` as default goal, `SHELL := /usr/bin/env bash`.
- **No `scripts/` directory existed** — created. Convention: executable bash scripts go here.
- **No `testdata/` directory existed** — created. `testdata/vectors/` is the first entry; this is the canonical path for vendored conformance vectors per OQ-M1.
- **CI framework:** GitHub Actions. Existing workflow at `.github/workflows/e2e.yml` uses `actions/checkout@v4`, `ubuntu-latest`, `actions/setup-go@v5` with `go-version: '1.21'`.
- **Temp dirs in Makefile:** `.gitignore` lists `tmp/` and `.temp/`. I used `.verify-vectors-tmp/` as verify scratch — always cleaned up by the target, no .gitignore entry needed.
- **Both repos in same org:** `ormasoftchile/gert` + `ormasoftchile/gert-private`. `GITHUB_TOKEN` likely covers cross-repo reads within the org — relevant for the CI Option A/B decision.
- **gert-private is read-only** from the runtime repo perspective — never commit to it from a gert worktree task.
- **Worktree pattern:** Phase A uses dedicated worktrees (`gert-phase-a-drift`, `gert-phase-a-pjvm`) branched from main. Don't `cd` to main repo for edits; the worktree IS the working directory.

## 2026-06-05 Phase 1 Complete

Phase 1 closed before my first session. TESS-AMBIG-3..6 resolved:
- **GXL-TYPE-005:** Boolean ordered comparison forbidden
- **GXL-TYPE-001:** Extended to forbid list/object equality (scalars+null only)
- **GXL-PATH-004:** Field access on non-object/null
- **Corpus:** 211 GXL vectors, 0 TBD (83 parse + 92 eval + 36 path)
- **Dogfood:** All 22 runbook fixtures clean (P1–P5 all zero)

Don + I cleared to start Phase A in `ormasoftchile/gert`. PJVM + Clock + harness + DRIFT-DETECTION-001 are first deliverables.

## 2026-06-05 — Phase A: DRIFT-DETECTION-001 Shipped (PR #8, First Shipment)

**Worktree:** `gert-phase-a-drift`  
**PR:** https://github.com/ormasoftchile/gert/pull/8 (draft, awaiting merge)  
**Status:** First ever shipment (pre-Phase-B)

**Shipped:**
- `scripts/sync-vectors.sh` — syncs tv-*.yaml + schema.json from gert-private; writes `VECTORS_SHA` (pinned to 3ce53431)
- `Makefile` — `sync-vectors` + `verify-vectors` targets (no prior Makefile in repo)
- `.github/workflows/verify-vectors.yml` — CI gate (PR + main + weekly cron)
- `testdata/vectors/README.md` — runbook (sync flow, drift detection, upgrade path)

**Option Chosen: A (best-effort).** GITHUB_TOKEN + sentinel-SHA fallback. Rationale: immediate within org; upgradeable to Option B (deploy-key) if token insufficient. Local `make verify-vectors` always fully enforced (dev gate during active dev).

**Test Loop Verified:** sync ✅ → verify-clean ✅ → hand-edit + verify-drift ✅ → restore + verify-clean ✅

**Env Contract:** `GERT_PRIVATE_PATH` (default: `../gert-private`)

**Learnings
