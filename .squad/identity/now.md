---
updated_at: 2026-08-19T16:57:48-07:00
focus_area: gert-vscode — first live ICM MCP run (unblocked, new machine)
active_issues: []
---

# ACTIVE FOCUS (2026-08-19) — Live ICM MCP execution from gert-vscode

> ✅ **UNBLOCKED.** New machine (`CPC-crist-LKO5U`). `icm-mcp` **starts here.**

**The goal:** Cristián runs a SQL Live-Site runbook from VS Code using his
already-authenticated ICM MCP tool. This has **never once succeeded live.**

## Machine change (2026-08-19 16:57) — READ FIRST

The previous machine is **broken and retired**. Live evidence: the ICM
test/diagnostic invocations **corrupted the identity broker** on that box.
Not a session-scoped restart-budget problem as previously recorded — it was
unrecoverable by Reload Window *and* by reboot, and cost the machine.

Consequences:
- The `401 ... icm-mcp-prod.azure-api.net/v1/` blocker was **machine-local to
  the retired box.** It is NOT an active blocker. Do not chase it.
- The standing "never burn live invokes on diagnostics" rule is now a hard
  law with a known price tag. `/arm-mcp` (zero invoke budget) or mocks only.
- `/probe-token` stays deleted, permanently. Its removal rationale in
  `decisions.md` understates the blast radius — see the inbox entry
  `coordinator-icm-identity-broker-corruption.md`.

## Code state (carried over — unchanged, still valid)

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

**Blocker — RESOLVED by machine replacement.** The `401` from
`icm-mcp-prod.azure-api.net` was a symptom of the retired machine's corrupted
identity broker. `icm-mcp` starts on the new machine. Gert's
`provider_unavailable` classification (fixtures T1/T2 in
`test/mcpBridge.test.js` F4) was correct and needs no change.

## Next steps, in order — ONE live run, deliberately

1. `MCP: List Servers` → confirm `icm-mcp` is **Running** on this machine.
2. `@gert /arm-mcp` — **zero invoke budget**, dumps `vscode.lm.tools`. Confirm
   an ICM tool name appears (e.g.
   `mcp_icm_mcp_serve_get_incident_details_by_id`).
3. Then **ONE** `@gert /run <runbook.yaml>`. Watch **View → Output → "gert"**,
   not the chat. Error codes: `tool_unavailable` = name mismatch vs the YAML
   `vscode_tool` field · `tool_not_found` = registry didn't load ·
   `input_validation_error` = arg shape mismatch · clean output = **first live
   success**.
4. If it fails: **stop.** Diagnose from the gert Output channel and unit
   tests. Do not re-invoke to "see if it was a fluke."

## Standing rules for this effort

- **Never burn live invoke attempts on diagnostics.** Established cost of
  violation: one development machine. Zero-invoke paths or mocks only.
- Green unit tests are NOT evidence the live path works. Say so plainly.
- Watch for vacuous tests: assert against real mutations at the production
  call site, not on pure helpers whose signature proves they never see the data.

## Open (non-blocking) — `gert-private` working tree

✅ **The design work SURVIVED the machine loss.** Commit `3bef9e2 protect` is
pushed to `origin/main` and captured everything before the old box died: the
`design/gert/` schema/LaTeX/EBNF changes, `tv-enum.yaml` + `tv-pkg-resolve.yaml`
conformance corpora, `analysis/` (including `tracked-modifications.patch` files
holding uncommitted work from the sibling `gert` and `gert-vscode` repos), and
the GCP grammar additions.

Consequence: `3bef9e2` is a **snapshot, not a curated commit.** Real spec work
is mixed with scratch output and rejected patches. It still needs a triage pass
to separate normative artifacts from debris — Cristián declined that pass on
2026-08-19. The `analysis/*/tracked-modifications.patch` files are the recovery
path for any sibling-repo work that was in-flight when the machine failed.

The current dirty working tree on this machine is a **Squad upgrade**
(`.github/skills/`, `.mcp.json`, ~40 modified templates, Rai + Fact Checker
agents, 7 new workflows) — unrelated to gert design work, also untriaged.

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
