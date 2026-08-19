# SKILL: Canonical Decisions Ledger Location

**Domain:** Squad process / Scribe responsibilities  
**Author:** Scribe  
**Date:** 2026-08-19  
**Binding Rule:** The canonical decisions ledger has a fixed, non-negotiable path. Stat it explicitly in every PRE-CHECK.

---

## The Problem

The canonical decisions ledger is `.squad/decisions.md` at the squad root:
```
C:\One\OpenSource\gert-private\.squad\decisions.md  ← CORRECT
```

NOT:
```
C:\One\OpenSource\gert-private\.squad\decisions\decisions.md  ← WRONG
```

The `.squad/decisions/` directory exists **only** to hold `inbox/` subfolder for pending decisions. It is not a ledger storage path.

---

## Why This Matters

When Scribe measures file size for the archiving HARD GATE in Step 1 (PRE-CHECK):
- Measuring `.squad/decisions.md` (canonical, 217KB+) correctly triggers archiving rules
- Measuring `.squad/decisions/decisions.md` (wrong file, 19KB) silently bypasses archiving logic

This error recurred in commits:
- **3e200c2** (2026-08-19T08:54:04Z): "Scribe: Repair b62d9f5 — restore root decisions.md"
- **c6dd227** (2026-08-19T09:24:06Z): "REPAIR — Append probe-token entries to canonical decisions.md"

---

## Binding Rule

**In every Scribe spawn, PRE-CHECK task must stat:**
```powershell
$canonical = "C:\One\OpenSource\gert-private\.squad\decisions.md"
$size = (Get-Item $canonical | Measure-Object -Property Length).Sum
```

**NOT:**
```powershell
# WRONG — do not use this path
Get-Item "C:\One\OpenSource\gert-private\.squad\decisions\decisions.md"
```

**Archiving thresholds (task 1 HARD GATE):**
- If canonical `decisions.md` >= 20,480 bytes: archive entries older than 30 days
- If canonical `decisions.md` >= 51,200 bytes: archive entries older than 7 days

---

## Prevention Checklist

- [ ] PRE-CHECK explicitly stats `.squad/decisions.md` (not `.squad/decisions\decisions.md`)
- [ ] Archiving logic reads from canonical path only
- [ ] Inbox merging appends to canonical path (`.squad/decisions.md`)
- [ ] After merge, delete inbox files and the **empty** `.squad/decisions/` directory is left intact (directory itself is not deleted)
- [ ] Do NOT create or write to `.squad/decisions/decisions.md`

---

## Historical Record

| Commit | Issue | Fix |
|--------|-------|-----|
| 3e200c2 | First recurrence: wrote to wrong file | Restored canonical, merged correctly |
| c6dd227 | Second recurrence: PRE-CHECK measured wrong file | Deleted wrong file, appended to canonical, documented rule |

This skill was created to prevent a third occurrence.
