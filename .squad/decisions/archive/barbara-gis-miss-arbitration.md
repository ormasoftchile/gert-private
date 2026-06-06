# GIS Miss Semantics Arbitration — Option A Ratified

**From:** Barbara — Spec Arbiter  
**Date:** 2026-06-06T01:40:00-04:00  
**Subject:** SPEC-AMBIG resolution for Phase G integration (Don PR #15)  
**Status:** ✅ RATIFIED — Option A (mandatory-miss-as-error)

---

## Context

During Phase G runtime integration, Don identified a semantic divergence between Ken's GIS engine (`internal/eval/gis/`) and the legacy Go `text/template` engine:

| Engine | Behavior on Missing Mandatory Path |
|--------|-----------------------------------|
| **Legacy** (`TemplateEvaluator`) | Returns empty string (`missingkey=zero`) — silent degradation |
| **New** (GIS as shipped) | Returns `GIS-PATH-MISSING` error — hard failure |

Don flagged this as a SPEC-AMBIG: "If existing runbooks rely on missing-key-as-empty-string semantics, they will error under `gxl`."

---

## Ruling: Option A — Ratify Ken's Design

**The spec already mandates hard errors for mandatory path misses.**

Evidence:
1. `gis.ebnf` §4.3: "Silent empty-string substitution is FORBIDDEN by default. Missing variables and path segments are hard errors unless the author has explicitly marked the segment optional."
2. `03b-interpolation-syntax.tex` §Unresolved Variables: "Silent empty-string substitution for missing variables is **forbidden**. Every unresolved reference is a hard error."
3. `tv-gis-path.yaml` TV-GIS-PATH-004: Tests `${a.b?.c.d}` with missing root → expects `GIS-PATH-MISSING` error. All 15 vectors PASS.

**Legacy zero-value-on-miss was an under-specified implementation leak, not a contract.**

The Go template `missingkey=zero` option was a runtime convenience that was never specified. Authors who relied on it were depending on undefined behavior.

---

## Spec Files Modified

| File | Section | Change |
|------|---------|--------|
| `design/gert/grammar/gis.ebnf` | Header | Updated date, added Arbitration note referencing this decision |
| `design/gert/sections/03b-interpolation-syntax.tex` | §Unresolved Variables | Added `\subsection{Migration Note: Legacy Zero-Value Semantics}` with explicit ruling and migration path |

**No grammar changes required.** The grammar already correctly specifies the behavior.

**No conformance vector changes required.** TV-GIS-PATH-004 already tests mandatory-miss-as-error; all 15 vectors pass as authored.

---

## Impact on Don (Phase H)

**No code changes to GIS engine.** Ken's implementation is correct.

**No CLI flag needed.** Option C (compat mode with `GERT_GIS_LEGACY_MISS=1`) is rejected.

**No parser changes needed.** Option B (explicit dual-mode with new operator) is rejected.

**Phase H proceeds as planned:**
1. Remove build tag `//go:build gxl`
2. Delete legacy engine (`internal/expr/`, `evaluators_legacy.go`)
3. Canonicalize runbooks (rename `-gxl.runbook.yaml` variants)
4. Document migration in CHANGELOG

---

## Impact on Ken (Phase E)

**No changes to `internal/eval/gis/`.** The engine is correct.

---

## Migration Path for Runbook Authors

Runbooks that relied on silent empty-string substitution MUST be updated:

### Option 1: Optional Chaining (preferred for interpolation-local tolerance)

```yaml
# Before (legacy — fails under GIS)
title: "Hello ${user.email}"

# After (explicit opt-in soft-miss)
title: "Hello ${user?.email}"
```

### Option 2: Capture Default (preferred when variable should have meaningful default)

```yaml
capture:
  user_email:
    path: local.json.user.email
    default: "unknown@example.com"

# Then use normally — variable always exists
title: "Hello ${user_email}"
```

### Option 3: Guard with `when:`

```yaml
- step:
    id: send_email
    type: tool
    when: 'user.email != null'
    tool:
      name: mailer
      args:
        to: "${user.email}"
```

---

## Survey: Legacy Runbook Exposure

Surveyed 10 runbooks in `ormasoftchile/gert/examples/`:

| Runbook | Template Syntax | GIS Exposure | Risk |
|---------|-----------------|--------------|------|
| collect-health.runbook.yaml | `{{ .svc }}` (Go template) | None | ✅ Already migrated to `-gxl.runbook.yaml` |
| check-service.runbook.yaml | `{{ .service_host }}` | None | ✅ Already migrated |
| incident-triage.runbook.yaml | `{{ .service_name }}` | None | No GIS paths |
| network.runbook.yaml | `{{ .service_name }}` | None | No GIS paths |
| multi-region-rollout.runbook.yaml | `{{ .region }}` | None | No GIS paths |
| service-health-branching.runbook.yaml | `{{ .hostname }}` | None | No GIS paths |

**Conclusion:** All legacy runbooks use Go template `{{ }}` syntax, not GIS `${}`. The `-gxl.runbook.yaml` variants already exist for GIS testing. No production runbooks are at risk from this ruling.

---

## Decision Summary

| Aspect | Value |
|--------|-------|
| **Ruling** | Option A — mandatory-miss-as-error (Ken's design ratified) |
| **Grammar Changes** | None |
| **Conformance Vector Changes** | None (15/15 pass) |
| **Phase H Impact** | None — proceed as planned |
| **Migration** | Authors use `?.` or `capture.default:` for soft-miss |

---

## References

- `design/gert/grammar/gis.ebnf` §4.3 (Unresolved Variables)
- `design/gert/sections/03b-interpolation-syntax.tex` §Unresolved Variables
- `design/gert/conformance/tv-gis-path.yaml` TV-GIS-PATH-004
- `.squad/decisions.md` "Phase G Runtime Integration (Don PR #15)" — handoff statement
- `.squad/decisions.md` "Phase E + F speculative kickoff" — Ken's design
- `.squad/decisions.md` "Runtime Migration Plan RATIFIED" — OQ-M4 (hard cutover)
