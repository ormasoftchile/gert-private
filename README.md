# gert-private

Private companion repo to [`ormasoftchile/gert`](https://github.com/ormasoftchile/gert).

Contains:

- `design/` — LaTeX sources for the gert detailed design document.
  Build with `make -C design/gert build` (requires a TeX Live
  distribution + `latexminted` + Pygments).
- `deliverables/` — Consumer-facing tool binding artifacts for each
  phase of the SQL Live-Site Operations integration.
- `scripts/` — Repository maintenance scripts.

## Deliverable parity check

`deliverables/phase2/*.vscode-mcp.tool.yaml` and the corresponding test
fixtures in `gert/internal/tool/testdata/` must be byte-identical.
`gert`'s testdata is authoritative for `gert`'s own tests; the
deliverable is the consumer-facing artifact. The script proves they
agree.

```sh
node scripts/check-deliverable-parity.js <path-to-gert-checkout>
# or
GERT_CHECKOUT=<path> node scripts/check-deliverable-parity.js
```

The `gert` checkout path must always be supplied — the script exits
non-zero with an actionable message if it is absent or the directory is
not found. It never skips the comparison.

Example (both files match):

```
Comparing 2 deliverable(s):
  deliverables : .../gert-private/deliverables/phase2
  gert testdata: .../gert/internal/tool/testdata

  OK       icm.vscode-mcp.tool.yaml
  OK       tsg-recommendation.vscode-mcp.tool.yaml

Result: 2 matched, 0 drifted
PASS: all deliverables match their gert testdata fixtures.
```
