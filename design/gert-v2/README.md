# Gert v2 Design Docs

This folder contains the LaTeX design document scaffold for Gert v2.

## Prerequisites

- `make`
- Python 3
- `tectonic` in `PATH`

Tectonic install instructions:

- https://tectonic-typesetting.github.io/

## Build

```sh
make check-tools
make build
```

Output PDF is generated in `build/`.

By default, the build helper auto-selects the first available engine in this order:

1. `tectonic`
2. `latexmk`
3. `pdflatex`

You can force an engine explicitly:

```sh
make build ENGINE=tectonic
make build ENGINE=latexmk
make build ENGINE=pdflatex
```

If your environment does not expose `python3`, override the Python command:

```sh
make check-tools PYTHON=python
make build PYTHON=python
```

## Live Preview (Recommended for Editing)

### Option 1: VS Code LaTeX Workshop (Easiest)

1. Install [LaTeX Workshop](https://marketplace.visualstudio.com/items?itemName=James-Yu.latex-workshop) extension in VS Code
2. Open `main.tex` in VS Code
3. LaTeX Workshop auto-previews on every save
4. PDF viewer opens in VS Code side panel or external viewer

### Option 2: Terminal Watch Mode

Run continuous preview in a dedicated terminal:

```sh
make watch
```

This uses `latexmk -pvc` to rebuild and auto-open the PDF every time you save.

Requires `latexmk` to be installed.


## Release

Create a distributable package of the design document:

```sh
make release
```

Outputs a timestamped zip archive (e.g., `gert-v2-design-20260418-123456.zip`) containing:

- Compiled PDF
- Source files (main.tex + all sections)
- Build instructions
- Metadata manifest

## Clean

```sh
make clean
```

## Notes on Windows

This build flow is designed to work on Windows as long as:

- `make` is available (for example via MSYS2, Git Bash, or similar)
- Python 3 is available in `PATH`
- `tectonic` is installed and available in `PATH`

The Makefile delegates all file operations to `scripts/latex.py`, which avoids shell-specific path and remove commands.

On Windows, this commonly works:

```powershell
make check-tools PYTHON="py -3"
make build PYTHON="py -3"
```

## Runbook Previsualization Sandbox

This design folder now includes a React Flow sandbox for speculative runbook visual design:

- Location: `previsualization/`
- Goal: quickly test visual stereotypes and layout primitives before runtime implementation

Run locally:

```sh
cd previsualization
npm install
npm run dev
```

Build for static preview:

```sh
cd previsualization
npm run build
npm run preview
```
