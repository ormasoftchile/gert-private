# `.squad/decisions/`

This directory holds the **decision inbox** and **archives**. It does NOT hold the canonical ledger.

## Canonical location

The authoritative decision ledger is **`.squad/decisions.md`** — one level up, in `.squad/`.

Every agent reads that file at spawn time. It is the only file that carries decisions into future sessions.

## Directory contents

| Path | Purpose |
|---|---|
| `.squad/decisions.md` | **CANONICAL LEDGER.** Agents read this at spawn. Scribe merges into it. |
| `.squad/decisions/inbox/` | Drop-box. Agents write individual decision files here; Scribe merges them into the canonical ledger, then deletes them. |
| `.squad/decisions/archive/` | Retired entries, preserved for reference. Never read at spawn. |

## Why this file exists

On 2026-08-17 a stale `decisions.md` was found sitting **inside this directory**, alongside `inbox/`. It was a legacy artifact from earlier efforts (enum parity, domain-kit, GXL) and had diverged from the canonical ledger.

Scribe merged a ruling into that stale file instead of the canonical one. The ruling never reached the file agents actually read. The failure was silent — the commit looked correct, the health report cited plausible byte counts, and nothing broke until the next spawn would have read a ledger missing the decision.

The stale file was archived as `archive/legacy-decisions-pre-2026-08.md` and this README was added so the ambiguity cannot recur.

## Rule

**Never create a `decisions.md` inside this directory.** If you are writing a decision, write it to `inbox/`. If you are merging decisions, merge into `../decisions.md`.
