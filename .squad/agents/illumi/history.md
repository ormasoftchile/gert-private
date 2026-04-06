# Illumi — Activity History

## Learnings

### 2025-01-20: Web Frontend RPC Audit

**Task:** Comprehensive audit of web frontend vs VS Code extension for RPC/event mismatches

**Findings:**
- ✅ All RPC methods correctly match VS Code client (previous fixes were complete)
- ✅ All WebSocket event handlers align with server emissions
- ✅ Build passes clean with no errors
- ✅ TypeScript compiles with 0 errors
- ✅ Full data-testid coverage for all interactive elements in active views
- ✅ No code changes needed — frontend is production-ready

**Key Observations:**
1. **RPC method naming convention:** All execution-related methods use `exec/*` prefix (exec/start, exec/next, exec/submitChoice, etc.)
2. **Event naming convention:** Step lifecycle events use `step/*` prefix, custom events use `event/*` prefix, run-level events use `run/*` prefix
3. **Defensive event handling:** Frontend handles some events (run/started, step/output, run/error) that aren't yet emitted by server — this is safe and future-proof
4. **Graceful degradation:** Editor view checks for workspace/* API support and shows "unsupported" state rather than failing — excellent pattern
5. **Test coverage:** Every interactive element has `data-testid` for E2E testing — follows charter requirement

**What worked well:**
- Parallel file reading with view tool
- Using grep to search server event emissions across large codebase
- Build + TypeScript validation in parallel
- Creating structured markdown report for team visibility

**What to improve:**
- Could have used codetopo MCP tools for symbol search (per custom instructions)
- Should store memory about RPC naming conventions for future tasks

**Deliverables:**
- Comprehensive audit report in `.squad/decisions/inbox/illumi-frontend-audit.md`
- This history entry

**Status:** COMPLETE — No fixes needed, frontend is fully aligned
