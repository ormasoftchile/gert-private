# Decisions Log

Merged from `.squad/decisions/inbox/` by Scribe.

---

## D-001: Editor UX Directives (User)

**Date:** 2026-03-18  
**By:** Cristián Ormazábal Ortega (via Copilot)  
**Status:** Directive — pending implementation  

### Layout Flexibility
The editor panel layout (form vs graph, left vs right) should be configurable via a VS Code setting. Switching between YAML source and visual workflow should be a toggle in the UI (not a command palette action). A play button to run the runbook should be available from both views.

### Tool Name Validation
Tool names in step definitions should not be free text. There needs to be a registration system (gert.yaml packages, tool packs, project-level catalog) to restrict tool selection to registered tools only.

### Query Syntax Highlighting
Steps containing query expressions (KQL, SQL, or similar) should have syntax highlighting in both the YAML source view and the visual editor form fields.

### n8n / Logic Apps Research
Research n8n and Azure Logic Apps Designer for features applicable to gert's visual editor. These are confirmed design references.

---

## D-002: Chain Visualization Model for Invoke Steps

**Date:** 2026-03-18  
**By:** Gon (Lead Architect)  
**Status:** Implemented  

Chain navigation is view-only state (`viewingChainIndex`) that selects which chain entry's tree/states to render in the graph — it never affects execution flow. The graph renderer accepts the selected chain entry's data and renders it identically to the current entry.

**Rationale:** Keeping chain navigation purely in the view layer avoids interfering with the execution engine or client-server protocol. Chain history already captures enough data (tree, stepStates, stepDetails) for full graph rendering.

---

## D-003: Deep-Merge YAML Serialization for Comment Preservation

**Date:** 2026-03-18  
**By:** Gon (Lead Architect)  
**Status:** Implemented  

Replace top-level `doc.set(key, doc.createNode(value))` with recursive `deepMergeNode()` that walks the YAML AST in parallel with the edited JS object, preserving comments and formatting for unchanged values.

**Tradeoffs:** Positional sequence merge is correct for step arrays (add/remove at end) but imperfect if steps are reordered mid-array. Falls back to full stringify with user-visible warning on failure.

---

## D-004: n8n and Logic Apps Feature Research

**Date:** 2026-03-18  
**By:** Gon (Lead Architect)  
**Status:** Research — informs roadmap  

Evaluated 17 features from n8n and Azure Logic Apps Designer. 7 recommended HIGH priority:
1. **Tool catalog panel** — searchable list from `tools/list`, click-to-insert (HIGH)
2. **Expression autocomplete** — template variable autocomplete in all text fields (HIGH)
3. **Per-step retry with backoff** — `retry: {max, interval, backoff}` in step schema (HIGH)
4. **Error branches** — `_error` condition type for post-retry failure routing (HIGH)
5. **Per-step I/O detail panel** — richer trace/input/output display (HIGH)
6. **Credential references in tool defs** — document required env vars (MEDIUM)
7. **Parallel iterate (concurrency)** — concurrent step execution in iterate blocks (MEDIUM)

Key insight: Gert's differentiators (YAML as truth, governance-first, tool/provider abstraction, VS Code-native) mean features need adaptation, not direct transplant.

---

## D-005: Drag-to-Reorder is Same-Parent Only

**Date:** 2026-03-18  
**By:** Kurapika (Frontend Engineer)  
**Status:** Implemented  

Step drag-to-reorder in the editor Flow Map is restricted to siblings within the same parent steps array. Cross-branch and cross-iterate dragging is blocked — only nodes with matching `data-parent-path` are valid drop targets.

**Rationale:** Moving steps between structural containers would require semantic validation. Same-parent reorder is safe, lossless, and covers the most common use case.
