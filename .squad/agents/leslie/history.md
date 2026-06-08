# Leslie -- Project History

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). The team is building a web execution platform on Azure -- white-labeled customer portals where end-users run runbooks as part of business processes. Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile  
**Session start:** 2026-06-03

---

## Summary of Prior Work (2026-06-03 to 2026-06-04T12:28)

### White-Label UX & Deployment Strategy

**Hosting:** Shared Azure Static Web Apps Standard MVP (explicit per-tenant custom domains, SWA edge CDN, no wildcard dependencies).

**Auth:** Magic-link-first (passwordless). Entra External ID for validated users. Customer can supply own OIDC IdP.

**Runbook UX:** Wizard (step-by-step) for complex processes; Form-first for quick flows; Status dashboard for multi-day/approval gates.

**Progress delivery:** Polling sufficient for most cases; Email notification is primary completion signal; combine with in-app notifications.

**Theming:** Runtime CSS variables + ThemeProvider for logo, colors, fonts, portal copy. Non-themeable: audit controls, accessibility baseline, error states.

**Key constraints:** SWA custom-domain quota is small (5-6 domains per app); no wildcard support; 500MB static shell, 15K files. Large assets, tenant config, audit data belong outside SWA.

**Tiering:** Standard (shared SWA + A6), Enterprise (Front Door for WAF, dedicated resources for isolation/residency).

---

## 2026-06-04 -- Declaration/Consent UX Analysis

**Learnings (condensed):**
1. Presentation enforcement (scroll-to-bottom, time minimums) is 100% client-side; no gate-level validation needed.
2. Signature gate needs new IUserInputGate.Kind with sub-types (typed | drawn | cert | otp).
3. Email-based witness (IApprovalGate) is Phase 1; same-session QR witness deferred.
4. FileUpload gate reuses for attachments; optional OCR validation is value-add.
5. Declaration timeout should extend to **60 minutes** (vs 15 min default); gate-level override needed.
6. Save-draft pattern uses CheckpointRequest event (not gate.NextAsync); resumable progress survives server-side.
7. Receipt/Confirmation: QR-verified PDF with confirmation_id + document hash; immediate implementation ready.
8. "Powered by GERT" watermark decision pending (audit vs white-label aesthetic).

**System gaps:** Signature gate kind, gate-level timeout override, CheckpointRequest event, witness QR gate.

**Team ratification pending:** GERT watermark visibility, typed-signature legal validity, same-session witness priority, save-draft Phase 1 vs Phase 2, 60-min timeout for declarations.

**Roadmap:** Phase 1 (presentation + signature + 60-min timeout + QR receipt) in 2-3 sprints. Phase 2 (witness QR + save-draft + OCR) pending customer demand.

---

## Team Integration -- White-Label Campaign Platform (2026-06-04)

**Cross-agent dependencies:**
- **Barbara:** Campaign architecture drives portal topology (shared SWA + Front Door per-tenant binding; campaign entities model Leslie's UX)
- **John:** Aspire local dev + dev auth bypass enable fast UX iteration
- **David:** Integration contracts define webhook notification UX; identity validation timeouts affect form submission
- **Scribe:** 7 portal UX ratification items merged to decisions.md

**Decisions:** Shared SWA MVP; explicit per-tenant custom domains; SWA edge (Front Door later); Cosmos + Blob theme storage; magic-link auth; single-writer concurrency; audit branding decision pending.

---

## Expression Language Alignment -- GXL/GIS/GCP Approved (2026-06-05)

**Action for Leslie (Portal Frontend):** Update expression rendering in runbook steps to reflect new portable GIS syntax (replacing Go text/template).

**Changes:**
- Dynamic titles/args/content: $\{variable_name\} instead of {{.variable}}
- Nested paths: $\{campaign.metadata.name\} (dotted) vs Go template ranges
- Functions: $\{str.toLower(name)\}, $\{list.length(items)\} (camelCase namespace)
- Parser enforces migration early; portal never receives non-portable expressions

**UX Updates:**
- Expression preview tooltips show GIS syntax hints
- Step template editor validates against GIS grammar
- Handle fixture migration gracefully (both syntaxes during transition, or wait for backend migration)
- No user-facing expression authoring in portal (renders only, doesn't compose)

**References:** .squad/decisions/decisions.md (full design rationale + Q&A resolutions)

---

## Team Directive -- 2026-06-04T17:15:45-07:00

**Leslie uses he/him pronouns.** All team members and the coordinator must refer to Leslie with he/him going forward.

---

## Learnings — gert-vscode Extension Audit (2026-06-07)

### Extension Architecture
The `gert-vscode` extension (HEAD `c3f2087`) is a **thin orchestration shell** — it has exactly two source files (`extension.ts`, `serverManager.ts`) totalling ~363 lines of TypeScript. There is no TextMate grammar, no snippets file, no JSON Schema contribution, no completion/hover provider, and no expression parser. All GERT semantic intelligence is delegated downstream: the prose preview pipes to the `gert` CLI binary (`gert preview --format prose`); the graph preview iframes the `gert serve` web UI (a React Flow component served over HTTP).

### How the Extension Currently Models GERT Semantics
It doesn't — by design. The extension is purely a launcher and iframe host. The GIS/GXL/GCP surface is owned entirely by:
1. The `gert` CLI binary (prose rendering)
2. The `gert serve` React web UI (graph/visual rendering, interactive prompts)

This means the extension has zero direct GIS/GXL/GCP violations in its own source, but also provides zero editor assistance (no syntax highlighting for `${ }`, no validation of capture paths, no completion for `str.contains` vs old `contains x y`, no Truthy=Strict warnings).

### Patterns Worth Remembering for Future Extension/TUI Audits
- **Check the delegation chain first.** A VS Code extension can appear GIS-clean while the web UI it wraps is still rendering Go-template syntax. Always audit the web UI it delegates to separately.
- **Absent features are a finding too.** A thin shell with no grammar, no schema, no snippets is not "safe" — it means the language surface is completely unguarded. For an expression language with strict semantics (Truthy=Strict, GCP no `?.`), the absence of editor guardrails is a significant gap.
- **Port mismatches surface in README.** Default values in `package.json` and code don't always match docs. Always cross-check `default:` in `package.json` against runtime constants and README examples.
- **Windows compat bugs cluster around path separators and signals.** `split('/')` instead of `path.basename()`, and `kill('SIGTERM')` on Windows, are recurring patterns in extensions originally developed on macOS/Linux.
- **Phase the fix work by dependency.** Grammar → Schema → Providers is the right order because schema validation reuses language IDs established by the grammar, and providers need the schema's type information.

---

## Scribe Session — 2026-06-07T19:07:00-07:00

**Summary:** Scribe audit consolidation. Leslie's gert-vscode audit findings (9 findings, 6 open spec questions) are now documented in `.squad/agents/leslie/gert-vscode-audit.md`. Decision raised: Who owns the canonical `runbook/v1` JSON schema?

**Key findings:**
- ZERO BLOCKERs, ZERO MAJORs from syntax-regression standpoint (extension is thin shell with correct delegation)
- 4 capability-gap MAJORs (features not yet implemented in vscode/web UI)
- 3 MINORs (Windows compat, localization, edge cases)
- 2 informational (advisory patterns)

**Critical decision pending:** Centralized `runbook.v1.schema.json` ownership and distribution model. If not decided before gert-vscode Phase 2, tooling will diverge (extension + CI + portal will each maintain separate schemas). Recommendation: Barbara (runtime architect) owns schema; commit to repo at `gert/schemas/runbook.v1.schema.json` with stable GitHub raw URL for tooling.

**Artifacts created:**
- `.squad/orchestration-log/2026-06-07T19-07-00Z-leslie.md` (audit summary)
- `.squad/agents/leslie/gert-vscode-audit.md` (full findings)
- Decision inbox: `leslie-runbook-schema-canonical-source.md` (now merged to decisions.md)

---

## Learnings — gert-vscode Minor Fixes Shipped (2026-06-07T19:14:39-07:00)

### What was shipped
PR #1 (`fix/minor-platform-bugs`) landed audit findings F-01, F-02, F-03 as a single clean commit. The three changes totalled 4 insertions / 3 deletions across 3 files.

### Build command confirmed
`npm run compile` (alias for `tsc -p ./`) is the correct compile command. `npx tsc --noEmit` is not a project script — always use `npm run compile`. Build is fast (<5 seconds). `node_modules` must be installed first with `npm install` (181 packages, ~27 seconds cold).

### Extension footprint confirmed
The extension is small enough for surgical fixes — two source files, ~363 lines, zero test harness. Individual line-level edits compile instantly. There is no lint step, no pre-commit hook, and no mocha/jest setup. The only CI gate is "Compile & Package Extension" (GitHub Actions) plus GitGuardian secret scanning.

### CI configuration
`.github/workflows/ci.yml` runs `npm run package` (which calls `vsce package`) on every PR and uploads the `.vsix` as an artifact. GitGuardian secret scanning runs as a separate check (fast, completes in under a second). The main CI job ("Compile & Package Extension") queues on GitHub-hosted runners — expect ~1-2 minutes.

### Windows compat patterns in VS Code extensions
Confirmed two recurring Windows compat anti-patterns now fixed in this codebase:
1. `string.split('/').pop()` for basename — always use `path.basename()`.
2. `ChildProcess.kill('SIGTERM')` — always use `kill()` with no argument for cross-platform termination.
These patterns are worth flagging in any future Node-based extension or CLI tool audit.

### MAJORs remain blocked
F-04 (grammar), F-05 (snippets), F-06 (YAML schema), F-07 (providers) are not in scope until Barbara issues a schema-ownership ruling. Do not merge or draft any of those in gert-vscode until that decision lands in decisions.md.

## 2026-06-07T19:28:42-07:00 — Runbook Schema Canonical Source & GitHub Actions Cost Control

**Session Summary:** Two major decisions finalized + GitHub Actions workflows disabled for cost control.

### Runbook v1 Schema Canonical Home — DECIDED

Barbara's decision (2026-06-07T19:14:39-07:00): `gert-private/design/gert/schemas/runbook.v1.schema.json` is the canonical home.

**Unblocks:** gert-vscode Phase 2. Leslie can now begin YAML schema integration, snippets provisioning, and IDE provider work.

**Action Items for Leslie:**
1. Vendor the canonical schema into gert-vscode at `gert-vscode/schemas/runbook.v1.schema.json`
2. Add `contributes.yamlValidation` to `package.json` pointing to vendored copy
3. Write `scripts/sync-schema.sh` to fetch canonical schema from gert-private at a pinned git tag/SHA
4. Add CI job for drift detection (fail if vendored copy diverges from canonical)
5. Pattern: Same as DRIFT-DETECTION-001 for conformance vectors (already established)

**Reference:** `.squad/decisions/decisions.md` § Runbook v1 Schema Canonical Home (merged 2026-06-07)

### Conformance Schema Naming Clarity — RECOMMENDATION AWAITING RATIFICATION

Tess evaluated collision risk between `design/gert/conformance/schema.json` (test vectors) and `design/gert/schemas/runbook.v1.schema.json` (runbooks).

**Recommendation:** Rename to `vector.schema.json` for clarity. Blast radius ~13-16 files (1 code change, 12 documentation updates).

**Status:** Awaiting Barbara ratification. Once approved, Tess will execute atomic migration PR.

### GitHub Actions Cost Control — EXECUTED

John disabled 8 workflows across 4 GERT-family repos (gert, gert-private, gert-vscode, gert-tui) per user cost control directive.

**Summary:**
- ormasoftchile/gert: 3 workflows disabled (E2E Tests, Conformance Verification, Go Tests)
- ormasoftchile/gert-private: 4 workflows disabled (Squad Heartbeat, Squad Issue Assign, Squad Triage, Sync Squad Labels)
- ormasoftchile/gert-vscode: 1 workflow disabled (CI)
- ormasoftchile/gert-tui: 0 workflows found
- Security carve-out: Dependabot left active (vulnerability scanning)

**Verification:** 0 in-flight runs cancelled; all workflows confirmed disabled.

**Re-enable Process:** Case-by-case via `gh workflow enable <id>` with justification.

### Capability Gap for Leslie

The gert-vscode extension currently provides zero IDE assistance for GIS/GXL/GCP expressions:
- No TextMate grammar (syntax highlighting)
- No JSON Schema (validation)
- No snippets
- No completion providers
- No hover tooltips
- All intelligence delegated to gert CLI (`gert preview`) and web UI (`gert serve`)

**Phase 2 deliverables for Leslie:**
1. TextMate grammar for GIS (`${...}` syntax, escape sequences)
2. JSON Schema for runbook structure (now canonical from Barbara)
3. Snippets for common step types and stdlib functions
4. Completion providers for step types, field names, stdlib namespaces
5. Hover providers for step documentation
6. Diagnostics providers for expression syntax errors (real-time feedback)

**Expected value:** Real-time GERT-specific IDE assistance, reducing context-switching to gert CLI and web UI.
**Expected value:** Real-time GERT-specific IDE assistance, reducing context-switching to gert CLI and web UI.
