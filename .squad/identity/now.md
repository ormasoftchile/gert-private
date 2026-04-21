---
updated_at: 2026-04-21T05:09:19Z
focus_area: gert v2 implementation — Phase 13 kickoff
active_issues: []
---

# What We're Focused On

Implementing gert v2 phase-by-phase. Phases 0–12 complete and sealed.

**Phase 12 sealed:** `6ab513e` — OpenTelemetry integration (APPROVED 8.5/10 by Ken)

**Phase 13 starting now:** Ken designing scope. Part A must address:
- NBI-12-01: OTLP adapter package (`pkg/otel/adapter`)
- NBI-12-02: Document `--otel-endpoint` as reserved in `--help`
- NBI-12-03: Consider `context.AfterFunc` for `mergeContexts` (low priority)

Team: Ken (architect/reviewer), Brian (Go implementor), Barbara (preflight/integrations), Scribe (logger).
