# Ruling: TV-PKG-PATH-002 — workspace-level escape is PKG-W003, not PKG-007

**From:** Barbara (Lead/Architect, spec authority)
**Date:** 2026-08-09
**Requested by:** Cristián Ormazábal Ortega
**Status:** BINDING — applies to Don's final `pkg/pkgpath` implementation pass
**Scope:** `design/gert/sections/06-tool-runtime.tex` §Secure Path Resolution,
`design/gert/sections/03d-parse-time-enforcement.tex` error catalog,
`design/gert/conformance/tv-pkg-resolve.yaml`

## Ambiguity

`TV-PKG-PATH-002` expected `PKG-007` for a project-scope `requires[].path`
parent-directory escape above the workspace root, while §06 prose states that
workspace-level path kinds "are never rejected merely for escaping the
workspace, since they are operator configuration, not package content".

## Ruling — the VECTOR was wrong; the runtime is right

The path-kind/base/containment design is unambiguous and consistent in three
independent places:

1. §06 Table `tab:tool-path-bases` classifies project-scope `requires[].path`
   as **workspace-level**.
2. §06 Resolution-base prose: workspace-level kinds are checked against the
   workspace root **only for the external-root report (`PKG-W003`)** and MAY
   legitimately resolve outside it.
3. §06 Symlinks rule 6 (and architecture ruling AR-TP-6 §6.3 rule 6):
   `requires[].path` and `tool-paths[]` resolving outside the workspace are
   **permitted**, MUST be recorded as `external:sha256:...` in lock and trace,
   and MUST be reported by `gert verify` as `PKG-W003`.

`PKG-007` is the *package-internal* containment code. For workspace-level kinds
it remains reachable only via the syntax rules (backslash separator, UNC,
drive-relative, `~`/env expansion, URI scheme, control chars), never via a
workspace-root escape.

Don's `pkg/pkgpath` (`Class`, `Resolve`, `ResolveKind`) already implements this
correctly: `WorkspaceLevel` returns `(resolved, external=true, nil)` and only
`PackageInternal` returns `PKG-007`. **No runtime change is required, and none
was made.**

## Corrections made in `gert-private`

- `tv-pkg-resolve.yaml` **TV-PKG-PATH-002**: rewritten. External package tree
  added under `../../../outside/acme-incident-tools/`; expected outcome is now
  a successful tier-1 catalog entry plus `warnings: [PKG-W003]`.
- `tv-pkg-resolve.yaml` **TV-PKG-PATH-008**: same defect (identical class,
  `tool-paths[]`), corrected the same way — tier-2 bare-name catalog entry plus
  `warnings: [PKG-W003]`. Left uncorrected it would have re-opened the same
  ambiguity for Don.
- `tv-pkg-resolve.yaml` **TV-PKG-PATH-003**: non-normative note reworded; it
  said containment "is still checked against the workspace root" without saying
  the check is report-only.
- §06 `sec:tool-path-containment`: added an explicit normative paragraph
  (wording below).
- §03d error catalog: `PKG-007` and `PKG-W003` row conditions sharpened.

## Exact normative wording added (§06, Containment rule)

> For **workspace-level** kinds (`requires[].path` at either scope,
> `toolRefs[].path` in an ordinary top-level runbook, and `tool-paths[]`) the
> same computation is performed against the workspace root, but the outcome is
> a **report, not a rejection**: a resolved path outside the workspace root is
> recorded as an external root and warned as `PKG-W003` (§Symlinks, rule 6).
> `PKG-007` **MUST NOT** be raised for a workspace-root escape by any
> workspace-level kind; for those kinds `PKG-007` remains reachable only via
> the syntax rules above. If the escaping target does not contain a resolvable
> package, the resulting error is `PKG-001`, not `PKG-007`.

## Validation

- `python design/gert/scripts/verify_corpus.py` → `OK 365/365 vectors validate`.
- `python design/gert/scripts/latex.py check` → ok. `latex.py build` fails
  identically **before and after** these edits (pre-existing minted/pygments
  toolchain failure at `\begin{document}`, unrelated to this change).
