# Phase F — GCP Capture Path Engine

**From:** Ken (Backend Dev)  
**Date:** 2026-06-06T00:31:40-04:00  
**PR:** ormasoftchile/gert#14 (draft, `phase-f-gcp`, stacked on `phase-a-pjvm`)  
**Status:** ✅ SHIPPED — 41/41 GCP-PATH vectors PASS on first pass.

---

## GCP Engine File Paths

| Path | Purpose |
|------|---------|
| `internal/eval/gcp/doc.go` | Package documentation |
| `internal/eval/gcp/gcp_gxl.go` | Error codes (`GCP-PARSE-*`, `GCP-RESOLVE-*`, `GCP-DEFAULT-SUBTREE`, `GCP-TYPE-001`), `GCPPath` AST types, `ResolveString` public entry point |
| `internal/eval/gcp/parser_gcp_gxl.go` | Recursive-descent parser for all four source prefixes: `LocalCapture`, `StepCapture`, `HttpCapture`, `EventCapture` |
| `internal/eval/gcp/resolver_gcp_gxl.go` | PJVM resolver; JSON/YAML parsing; GDP traversal; §6 default policy enforcement |
| `internal/eval/gcp/gcp_unit_gxl_test.go` | 15 edge-case unit tests |
| `internal/eval/harness/path_runner_gcp_gxl_test.go` | Conformance harness runner (`gcpPathRunner`) |
| `internal/eval/harness/conformance_gxl_test.go` | *(minimal edit)* — wired GCP-PATH runner (replaced Phase A stub) |

All files carry `//go:build gxl`.

---

## Vector Results

**41 / 41 PASS** — zero failures, zero skips on GCP-PATH corpus.

```
Conformance totals: total=267 pass=41 fail=0 skip=226
  GCP-PATH   41/41 ✅ (Phase F — this PR)
  GIS-PATH   15   SKIP ← Phase E (Ken-E, parallel branch)
  GXL-PARSE  83   SKIP ← Phase B (Don)
  GXL-EVAL   92   SKIP ← Phase C (Don)
  GXL-PATH   36   SKIP ← Phase D (Don)
```

All 41 GCP-PATH vectors classified and passed:
- Local captures: stdout/stderr/exit_code/json/yaml, bare root, mixed GDP (001–020)
- Parse errors: unknown prefix, invalid suffix, negative index, trailing dot, empty segment, invalid header name (021–025, 031, 038)
- HTTP captures: status, body (JSON body), headers, absent-header soft null (026–031)
- Event captures: id, body (JSON body), headers (032–034)
- Step captures: json, stdout legacy form, exit_code, cross-step GDP (035–037)
- YAML scalar edge cases: timestamp as string per OI-GCP-06 (039)
- Default policy: GCP-DEFAULT-SUBTREE for object/array capture (040), GCP-TYPE-001 for scalar type mismatch (041)

---

## Design Choice: Separate Package (Option A)

GCP lives in `internal/eval/gcp/` — fully independent of `internal/eval/gxl/` and `internal/eval/gis/`. Rationale:

1. **Different surface languages.** GCP has source prefixes (`stdout`, `json`, `http.*`, `event.*`, `step.*`) that are entirely absent from GXL/GIS. Sharing code would require a confusing abstraction layer.
2. **Different resolution semantics.** HTTP/event headers have soft-null resolution (absent → null). §6 default policy (GCP-DEFAULT-SUBTREE, GCP-TYPE-001) is GCP-only. GIS has optional-chaining semantics. GXL uses variable bindings. These diverge enough to make sharing harmful.
3. **Don's precedent (Phase D thin-entrypoint).** Phase D reused Phase C's evaluator with a thin wrapper. GCP has no Phase C analogue to reuse — it starts from scratch on an entirely different grammar. Separate is cleaner.

The only shared code is `internal/eval/core` (PJVM value model, `FromYAML`) — which is appropriate since PJVM is the common representation layer.

**GDP traversal:** Implemented locally in `resolver_gcp_gxl.go::traverseGDP`. The GCP traversal semantics differ slightly from GXL (GCP uses GCP-RESOLVE-003 for both non-array and out-of-bounds; GXL distinguishes these). No sharing with GXL path logic.

---

## Implementation Notes

### OI-GCP-06: YAML Timestamps as Strings

yaml.v3 tags date-like scalars (`2026-06-04`) as `!!timestamp` using YAML 1.1 conventions. The spec (OI-GCP-06) requires YAML 1.2 core schema behaviour: no timestamp type, dates stay as strings. Implemented via `gcpFromYAML()` — a GCP-local YAML-to-PJVM converter that maps `!!timestamp` (and any other `core.ErrUnsupportedYAMLTag` scalar) to `core.NewString(node.Value)`. TV-GCP-PATH-039 passes.

### §6 Default Policy

Checked post-resolution in `checkDefaultPolicy()`:
- If `capture_default` present AND result is KindObject/KindArray → `GCP-DEFAULT-SUBTREE`
- If `capture_default` present AND resolved scalar type ≠ default type (and neither is null) → `GCP-TYPE-001`

The harness passes `capture_default` as a key in the variables map; the resolver reads it and applies the policy. TV-GCP-PATH-040 and -041 pass.

### GCP-PARSE-006 Step Check

The step-existence check (`checkStepExists`) inspects `variables["steps"][stepID]` at parse time, simulating plan-time runbook validation. TV-GCP-PATH-038 passes.

### HTTP/Event Header Soft-Null

Per gcp.ebnf §7.2 and TV-GCP-PATH-030, absent HTTP/event headers resolve to `null` rather than raising GCP-RESOLVE-002. Implemented in `resolveHTTP` and `resolveEvent` header branches. Header lookup is case-insensitive (RFC 7230 §3.2) via `strings.ToLower`.

---

## Handoffs

**→ Barbara (Spec):** None required. All error codes (`GCP-PARSE-001..006`, `GCP-RESOLVE-001..004`, `GCP-DEFAULT-SUBTREE`, `GCP-TYPE-001`) were pre-ratified in Phase 1. No new ambiguities surfaced. No spec gaps found.

**→ Tess (Corpus):** None required. All 41 vectors passed as authored. Zero corpus bugs discovered. No amendments needed.

**→ Don (Phase G):** GCP engine ready. Merge order: #9 → #14. Phase G (cutover) unblocks once E (GIS, Ken-E) and F (this PR) are merged.

---

## Stack Order & Merge Path

```
main
 └─ phase-a-pjvm       (PR #9)  ← merge first
     ├─ phase-e-gis    (Ken-E, 15 GIS vectors)    — parallel
     └─ phase-f-gcp    (PR #14, this PR)           — parallel
         
Don's GXL critical path (stacked B→C→D) is independent:
main → phase-a-pjvm → phase-b → phase-c → phase-d
```

PR #14 is independent of #10/#11/#12 and Phase E. Merge in any order once #9 lands.

---

## Phase F Exit-Criteria Readiness

✅ **Complete:**
- GCP capture engine shipped (`internal/eval/gcp/`)
- Self-contained: no entanglement with gxl/ or gis/
- Parser covers all four source prefixes + all error codes from gcp.ebnf §7
- Resolver covers all PJVM sources + GDP traversal + §6 default policy
- OI-GCP-06 timestamp handling implemented
- 41/41 GCP-PATH vectors PASS
- 15 unit tests PASS
- `go test ./...` (no tag): no regressions introduced (pre-existing markdown test failure unchanged)
- Build tag discipline: all files `//go:build gxl`

✅ **No Action Items:** No Barbara action. No Tess action. No charter/skills/team changes required.
