### 2026-03-18: Design principle — workflow view is the default for both authoring and execution
**By:** ormasoftchile (via Copilot)
**What:** A tree view adds little value over editing YAML directly. Both the authoring experience (RunbookEditorPanel) and the execution experience (RunbookPanel) should be workflow-graph-first. The tree is just YAML-with-boxes; a workflow graph reflects the actual execution mental model — conditions, branching paths, sequence.
**Why:** User explicitly stated: if someone wants a tree, they can edit the YAML. The visual tool's value proposition is showing the workflow, not the document structure.
