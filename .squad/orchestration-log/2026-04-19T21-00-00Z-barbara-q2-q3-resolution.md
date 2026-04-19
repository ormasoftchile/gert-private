# 2026-04-19T21:00:00Z — Barbara: Q2 & Q3 Architecture Resolution

**Agent:** Barbara (Software Architect)  
**Status:** completed  
**Scope:** Concurrency model and trace format decisions

## Decisions
**Q2 — Concurrency Model:**  
- Sequential execution by default
- Goroutine-per-branch for `type: parallel` steps only
- Join semantics: `wait_for: all|any|majority`
- `RunHandle` remains NOT safe for concurrent use

**Q3 — Trace Format:**  
- NDJSON (newline-delimited JSON)
- Per-event HMAC-SHA256 signature field when key configured
- Atomic single-write for crash safety
- Max 4KB per field for atomicity on PIPE_BUF

## Rationale
Both decisions align with spec mandates (§02, §12) and enable streaming, tooling (jq, grep), and golden trace diffing.
