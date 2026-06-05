# GERT Expression Evaluation Audit

**Date:** 2026-06-04  
**Author:** Don — Backend Dev  
**Scope:** Audit only. No runtime code was modified.

## Executive Summary

The repository currently available for this audit does **not** contain a Go runtime source tree (`pkg/`, `internal/`, `cmd/`, `go.mod`, or `*.go` files were not present). This audit therefore covers the normative design/specification files, web-platform parity notes, tool definitions, and runbook corpus under `design/gert/`.

There are two major author-supplied evaluation surfaces:

1. **Conditional expressions** using `expr-lang/expr` for `when`, `condition`, and `until` fields.
2. **String/template interpolation** using Go `text/template` for non-conditional fields such as `title`, `args`, tool `argv`, display content, and assertion subjects.

The portability risk is real. Even where the spec says `expr` is "Python-like", the exposed syntax includes Go/C-family operators (`&&`, `||`, `!`, `==`, `!=`) and test fixtures also use `expr`'s non-normative infix `contains`. More importantly, Go `text/template` is explicitly part of the authored runbook/tool surface, including dot traversal, template variables, custom functions, and Go template error/default behavior.

## Source Availability Finding

| Area requested | Finding | Evidence |
|---|---|---|
| Go runtime source tree | Not present in this repository snapshot | No `*.go` files or `go.mod` matched under the repository root. |
| Runtime evaluator implementation | Not directly auditable from code | Design docs describe evaluator contracts but no implementation files are available. |
| Runbook DSL/spec parser | Spec-level contracts are present | `design/gert/sections/03-schema-vnext.tex:412`, `design/gert/sections/02-architecture.tex:150` |
| Tests/examples | Runbook corpus present | `design/gert/testdata/runbooks/` |

## Inventory of Evaluation Sites

| # | Evaluation site | Author-supplied field(s) | Evaluator/mechanism | Go-style? | References |
|---|---|---|---|---|---|
| 1 | General condition language | All conditional fields: `when`, `condition`, `until` | `expr-lang/expr` | Partly. Uses Go/C-style `&&`, `||`, `!`, comparison operators; not Go runtime internals. | `design/gert/sections/03-schema-vnext.tex:412-426`, `design/gert/sections/03-schema-vnext.tex:435-440` |
| 2 | Common step guard | `step.when` | `expr-lang/expr` per global condition language | Yes-like operator surface (`==`, `&&`, `||`, `!`) | `design/gert/sections/03-schema-vnext.tex:695-710` |
| 3 | Branch routing | `branches[].condition` | Intended condition evaluator, but schema table still says "Go template" | Conflicted. Examples use infix `expr`; table says Go template. | `design/gert/sections/03-schema-vnext.tex:2518-2532`, `design/gert/sections/03-schema-vnext.tex:2543-2560` |
| 4 | Iterate convergence | `iterate.until` | `expr-lang/expr` condition expression | Yes-like operators where used | `design/gert/sections/03-schema-vnext.tex:2615-2635`, `design/gert/sections/03-schema-vnext.tex:2664-2669` |
| 5 | Iterate list source | `iterate.over` | Template/string/path mechanism is inconsistent across docs/examples | Yes if Go template; separate JSONPath-like surface in fixtures | `design/gert/sections/03-schema-vnext.tex:2628`, `design/gert/sections/03-schema-vnext.tex:2643-2650`, `design/gert/testdata/runbooks/r11-iterate-loop/schema.yaml:28-35` |
| 6 | Include control | `include.when`, `include.with` values | `include.when` is boolean expression; `include.with` values are described as expressions but examples use Go templates | Mixed expression/template surface | `design/gert/sections/03-schema-vnext.tex:2322-2343`, `design/gert/sections/03-schema-vnext.tex:2348-2357` |
| 7 | Collector dynamic fields | `fields[].when` | Conflicted: field table says Go template; dynamic-form section says `when` expression | High risk because browser/backend must agree for live forms | `design/gert/sections/03-schema-vnext.tex:1250-1262`, `design/gert/sections/03-schema-vnext.tex:1333-1354` |
| 8 | CLI step interpolation | `title`, `args`, `stdin`, `run` selected script | Go `text/template` | Yes. Dot traversal and Go template semantics are explicitly exposed. | `design/gert/sections/03-schema-vnext.tex:562-568`, `design/gert/sections/03-schema-vnext.tex:875-913`, `design/gert/sections/02-architecture.tex:467-477` |
| 9 | Tool step argument interpolation | `tool.args` values | Go `text/template` | Yes | `design/gert/sections/03-schema-vnext.tex:2138-2146`, `design/gert/sections/03-schema-vnext.tex:2160-2170` |
| 10 | Native/stdio tool argv rendering | `actions.<name>.argv[]` in tool definitions | Go `text/template` rendered with `step.tool.args` | Yes | `design/gert/sections/06-tool-runtime.tex:112-117`, `design/gert/sections/03-schema-vnext.tex:3154-3158`, `design/gert/sections/03-schema-vnext.tex:3317-3323` |
| 11 | Assertion operands | `assert[].subject`, `assert[].expected`; `json_path.path` | Go-template-rendered strings plus fixed assertion operators | Yes for operands; assertion operators are portable if specified | `design/gert/sections/03-schema-vnext.tex:2865-2891` |
| 12 | Display step content | `display.content` | Go `text/template` | Yes | `design/gert/sections/02-architecture.tex:586-587`, `design/gert/sections/02-architecture.tex:1385-1412` |
| 13 | Tool and event capture paths | `capture` values such as `json.items[0].metadata.name`, `json.items | length`, `stdout.incident.id` | Path extractor / pipeline-like expression; exact evaluator unspecified | Not Go-style, but under-specified and non-portable unless formalized | `design/gert/sections/03-schema-vnext.tex:2146-2173`, `design/gert/sections/06-tool-runtime.tex:282-299`, `design/gert/sections/02-architecture.tex:1030-1040` |
| 14 | Wait-for-event filters | `event.filter` | Literal key-value predicate map, not expression language | No | `design/gert/sections/03-schema-vnext.tex:2215-2247`, `design/gert/sections/02-architecture.tex:1030-1038` |
| 15 | Provider input binding | input `from:` | Provider prefix dispatch, not expression evaluation | No | `design/gert/sections/15-input-provider-framework.tex:4-8`, `design/gert/sections/15-input-provider-framework.tex:98-100` |

## Grammar Surface Exposed

### Conditional fields (`expr-lang/expr`)

Documented syntax:

- Identifiers: direct variable names from the flat run variable map (`hostname`, `severity`, `count`).
- Literals: strings, numbers, booleans (`"critical"`, `1`, `true`).
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`.
- Logical: `&&`, `||`, `!`.
- Grouping: parentheses.
- Built-ins/predicates: `len(s)`, `matches(s, pattern)`.
- GERT string helper namespace: `str.contains`, `str.startsWith`, `str.endsWith`, `str.toLower`, `str.toUpper`, `str.trim`.
- Underlying `expr` infix operators such as `x contains "y"` are explicitly excluded from the normative spec, but fixtures still use them.

References: `design/gert/sections/03-schema-vnext.tex:428-560`.

### Go template interpolation fields

Documented syntax/mechanisms:

- Dot traversal: `{{ .hostname }}`, `{{ .vars.name }}`.
- Template variables/assignment: `{{ $cfg := fromJSON .service_config }}`.
- Structured parse helpers: `fromJSON`, `fromYAML` returning `interface{}`.
- Full Go template pipelines and custom function maps are described as intentional.
- Missing-variable behavior is a parity concern in C# notes.

References: `design/gert/sections/03-schema-vnext.tex:562-590`, `design/web-platform/c-sharp-governance-parity.md:485-540`.

### Assertion operator surface

Assertions are structured, not a free-form expression language:

- Types: `contains`, `not_contains`, `matches`, `eq`, `ne`, `lt`, `gt`, `exit_code`, `json_path`.
- However, `subject` and `expected` are template strings in examples.

References: `design/gert/sections/03-schema-vnext.tex:2865-2891`.

### Capture path surface

Capture paths include:

- Fixed streams: `stdout`, `stderr`, `exitCode` / `exit_code`.
- Dot/index paths: `json.items[0].metadata.name`.
- Pipeline-like form: `json.items | length`.
- Tool-runtime dot paths: `stdout.incident.id`.

This surface is not Go-style, but it is under-specified. It needs one portable grammar if non-Go runtimes must produce identical captures.

References: `design/gert/sections/03-schema-vnext.tex:2146-2173`, `design/gert/sections/06-tool-runtime.tex:292-299`.

## Examples and Test Coverage Found

| Surface | Examples / tests found | Notes |
|---|---|---|
| `step.when` | `design/gert/testdata/runbooks/r06-db-migration/schema.yaml:414`; `design/gert/testdata/runbooks/r07-financial-approval/schema.yaml:248` | Uses `==`, `>=` style expressions. |
| Branch `condition` | `design/gert/testdata/runbooks/r02-canary-deploy/schema.yaml:226`; `design/gert/testdata/runbooks/r07-financial-approval/schema.yaml:161`; `design/gert/testdata/runbooks/r10-gdpr-deletion/schema.yaml:393` | Uses `&&`, `||`, comparison, negation. |
| Non-normative `contains` infix | `design/gert/testdata/runbooks/r01-k8s-incident/schema.yaml:152` | Direct portability problem: spec says authors MUST use `str.contains`, but fixture uses `pod_json contains ...`. |
| Assert templating | `design/gert/testdata/runbooks/r14-assert-compensate/schema.yaml:53-58`; `design/gert/testdata/runbooks/r17-tool-transport/schema.yaml:33-101` | Assertions are structured, but operands are Go templates. |
| Iterate over path | `design/gert/testdata/runbooks/r11-iterate-loop/schema.yaml:28-35` | Fixture uses `$.services`, while spec example uses `{{ .service_list }}`. |
| Template interpolation in artifacts | `design/gert/testdata/runbooks/r22-evidence-replay/schema.yaml:88` | Artifact path embeds `{{ .build_id }}`. |
| Tool command interpolation | `design/gert/testdata/tools/echo.tool.yaml`, `jsonrpc-server.tool.yaml`, `mcp-server.tool.yaml` | These use `${GERT_TOOLS_DIR}` environment-style substitution, distinct from Go templates. |
| C# parity test vectors | `design/web-platform/c-sharp-governance-parity.md:485-540` | Confirms Go `text/template` is treated as a portability gap. |

The testing section calls for table-driven tests for expression evaluation but does not provide implementation tests in this repository snapshot: `design/gert/sections/09-testing-and-acceptance.tex:20-21`.

## Distinguishing Go Semantics from Go Implementation

| Finding | Problem? | Why |
|---|---|---|
| Runtime is designed in Go | No | Implementation language alone is not a portability concern. |
| Use of `expr-lang/expr` library | Medium | Safe sandboxed evaluator, but runtime parity requires every implementation to reproduce `expr` semantics or define a strict GERT subset. |
| Conditional syntax using `&&`, `||`, `!`, `==` | Medium | Go/C-style tokens are portable if specified, but they are not neutral YAML/JSON policy syntax and may differ in truthiness/type coercion across runtimes. |
| Explicit Go `text/template` for runbook/tool fields | High | This exposes Go template grammar, data lookup, function maps, missing-key behavior, pipelines, and parse/runtime errors to authors. A non-Go runtime must emulate Go, call Go/WASM, or break behavior. |
| `fromYAML` / `fromJSON` returning `interface{}` | High | The observable traversal/type behavior depends on Go template data semantics unless tightly specified. |
| Fixture use of `contains` infix | High | The normative spec forbids it as non-portable, but examples already rely on it. |
| Capture path mini-language | Medium | Not Go-specific, but currently underspecified and inconsistent (`json.<path>`, `stdout.<path>`, `| length`). |
| `event.filter` map and provider `from:` dispatch | Low | These are literal/prefix contracts, not general expression evaluation. |

## Portability Risks

1. **C# runtime parity is already known to be fragile.** The C# parity document says Go `text/template` has no exact C# equivalent and defines template parity vectors as a gate (`design/web-platform/c-sharp-governance-parity.md:291-336`).
2. **Spec conflict creates implementation ambiguity.** The global expression section says all `when`/`condition`/`until` fields use `expr-lang/expr`, but branch and collector tables still label some fields as Go templates (`design/gert/sections/03-schema-vnext.tex:415-416`, `design/gert/sections/03-schema-vnext.tex:1261`, `design/gert/sections/03-schema-vnext.tex:2531`).
3. **The runbook corpus uses syntax outside the normative subset.** `pod_json contains "CrashLoopBackOff"` appears in fixtures even though the spec requires `str.contains(...)`.
4. **Template interpolation is broad.** The spec intentionally keeps Go template pipelines, custom function maps, and `fromJSON`/`fromYAML`; that is effectively a commitment to Go template compatibility across runtimes.
5. **Capture paths need formalization.** `json.items | length` looks pipeline-like but the evaluator is not specified; different runtimes could reasonably implement different semantics.
6. **Browser/backend dynamic forms can diverge.** `collector.fields[].when` must be evaluated live in UI and authoritatively in the runtime. If this remains Go template or `expr` without a portable browser evaluator, the UI can show a field the backend skips, or vice versa.

## Options to Consider

No recommendation yet; these are directions for team evaluation.

### Option A — Adopt CEL for all boolean expressions

Use CEL for `when`, `condition`, `until`, `include.when`, collector field guards, and possibly assertions. Keep interpolation separate.

- Pros: Portable, typed, multi-language implementations, policy-friendly.
- Cons: Migration required from `expr` syntax; need a compatibility story for existing fixtures; CEL still permits a non-trivial grammar.

### Option B — Define a tiny GERT-native expression language

Specify a minimal grammar: identifiers, string/number/bool literals, `==`, `!=`, comparisons, `and/or/not` or another chosen spelling, parentheses, and allowlisted helper namespaces (`str.*`).

- Pros: Fully portable and auditable; small enough for Go/C#/TS evaluators.
- Cons: GERT must own parser/evaluator conformance tests; authors lose convenience features from `expr`.

### Option C — Allowlist a portable subset and reject everything else

Keep `expr-lang/expr` as the Go implementation detail, but validate authored expressions against a GERT-defined subset and reject `contains` infix, unapproved functions, dynamic dispatch, pipes, and advanced constructs.

- Pros: Smaller migration from current docs; fastest path if `expr` remains in Go.
- Cons: Still risks depending on `expr` edge behavior unless conformance tests are exhaustive.

### Option D — Remove general expression evaluation

Replace expressions with literal/lookup-only controls: structured predicates such as `{var, op, value}`, branch predicates as arrays of clauses, and no free-form templates except simple `${var}` substitution.

- Pros: Maximum portability and governance auditability.
- Cons: Biggest authoring change; runbooks become more verbose; may need more built-in step types for common logic.

## Immediate Follow-up Questions

1. Is the committed design goal now **zero Go template syntax in authored runbooks/tools**, or only **no Go-style boolean expression evaluation**?
2. Should existing fixtures be treated as compatibility commitments or disposable design examples?
3. Should `text/template` interpolation be replaced at the same time as `expr`, or audited as a separate migration?
4. Should dynamic UI guards (`collector.fields[].when`) use the same evaluator as backend guards, with a browser-compatible implementation?
5. Should `capture` path syntax be normalized to JSON Pointer, JSONPath, JMESPath, or a GERT-native path grammar?

## Don's Governance Note

The governance layer is only safe if every runtime evaluates the same runbook the same way. Letting authored runbooks depend on Go template behavior or library-specific `expr` extensions makes policy, audit, replay, and non-Go workers drift. If we move away from Go-style evaluation, the migration must be enforced at parse/plan time, before any step reaches `RunHandle.Next`.