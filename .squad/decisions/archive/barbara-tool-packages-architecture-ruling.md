# Architecture Ruling — GERT Tool Packages (MVP)

**From:** Barbara (Lead / Architect)
**Date:** 2026-08-09T14:09:01-07:00
**Requested by:** Cristián Ormazábal Ortega
**Status:** RATIFIED — binding input for Edith's normative spec authoring
**Supersedes:** open items in `.squad/orchestration-log/2026-08-09T14-04-49Z-barbara-gert-tool-packages.md`
and `...-edith-gert-tool-packages.md`

These are decisions, not options. Where the current spec text conflicts, this ruling wins and the
spec text is to be rewritten.

---

## 0. MVP Boundary (non-negotiable)

**IN:** local tool packages on disk, `requires:` declaration, deterministic catalog construction,
collision rules, version constraint checking, runbook-backed action substitution, secure path rules,
package digests, evidence/trace/resume behaviour, plan-time error taxonomy, conformance vectors.

**OUT (MUST NOT appear in normative text except as an explicit non-goal):** remote catalogs and
registries, publishing/packaging workflows, transitive package dependencies, tool aliases, action
allowlists as an *enforcement* mechanism, packaging of non-tool assets (docs, dashboards, policy
bundles, binaries-as-payload).

Rationale: GERT is pre-1.0 with no external consumers (ratified OQ-M4, "just ship"). MVP must pin
determinism and safety, not distribution.

---

## 1. `toolPackages` vs `requires:` vs `toolRefs` — AR-TP-1

### Ruling

| Key | Status | Meaning |
|---|---|---|
| `requires:` | **CANONICAL.** Runbook top-level and `.gert/config.yaml` top-level. | Declares *package* dependencies. Contributes tool definitions to the catalog. |
| `toolRefs:` | **RETAINED.** Runbook top-level only. | Declares which *tool names* this runbook file may reference from `flow`, and optionally pins where each comes from. Does not itself add packages. |
| `toolPackages:` | **REJECTED. MUST NOT be introduced.** | Rejected as a third concept with the same job as `requires:`. |

`requires:` supplies; `toolRefs:` binds and constrains. One-way relationship: `toolRefs` may *narrow*
what `requires:` supplied; it may never widen it.

### Migration path

`toolPackages:` never shipped in any released schema. There is **no alias, no dual-read, no
deprecation window**. Any document containing `toolPackages:` is rejected with `PKG-020`
(`unsupported key: use requires:`) so the mistake is diagnosable rather than silently ignored.
`PKG-020` is a permanent hard error, not a transitional one.

### `ToolRef` field disposition

| Field | Ruling |
|---|---|
| `name` | REQUIRED. The name used in `step.tool.name`. |
| `package` | **NEW.** Optional. Reverse-DNS package name that MUST have supplied this tool. Pins the tier and the supplier; makes a collision impossible by construction. |
| `version` | **NEW.** Optional. Constraint (§4 grammar) checked against the resolved *tool definition's* `meta.version`. |
| `path` | RETAINED. Optional. Direct path to a `.tool.yaml`, resolved per §6. Highest tier (§2). Mutually exclusive with `package`. |
| `actions` | RETAINED, **semantics changed.** Documentation + plan-time reachability assertion only. Every listed action MUST exist on the resolved tool (`PKG-012` if not). It is **NOT** an allowlist and MUST NOT gate invocation in MVP. Emits deprecation warning `PKG-W001` when present. |
| `source` | RETAINED, **semantics changed.** Opaque provenance string. MUST NOT participate in resolution. Emits deprecation warning `PKG-W002`. Replaced by `package`. |
| `alias` | **REMOVED** from the schema. Out of MVP. Presence is `PKG-021`. |

No fixture currently uses `alias`. Fixtures r01/r05 use `source`/`actions`; they remain valid and
emit warnings, which is the intended migration signal.

---

## 2. Deterministic Resolution and Collision — AR-TP-2

### 2.1 Two-phase model

Resolution is split so that "what exists" is decided before "what a step means".

- **Phase C (Catalog construction)** — plan time, before any step is planned. Produces the
  `ResolvedToolCatalog`: a map from *fully-qualified tool name* to a single `ToolDefinition`, plus a
  frozen package map (§3.5). Pure function of (workspace files, `requires:`, extension contributions,
  MCP `tools/list` responses). No step context.
- **Phase B (Binding)** — plan time, per runbook file. Each `toolRefs` entry binds one catalog entry
  to one local name for that file. Each `step.tool.name` MUST resolve to a bound name.

A runtime MUST NOT perform catalog mutation after Phase C completes. Late contribution is `PKG-017`.

### 2.2 Tiers

Tools enter the catalog from exactly five tiers. Higher number = higher precedence.

| Tier | Source | Namespacing |
|---|---|---|
| 0 | Built-in registry compiled into the host | bare names |
| 1 | Packages from `requires:` (project-level, then runbook-level) | `<package>/<tool>` + bare name |
| 2 | Project tools: `.gert/config.yaml` `tool-paths` (in listed order), then `<workspace>/tools/` | bare names |
| 3 | Dynamic: extension contributions (`capability/tool-registration`), then MCP `tools/list` | `<extension>/<tool>`, `<server>/<tool>` — mandatory |
| 4 | `toolRefs[].path` in the runbook being planned | bare name, file-local |

Every catalog entry carries **both** a fully-qualified name (`<origin>/<tool>`) and, for tiers 0/1/2/4,
a bare name. Tier 3 entries have **no** bare name — dynamic contributions are always qualified. This
alone removes the entire class of "extension silently shadows kubectl" failures.

### 2.3 Precedence rule (replaces "first match wins + warning")

1. **Same-tier collision on the same name is a hard error** (`PKG-006`). No warning-and-continue. Two
   packages exporting bare name `kubectl` collide; the runbook MUST disambiguate with
   `toolRefs[].package` or with the qualified name.
2. **Cross-tier shadowing is permitted and silent only when the shadowing entry is at a strictly
   higher tier AND the runbook declares the intent** via `toolRefs[].package` or `toolRefs[].path`.
   Undeclared cross-tier shadowing is `PKG-022` (error), not a warning.
3. Fully-qualified names never collide and never shadow; they are always addressable even when the
   bare name is ambiguous.
4. **Direction is uniform across the spec:** higher tier wins, ties are errors. §05 Extension
   Discovery's current "later entries take precedence" and §06 Tool Discovery's "first match wins"
   are **both wrong** and MUST be rewritten to this single rule. This inconsistency is a spec defect,
   not a subsystem difference.

### 2.4 Determinism of enumeration

Non-determinism from filesystem iteration order is a defect. Normative:

- Directory scans are recursive over `*.tool.yaml`, and results are sorted by the **workspace-relative
  POSIX path**, NFC-normalised, compared by Unicode code point, ascending.
- `requires:` entries are processed in declaration order; project-level `requires:` precede
  runbook-level `requires:`.
- `tool-paths` entries in declaration order.
- Extension contributions in extension-resolution order; within one extension, in the order returned
  by `contributions/list`.
- MCP servers in declaration order; within one server, tools sorted by name (the server's array order
  is NOT trusted, since MCP servers make no ordering guarantee).
- Two runs over an identical file tree MUST produce byte-identical catalog digests (§7).

### 2.5 Builtins

Built-ins are tier 0 and are always present. A package or project tool with the same bare name as a
built-in shadows it, and this requires declaration per rule 2. There is no "disable builtins" switch
in MVP.

---

## 3. Schemas — AR-TP-3

All new schemas are hand-authored, language-neutral JSON Schema (Draft 2020-12), consistent with
`runbook.v1.schema.json`, `additionalProperties: false` throughout.

### 3.1 Package manifest — `gert-package.yaml` (`tool-package.v1.schema.json`)

```yaml
apiVersion: package/v1          # REQUIRED, const "package/v1"
meta:                           # REQUIRED
  name: acme.incident-tools     # REQUIRED. Reverse-DNS: ^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$
  version: "1.4.2"              # REQUIRED. Strict SemVer 2.0.0
  description: "..."            # OPTIONAL
  license: "Apache-2.0"         # OPTIONAL, SPDX identifier
exports:                        # REQUIRED
  tools:                        # REQUIRED, minItems 1, unique by both id and name
    - id: kubectl               # REQUIRED. Bare name exported. ^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$
      path: tools/kubectl.tool.yaml   # REQUIRED. Package-root-relative, §6-safe
dependencies: []                # OPTIONAL. MUST be absent or empty in MVP; non-empty is PKG-016
```

Rules:
- The package root is the directory containing `gert-package.yaml`.
- `exports.tools[].id` is the *only* way a tool becomes visible. A `.tool.yaml` inside the package
  that is not exported is invisible (`PKG-011` if referenced). No implicit `tools/` scanning inside a
  package — implicit scanning is what makes package contents an accidental API surface.
- The referenced `.tool.yaml`'s `meta.name` (or `name`) MUST equal the export `id` (`PKG-023`).
- The qualified name is `<meta.name>/<export id>`.
- Non-tool assets are not describable in MVP. Any other key under `exports` is a schema error.

### 3.2 Project binding — `.gert/config.yaml` (`project-config.v1.schema.json`)

`.gert/config.yaml` currently has no normative schema. It gets one, covering only the fields this
ruling touches; other existing keys (`defaults`, `tools.mcp-servers`) are carried over as-is.

```yaml
apiVersion: config/v1
requires:
  - package: acme.incident-tools   # REQUIRED
    version: "^1.4.0"              # REQUIRED (§4 grammar)
    path: ./vendor/acme-incident-tools   # REQUIRED in MVP; §6-safe; must contain gert-package.yaml
tool-paths:
  - ./ops/tools
```

`path` is REQUIRED in MVP precisely because there is no remote catalog. When catalogs land,
`path` becomes optional and its absence means "resolve from catalog" — that is the designed
extension point, and it is forward-compatible without a schema break.

### 3.3 Runbook binding — additions to `runbook.v1.schema.json`

```yaml
requires:                    # NEW top-level, optional, array
  - package: acme.incident-tools
    version: "^1.4.0"
    path: ./vendor/acme-incident-tools   # optional here; if omitted, MUST be supplied by project config
toolRefs:
  - name: kubectl
    package: acme.incident-tools
    version: "^1.4.0"
```

A runbook-level `requires:` entry with no `path` and no matching project-level entry is `PKG-001`.
A runbook-level entry whose `path` disagrees with the project-level entry for the same package name
is `PKG-024` (conflicting binding) — the project is authoritative and disagreement is an error, not a
silent override.

### 3.4 Duplicate package declarations

Same `package` name twice within one `requires:` list is `PKG-005`, unconditionally, even if the
constraints and paths are identical. Duplicates are always an authoring mistake.

Same `package` name in project-level and runbook-level lists is legal; the constraints are
**intersected**. Empty intersection is `PKG-002`.

### 3.5 Package map / lock — `.gert/packages.lock.json` (`package-lock.v1.schema.json`)

Generated, committed, and verified. Not a solver output (there is nothing to solve in MVP) — it is a
**frozen integrity record**.

```json
{
  "apiVersion": "package-lock/v1",
  "generatedBy": "gert/<version>",
  "packages": [
    {
      "name": "acme.incident-tools",
      "version": "1.4.2",
      "root": "vendor/acme-incident-tools",
      "digest": "sha256:<64 hex>",
      "exports": [
        { "id": "kubectl", "path": "tools/kubectl.tool.yaml", "digest": "sha256:<64 hex>" }
      ]
    }
  ],
  "catalogDigest": "sha256:<64 hex>"
}
```

- `root` is workspace-relative POSIX. Packages outside the workspace are recorded as
  `"external:sha256:<digest of the absolute realpath, UTF-8, NFC>"` — never as a raw absolute path,
  because absolute paths leak usernames and host layout into committed artefacts and traces.
- Lock presence is OPTIONAL for `gert run`. When present, mismatch is `PKG-009`.
- `gert verify` (existing command, §08) is extended to check the lock.

---

## 4. Version Constraint Grammar — AR-TP-4

### 4.1 Versions

Package and tool versions MUST be strict SemVer 2.0.0. Non-conforming is `PKG-025`. Leading `v` is
rejected, not stripped.

### 4.2 MVP constraint grammar (normative EBNF, to be added to the spec, not to a `.ebnf` grammar file
— this is not an expression language)

```
Constraint   = Comparator , { " " , Comparator } ;
Comparator   = Caret | Tilde | Relational | Exact ;
Caret        = "^" , Version ;
Tilde        = "~" , Version ;
Relational   = ( ">=" | "<=" | ">" | "<" ) , Version ;
Exact        = [ "=" ] , Version ;
Version      = Major , "." , Minor , "." , Patch , [ "-" , Prerelease ] ;
```

- Space-separated comparators are **conjunctive (AND)**. There is no disjunction.
- **NOT in MVP** (each is `PKG-003`): `||`, `*`, `x`/`X` wildcards, hyphen ranges (`1.2.3 - 1.5.0`),
  partial versions (`^1`, `~1.2`), `!=`, whitespace variants other than a single ASCII space,
  build metadata in constraints.
- Partial versions are excluded deliberately: `^1` vs `^1.0.0` differ across ecosystems, and
  ambiguity in a governance-bearing artefact is unacceptable.

### 4.3 Semantics

- `^1.4.2` → `>=1.4.2 <2.0.0`. For `0.y.z`: `^0.4.2` → `>=0.4.2 <0.5.0`; `^0.0.3` → `>=0.0.3 <0.0.4`.
  The 0.x narrowing is explicit because it is the most commonly mis-implemented rule.
- `~1.4.2` → `>=1.4.2 <1.5.0`.
- Ordering is SemVer 2.0.0 §11 precedence.
- **Build metadata** (`+meta`) is ignored for ordering and for satisfaction, and MUST NOT be used for
  selection. It is retained verbatim in the lock and trace for provenance.
- **Prereleases:** a version with a prerelease component satisfies a comparator only if the
  comparator's own version has a prerelease component **and** shares the same `major.minor.patch`.
  So `1.5.0-rc.1` does not satisfy `^1.4.0`, but does satisfy `>=1.5.0-rc.1 <1.6.0`. This is the
  Cargo/npm rule and it is the safe one for operational runbooks.

### 4.4 Constraint checking, not solving

MVP has exactly one candidate per package (the local path). The constraint is a **predicate on that
candidate**, evaluated after loading the manifest. There is no backtracking, no version selection, no
SAT. Failure is `PKG-002` and reports: package name, resolved version, effective constraint, and the
list of declaration sites that contributed to the intersection.

---

## 5. Runbook-Relative Replacement Contract — AR-TP-5

A tool action may be implemented by a GERT runbook instead of a process. This is "substitution".

### 5.1 Declaration

Declared **in the `.tool.yaml`**, per action — never in the calling runbook. The caller must not be
able to see or change how an action is implemented.

```yaml
actions:
  - name: drain-node
    impl:
      kind: runbook                    # process | runbook  (default: process)
      path: runbooks/drain-node.yaml   # relative to the DIRECTORY OF THIS .tool.yaml
    inputs:
      node: { type: string, required: true }
    outputs:
      drained: { type: boolean }
```

`impl.path` resolves relative to the declaring `.tool.yaml`'s directory, **not** the calling runbook,
**not** the workspace root, **not** the process cwd. This is the "runbook-relative" contract: a
package is relocatable because every internal reference is package-internal. Resolution obeys §6.

### 5.2 Input contract

- The substitute runbook's `inputs:` map MUST be a **superset-free exact match by name** with the
  action's `inputs:`: same key set, no extras, no omissions (`PKG-013`).
- Types MUST be identical PJVM types. No widening, no coercion (`PKG-013`).
- `required` on the action side MUST imply `required` on the substitute side.
- Substitute inputs MUST have `from:` absent or `context`. `from: prompt` or `from: env` in a
  substitute is `PKG-026` — a tool action must not covertly prompt the operator or read host env.
- Argument values are bound positionally-by-name from `step.tool.args` after GIS interpolation in the
  **caller's** scope. The substitute receives values, never expressions. The caller's variable scope
  is not visible inside the substitute.

### 5.3 Output contract

- The substitute's `outputs:` map MUST exactly match the action's `outputs:` key set and types
  (`PKG-013`).
- Every declared output MUST be produced on every terminal path of the substitute, or declare a
  default. Statically unproducible outputs are `PKG-027`.
- Captures inside the substitute are not visible to the caller. Only declared outputs cross the
  boundary. Trace shows the substitute's steps as nested spans (§7.3), so auditability is preserved
  without leaking scope.

### 5.4 Governance contract (invariants — non-negotiable)

Substitution MUST NOT be a privilege-escalation path. The substitute's effective policy is derived,
never adopted:

| Field | Composition |
|---|---|
| `require_approval` | logical **OR** of caller-effective and substitute |
| `deny_commands` | **union** |
| `deny_env_vars` | **union** |
| `allow_commands` | **intersection** |
| `redact` | **union** |
| `rules` | caller-effective rules evaluated first, then substitute rules; a substitute rule MUST NOT convert a deny into an allow |
| capabilities | substitute's effective capability set = caller-effective ∩ tool `governance.requires-capabilities` closure; never a superset |

Any substitute construct whose composition would **widen** the caller's effective policy is
`PKG-014`, raised at plan time by static comparison — not at runtime. Widening detectable only at
runtime is a design defect; this is why `allow_commands` is intersected rather than merged.

The tool's own `governance.allowed-environments` and `requires-approval` continue to gate the action
as for process-backed actions. A `dry-run` run MUST NOT execute a substitute's side-effecting steps.

### 5.5 Action reachability

At plan time, for every `step.tool` in every reachable step of every runbook in the include graph:

1. `tool.name` MUST be bound by a `toolRefs` entry in the **same runbook file** (`PLAN-010`).
2. `tool.action` MUST be declared by the bound tool definition (`PKG-012`).
3. If the action is `kind: runbook`, the substitute MUST parse, validate, and satisfy §5.2–5.4
   (`PKG-013`/`PKG-014`/`PKG-027`).
4. Every name listed in `toolRefs[].actions` MUST exist (`PKG-012`), even if no step uses it.

Reachability uses the existing static-reachability analysis behind `PLAN-009`. Unreachable steps are
still validated: validity is not conditional on execution path.

### 5.6 Recursion

Substitution frames are tracked as `(package, tool, action)` triples. A cycle is `PKG-015`, detected
at plan time by DFS. Maximum substitution depth in MVP is **4**; exceeding it is `PKG-028`. Depth is
bounded because each frame multiplies the governance-composition surface.

---

## 6. Secure Path Resolution — AR-TP-6

Applies uniformly to `requires[].path`, `exports.tools[].path`, `impl.path`, `toolRefs[].path`,
`tool-paths[]`, and include/import paths inside packages.

### 6.1 Syntax rules (checked before touching the filesystem)

1. Separator in authored artefacts is `/`. Backslash in an authored path is `PKG-007` — not
   normalised, because `a\b` is a legal single filename on POSIX.
2. Absolute paths are rejected in **package-internal** references (`exports`, `impl`,
   package-internal includes): `PKG-007`. Absolute paths are permitted in `requires[].path` and
   `tool-paths[]` (workspace-level operator configuration) but are then subject to §6.3.
3. Rejected everywhere: drive-relative (`C:foo`), UNC (`\\host\share`), extended-length
   (`\\?\`, `\\.\`), device namespaces, `~` expansion, environment-variable expansion, URI schemes,
   NUL and control characters, and any segment equal to `.` or `..` **after** normalisation that
   escapes the root.
4. Rejected filenames (all platforms, for portability): reserved DOS names `CON PRN AUX NUL COM1–COM9
   LPT1–LPT9` with or without extension, and any segment with a trailing dot or trailing space.
   `PKG-018`.
5. Paths are NFC-normalised for comparison. Two exports whose paths differ only by Unicode
   normalisation form, or only by ASCII case, are `PKG-018` — this makes a package that works on
   Linux but breaks on Windows/macOS fail at authoring time, on every platform.

### 6.2 Containment rule

For every package-internal reference: `realpath(join(root, p))` MUST be lexically inside
`realpath(root)`, compared segment-wise on NFC-normalised, case-folded segments when the host
filesystem is case-insensitive, and exactly otherwise. Violation is `PKG-007`.

Containment is checked on the **resolved** path, after link resolution — checking the lexical path
only is the classic bypass.

### 6.3 Symlinks, junctions, reparse points

1. Symlinks are **resolved, then containment-checked**. A link whose final target is inside the root
   is allowed. A link whose final target is outside the root is `PKG-008` — an **error**, never a
   silent skip, because silent skipping produces a catalog that differs between machines.
2. Windows directory junctions, mount points, and any `IO_REPARSE_TAG_*` reparse point are treated
   exactly as symlinks. AppExecLink and OneDrive placeholder reparse points are `PKG-008`.
3. Link chains are bounded at **8** hops; exceeding is `PKG-008`. Cycles are `PKG-008`.
4. Resolution MUST be performed with a TOCTOU-resistant sequence: resolve once, then read via the
   resolved handle/path; the digest (§7) is computed over the same resolved bytes used for loading.
5. Hard links are not detectable and are out of scope; the digest covers content, which is the
   property that matters.
6. `requires[].path` and `tool-paths[]` that resolve outside the workspace are permitted but MUST be
   recorded in the lock and trace in the `external:sha256:...` form (§3.5), and `gert verify` MUST
   report them as a distinct "external root" finding.

---

## 7. Digest, Evidence, Trace, Resume — AR-TP-7

### 7.1 File digest

`sha256` over the file's **raw bytes**. No line-ending normalisation, no whitespace normalisation, no
YAML canonicalisation. Encoded `sha256:` + 64 lowercase hex.

Rationale for raw bytes: any normalisation is a second parser, and a second parser is a second bug.
The Windows CRLF hazard is handled by a documented `.gitattributes` recommendation (`*.tool.yaml
text eol=lf`, `gert-package.yaml text eol=lf`), not by a normalisation rule.

### 7.2 Package digest (deterministic, order-free)

Given package root `R`, the digest set is: `gert-package.yaml`, every `exports.tools[].path`, and
every file transitively referenced by an exported tool via `impl.path` and package-internal includes.
Files present in the package but not in this closure are **not** hashed — the digest covers the API
surface, not incidental contents.

```
for each file f in closure:
    line(f) = hex(sha256(bytes(f))) + "  " + relpath_posix_nfc(f, R)
sort lines ascending by UTF-8 byte order of the whole line
blob = concat(line + "\n" for each sorted line)
packageDigest = "sha256:" + hex(sha256(blob))
```

Two spaces between digest and path (coreutils `sha256sum` convention, so the blob is
independently reproducible with standard tools).

### 7.3 Catalog digest

Same construction over lines of `qualifiedName + "  " + packageDigestOrFileDigest + "  " + tierNumber`,
sorted ascending. `catalogDigest` is the single value that proves two runs saw the same tool universe.

### 7.4 Trace and evidence

New events in §07 (Extension and Tool Events subsection):

- `package/resolved` — one per package, emitted at plan time, **before** `run/started` side effects:
  `{ name, version, digest, root, external: bool, constraintSources: [...] }`.
- `catalog/frozen` — one per run: `{ catalogDigest, toolCount, tiers: {0..4 counts} }`.
- `tool/substituted` — emitted when a `kind: runbook` action is entered:
  `{ tool, action, substituteDigest, depth, effectiveGovernance: {...} }`. The composed governance
  is recorded explicitly, because "what policy actually applied" is the audit question.

Substitute steps appear in the trace as nested steps with an `Origin` breadcrumb
`[substituted from: <package>/<tool>#<action>]`, mirroring the existing include breadcrumb. They are
**not** hidden: an audit trail that omits executed steps is not an audit trail.

Package roots, digests, and the catalog digest are part of the run manifest (`run.yaml`) and are
covered by the existing HMAC trace chaining. Package paths are subject to the standard redaction
pipeline; the `external:sha256:` form is applied *before* redaction, not after.

### 7.5 Resume

On `gert exec --resume`:

1. Recompute every package digest and the catalog digest from the current filesystem.
2. Compare against the values in the run manifest.
3. **Mismatch is a hard refusal** — `PKG-009` — and the run stays resumable. It does not partially
   resume, and it does not silently re-plan. Resuming a run against changed tool definitions
   invalidates the evidence chain, which is the entire point of the trace.
4. `--allow-package-drift` overrides, and when used MUST emit a `governance/packageDriftAccepted`
   event recording both digests and the operator identity. Drift acceptance is an auditable act.
5. Missing packages on resume are `PKG-001`, never a fallback to a different tier.

### 7.6 Replay

Replay mode does not invoke tools, so digests are **recorded and compared but non-fatal**: a mismatch
emits a `replay/packageDrift` warning event and replay proceeds. Replay determinism is defined by the
scenario file, not by the on-disk catalog.

---

## 8. Include / Import Graph — AR-TP-8

### 8.1 Lexical scoping of tool names

**Ruling: tool name binding is lexically scoped to the runbook file that declares it.** An included
child runbook does NOT inherit the parent's `toolRefs`. If a child uses `kubectl`, the child declares
`kubectl` in its own `toolRefs`.

This is the decisive call. Dynamic (inherited) scoping makes a child runbook's meaning depend on who
included it, which destroys both composability and static reachability analysis.

### 8.2 Global package set

Packages are **globally resolved, per run, at the root runbook**. The effective package set is the
union of the project-level `requires:` and the runbook-level `requires:` of *every runbook in the
include/import closure*, computed at plan time. Constraints for the same package name across the
closure are intersected (§3.4); empty intersection is `PKG-002`.

Consequence: a package is loaded once per run, at one version, with one digest. Two runbooks in one
run cannot use two versions of the same package. This is a deliberate MVP restriction — multi-version
coexistence requires a tier-aware qualified namespace and is deferred.

### 8.3 Graph rules

- Cycle detection is over `realpath` of resolved include/import targets (extends the existing
  load-time DFS). Cycles remain a hard error.
- The include closure MUST be fully enumerable at plan time. **Lazy includes** (`expand: lazy`) are
  still *resolved and package-analysed* at plan time even though they are *expanded* at run time.
  A lazily-expanded include that would introduce a package not in the frozen set is `PKG-017`.
- Packages MUST NOT contain runbooks that include files outside the package root (§6.2). A package's
  runbooks are self-contained.
- `imports:` (alias→path) remains a runbook-local alias map for child runbooks and is unrelated to
  packages. It MUST NOT be repurposed for package aliasing. Alias-based package naming is out of MVP.
- Maximum include depth is unchanged; substitution depth (§5.6) is counted separately and both
  bounds apply independently.

---

## 9. Error Taxonomy — AR-TP-9

All codes are plan-time unless noted, and MUST use the structured error format of
§03d `sec:parse-gate:error-format` (`code`, `message`, `location{file,line,column,span_length}`,
`snippet`, `suggestion`). `location.file` is workspace-relative; for errors inside a package it is
`<packageRoot>/<relpath>`.

| Code | Raised by | Condition | Remediation |
|---|---|---|---|
| `PKG-001` | Resolver | Required package not resolvable (no path, missing root, missing manifest) | Declare `path` or install the package |
| `PKG-002` | Resolver | Version constraint unsatisfied, or empty constraint intersection | Align the constraint with the resolved version |
| `PKG-003` | Resolver | Malformed version constraint (grammar §4.2) | Use a supported comparator form |
| `PKG-004` | Resolver | Package manifest fails schema validation | Fix `gert-package.yaml` |
| `PKG-005` | Resolver | Duplicate package name in one `requires:` list | Remove the duplicate entry |
| `PKG-006` | Catalog | Same-tier tool name collision | Rename, or bind via `toolRefs[].package` |
| `PKG-007` | Path | Unsafe or escaping path (syntax or containment) | Use a package-root-relative POSIX path |
| `PKG-008` | Path | Symlink/junction escape, cycle, or hop limit exceeded | Remove the link or move the target inside the root |
| `PKG-009` | Verify/Resume | Package or catalog digest mismatch | Re-verify, or resume with `--allow-package-drift` |
| `PKG-010` | Resolver | Export references a missing file | Fix `exports.tools[].path` |
| `PKG-011` | Binding | Referenced tool is not exported by the package | Add it to `exports.tools` |
| `PKG-012` | Binding | Action not declared by the resolved tool (incl. `toolRefs[].actions` entries) | Correct the action name |
| `PKG-013` | Substitution | Substitute runbook input/output signature mismatch | Align `inputs`/`outputs` with the action |
| `PKG-014` | Substitution | Substitute would widen effective governance | Narrow the substitute's policy |
| `PKG-015` | Substitution | Substitution cycle | Break the `(package,tool,action)` cycle |
| `PKG-016` | Resolver | Non-empty `dependencies:` (transitive deps out of MVP) | Flatten dependencies into `requires:` |
| `PKG-017` | Catalog | Late package binding after catalog freeze (incl. lazy include) | Declare the package at the root |
| `PKG-018` | Path | Reserved name, case-only or NFC-only collision, trailing dot/space | Rename the file |
| `PKG-019` | *(reserved)* | — | — |
| `PKG-020` | Parser | `toolPackages:` key present | Use `requires:` |
| `PKG-021` | Parser | `toolRefs[].alias` present | Remove; aliases are out of MVP |
| `PKG-022` | Catalog | Undeclared cross-tier shadowing of a bare tool name | Declare `toolRefs[].package` or `.path` |
| `PKG-023` | Resolver | Export `id` ≠ tool definition `name` | Make them identical |
| `PKG-024` | Resolver | Runbook `requires[].path` conflicts with project binding | Remove the runbook-level `path` |
| `PKG-025` | Resolver | Version is not strict SemVer 2.0.0 | Use `MAJOR.MINOR.PATCH` |
| `PKG-026` | Substitution | Substitute declares `from: prompt` or `from: env` | Pass the value as an action input |
| `PKG-027` | Substitution | Declared substitute output not produced on all terminal paths | Produce it or give it a default |
| `PKG-028` | Substitution | Substitution depth > 4 | Flatten the substitution chain |
| `PLAN-010` | Planner | `step.tool.name` has no `toolRefs` binding in the declaring file | Add a `toolRefs` entry |

Warnings (non-fatal, MUST appear in the plan report and in the trace):

| Code | Condition |
|---|---|
| `PKG-W001` | `toolRefs[].actions` present — documentation-only in MVP, not an allowlist |
| `PKG-W002` | `toolRefs[].source` present — deprecated, superseded by `package` |
| `PKG-W003` | Package root resolves outside the workspace (external root) |

`PLAN-010` slots into the existing `PLAN-*` series in §03d. `PKG-*` is a new series in the same
normative catalog.

---

## 10. Minimum Conformance Vectors — AR-TP-9b

New vector categories added to `design/gert/conformance/vector.schema.json` `category` enum:
`PKG-RESOLVE`, `PKG-COLLIDE`, `PKG-VERSION`, `PKG-PATH`, `PKG-SUBST`, `PKG-DIGEST`, `PKG-ERROR`.

New corpus file: `design/gert/conformance/tv-pkg-resolve.yaml`. **Minimum 46 vectors**, distributed:

**PKG-VERSION (12)** — `^1.4.2` bounds; `^0.4.2` bounds; `^0.0.3` bounds; `~1.4.2` bounds;
conjunctive `>=1.2.0 <2.0.0`; exact `=1.4.2` and bare `1.4.2` equivalence; prerelease excluded from
`^1.4.0`; prerelease included by same-tuple constraint; build metadata ignored in ordering;
`PKG-003` for `||`; `PKG-003` for `^1`; `PKG-025` for `v1.4.2`.

**PKG-RESOLVE (8)** — tier-2 shadows tier-1 with declaration; tier-1 over tier-0; qualified name
resolves when bare name is ambiguous; project+runbook constraint intersection; `PKG-002` on empty
intersection; `PKG-005` duplicate; `PKG-024` conflicting path; `PKG-011` unexported tool.

**PKG-COLLIDE (6)** — same-tier bare collision → `PKG-006`; undeclared cross-tier → `PKG-022`;
declared cross-tier via `package` → OK; extension tool with bare name is unaddressable; MCP tool
qualified as `<server>/<tool>`; two MCP servers exporting the same tool name → both addressable.

**PKG-PATH (8)** — `..` escape → `PKG-007`; absolute in `exports` → `PKG-007`; backslash separator →
`PKG-007`; symlink inside root → OK; symlink outside root → `PKG-008`; link cycle → `PKG-008`;
reserved name `AUX.tool.yaml` → `PKG-018`; case-only collision `Kubectl` vs `kubectl` → `PKG-018`.

**PKG-SUBST (8)** — valid substitution round trip; input key mismatch → `PKG-013`; input type
mismatch → `PKG-013`; output not produced on a branch → `PKG-027`; governance widening via
`allow_commands` → `PKG-014`; `require_approval` OR composition applied; substitution cycle →
`PKG-015`; depth 5 → `PKG-028`.

**PKG-DIGEST (4)** — digest stability across enumeration order; digest changes on content change;
digest unchanged when a non-exported file changes; catalog digest stability across two runs.

Vectors follow the existing `TestVector` shape (`id, category, description, input, variables,
expected`), with `variables` carrying the synthetic filesystem tree (path → content) and the
`requires:`/`toolRefs:` declarations, and `expected` carrying either the resolved catalog projection
or `{ error: { code } }`. Tess owns authoring; this ruling fixes the categories, the count floor, and
the required scenarios.

---

## 11. Backward Compatibility and Deprecation — AR-TP-10

Pre-1.0, no external consumers, ratified OQ-M4 ("just ship"). Therefore:

1. **No compatibility shims.** `toolPackages:` and `toolRefs[].alias` are hard errors from day one
   (`PKG-020`, `PKG-021`). Diagnosable rejection, not silent acceptance.
2. **Soft deprecations only where fixtures exist.** `toolRefs[].source` and `toolRefs[].actions`
   remain schema-valid and emit `PKG-W002` / `PKG-W001`. They lose all resolution and enforcement
   power immediately. Removal is scheduled for the first release after the fixture corpus is
   migrated; Edith records this as a numbered deprecation with a removal target, not "eventually".
3. **`requires:` was already normative** in §06 (Tool Versioning). This ruling makes it schema-backed
   and precise; it is a tightening, not an introduction. Existing `requires:` documents that used
   `package: github.com/acme/gert-tools` must migrate to reverse-DNS names — URL-shaped names are
   now `PKG-004`, because a name that looks like a location invites a remote fetch that MVP does not
   perform.
4. **Additive-only for runbooks that use none of this.** A runbook with no `requires:` and only
   tier-0/tier-2 `toolRefs` is unaffected except that undeclared cross-tier shadowing now errors
   (`PKG-022`). That behaviour change is intentional and is the single breaking change in this
   ruling.
5. Schema versions: `runbook.v1` gains optional fields — stays `v1`. New schemas ship as
   `tool-package/v1`, `config/v1`, `package-lock/v1`.
6. Every code and warning in §9 is permanent once shipped. Codes are never renumbered or reused;
   `PKG-019` is reserved and left unassigned to make that policy visible.

---

## 12. Target Artefacts for Edith

**Modify:**

| File | Change |
|---|---|
| `design/gert/sections/06-tool-runtime.tex` | Rewrite §Tool Discovery and §Name Resolution Order to the two-phase/five-tier model (§2). Delete "first match wins and a warning is emitted". Rewrite §Tool Versioning to §4. Add §Tool Packages (schemas §3), §Action Substitution (§5), §Secure Path Resolution (§6). |
| `design/gert/sections/05-extension-runtime.tex` | §Extension Discovery: replace "later entries taking precedence" with the uniform higher-tier-wins/ties-are-errors rule; state that extension-contributed tools are always qualified and never bind a bare name. |
| `design/gert/sections/03d-parse-time-enforcement.tex` | Add the `PKG-*` table and `PLAN-010` to the normative master error catalog; add the `PKG-W*` warning table. |
| `design/gert/sections/07-runtime-events.tex` | Add `package/resolved`, `catalog/frozen`, `tool/substituted`, `governance/packageDriftAccepted`, `replay/packageDrift` to the event catalog with full envelopes. |
| `design/gert/sections/08-security-and-trust.tex` | §Supply Chain Security: add package digests (§7.1–7.3), the lock file, symlink/reparse rules (§6.3), external-root reporting; extend the `gert verify` contract. |
| `design/gert/sections/12-governance-policy.tex` | Add the substitution governance composition table (§5.4) as a normative invariant. |
| `design/gert/sections/13-evidence-tracing-resumption.tex` | Add package map to the run manifest and to Minimum Resumption State; specify resume refusal on digest mismatch and the drift override (§7.5–7.6). |
| `design/gert/sections/02-architecture.tex` | §include dispatch: add lexical tool-name scoping (§8.1) and plan-time package analysis of lazy includes (§8.3). |
| `design/gert/schemas/runbook.v1.schema.json` | Add top-level `requires`; add `$defs/PackageRequirement`; `ToolRef` gains `package`, `version`; drop `alias`; redescribe `source`/`actions` as deprecated/non-enforcing. |
| `design/gert/schemas/README.md` | Register the three new schemas and their apiVersions. |
| `design/gert/conformance/vector.schema.json` | Add the seven `PKG-*` categories to the `category` enum. |
| `design/gert/sections/10-open-questions.tex` | Close the tool-package open questions; record the deferred items (remote catalogs, transitive deps, multi-version coexistence, aliases, action allowlists, non-tool assets) as explicit non-goals with rationale. |
| `design/gert/testdata/runbooks/r01-k8s-incident/schema.yaml`, `r05-security-breach/schema.yaml` | No structural change required; annotate that `source`/`actions` now emit `PKG-W001`/`PKG-W002`. |

**Add:**

| File | Purpose |
|---|---|
| `design/gert/schemas/tool-package.v1.schema.json` | Package manifest (§3.1) |
| `design/gert/schemas/project-config.v1.schema.json` | `.gert/config.yaml` (§3.2) |
| `design/gert/schemas/package-lock.v1.schema.json` | `.gert/packages.lock.json` (§3.5) |
| `design/gert/schemas/examples/acme-incident-tools/gert-package.yaml` | Reference package manifest |
| `design/gert/schemas/examples/acme-incident-tools/tools/kubectl.tool.yaml` | Exported tool with a `kind: runbook` action |
| `design/gert/schemas/examples/acme-incident-tools/runbooks/drain-node.yaml` | Substitute runbook |
| `design/gert/testdata/runbooks/r23-tool-package/schema.yaml` | Fixture consuming a package + substitution |
| `design/gert/conformance/tv-pkg-resolve.yaml` | ≥46 vectors (§10) — Tess authors, Edith registers |

---

## 13. Sequencing

1. Edith authors §3 schemas + §9 error catalog first — Tess is blocked on the category enum and the
   codes, nothing else.
2. Edith then rewrites §06/§05 discovery and precedence (§1, §2).
3. Edith then §5 substitution + §12 governance invariants; these are the highest-risk text and
   deserve a dedicated review pass by me before merge.
4. Tess authors `tv-pkg-resolve.yaml` against §9/§10 once step 1 lands.
5. I review §5 and §7 specifically for governance-widening and digest-determinism holes before the
   phase gate.

---

**Ruling authority:** Barbara (Lead / Architect). Effective immediately. Deviations require a new
inbox entry, not an inline spec edit.
