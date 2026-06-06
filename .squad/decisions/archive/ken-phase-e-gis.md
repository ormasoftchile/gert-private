# Ken — Phase E GIS Path Engine — Ship Memo

**From:** Ken (Backend Dev)
**Date:** 2026-06-06T00:45:00-04:00
**PR:** ormasoftchile/gert#13 (draft, `phase-e-gis`, stacked on `phase-a-pjvm`)
**Status:** ✅ SHIPPED — 15/15 GIS-PATH vectors PASS on first pass.

---

## GIS Engine File Paths

| Path | Purpose |
|------|---------|
| `internal/eval/gis/doc.go` | Package doc + error code catalogue |
| `internal/eval/gis/errors.go` | ParseError, PathError, TypeError + stable codes |
| `internal/eval/gis/lexer.go` | Inner-expression tokenizer; longest-match `?.` / `?.[` |
| `internal/eval/gis/ast.go` | PathExpr/Segment, CallExpr, LiteralExpr |
| `internal/eval/gis/parser.go` | Two-level parser: template string + GIS expression |
| `internal/eval/gis/eval.go` | Path resolver + str stdlib (toLower, toUpper, trim, …) |
| `internal/eval/gis/render.go` | `Render(template, bindings) → core.Value` entry point |
| `internal/eval/harness/gis_runner_gxl_test.go` | Harness runner + `vectorBindings` helper |
| `internal/eval/harness/conformance_gxl_test.go` | *(minimal edit)* GIS stub replaced with `gisPathRunner` |

All files carry `//go:build gxl`.

---

## Design Choice: Separate Package (NOT shared with GXL)

**Decision:** `internal/eval/gis/` is self-contained. It does NOT import or share code with `internal/eval/gxl/`.

**Rationale:** GIS and GXL share a grammar family, but GIS optional-chaining semantics (null-as-miss, short-circuit-to-`""`) are fundamentally divergent from GXL's strict path-error model. A shared resolver would require forking its core traversal loop anyway. The separate package approach:
- Keeps each evaluator independently deployable
- Avoids coupling Phase E to Don's GXL critical path
- Matches the grammar's own design intent (`gis.ebnf` overrides GDP for GIS context only)

Don noted in his Phase D memo that "GIS adds optional chaining semantics which likely warrant separate resolver" — confirmed.

---

## Vector Results

```
go test -tags gxl ./internal/eval/...

GERT conformance: total=267 pass=15 fail=0 skip=252
  GCP-PATH     total=41  pass=0  fail=0  skip=41   ← Phase F (Ken, parallel stream)
  GIS-PATH     total=15  pass=15 fail=0  skip=0    ← ✅ Phase E complete
  GXL-EVAL     total=92  pass=0  fail=0  skip=92   ← Phase C (Don, not in base)
  GXL-PARSE    total=83  pass=0  fail=0  skip=83   ← Phase B (Don, not in base)
  GXL-PATH     total=36  pass=0  fail=0  skip=36   ← Phase D (Don, not in base)
```

**15/15 PASS** — zero failures, zero silent skips.

### Vector coverage

| Group | IDs | Semantics | Result |
|-------|-----|-----------|--------|
| Optional root miss | 001, 013 | Root not found, first seg `?.` → `""` | ✅ PASS |
| Deep optional chain | 002 | Miss mid-chain, tail skipped | ✅ PASS |
| Optional + mandatory tail | 003 | Optional miss short-circuits mandatory tail | ✅ PASS |
| Mandatory prefix error | 004 | Root mandatory, `?.` not yet reached | ✅ PASS |
| Mixed chain present | 005 | Mandatory prefix found, optional miss | ✅ PASS |
| Null as optional miss | 006 | `b = null` via `?.b` → `""` | ✅ PASS |
| Falsy values NOT miss | 007–010 | `""`, `false`, `0`, `[]` are present values | ✅ PASS |
| Optional bracket | 011–013 | `?.[N]` with hit, OOB, missing root | ✅ PASS |
| Stdlib + optional arg | 014 | `str.toLower(user?.name)` → `""` | ✅ PASS |
| Invalid optional root | 015 | `${?.root}` → GIS-PARSE-003 | ✅ PASS |

---

## Stack & Merge Path

```
main
 └─ phase-a-pjvm          (PR #9 — awaiting Germán merge)
     └─ phase-e-gis        (PR #13, this PR)
```

Also running in parallel (independent branches off phase-a-pjvm):
```
phase-a-pjvm
 └─ phase-e-gis  (this PR — GIS)
 └─ phase-f-gcp  (sibling Ken-F — GCP, 41 vectors)
```

**Merge order:** PR #9 must land before #13 can merge cleanly. Phase E and F are independent.

---

## Handoffs

**→ Barbara (Spec):** None required. All error codes (GIS-PARSE-001..004, GIS-PATH-MISSING, GIS-TYPE-001) were pre-ratified in Phase 1. No new ambiguities surfaced during implementation.

**→ Tess (Corpus):** None required. All 15 vectors passed as authored. No corpus bugs found. One observation: TV-GIS-PATH-007 and TV-GIS-PATH-006 produce identical expected output (`""`) but for different reasons (present-empty-string vs null-as-miss) — this is intentional per the notes. The harness correctly distinguishes by evaluating the actual PJVM value path, not by special-casing the output.

**→ Ken-F (Phase F / GCP):** GCP runner remains `stubRunner` in this PR — Phase F is your lane, untouched.

**→ Don (Phase G / Cutover):** No dependencies. Phase G can proceed once A–F are merged. GIS package is fully wired; no further harness changes needed on my end for the migration.

---

## `go test ./...` (no tag) Baseline

The `pkg/preview/render/markdown` test failure (`change-request.golden.yaml: no such file`) is **pre-existing** (confirmed by running on the bare `phase-a-pjvm` base before my changes). My changes do not affect any non-`gxl`-tagged code.
