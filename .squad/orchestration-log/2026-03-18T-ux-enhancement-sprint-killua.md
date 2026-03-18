# Orchestration Entry: Killua

### 2026-03-18 — Per-step retry with backoff + parallel iterate (concurrency) + error branches (_error condition)

| Field | Value |
|-------|-------|
| **Agent routed** | Killua (Backend Engineer) |
| **Why chosen** | Backend engineer owns engine execution semantics — retry logic, iterate concurrency, and error branch evaluation are engine-layer changes |
| **Mode** | `background` |
| **Why this mode** | Independent of frontend work; operates on Go engine internals and schema definitions |
| **Files authorized to read** | `pkg/engine/engine.go`, `pkg/engine/types.go`, `pkg/schema/schema.go`, `pkg/schema/validate.go`, `ext/serve/pkg/serve/serve.go` |
| **File(s) agent must produce** | `pkg/engine/engine.go` (retry loop, parallel iterate, error branch routing), `pkg/schema/schema.go` (retry + concurrency fields), `pkg/schema/validate.go` (new field validation) |
| **Outcome** | Pending |
