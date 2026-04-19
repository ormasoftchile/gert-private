### 2026-04-19: All code blocks converted to minted
**By:** the user (via Leslie)
**What:** All YAML/Go/JSON verbatim and lstlisting environments across all sections converted to minted. minted is now the sole mechanism for code blocks in this document.
**Count:** 260 blocks converted across 13 files (yaml: 96, json: 85, go: 24, bash: 13, text: 42). 42 blocks intentionally left as verbatim (ASCII trees, HTTP headers, CLI terminal output, error messages, file path trees).
**Diagrams added:**
- `fig:governance-enforcement-points` — §07 Security, 4-stage pipeline showing governance checkpoints
- `fig:event-flow-timeline` — §06 Runtime Events, vertical timeline of a two-step run with governance approval
- `fig:provider-resolution-flow` — §14 Input Provider Framework, 6-stage resolution pipeline for `from:` bindings
**Notes:**
- Removed `\usepackage{listings}` from main.tex (conflicted with minted v3's tocbasic lol extension)
- Added `scripts/pypath/python3` wrapper routing latexminted to Python 3.13 (Python 3.14 breaks latexminted 0.5.0 argparse API)
- Global TikZ styles updated: `text centered` → `align=center` to support multiline `\\` in node labels
- Added `scripts/convert_to_minted.py` as a reusable conversion tool
