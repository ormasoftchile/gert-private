# Orchestration Entry: Killua

### 2026-03-18 — Tool catalog API (tools/list, tools/detail) + schema bundle endpoint + dry-run endpoint in Go serve

| Field | Value |
|-------|-------|
| **Agent routed** | Killua (Backend Engineer) |
| **Why chosen** | Backend Go RPC owner — tool catalog API, schema bundle, and dry-run are new JSON-RPC methods in the serve layer |
| **Mode** | `background` |
| **Why this mode** | Backend-only changes; all endpoints are additive and stateless with no frontend dependencies |
| **Files authorized to read** | `ext/serve/pkg/serve/serve.go`, `pkg/tools/*.go`, `pkg/schema/*.go`, `pkg/engine/*.go`, `schemas/*.json` |
| **File(s) agent must produce** | `ext/serve/pkg/serve/serve.go` (tools/list, tools/detail, schema/bundle, exec/dryRun endpoints) |
| **Outcome** | Pending |
