# Decision: Phase 5 Schema Gap Resolutions (v0.1.0)

**Date:** 2026-04-24  
**Agent:** John (YAML and Schema Specialist)  
**Status:** ✅ RESOLVED  
**Impact:** Schema v0 finalizations before v0.1.0 tag

---

## Context

Phase 4 design review identified 3 schema ambiguities in `specs/gert-domain-home/schema.json`. These gaps risk downstream misinterpretation of home maintenance configurations. All 3 gaps have been resolved with minimal, surgical changes.

## Gap 1: Routine Scope Ambiguity

### The Problem
- Routine has optional `zone` field and optional `asset` field
- Both can theoretically be present simultaneously
- Semantic intent: exactly ONE scope (zone XOR asset XOR property-wide)
- JSON Schema couldn't enforce XOR across independent optional fields

### The Decision
**Add JSON Schema `not` constraint to enforce mutual exclusivity**

```json
"not": {
  "required": ["zone", "asset"]
}
```

### Why This Works
- ✅ Rejects both-set case
- ✅ Allows either-one case
- ✅ Allows neither case (property-wide)
- ✅ JSON Schema Draft 2020-12 native support
- ✅ No runtime validation needed

### Applied Location
`specs/gert-domain-home/schema.json`, lines 203-211 in `$defs/routine`

### Validation
- ✅ casa-santiago.home.yaml: every routine has exactly one scope (passes)
- ✅ apartment-minimal.home.yaml: every routine has exactly one scope (passes)

## Gap 2: Decision Routing Clarity

### The Problem
- Decision step has `choices` array with optional `next_step` on each choice
- Workflow execution requires explicit routing: which step runs after this choice?
- If `next_step` is omitted, routing is undefined (stall? error? silent fallthrough?)

### The Decision
**Make `next_step` required on every choice item**

Updated schema:
- Added `"next_step"` to `required` array in choice object
- Updated description: "Each choice must specify 'next_step' to define routing"

### Why This Works
- ✅ Eliminates routing ambiguity completely
- ✅ Authoring model is clearer: every decision must route
- ✅ Aligns with existing examples (already had next_step everywhere)

### Applied Location
`specs/gert-domain-home/schema.json`, lines 416-436 in `$defs/incident_step[properties][choices]`

### Validation
- ✅ apartment-minimal.home.yaml plumbing template: all decision choices have next_step (passes)
- ✅ casa-santiago.home.yaml incident templates: all decision choices have next_step (passes)

## Gap 3: Step Execution Ordering Documentation

### The Problem
- IncidentTemplate has `steps` array with arbitrary order
- Runtime execution is sequential top-to-bottom (DAG-based)
- YAML schema had no documentation of this ordering guarantee
- Authors could be confused about step sequence

### The Decision
**Clarify in schema description that steps execute in array order**

Updated text:
```
"Steps are executed sequentially in array order (top-to-bottom)."
```

### Why This Works
- ✅ Documentary fix aligns schema docs with runtime behavior
- ✅ No validation change (pure clarity)
- ✅ Guides authoring best practices (topological ordering)

### Applied Location
`specs/gert-domain-home/schema.json`, line 359 in `$defs/incident_template[properties][steps]`

### Validation
- ✅ Documentary change only — all existing examples remain valid

---

## Summary of Changes

| Gap | Type | Enforcement | Impact |
|-----|------|-----------|--------|
| Gap 1: Routine Scope XOR | Schema | JSON Schema `not` constraint | Reject ambiguous; accept clear |
| Gap 2: Decision Routing | Schema | Required field on choices | Enforce explicit routing |
| Gap 3: Step Ordering | Documentation | Description text | Clarify existing behavior |

## Artifacts

- **Modified:** `specs/gert-domain-home/schema.json` (3 targeted edits)
- **Updated:** `specs/gert-domain-home/schema-notes.md` (added Phase 4 Gap Resolution section)
- **Validated:** Both casa-santiago.home.yaml and apartment-minimal.home.yaml pass updated schema
- **Committed:** `commit 6d2599b...` with detailed message

## Recommendations

1. **Before v0.1.0 tag:** Include these schema changes in release
2. **Documentation:** Link schema-notes.md Phase 4 Gap Resolution section in authoring guide
3. **Versioning:** No semver bump needed (all gaps were pre-release ambiguities, not breaking changes)

---

## Decisions Signed Off

✅ Routine scope will be enforced as XOR at schema level  
✅ Decision routing will require explicit `next_step` per choice  
✅ Step execution order will be documented as sequential top-to-bottom

Ready for v0.1.0 release.

