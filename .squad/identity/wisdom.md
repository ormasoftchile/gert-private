---
last_updated: 2026-06-05T18:13:09-07:00
---

# Team Wisdom

Reusable patterns and heuristics learned through work. NOT transcripts — each entry is a distilled, actionable insight.

## Patterns

**Repository scope is a pre-planning gate.** When scoping work for a repository that already contains language-specific source files (e.g., `main.go`, `go.mod`), do NOT assume that's where new work of that language should go. Instead:
1. Verify the repo's stated scope in project context (README, charter, audits)
2. Check for sibling repositories that might be the actual home for runtime/implementation work
3. Confirm with stakeholders before planning multi-day implementation into that repo

**Failure example:** Coordinator misplaced 4 days of Phase 2 Go runtime work into gert-private, which is design-only. The signals were all there: original audit said no runtime, project structure showed separate `gert` repo for actual runtime, and `cmd/gert/` was an empty stub. Scope verification before planning would have caught this. (See orchestration-log `2026-06-05T181309Z-coordinator-scope-correction.md` and session log `2026-06-05-design-only-scope-correction.md` for details.)
