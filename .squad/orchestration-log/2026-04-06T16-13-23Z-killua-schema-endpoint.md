# Orchestration Log: killua-schema-endpoint

**Timestamp:** 2026-04-06T16:13:23Z  
**Agent:** Killua (Backend Engineer, Go)  
**Task:** Implement schema/runbook RPC endpoint  
**Status:** ✅ Complete

## Dispatch

Killua was tasked to add a `schema/runbook` JSON-RPC endpoint to the serve layer to support web frontend input collection UI. The web app needs to discover user inputs before calling `exec/start`, requiring a contract that returns input schema and runbook metadata without execution.

## Work

1. Designed endpoint contract (`schema/runbook`):
   - HTTP POST `/rpc` and WebSocket transport
   - Request: `{"method": "schema/runbook", "params": {"runbook": "path/to/foo.runbook.yaml", "cwd": "/optional/dir"}}`
   - Response: Returns `inputs` array with type, required, description for each `from: 'user'` input, plus `kind`, `description`, `title` from metadata

2. Implemented in `serve.go`:
   - Integrated with existing JSON-RPC server (same Killua built for `exec/start`)
   - Calls `LoadRunbookFlexible()` to parse runbook YAML
   - Extracts user inputs from `meta.inputs` array filtering on `from: 'user'`
   - Returns structured schema for frontend form generation

3. Error handling:
   - File not found: returns `{"code": -32603, "message": "runbook not found"}`
   - Parse error: returns parse diagnostics
   - Works with cwd parameter for project-relative paths

## Files Changed

- `ext/serve/pkg/serve/serve.go` — Added schema/runbook RPC method

## Commits

- `fc30ff1` — feat: add schema/runbook RPC endpoint for web input preflight

## Status

Endpoint live and testable. Web frontend can now call `schema/runbook` to populate input collection form before `exec/start`.
