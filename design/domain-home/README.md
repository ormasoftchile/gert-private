# Domain Kit Manual

This folder contains the LaTeX source for this domain kit's manual.

## Quick start

```sh
# One-time setup (copy MastersThesis.cls and substitute placeholders):
make init DOMAIN_NAME="My Kit" DOMAIN_SLUG=my-kit

# Build the PDF:
make build

# Continuous preview (requires latexmk):
make watch
```

## Requirements

- Python 3 and a LaTeX distribution (TeX Live / MiKTeX) with `latexmk` and `minted`
- The `gert` repo checked out as a sibling directory (provides `MastersThesis.cls` and `scripts/latex.py`)

## Structure

| File / Folder | Purpose |
|---|---|
| `main.tex` | Document root — preamble and section includes |
| `Makefile` | Build targets |
| `MastersThesis.cls` | Document class (copied from gert repo by `make init`) |
| `sections/` | One `.tex` file per chapter |
| `build/` | Generated artefacts (git-ignored) |
