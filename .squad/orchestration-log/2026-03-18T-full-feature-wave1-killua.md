# Orchestration Entry: Killua

### 2026-03-18 — Tool catalog API + dry-run endpoint + schema-driven forms backend

| Field | Value |
|-------|-------|
| **Agent routed** | Killua (Backend Engineer) |
| **Why chosen** | Backend Go RPC owner — all three tasks are new JSON-RPC methods in serve.go |
| **Mode** | `background` |
| **Why this mode** | Backend-only changes with no frontend dependencies; 3 new RPC methods are additive |
| **Files authorized to read** | `ext/serve/pkg/serve/serve.go`, `pkg/tools/*.go`, `pkg/schema/*.go`, `pkg/engine/*.go` |
| **File(s) agent must produce** | `ext/serve/pkg/serve/serve.go` (3 new RPC methods: tool catalog, dry-run, schema-driven forms) |
| **Outcome** | Pending |
