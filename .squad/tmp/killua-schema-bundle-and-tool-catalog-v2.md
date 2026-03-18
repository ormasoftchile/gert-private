# Decision: Schema bundle endpoint and tool catalog array format

**By:** Killua (Backend Engineer)
**Date:** 2026-03-18

## What

1. `schema/bundle` generates all three JSON schemas (runbook, tool, project) from Go structs at runtime rather than reading static JSON files from disk.
2. `tools/list` returns actions as an array `[{name, description, args}]` instead of a map keyed by action name.
3. `exec/dryRun` returns a `steps` array with `{id, type, title, dependencies}` derived from template reference scanning, plus a separate `warnings` array.

## Why

- Generating schemas from Go structs guarantees the schemas always match the Go types — no drift between code and static schema files.
- Array format for actions is more natural for frontend rendering (dropdown lists, ordered iteration) and includes the name inline.
- The `steps` + `dependencies` array in dry-run enables the visual editor to render a step dependency graph without needing to parse the tree structure itself.
- Splitting `warnings` from `errors` lets the UI show non-blocking issues differently from blockers.

## Impact

- Frontend consumers should use `tools/list` with array-format actions (not the previous map format).
- `tools/detail` is now available as an alias for `tools/get` — both work identically.
- `schema/bundle` is stateless and can be called before any execution session.
