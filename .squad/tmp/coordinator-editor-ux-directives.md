### 2026-03-18: User directive — editor layout flexibility
**By:** Cristián Ormazábal Ortega (via Copilot)
**What:** The editor panel layout (form vs graph, left vs right) should be configurable via a VS Code setting. Users should be able to swap which side has the form and which has the graph. Switching between YAML source and visual workflow should be a toggle in the UI (not a command palette action). A play button to run the runbook should be available from both the YAML view and the visual editor view.
**Why:** Different users have different spatial preferences. The visual editor and YAML source are two views of the same file — switching between them should be frictionless, not a command search.

### 2026-03-18: User directive — tool name validation and registration
**By:** Cristián Ormazábal Ortega (via Copilot)
**What:** Tool names in step definitions should not be free text. There needs to be a way to register which tools are available — via a toolkit reference, a package system, or a project-level tool catalog. The editor should restrict tool selection to registered tools only.
**Why:** Free-text tool names lead to typos and invalid references. A registration system (gert.yaml packages, tool packs, etc.) creates a closed catalog for validation and autocomplete.

### 2026-03-18: User directive — query syntax highlighting
**By:** Cristián Ormazábal Ortega (via Copilot)
**What:** Steps that contain query expressions (KQL, SQL, or similar) should have syntax highlighting in the editor. This applies to both the YAML source view and the visual editor form fields.
**Why:** Query expressions are code. Showing them as plain text makes them harder to read and debug.

### 2026-03-18: User directive — research n8n and Logic Apps features
**By:** Cristián Ormazábal Ortega (via Copilot)
**What:** Research n8n and Azure Logic Apps Designer for features that could enhance gert's visual editor. Identify what they do well that gert could adopt.
**Why:** These are the confirmed design references for the visual editor UX.
