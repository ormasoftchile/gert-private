# gert v2 — Implementation Plan

**Approach:** Build in phases. Review after each phase. Replan as needed.

**Architecture decisions locked:**
- Concurrency: sequential steps; goroutine-per-branch for explicit `parallel` blocks only
- Trace format: NDJSON (`trace.jsonl`) with per-event HMAC-SHA256

## Phases

| # | Phase | Delivers |
|---|-------|----------|
| 0 | Foundation | Go module, shared types, all interfaces, JSON Schema artifact |
| 1 | Parser & Schema | v2 parser, structural/semantic validation |
| 2 | Planner | Import resolution, tool discovery, `ExecutionPlan` |
| 3 | Runtime Core & Trace | Engine loop, run stores, event bus, `trace.jsonl` writer |
| 4 | Governance Engine | Allowlist/denylist, env blocking, redaction, RBAC, approval gates |
| 5 | Step Types | All 14 v2 step types |
| 6 | Tool Runtime | stdio/jsonrpc/mcp transports, capability gate |
| 7 | Extension Host | Manifest lifecycle, Ed25519 trust, crash isolation |
| 8 | Input Provider | Provider framework, 4 built-in providers |
| 9 | `gert serve` | exec/v2 RPC, WebSocket events, RBAC, TLS |
| 10 | Adapters | Bubble Tea TUI + VS Code TypeScript client |
| 11 | Evidence & Replay | Snapshots, `--resume`, `gert test`, HMAC chaining |
| 12 | Observability | OTel, Prometheus, structured logs, `gert verify` |
| 13 | Acceptance | Integration corpus, golden traces, perf benchmarks |

## Spec reference

All spec files: `design/gert-v2/spec/` (16 files, agent-readable)
Full design doc: `design/gert-v2/main.pdf` (328 pages)
