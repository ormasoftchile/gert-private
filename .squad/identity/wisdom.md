---
last_updated: 2026-06-07T06:30:40-07:00
---

# Team Wisdom

Reusable patterns and heuristics learned through work. NOT transcripts — each entry is a distilled, actionable insight.

## Patterns

**Repository scope is a pre-planning gate.** When scoping work for a repository that already contains language-specific source files (e.g., `main.go`, `go.mod`), do NOT assume that's where new work of that language should go. Instead:
1. Verify the repo's stated scope in project context (README, charter, audits)
2. Check for sibling repositories that might be the actual home for runtime/implementation work
3. Confirm with stakeholders before planning multi-day implementation into that repo

**Failure example:** Coordinator misplaced 4 days of Phase 2 Go runtime work into gert-private, which is design-only. The signals were all there: original audit said no runtime, project structure showed separate `gert` repo for actual runtime, and `cmd/gert/` was an empty stub. Scope verification before planning would have caught this. (See orchestration-log `2026-06-05T181309Z-coordinator-scope-correction.md` and session log `2026-06-05-design-only-scope-correction.md` for details.)

**Filenames must be Windows-NTFS-safe.** When writing log, orchestration-log, or any other file with a timestamp in its name, NEVER include `:` in the path component — Windows NTFS forbids colons in filenames and any pull or checkout on a Windows clone will fail with `error: invalid path`. Use this exact format for ISO-8601 timestamps in filenames:

- ✅ `2026-06-06T02-31-50Z-phase-1-closed.md` (colons replaced with hyphens)
- ❌ `2026-06-06T02:31:50Z-phase-1-closed.md` (raw ISO with colons — breaks on Windows)

Also avoid: `<`, `>`, `"`, `|`, `?`, `*`, and trailing `.` or space. Stick to `[A-Za-z0-9._-]` plus `/` as the path separator and you're safe on every platform.

**Failure example:** Scribe wrote three log files with raw `T02:31:50Z` timestamps on 2026-06-06. The subsequent `git pull` on Brady's Windows machine errored with `invalid path` for each, blocking the entire fast-forward. Recovery required a plumbing-level index rebuild (`git update-index --cacheinfo`) to rename the paths without checking out the offending content. This is the second time this category of bug has hit the team (first time was an earlier session's `:` in a different log path). Add `-` substitution at filename generation time — not as a post-hoc fix.
