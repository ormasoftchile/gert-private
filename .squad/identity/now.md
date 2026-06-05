---
updated_at: 2026-06-05T09:25:14-07:00
focus_area: Phase 2 — Go runtime implementation of GXL/GIS/GCP
active_issues: []
---

# What We're Focused On

**Phase 1 (spec + grammar + conformance corpus) is complete.** All three grammars
(GXL, GIS, GCP) are normative; 208 baseline conformance vectors + 15 GIS
optional-chaining vectors = 223 total; 22 runbook fixtures migrated and frozen.

**Phase 2 kickoff:** Go runtime implementation of the evaluators.
- Don owns Go runtime build-out (parser + evaluator + path resolver)
- Conformance corpus (`design/gert/conformance/tv-gxl-*.yaml`, `tv-gis-path.yaml`)
  is the bar — runtime is conformant when all 223 vectors pass
- Stream E (migrator) was removed mid-Phase-1; not coming back
- C# runtime parity unblocked but not started; A6 web platform runs parallel
