---
updated_at: 2026-08-19T17:44:46-07:00
focus_area: gert-vscode — first live ICM MCP run (serve starts; catalog contract gap remains)
active_issues:
  - serve/requires catalog parity spec defect
---

# ACTIVE FOCUS (2026-08-19) — Live ICM MCP execution from gert-vscode

> ✅ **F5 works.** ✅ **`gert serve` starts.** ⚠️ Remaining gap: `serve` does not resolve `requires:` catalogs, so the tactical live run cannot prove package-exported TSG execution.

## Current state — read first

- Work is on new machine `CPC-crist-LKO5U`; `icm-mcp` starts here. The old-machine `401` identity-broker blocker is retired and must not be chased.
- F5 now works from a clean clone. Coordinator verified after deleting `out/extension.js` and `gert.exe`; TypeScript and Go artifacts rebuilt. Go fallback handled a stale inherited process environment where `go` was absent from PATH even though `C:\Program Files\Go\bin` was already on Machine PATH. No PATH edit was made.
- `gert serve` starts with the serve-compatible SQL Live-Site map. Coordinator verified `gert serve: listening on 127.0.0.1:65191`.
- The extension setting `gert.packageMap` in `C:\One\gert-sqllivesite\.vscode\settings.json` points to `packages/incident-routing.vscode-mcp.serve-package-map.yaml`.

## Remaining contract defect

Barbara ruled the current `serve` rejection of `requires:` is a deliberate implementation guard, but the durable dialect split is a **gert-private spec defect**. `tool-paths:` cannot carry package version constraints, package identity/provenance, package digest semantics, or exported runbooks from `sql-livesite-tsgs`. Long-term fix: teach `serve` to resolve `requires:` per run with run/plan catalog semantics, or formally specify a serve-only restricted dialect with schema and conformance coverage.

Don made the gap loud in the Go runtime: SERVE-W001 warns when served runbooks declare `requires:`; DINC-002 continued dynamic-include misses now emit explicit warning/stderr and skipped-reason output. This prevents another false-green where `icm-tsg-routing-complete` appears without executing the TSG.

## What a live run today can prove

A deliberate single `@gert /run <runbook.yaml>` today can prove:

- VS Code extension F5/debug host path starts.
- `gert serve` starts and the extension can talk to it.
- The live VS Code MCP ICM retrieval path works if `get_icm_incident` is reached and authorized.
- The local/native `tsg-recommendation` path works from `packages/incident-routing-vscode-mcp` (currently expected to return `no-suggestion`).

It **will not prove**:

- `sql-livesite-tsgs` package catalog export resolution.
- Execution of GEODR0001 through `resolve_from: catalog`.
- Full run/plan/serve package-map parity.

## Standing rules for this effort

- **Never burn live invoke attempts on diagnostics.** Established cost of violation: one development machine. Zero-invoke paths or mocks only.
- Stop after one live run attempt. Diagnose from the gert Output channel and unit tests; do not re-invoke to see if it was a fluke.
- Treat false-green/vacuity as the defining defect class of this engagement. Current count: sixth in the runtime/extension path, plus Barbara's independent seventh in Squad archival tooling. Every success claim must identify the load-bearing path it actually exercised.

---

# Background — this repo's standing role

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
