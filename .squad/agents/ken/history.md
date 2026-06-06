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

(none yet — first session)

## 2026-06-05 Phase 1 Complete

Phase 1 closed before my first session. TESS-AMBIG-3..6 resolved:
- **GXL-TYPE-005:** Boolean ordered comparison forbidden
- **GXL-TYPE-001:** Extended to forbid list/object equality (scalars+null only)
- **GXL-PATH-004:** Field access on non-object/null
- **Corpus:** 211 GXL vectors, 0 TBD (83 parse + 92 eval + 36 path)
- **Dogfood:** All 22 runbook fixtures clean (P1–P5 all zero)

Don + I cleared to start Phase A in `ormasoftchile/gert`. PJVM + Clock + harness + DRIFT-DETECTION-001 are first deliverables.
