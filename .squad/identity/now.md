---
updated_at: 2026-06-05T18:13:09-07:00
focus_area: GERT design contracts — grammars, specs, conformance, fixtures
active_issues: []
---

# What We're Focused On

**This repo (`gert-private`) is DESIGN ONLY.**

It holds normative GERT design artifacts that every runtime implementation
(Go, C#, TS, …) consumes:
- Grammars: `design/gert/grammar/*.ebnf` (GXL, GIS, GCP)
- Spec sections: `design/gert/sections/*.tex`
- Conformance corpora: `design/gert/conformance/tv-*.yaml` (264 vectors total)
- Runbook fixtures: `design/gert/testdata/runbooks/`
- Proposals + audits: `design/gert/*.md`
- Web-platform design: `design/web-platform/`

**No language-specific runtime code lives here.** The Go runtime lives in
`ormasoftchile/gert` (a sibling repo). Future C#/TS runtimes will live in
their own repos. All of them implement against the artifacts in THIS repo.

## Phase 1 Status

✅ COMPLETE. GXL/GIS/GCP grammars, normative spec sections, 264 conformance
vectors, GIS optional-chaining extension, fixture migration. All ratified
and shipped.

## Phase 2 Status

**Phase 2 is runtime implementation. It happens in the `gert` repo, not here.**

If runtime implementers find spec ambiguities or missing test vectors during
their work, the response is:
- Open an issue or proposal in THIS repo to fix the design
- Add conformance vectors HERE that pin down the disputed behavior
- Then the runtime repo implements against the corrected design

The 4 days of Phase 2 Go code that briefly lived in this repo's
`internal/eval/` (commits 97ce48b..5c550c0, removed) are available for
cherry-pick into the `gert` repo if any of it is useful as a starting
point. Treat it as a sketch, not a contract.

## Backlog (design-side)

- Barbara: parse-gate proposal (OPQs around in-flight grammar upgrades,
  statically-reachable steps, plan persistence) — was deferred during
  Phase 2 planning, still on backlog
- Tess: keep extending corpus when runtime implementers surface
  unspecified behavior
