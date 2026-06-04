# David — Project History

## Learnings

**2026-06-03 — Webhook Delivery Patterns Review**
- Analyzed 4 dimensions of event delivery (webhook models, subscription types, payload designs, security)
- Key insight: retry + exponential backoff is the sweet spot for v1 — simple, survives transient failures, unblocks shipping
- Defensive design: always plan for "customer endpoint is down" before writing code; DLQ is non-negotiable in v1
- Per-execution webhooks scale well initially; tenant-level webhooks reduce blast radius for v1.1; Service Bus moves responsibility to Azure in v2
- HMAC-SHA256 + HTTPS is table-stakes; mTLS and Managed Identity defer to v2
- Summary events (not full stream) in v1 keeps payload small and integrations simple; full audit trace available on-demand via polling endpoint
- Baseline: 5 retries over 24h with exponential backoff; customer admin can replay from DLQ indefinitely

**2026-06-04:** Brainstorm output merged to decisions.md. Webhook & event delivery baseline documented with v1–v2 roadmap. Rollout plan recorded. Security checklist included. Orchestration log created.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03
