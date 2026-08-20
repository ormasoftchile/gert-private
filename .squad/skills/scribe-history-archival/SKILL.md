# Scribe — History Archival Binding Rules

**Purpose:** Document the correct procedure for archiving agent history files when they exceed size thresholds.

## Decisions Ledger Archive Gate — Honest Reporting Rule

The canonical decisions ledger is `.squad/decisions.md`. Its own Archive Policy is authoritative: decisions are archived **by effort completion**, not by file size. Size thresholds are therefore a **WARN-and-escalate signal**, not automatic permission to move content.

When `.squad/decisions.md` is over a size threshold, Scribe must measure the visibility of any date-based scan before reporting the result:

1. Count candidate entry headings (`##` and `###`, excluding structural headings such as Archive Policy / Archive Index).
2. Count how many candidate entries carry a parseable `YYYY-MM-DD` or ISO-8601 date in the heading.
3. Report the dated-entry fraction in the health report, e.g. `12/403 headings (2.98%)`.
4. If the file is over threshold but the dated-entry fraction is low, or the scan matches few/no entries, **do not report a clean/no-op archive result.** Emit an explicit warning such as:

   > WARNING: `.squad/decisions.md` is 250,237 bytes over threshold, but only 12/403 candidate entries carry parseable dates. Date-based archival cannot see most of this file; escalating for effort-completion review.

5. Do **not** archive, move, or delete ledger content based only on file size when the ledger policy says effort-completion controls archival. Wait for an explicit effort-completion ruling (for example from Barbara) and then execute that ruling precisely.

This rule exists because a prior date-heading scan produced a false-clean result: the ledger was far over threshold, but most entries used undated topic headings (`### Approval Gate Precedence Table`, `### Tiered Preflight Design`, etc.), so the scan saw almost none of the data it claimed to evaluate.

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

## Archive File Create vs. Append Path

`history-archive.md` may not exist the first time an agent crosses the summarization gate. The create path is the same append-only operation with an empty prior archive:

1. If `history-archive.md` is missing, create it with **Generation 1** and the full original `history.md` content verbatim after the generation header.
2. If `history-archive.md` exists, scan existing `## Generation N` headers, choose `max(N)+1`, and append the new generation to the end.
3. Never open an existing archive in truncate/overwrite mode. Use append semantics only. If the available file API cannot append, read the existing archive, concatenate `existing + new_generation`, and verify the previous bytes are an unchanged prefix before writing.
4. After either branch, verify the archived payload contains the exact full original `history.md` bytes for that generation before replacing live `history.md` with the summary.
5. A second archival after a first-time create must produce **Generation 2** and preserve Generation 1 byte-for-byte.

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
