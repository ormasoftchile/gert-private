# gert-private

Private companion repo to [`ormasoftchile/gert`](https://github.com/ormasoftchile/gert).

Contains:

- `design/` — LaTeX sources for the gert detailed design document and the
  domain kit guide. Build with `make -C design/gert build` (requires a
  TeX Live distribution + `latexminted` + Pygments).
- `.squad/` — squad agent system data (decisions, agents, logs, skills,
  templates) and the four squad GitHub Actions workflows under
  `.github/workflows/squad-*.yml`.

History was extracted from `gert` with `git filter-repo` on
2026-05-09 and removed from the public repo on the same day. The original
history snapshot is preserved as the `backup/pre-extract-design-squad`
tag and branch on `gert`.
