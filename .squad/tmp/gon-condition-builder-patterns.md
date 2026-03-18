# Decision: Branch condition builder expression patterns

**Date:** 2026-03-18
**By:** Gon (Lead Architect)
**Status:** Proposed

## What
The branch condition builder generates Go template expressions using a fixed set of patterns:
- `{{ contains .var "value" }}` / `{{ not (contains .var "value") }}`
- `{{ eq .var "value" }}` / `{{ ne .var "value" }}`
- `{{ regexMatch "pattern" .var }}`
- `{{ gt .var value }}` / `{{ lt .var value }}`

The parser recognizes these same patterns for round-trip fidelity. Unrecognized patterns fall through to raw expression editing (Advanced mode).

## Why
These patterns cover the most common branch conditions observed in example runbooks (`contains`, `not contains`, equality). The Go template function names (`contains`, `eq`, `ne`, `gt`, `lt`, `regexMatch`) match the Sprig library used by the gert engine. Power users can always toggle to Advanced mode for arbitrary expressions.

## Impact
- Killua: if the engine's template function set changes, the builder patterns should be updated.
- Kurapika: UX may want to add visual indicators when a condition can't be parsed.
- Hisoka: round-trip test coverage should include builder-generated → save → reload → parse cycle.
