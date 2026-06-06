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

## 2026-06-06 — Phase E + F Shipped (PR #13 + #14) — 56/56 PASS

**Status:** Two parallel shipments — GIS (Phase E, 15 vectors) + GCP (Phase F, 41 vectors) — both 100% conformance.

**PRs:** ormasoftchile/gert#13 (phase-e-gis) + #14 (phase-f-gcp), both draft, stacked on phase-a-pjvm (PR #9).

**My Track Record:**
- **Phase A (PR #8):** DRIFT-DETECTION-001 (sync-vectors.sh + verify-vectors CI gate) ✅
- **Phase E (PR #13):** GIS path resolver (internal/eval/gis/, 15/15 vectors) ✅
- **Phase F (PR #14):** GCP capture engine (internal/eval/gcp/, 41/41 vectors) ✅

Total: 3 shipments, 56 conformance vectors, zero defects.

### Key Design Pattern: Three Separate Packages, Single Shared Foundation

**Architecture:** gxl, gis, gcp are three independent evaluators, each in its own package:
- `internal/eval/gxl/` — variable bindings + operator precedence + stdlib
- `internal/eval/gis/` — optional chaining + null-as-miss semantics
- `internal/eval/gcp/` — four source prefixes + GDP traversal + §6 default policy

**Why separate?** Each has fundamentally different resolution semantics. Sharing would require abstraction layers that obfuscate the logic. Attempting to fork a "generic resolver" across all three creates coupling and refactor overhead without benefit. Keep each self-contained.

**Only shared layer:** `internal/eval/core` (PJVM value model, `FromYAML`, `core.Value` interface). This is the right level of abstraction — the common representation, not the traversal logic.

**Implication for future work:** Don't expect GIS/GCP to reuse GXL path logic. When Phase G integrates all three into the request/response flow, each stays independent, each gets its own harness runner entry point.

### Implementation Notes

**GIS (Phase E):**
- Two-level parser: outer template string layer, inner GIS expression tokenizer
- Lexer uses longest-match for `?.` and `?.[` to avoid ambiguity
- Eval implements null coercion (null-as-miss) + short-circuit semantics per optional-chaining design
- Falsy values (`""`, `false`, `0`, `[]`) correctly treated as **present** values, not misses
- Stdlib integration: str functions (toLower, toUpper, trim, etc.) receive optional results and propagate `""` on miss

**GCP (Phase F):**
- Four source prefixes (local/http/event/step) parsed uniformly but resolved differently per source
- HTTP/event headers resolve to soft-null (absent header → null, not error) per RFC 7230 case-insensitivity
- YAML timestamp handling: yaml.v3 tags dates as `!!timestamp`, but spec (OI-GCP-06) requires YAML 1.2 strings. Implemented local YAML converter (`gcpFromYAML`) to map timestamp tags back to strings
- §6 default policy: post-resolution check (`checkDefaultPolicy`) handles both GCP-DEFAULT-SUBTREE (object/array capture with default) and GCP-TYPE-001 (scalar type mismatch with default)
- GDP traversal implemented locally in `traverseGDP` — different semantics than GXL (GCP uses single error code for both non-array and out-of-bounds)

### Learnings for Phase G

1. **Separate-package pattern scales.** When Don integrates all three engines into the request/response flow, he won't be juggling a monolithic resolver or a tangle of conditional branches. Three independent entry points, three independent error code sets, three independent unit test suites.

2. **Build tag discipline holds.** All files `//go:build gxl`. The tag stays until Phase H cutover. Once old engine is deleted, tags come off and we run both systems side-by-side in CI to verify behavior parity.

3. **Error code pre-ratification critical.** All 56 vectors passed without corpus bugs or spec gaps. This happened because Barbara's Phase 1 arbitrations (TESS-AMBIG-3..6) locked down all error codes upfront. Phase G can proceed without returning to Spec.

4. **No handoff surprises.** Barbara: no action. Tess: no action. Each phase just ships. This is what pre-ratification looks like.

### Next: Phase G + H (Don)

Don takes integration (Phase G) and cutover (Phase H). My assignment (Phases E–F) closes once #9 merges and my PRs (#13/#14) pass final review. Wallclock on Phases A–F: ~6 days (2026-06-01 to 2026-06-06). Target was 10–15 days total with parallel streams; we're on track.

