### 2026-04-19: minted adopted for syntax highlighting
**By:** the user (via Leslie)
**What:** minted package (Pygments-based) for YAML/Go/JSON syntax highlighting in code blocks. Requires --shell-escape on pdflatex and Python 3 + Pygments on build machine.
**Style:** friendly, frame=leftline, no line numbers, small font
**Languages configured:** yaml, go, json
**Why:** Better readability for YAML runbook examples — proper colour-coded output vs plain verbatim
**Migration:** blocks converted to minted on-demand, not all at once
