### 2026-04-19: type:extension is not a step type

**By:** John  
**What:** `type: extension` does not exist in the gert v2 schema. Extension is a field-annotation
convention (`x-<namespace>:` prefix) only — it annotates runbook or step objects with metadata.
It is not a step type that can appear in the `flow:` array.

- **Outbound notify/send** steps (Slack, PagerDuty, email, external API calls) → `type: tool`
  with the tool declared in `toolRefs`.
- **Inbound event-receive** steps (wait for webhook, wait for SIEM alert, wait for callback from
  external system) → GAP-3: `type: wait_for_event` is not yet specified. Use a `type: cli` stub
  with a `# GAP: no wait_for_event step type yet` comment as placeholder.

**Why:** Correcting testdata YAML (r01-k8s-incident, r05-security-breach) that used
`type: extension` as a catch-all for steps that didn't fit other types. Prevents future confusion
during implementation and ensures testdata accurately reflects the schema's real capabilities
and gaps.
