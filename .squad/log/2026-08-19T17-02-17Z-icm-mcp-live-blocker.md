# Session: ICM MCP Live Blocker

**Timestamp:** 2026-08-19T17:02:17Z

## Executive

Ken identified two cleanly-separated problems:

1. **Provider-side blocker (current):** VS Code cannot start `icm-mcp` (401 from `icm-mcp-prod.azure-api.net`). Full recovery procedure documented in decisions.md.

2. **Gert-side blocker (fixed):** The `/run` handler was completely missing from extension.ts after the Petals port. Fixed; tests pass 186/186.

## Status

Gert's code path is now ready for the first live test once the provider recovers.

