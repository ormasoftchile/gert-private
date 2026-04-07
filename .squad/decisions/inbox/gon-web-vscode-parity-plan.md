# Decision: Web/VS Code Panel Parity Implementation Plan
**By:** Gon (Lead Architect)  
**Date:** 2025-07-24  
**Context:** Audit requested by ormasoftchile — 3 panels where web app diverges from VS Code extension (the reference truth).

---

## Summary

After a full code audit comparing `web/src/views/runbookRunner.ts` against the VS Code panel sources (`html.ts`, `prose.ts`, `renderHelpers.ts`, `graphRenderer.ts`, `html.css.ts`), the following decisions were made:

---

## Decision 1: Adopt `workflow-header` div pattern across all three panels

**What:** Replace `class="panel-header"` divs in the web with `class="workflow-header"` matching the VS Code pattern. This unifies panel chrome rendering.

**Why:** VS Code's `workflow-header` serves triple duty: panel title + kind badge + action buttons (Restart, Back, etc). Adopting it in the web makes future parity patches mechanical.

**Applies to:**
- Prose panel header (`runbookRunner.ts:492-496` → adopt `html.ts:234`)
- Workflow map header (`runbookRunner.ts:505-508` → adopt `html.ts:239`)
- Active step panel header (currently missing → adopt `html.ts:252-260`)

---

## Decision 2: Port `runbookKind` from exec/start response

**What:** Add `runbookKind: string` to `RunState` and populate it from the `exec/start` server response. Use it to render kind badges (`.badge.mitigation`, `.badge.guide`, etc.) in prose and map headers.

**Why:** The web currently hardcodes "Runbook Instructions" with no contextual type information. VS Code shows the runbook kind badge prominently. This is visible and meaningful to operators.

**Risk:** Low. The `kind` field is returned by the server's `exec/start`; if absent, default to `''` (no badge rendered).

---

## Decision 3: Outcome panel uses VS Code structured layout, not emoji-icon layout

**What:** Replace the web's `outcome-banner + emoji` pattern with VS Code's `outcome-detail + type-badge outcome` structure. Use `outcomeIsConclusion()` to choose between `RESULT` and `RECOMMENDATION` badge text.

**Why:** The emoji-based layout (`✅ RESOLVED`) was a quick initial port. The VS Code layout communicates more clearly and is consistent with step headers.

**Impact:** Small rework of the `outcomeResult` branch in `renderActiveStepContent()` — strictly additive/replacement, no schema changes.

---

## Decision 4: Notes section is a first-class addition, not a stub

**What:** The Notes section (`renderStepNotesSection`) should be implemented properly in the web — with annotation state in `RunState`, tag chip selection, textarea, and local save. Server persistence is optional and can be wired later.

**Why:** The user explicitly called out the missing Notes section. The VS Code implementation is self-contained HTML with no VS Code APIs — it can be ported directly. Starting with local-only state is sufficient for MVP parity.

**Deferred:** Server-side annotation persistence (POST to `/rpc`) — add as a follow-up once the UI is in place.

---

## Decision 5: Chain breadcrumb and parent minimap are deferred

**What:** The chain breadcrumb (P1-B) and parent minimap (P2-G) are VS Code features tied to multi-TSG chain execution, which the web runner does not yet support.

**Why:** Porting these without the underlying chain execution state would produce dead/incorrect UI. Defer until the web runner supports `chainHistory`.

---

## Decision 6: Graph node when-condition subtitles require renderGraph.ts fix

**What:** The web's `renderGraph.ts` shows `node.stepType || 'step'` as node subtitle. VS Code's `stepNodeRenderer.ts` shows the branch condition (when expression) or resolved tool action as subtitle. This needs a fix in `web/src/shared/renderGraph.ts`, NOT just in `runbookRunner.ts`.

**Why:** The issue is in the shared rendering layer. Fixing it there benefits any future consumers of `renderExecutionGraphSvg`.

**Caution:** Verify that `treeToWorkflow` (or `treeToGraph`) passes the `when` condition through to the node object before fixing the renderer. If it doesn't, both files need changes.

---

## Learnings from audit

1. **The web and VS Code share `helpers.ts` logic** (`outcomeIsConclusion`, `outcomeHeadline`) — web already imports from `../shared/helpers`. Good parity foundation.
2. **The web's `syncProseActiveStep` is correct** — blue border CSS exists and DOM sync works. The gap is the missing `prose-arrow-marker` span elements and `indicator-arrow` CSS classes not being added to sections.
3. **Web `renderStepIO` is nearly identical to VS Code** — minor styling differences only.
4. **The web `inputs-summary` structure matches VS Code** — only cosmetic differences (uppercase label).
5. **Annotation/Notes is the largest gap** — requires new state, new render function, and event wiring. Estimate: 3–4 hours of focused work.
