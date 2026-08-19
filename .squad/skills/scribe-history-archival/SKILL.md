# Scribe — History Archival Binding Rules

**Purpose:** Document the correct procedure for archiving agent history files when they exceed size thresholds.

## Archival vs. Summarization

These are NOT the same operation:

- **Summarization** (live history.md): Create a condensed version that captures key points. Target <15,360 bytes.
- **Archival** (to history-archive.md): Preserve the **full, complete, verbatim text** of the original. APPEND-ONLY.

## Correct Procedure

1. Create a summary version (~1,500-3,000 bytes)
2. APPEND the full original file verbatim to history-archive.md with a generation header
3. Replace history.md with ONLY the summary
4. CRITICAL: Verify byte reconciliation before committing

## Byte Reconciliation (CRITICAL — BINDING RULE)

Before committing archival changes:

- Calculate: removed_bytes = history.md_before_size - history.md_after_size
- Calculate: added_bytes = history_archive.md_after_size - history_archive.md_before_size
- Verify: removed_bytes ≈ added_bytes (within ~2KB for headers)

Example:
- history before: 24,770 bytes
- history after: 1,550 bytes
- removed: 23,220 bytes
- archive before: 30,792 bytes
- archive after: 49,850 bytes
- added: 19,058 bytes
- gap: 4,162 bytes (acceptable with headers/formatting)

If gap > 5% of removed bytes: **STOP and investigate. Missing bytes = missing content.**

Recover lost content via git show HEAD^:path/to/file and append to archive as a new generation.

## Generation Headers

Each archival adds a new generation section to history-archive.md:

 ---

## Generation N — Brief Title (ISO-8601-timestamp)

**Source:** Full content from [commit/reference], archived from [task] task.

---

Do NOT overwrite or renumber existing generations.

## Binding Rules

- Archival must be byte-neutral (verified before commit)
- Always append, never overwrite archive
- Generate numbers only increase (1, 2, 3, ...)
- Include commit hash and timestamp in generation header
- Preserve key learnings that agents will need on next spawn
- If content seems lost, fail the task and recover before committing

## Common Mistakes

❌ Delete overflow without appending to archive first
✅ Always append to archive BEFORE removing from history.md

❌ Append only lines 23+ instead of the full original file
✅ Append entire pre-summarization file verbatim

❌ Skip byte reconciliation check
✅ Compute and verify reconciliation every time

❌ Overwrite or renumber existing generations
✅ Use new generation numbers only (append mode)

## See Also

- commit c868c5a — Prior history-archive repair example
- .squad/identity/wisdom.md — Failure history on data integrity defects
