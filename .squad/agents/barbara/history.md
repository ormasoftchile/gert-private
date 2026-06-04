# Barbara — Project History

## Learnings

- **2026-06-03:** The GERT runtime's governance enforcement happens INSIDE the Go binary, before subprocess fork. This means any hosting option that runs the unmodified binary preserves governance automatically — the web platform cannot bypass it unless it circumvents the JSON-RPC contract entirely.
- **2026-06-03:** GERT has four adapter surfaces: JSON-RPC exec contract (`exec/v1`), WebSocket event stream, tool invocation contract, and extension handshake. The web platform is just another adapter — it must consume these contracts, not invent new execution paths.
- **2026-06-03:** The event system is one-way (runtime → subscribers) and the trace file is the authoritative record (at-least-once to file, best-effort to subscribers). This means the web layer can tolerate event streaming failures as long as the trace file is persisted reliably.
- **2026-06-03:** Key architectural decision: WHERE the Go binary runs determines isolation, blast radius, and governance preservation. Produced 5-option architecture menu. Recommended Thin Relay (Container Apps Jobs) for MVP with Sidecar Agent as enterprise endgame.
- **2026-06-03:** Approval gates mid-execution are the hardest seam for the web platform. Jobs-based topologies need state serialization + resume; pool/sidecar topologies handle them naturally but at higher cost/complexity.
- **2026-06-04:** Brainstorm output merged to decisions.md. Architectural recommendation (Thin Relay MVP → Sidecar endgame) recorded. Four related agent outputs (John, Don, David) synchronized cross-agent. Orchestration log created.
- **2026-06-03:** Go-only constraint removed. User open to C# native runtime. Re-evaluated all 5 original topologies + 4 new C# options. Key insight: C# eliminates the binary lifecycle problem AND makes approval gates trivial (async/await vs checkpoint/resume). New MVP recommendation: B1 (App Service + BackgroundService) — zero cold start, trivial approval gates, single deployment, $55/mo. New enterprise endgame: B4 (App Service Per-Tenant) — same isolation as A4 Sidecar but without container orchestration complexity. Retired A2 (superseded by B2 Functions Isolated). Critical constraint: C# Runtime must be `internal sealed` to prevent governance bypass in shared-process topology. Don must validate governance parity. Decision output merged to decisions.md (2026-06-03T20:36:12). Orchestration log created.

---

## Project Context

GERT is a governed, executable, traceable runbook engine (Go binary, local-first). It has a VSCode extension, TUI runner, and mobile SDKs. The team focus is building a web execution platform on Azure — white-labeled customer portals where end-users run runbooks as part of business processes (applications, authorizations, compliance flows). Azure infra only (no k8s). Key concerns: queue-driven execution, webhook delivery, tenant isolation, audit traceability.

**User:** ormasoftchile
**Session start:** 2026-06-03
