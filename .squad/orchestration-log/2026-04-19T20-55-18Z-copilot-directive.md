# 2026-04-19T20:55:18Z — Copilot Directive: No v1 Compatibility

**Agent:** Copilot (Directive Captured)  
**Status:** completed  
**Scope:** gert v2 architecture decision

## Directive
User declared: **No v1 compatibility at all.** v1 compat mode is explicitly out of scope for gert v2.

## Impact
- v2 design space freed from backward-compatibility constraints
- API, trace format, config schema can diverge from v1 entirely
- Simplifies implementation and spec
