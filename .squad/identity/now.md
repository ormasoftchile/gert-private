---
updated_at: 2026-08-19T11:54:46-07:00
focus_area: gert-vscode — live ICM MCP execution blocker
active_issues: []
---

# ACTIVE FOCUS (2026-08-19) — Live ICM MCP execution from gert-vscode

> ⚠️ Cristián rebooted to recover the `icm-mcp` MCP server. Resume here.

**The goal:** Cristián runs a SQL Live-Site runbook from VS Code using his
already-authenticated ICM MCP tool. This has **never once succeeded live.**

## State at reboot

**Code — DONE and committed.** `gert-vscode` `ffce6da` on `main`, clean tree,
`npm run compile` 0, `npm test` 186/186.
- Restored the `/run` chat handler. It had been **completely absent** since the
  Petals lifecycle port (`742e368`) deleted the pump-based implementation
  without adding a replacement. `package.json` still declared `run`, so VS Code
  routed it, but the handler fell through `isArmCommand()` to "Unknown command"
  for every `@gert /run` since. 169 tests passed over a dead code path because
  none exercise the `extension.ts` handler body (needs a VS Code host).
  **5th occurrence of the vacuity defect class on this engagement.**
- Removed `/probe-token` entirely. Its 4 unconditional `invokeTool` calls
  exhaust VS Code's MCP restart budget and permanently disable `icm-mcp` for
  the session. Deleting it also fixed a latent compile error
  (`handleProbeToken` was called with no import).

**Blocker — provider-side, NOT gert.** VS Code cannot start `icm-mcp`:
`401 status sending message to https://icm-mcp-prod.azure-api.net/v1/`.
No gert code executes until the provider is healthy. Gert already classifies
this correctly as `provider_unavailable` (fixtures T1/T2 in
`test/mcpBridge.test.js` F4); no change needed there.

## Next steps, in order — DO NOT skip to /run

1. `MCP: List Servers` → is `icm-mcp` **Running**?
2. If not: re-authenticate there → `Developer: Reload Window`.
   ⚠️ Cristián's LIVE evidence is that Reload Window alone did NOT recover it
   after the probe incident; a reboot was required. Live evidence beats the
   theory recorded in the probe-removal decision.
3. Once Running: `@gert /arm-mcp` — **zero invoke budget**, dumps
   `vscode.lm.tools`. Confirm an ICM tool name appears
   (e.g. `mcp_icm_mcp_serve_get_incident_details_by_id`).
4. Then ONE `@gert /run <runbook.yaml>`. Watch **View → Output → "gert"**,
   not the chat. Error codes: `tool_unavailable` = name mismatch vs the YAML
   `vscode_tool` field · `tool_not_found` = registry didn't load ·
   `input_validation_error` = arg shape mismatch · clean output = **first live
   success**.

## Standing rules for this effort

- Never burn live invoke attempts on diagnostics. Stop-on-first-failure or mocks.
- Green unit tests are NOT evidence the live path works. Say so plainly.
- Watch for vacuous tests: assert against real mutations at the production
  call site, not on pure helpers whose signature proves they never see the data.

## Open (non-blocking) — `gert-private` working tree

~20 modified `design/gert/` files (schemas, LaTeX, EBNF) plus untracked
`analysis/`, new schema files, conformance vectors, and root-level debris
(`gert-core-*.md`, `*.jsonl`, `history-sizes.json`). Never triaged —
Cristián declined a Barbara triage pass on 2026-08-19. Real spec work is
likely mixed in with scratch output.

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
