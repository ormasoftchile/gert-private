# Scaffolding generator for design/gert/conformance/tv-enum.yaml, per Barbara's
# ratified ruling .squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md
# (AR-ENUM-1..15, §Handoff -> Tess). Mirrors the intent of the now-deleted
# design/gert/scripts/gen_pkg_vectors.py: a disposable authoring aid, safe to
# delete once the corpus is reviewed/stable. Not part of the verified corpus
# itself -- design/gert/scripts/verify_corpus.py is the actual conformance
# gate and does not import this file.
import json
import unicodedata as ud

OUT = []


def add(id_, category, description, input_, variables, expected, note=None, tags=None):
    v = {
        "id": id_,
        "category": category,
        "description": description,
        "input": input_,
        "variables": variables,
    }
    v["expected"] = expected
    if note:
        v["note"] = note
    if tags:
        v["tags"] = tags
    OUT.append(v)


# ---------------------------------------------------------------------------
# Fixture helpers. All runbooks are self-contained (no external files) unless
# a tool-package scenario is required (ENUM-PLAN literal-arg / ENUM-RUNTIME
# provider+substitution / ENUM-SUBST / ENUM-MOCK / some ENUM-TRACE vectors).
# ---------------------------------------------------------------------------

def runbook_input(input_yaml_block, extra_top="", flow_extra=""):
    return f"""apiVersion: runbook/v1
id: r
name: r
{extra_top}inputs:
{input_yaml_block}
flow:
{flow_extra}  - step:
      id: end
      type: end
      outcome: {{category: success, code: ok}}
"""


def runbook_output(output_yaml_block, flow_block):
    return f"""apiVersion: runbook/v1
id: r
name: r
outputs:
{output_yaml_block}
flow:
{flow_block}  - step:
      id: end
      type: end
      outcome: {{category: success, code: ok}}
"""


def indent(s, n=2):
    pad = " " * n
    return "\n".join(pad + line if line else line for line in s.splitlines())


PKG_YAML = """apiVersion: tool-package/v1
meta:
  name: acme.enum-tools
  version: "1.0.0"
exports:
  tools:
    - id: kubectl
      path: tools/kubectl.tool.yaml
"""


def tool_yaml(strategy_enum_caller, extra_args="", extra_outputs=""):
    """A tool.yaml with a process-backed action and a substituted
    'drain-node' action carrying an enum-constrained 'strategy' arg."""
    return f"""apiVersion: tool/v1
meta:
  name: kubectl
  version: "1.0.0"
transport:
  mode: stdio
governance:
  allowed-environments: ["real"]
  requires-approval: false
actions:
  - name: get-nodes
    argv: ["get", "nodes"]
    args: {{}}
  - name: drain-node
    execute:
      kind: runbook
      path: ../runbooks/drain-node.yaml
    args:
      node: {{type: string, required: true}}
      strategy:
        type: string
        required: false
        default: graceful
        enum: {strategy_enum_caller}
{extra_args}
    outputs:
      drained: {{type: boolean}}
{extra_outputs}
"""


def substitute_runbook_yaml(strategy_enum_sub, output_extra=""):
    return f"""apiVersion: runbook/v1
id: acme.enum-tools/drain-node
name: drain-node
inputs:
  node: {{type: string, required: true, from: context}}
  strategy:
    type: string
    required: false
    from: context
    default: graceful
    enum: {strategy_enum_sub}
outputs:
  drained: {{type: boolean, value: "step.drain.json.drained"}}
{output_extra}
flow:
  - step:
      id: drain
      type: cli
      command: bash
      args: ["-c", "echo '{{\\"drained\\": true}}'"]
      capture:
        drain: json
"""


def caller_runbook_yaml(node_value="node-1", strategy_value=None, capture_extra=""):
    strategy_line = f"\n          strategy: {strategy_value}" if strategy_value is not None else ""
    return f"""apiVersion: runbook/v1
id: r
name: r
requires:
  - package: acme.enum-tools
    version: "^1.0.0"
    path: ./vendor/acme-enum-tools
toolRefs:
  - name: kubectl
    package: acme.enum-tools
flow:
  - step:
      id: drain
      type: tool
      tool:
        name: kubectl
        action: drain-node
        args:
          node: {node_value}{strategy_line}
      capture:
        drained: outputs.drained
{capture_extra}  - step:
      id: end
      type: end
      outcome: {{category: success, code: ok}}
"""


def pkg_scenario(strategy_enum_caller, strategy_enum_sub, node_value="node-1", strategy_value=None,
                  extra_args="", extra_outputs="", output_extra=""):
    return {
        "runbook.yaml": caller_runbook_yaml(node_value=node_value, strategy_value=strategy_value),
        "vendor/acme-enum-tools/gert-package.yaml": PKG_YAML,
        "vendor/acme-enum-tools/tools/kubectl.tool.yaml": tool_yaml(
            strategy_enum_caller, extra_args=extra_args, extra_outputs=extra_outputs),
        "vendor/acme-enum-tools/runbooks/drain-node.yaml": substitute_runbook_yaml(
            strategy_enum_sub, output_extra=output_extra),
    }


def err(code):
    return {"error_class": "ENUM", "error_code": code}


def pkg013():
    return {"error": {"code": "PKG-013"}}


# ===========================================================================
# ENUM-DECL (min 12) -- well-formedness across all four sites.
# ===========================================================================

add(
    "TV-ENUM-DECL-001", "ENUM-DECL",
    "enum on a runbook input (S3) declared type: number is ENUM-001.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: number\n    required: false\n    enum: [\"1\", \"2\"]\n")},
    err("ENUM-001"),
    note="AR-ENUM-2: enum valid only when the declaration resolves to type: string.",
)

add(
    "TV-ENUM-DECL-002", "ENUM-DECL",
    "enum on a runbook input (S3) declared type: secret is ENUM-001 (C1): a secret's value domain must not be enumerable.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  api_key:\n    type: secret\n    required: false\n    enum: [\"tokenA\", \"tokenB\"]\n")},
    err("ENUM-001"),
    note="C1: enum forbidden on type: secret regardless of redact/sensitive_inputs status.",
)

add(
    "TV-ENUM-DECL-003", "ENUM-DECL",
    "enum on a tool action arg (S1) declared type: boolean is ENUM-001.",
    "runbook.yaml",
    {
        "runbook.yaml": caller_runbook_yaml(strategy_value=None),
        "vendor/acme-enum-tools/gert-package.yaml": PKG_YAML,
        "vendor/acme-enum-tools/tools/kubectl.tool.yaml": (
            "apiVersion: tool/v1\nmeta:\n  name: kubectl\n  version: \"1.0.0\"\n"
            "transport:\n  mode: stdio\ngovernance:\n  allowed-environments: [\"real\"]\n"
            "  requires-approval: false\nactions:\n  - name: get-nodes\n"
            "    argv: [\"get\", \"nodes\"]\n    args: {}\n  - name: drain-node\n"
            "    execute:\n      kind: runbook\n      path: ../runbooks/drain-node.yaml\n"
            "    args:\n      node: {type: string, required: true}\n"
            "      force:\n        type: boolean\n        required: false\n"
            "        enum: [\"true\", \"false\"]\n    outputs:\n      drained: {type: boolean}\n"
        ),
        "vendor/acme-enum-tools/runbooks/drain-node.yaml": substitute_runbook_yaml('["graceful", "force"]'),
    },
    err("ENUM-001"),
    note="S1: tool action args.<name>. Governing section: 06-tool-runtime.tex Tool Definition Schema.",
)

add(
    "TV-ENUM-DECL-004", "ENUM-DECL",
    "enum value that is a YAML mapping (not a sequence) at a runbook input (S3) is ENUM-002.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: {a: 1, b: 2}\n")},
    err("ENUM-002"),
    note="AR-ENUM-3(1): enum MUST be a YAML sequence.",
)

add(
    "TV-ENUM-DECL-005", "ENUM-DECL",
    "An empty enum sequence ([]) at a runbook input (S3) is ENUM-002 (minItems: 1 violated).",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: []\n")},
    err("ENUM-002"),
    note="AR-ENUM-3(2): a zero-member domain is always an authoring mistake, never 'no values allowed'.",
)

add(
    "TV-ENUM-DECL-006", "ENUM-DECL",
    "Unquoted enum members yes/no resolve to YAML 1.2 core-schema booleans, not strings: ENUM-002, no coercion.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  answer:\n    type: string\n    required: false\n    enum: [yes, no]\n")},
    err("ENUM-002"),
    note="AR-ENUM-3(3): the single most likely real-world authoring trap. Authors must quote: enum: [\"yes\", \"no\"].",
    tags=["yaml-1.2-core-schema"],
)

add(
    "TV-ENUM-DECL-007", "ENUM-DECL",
    "An enum member that is itself a nested YAML sequence is ENUM-002.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [[\"prod\"], \"stage\"]\n")},
    err("ENUM-002"),
    note="AR-ENUM-3(4): nested sequences/mappings as items are rejected.",
)

add(
    "TV-ENUM-DECL-008", "ENUM-DECL",
    "An enum member carrying an explicit YAML tag (!!int on a digit string) is ENUM-002: no coercion, no tag override.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [!!int \"1\", \"2\"]\n")},
    err("ENUM-002"),
    note="AR-ENUM-3(4): explicit tags on items are rejected regardless of the tag's target type.",
)

add(
    "TV-ENUM-DECL-009", "ENUM-DECL",
    "An enum member that is the empty string is ENUM-003.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [\"prod\", \"\"]\n")},
    err("ENUM-003"),
    note="AR-ENUM-3(5): empty string is indistinguishable from 'unset' across from: env/prompt/CLI sourcing.",
)

add(
    "TV-ENUM-DECL-010", "ENUM-DECL",
    "An enum member that is whitespace-only is ENUM-003.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [\"prod\", \"   \"]\n")},
    err("ENUM-003"),
    note="AR-ENUM-3(5): whitespace-only member.",
)

add(
    "TV-ENUM-DECL-011", "ENUM-DECL",
    "An enum member with leading/trailing whitespace (' prod') is ENUM-003: an invisible distinction humans reviewing an incident artifact must not have to spot.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [\" prod\", \"stage\"]\n")},
    err("ENUM-003"),
    note="AR-ENUM-3(5).",
)

add(
    "TV-ENUM-DECL-012", "ENUM-DECL",
    "Exact-duplicate enum members ('prod' twice) are ENUM-004.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [\"prod\", \"stage\", \"prod\"]\n")},
    err("ENUM-004"),
    note="AR-ENUM-3(6): members MUST be pairwise distinct after NFC normalisation (trivial case: exact duplicates).",
)

add(
    "TV-ENUM-DECL-013", "ENUM-DECL",
    "Case-only-distinct members ('Prod'/'prod') are legal and match case-sensitively; a validator MUST emit ENUM-W001.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    default: prod\n    enum: [\"Prod\", \"prod\"]\n")},
    {"value": "prod", "warnings": ["ENUM-W001"]},
    note="AR-ENUM-4: case-only-distinct pairs are NOT PKG-018 (no filesystem substrate); non-fatal warning only.",
)

add(
    "TV-ENUM-DECL-014", "ENUM-DECL",
    "enum on a runbook output (S4) declared type: integer is ENUM-001.",
    "runbook.yaml",
    {"runbook.yaml": runbook_output(
        "  drain_state:\n    type: integer\n    enum: [\"1\", \"2\"]\n",
        "  - step:\n      id: drain\n      type: cli\n      command: bash\n      args: [\"-c\", \"echo drained\"]\n      capture:\n        drain: stdout\n",
    )},
    err("ENUM-001"),
    note="S4: runbook outputs.<name>. Governing section: 03-schema-vnext.tex Output Declarations.",
)


# ===========================================================================
# ENUM-UNICODE (min 8)
# ===========================================================================

NFC_E = ud.normalize("NFC", "cafe\u0301")   # "café" precomposed (U+00E9)
NFD_E = ud.normalize("NFD", NFC_E)          # "cafe" + combining acute (U+0301)
assert NFC_E != NFD_E
assert ud.normalize("NFC", NFD_E) == NFC_E

BIDI_MEMBER = "prod\u202E"           # trailing RIGHT-TO-LEFT OVERRIDE (U+202E)
CONTROL_MEMBER = "prod\u0007"        # BEL (U+0007), a C0 control character
COMBINING_A = "a\u0301"              # 'a' + combining acute -- NOT the same
                                      # grapheme as the precomposed 'á'
                                      # (U+00E1); used for the "combining-mark
                                      # equality" vector below: after NFC
                                      # normalisation 'a\u0301' *is* 'á'.
PRECOMPOSED_A = ud.normalize("NFC", COMBINING_A)
assert PRECOMPOSED_A == "\u00e1"

add(
    "TV-ENUM-DECL-015", "ENUM-UNICODE",
    "An enum member in NFD (decomposed combining-acute form) instead of NFC is ENUM-005: declared members MUST already be NFC.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        f"  city:\n    type: string\n    required: false\n    enum: [\"{NFD_E}\", \"stage\"]\n")},
    err("ENUM-005"),
    note="AR-ENUM-4: normalisation is asymmetric -- the FILE itself must already be NFC.",
)

add(
    "TV-ENUM-UNICODE-001", "ENUM-UNICODE",
    "An NFD-form plan-time literal candidate matches an NFC-declared member: candidates are NFC-normalised before comparison, so this passes (asymmetric normalisation).",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller=json.dumps(["graceful", "force"]),
        strategy_enum_sub=json.dumps(["graceful", "force"]),
        node_value=json.dumps(NFD_E),
    ),
    {"value": NFD_E},
    note=(
        "AR-ENUM-4: 'declared members MUST already be NFC; candidate values from outside the file "
        "(env, prompt, provider, tool stdout, caller bindings) ARE NFC-normalised before comparison'. "
        "The literal caller-supplied `node` value here is a candidate, not a declared member, so its "
        "NFD form is normalised for the ENUM-007 plan-time comparison and passes; the stored bound "
        "value itself is not rewritten (normalisation is comparison-only), hence expected.value is "
        "still the original NFD codepoint sequence. Uses the 'node' arg (unconstrained by enum) purely "
        "as the vehicle for a deterministic Unicode-bearing plan-time literal; the enum-relevant "
        "assertion is that plan validation does not raise ENUM-007 for this scenario."
    ),
)

add(
    "TV-ENUM-UNICODE-002", "ENUM-UNICODE",
    "Invalid-UTF-8 enum members (ENUM-005) cannot be represented in this YAML/JSON corpus: the file itself must be valid UTF-8 to load. Pinned as a contract-level placeholder.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [\"prod\", \"stage\"]\n")},
    {"value": "prod"},
    note=(
        "UNSUPPORTED-UNDER-CORPUS-FORMAT: ENUM-005's 'invalid UTF-8' clause has no vector-representable "
        "witness -- design/gert/conformance/vector.schema.json vectors are YAML/JSON text, and both YAML "
        "and JSON require the document to be well-formed Unicode text before any parser sees it; an "
        "invalid byte sequence would fail file decoding before enum well-formedness checks ever run. "
        "This vector is a deterministic placeholder pinning that fact rather than inventing a raw-bytes "
        "field the schema does not define. A conformance harness implementer MUST instead exercise this "
        "condition, if desired, via a unit test that feeds raw non-UTF-8 bytes directly to the "
        "declaration parser -- outside this corpus format."
    ),
    tags=["untestable-in-corpus-format"],
)

add(
    "TV-ENUM-UNICODE-003", "ENUM-UNICODE",
    "An enum member containing a bidi/format control character (U+202E RIGHT-TO-LEFT OVERRIDE) is ENUM-005.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        f"  env_name:\n    type: string\n    required: false\n    enum: [\"{BIDI_MEMBER}\", \"stage\"]\n")},
    err("ENUM-005"),
    note="AR-ENUM-3(7)/AR-ENUM-4: bidi/format controls U+200E, U+200F, U+202A-U+202E, U+2066-U+2069 are rejected.",
)

add(
    "TV-ENUM-UNICODE-004", "ENUM-UNICODE",
    "An enum member containing a Unicode control character (U+0007 BEL, a C0 control) is ENUM-005.",
    "runbook.yaml",
    # The BEL control character (U+0007) is written as the YAML double-quoted
    # escape \a (literal backslash-a), not as a raw embedded byte: YAML's own
    # c-printable character set (spec production c-printable) excludes most
    # C0 controls from appearing unescaped in a document at all, so a raw
    # BEL byte would make even the surrounding YAML block ill-formed before
    # enum well-formedness is ever evaluated. \a is the normative way to
    # express this candidate member in a YAML document.
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [\"prod\\a\", \"stage\"]\n")},
    err("ENUM-005"),
    note="AR-ENUM-3(7): control characters U+0000-U+001F and U+007F are rejected.",
)

add(
    "TV-ENUM-UNICODE-005", "ENUM-UNICODE",
    "A combining-acute candidate ('a'+U+0301) NFC-normalises to the precomposed form and matches the declared member: combining and precomposed forms are the same member.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller=json.dumps([PRECOMPOSED_A, "force"]),
        strategy_enum_sub=json.dumps([PRECOMPOSED_A, "force"]),
        strategy_value=json.dumps(COMBINING_A),
    ),
    {"value": COMBINING_A},
    note=(
        "AR-ENUM-4: comparison is codepoint-wise equality on NFC forms; the candidate "
        "'a\\u0301' NFC-normalises to '\\u00e1' and matches the declared member exactly, so binding "
        "succeeds (no ENUM-008). The stored value is the original candidate ('a\\u0301'); normalisation "
        "is comparison-only."
    ),
)

add(
    "TV-ENUM-UNICODE-006", "ENUM-UNICODE",
    "Case-only-distinct members after NFC ('Café'/'café') are legal, match case-sensitively, and MUST emit ENUM-W001.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        f"  env_name:\n    type: string\n    required: false\n    default: \"{NFC_E}\"\n    enum: [\"Caf\\u00e9\", \"{NFC_E}\"]\n")},
    {"value": NFC_E, "warnings": ["ENUM-W001"]},
    note="AR-ENUM-4: case-only-distinct pairs are evaluated on their NFC forms, same rule as the ASCII case (ENUM-DECL-013).",
)

add(
    "TV-ENUM-UNICODE-007", "ENUM-UNICODE",
    "Canonical order (used for deterministic diagnostics/cross-runtime parity) is ascending codepoint order of NFC member forms, independent of declared order.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    default: stage\n    enum: [\"stage\", \"prod\", \"dev\"]\n")},
    {"value": "stage"},
    note=(
        "AR-ENUM-5: declared order here is [stage, prod, dev] (preserved for UI/diagnostics/plan carriage); "
        "canonical order for any textual/byte comparison (deterministic diagnostics, cross-runtime parity) "
        "is ascending codepoint order of NFC forms: [dev, prod, stage]. Both orderings are normative and "
        "distinct; this vector's declared member list is deliberately not already in canonical order, "
        "pinning that a conforming implementation MUST NOT silently re-sort 'enum:' as authored for "
        "declared-order-consuming surfaces (adapter/UI rendering, --output=json metadata, plan carriage) "
        "while still being able to derive canonical order on demand for comparison."
    ),
)


# ===========================================================================
# ENUM-DEFAULT (min 5) -- across S1/S3/S4, valid default, optional-absent.
# ===========================================================================

add(
    "TV-ENUM-DEFAULT-001", "ENUM-DEFAULT",
    "A runbook input (S3) default that is not a declared enum member is ENUM-006 at plan time.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    default: qa\n    enum: [\"prod\", \"stage\"]\n")},
    err("ENUM-006"),
    note="AR-ENUM-6: default present with enum present -> default MUST be a member (exact, NFC).",
)

add(
    "TV-ENUM-DEFAULT-002", "ENUM-DEFAULT",
    "A tool action arg (S1) default that is not a declared enum member is ENUM-006 at plan time.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller='["graceful", "force"]',
        strategy_enum_sub='["graceful", "force"]',
        extra_args="",
    ) | {
        "vendor/acme-enum-tools/tools/kubectl.tool.yaml": (
            "apiVersion: tool/v1\nmeta:\n  name: kubectl\n  version: \"1.0.0\"\n"
            "transport:\n  mode: stdio\ngovernance:\n  allowed-environments: [\"real\"]\n"
            "  requires-approval: false\nactions:\n  - name: get-nodes\n"
            "    argv: [\"get\", \"nodes\"]\n    args: {}\n  - name: drain-node\n"
            "    execute:\n      kind: runbook\n      path: ../runbooks/drain-node.yaml\n"
            "    args:\n      node: {type: string, required: true}\n"
            "      strategy:\n        type: string\n        required: false\n"
            "        default: aggressive\n        enum: [\"graceful\", \"force\"]\n"
            "    outputs:\n      drained: {type: boolean}\n"
        ),
    },
    err("ENUM-006"),
    note="S1: tool action args.<name>.default MUST be a member (AR-ENUM-6), independent of S3.",
)

add(
    "TV-ENUM-DEFAULT-003", "ENUM-DEFAULT",
    "A substituted action's declared output default (S2, tool.yaml outputs.<name>) not a member is ENUM-006 at plan time, independent of the substitute's own ENUM-009 check.",
    "runbook.yaml",
    pkg_scenario('["graceful", "force"]', '["graceful", "force"]') | {
        "vendor/acme-enum-tools/tools/kubectl.tool.yaml": (
            "apiVersion: tool/v1\nmeta:\n  name: kubectl\n  version: \"1.0.0\"\n"
            "transport:\n  mode: stdio\ngovernance:\n  allowed-environments: [\"real\"]\n"
            "  requires-approval: false\nactions:\n  - name: get-nodes\n"
            "    argv: [\"get\", \"nodes\"]\n    args: {}\n  - name: drain-node\n"
            "    execute:\n      kind: runbook\n      path: ../runbooks/drain-node.yaml\n"
            "    args:\n      node: {type: string, required: true}\n"
            "      strategy:\n        type: string\n        required: false\n"
            "        default: graceful\n        enum: [\"graceful\", \"force\"]\n"
            "    outputs:\n      drain_state:\n        type: string\n"
            "        default: unknown\n        enum: [\"drained\", \"cordoned\"]\n"
        ),
    },
    err("ENUM-006"),
    note=(
        "AR-ENUM-6: 'Applies at S1, S3, S4 and to a substituted action's declared output default.' S2 "
        "(tool action outputs.<name>, design/gert/sections/06-tool-runtime.tex) is the fourth of the four "
        "sites and, per C3, has no governing tool.v1.schema.json to structurally gate 'default' at the "
        "schema layer -- so this vector's fixture (unlike the runbook Input/Output $defs, which DO enforce "
        "structural shape) is a case where the ENUM-006 check itself is the only gate, exercised here "
        "purely at the semantic/plan-validation layer."
    ),
)

add(
    "TV-ENUM-DEFAULT-004", "ENUM-DEFAULT",
    "A runbook input (S3) default that IS a declared enum member validates successfully with no warning.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    default: prod\n    enum: [\"prod\", \"stage\"]\n")},
    {"value": "prod"},
    note="AR-ENUM-6 positive case.",
)

add(
    "TV-ENUM-DEFAULT-005", "ENUM-DEFAULT",
    "An optional, defaultless, unsupplied enum-constrained runbook input is absent, not a violation: enum constrains values, not presence, and does not imply required: true.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    enum: [\"prod\", \"stage\"]\n")},
    {"value": None},
    note=(
        "AR-ENUM-6: 'enum constrains values, not presence... an optional, defaultless, unsupplied "
        "declaration is absent; absence is not an enum violation.' expected.value models the absent "
        "input's resolved context value as PJVM null (there is no 'default' to surface); this is "
        "deliberately distinct from ENUM-DECL's rejection of null as a MEMBER (ENUM-002) -- absence of "
        "the whole input is not the same thing as a null enum member."
    ),
)


add(
    "TV-ENUM-DEFAULT-006", "ENUM-DEFAULT",
    "A runbook-level output (S4, runbook.v1.schema.json $defs.Output) declaring 'default' cannot be represented: the Output schema has no 'default' property, unlike Input's.",
    "runbook.yaml",
    {"runbook.yaml": runbook_output(
        "  drain_state:\n    type: string\n    enum: [\"drained\", \"cordoned\"]\n"
        "    value: \"step.drain.json.drain_state\"\n",
        "  - step:\n      id: drain\n      type: cli\n      command: bash\n"
        "      args: [\"-c\", \"echo '{\\\"drain_state\\\": \\\"drained\\\"}'\"]\n"
        "      capture:\n        drain: json\n",
    )},
    {"value": "drained"},
    note=(
        "UNSUPPORTED-UNDER-CORPUS-FORMAT / schema gap (adjacent, not part of AR-ENUM-1..15's own scope, "
        "surfaced while authoring this corpus): design/gert/schemas/runbook.v1.schema.json $defs.Output "
        "has no 'default' property (only type/description/value/enum), while $defs.Input does. AR-ENUM-6 "
        "states default-in-enum 'Applies at S1, S3, S4', and design/gert/sections/06-tool-runtime.tex "
        "§Output contract's PKG-027 prose describes a substitute output that may 'declare a default', "
        "which is itself an S4-shaped (runbook Output) declaration for the substitute's own outputs: "
        "map. A literal 'outputs.<name>.default:' fixture at the runbook-Output site is therefore not "
        "schema-representable today. This vector is a deterministic placeholder: it exercises the S4 "
        "site's enum production-time check on its actually-producible path (no default needed, since "
        "the step genuinely produces a member value) and documents the gap in 'note' rather than "
        "inventing an unsupported 'default' field on Output. Flagged to Edith/Barbara "
        "(.squad/decisions/inbox/tess-enum-corpus-notes.md) as a schema-completeness question, not an "
        "enum-semantics ambiguity: AR-ENUM-6 remains binding and is exercised at S1 (DEFAULT-002), S2 "
        "(DEFAULT-003), and S3 (DEFAULT-001) with a literal 'default:' field; S4 needs the same field "
        "added to $defs.Output before an equivalent literal fixture is possible."
    ),
    tags=["untestable-in-corpus-format", "schema-gap"],
)


# ===========================================================================
# ENUM-PLAN (min 7)
# ===========================================================================

add(
    "TV-ENUM-PLAN-001", "ENUM-PLAN",
    "A statically-known literal step.tool.args value (no GIS interpolation) bound to an enum-constrained arg, not a declared member, is ENUM-007 at plan time.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller='["graceful", "force"]',
        strategy_enum_sub='["graceful", "force"]',
        strategy_value='"aggressive"',
    ),
    err("ENUM-007"),
    note="AR-ENUM-7 P4: a constant step.tool.args value with no GIS interpolation is checked against the enum at plan time.",
)

add(
    "TV-ENUM-PLAN-002", "ENUM-PLAN",
    "A statically-known literal step.tool.args value that IS a declared enum member passes plan-time validation.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller='["graceful", "force"]',
        strategy_enum_sub='["graceful", "force"]',
        strategy_value='"force"',
    ),
    {"value": "force"},
    note="AR-ENUM-7 P4 positive case.",
)

add(
    "TV-ENUM-PLAN-003", "ENUM-PLAN",
    "A GIS-interpolated step.tool.args value (${strategy}) is NEVER checked against the enum at plan time: no partial evaluation or constant folding; a designed non-check, not a defect.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller='["graceful", "force"]',
        strategy_enum_sub='["graceful", "force"]',
        strategy_value='"${strategy}"',
    ),
    {"value": "${strategy}"},
    note=(
        "AR-ENUM-7: 'Explicitly NOT plan time: any value containing GIS interpolation. There is no "
        "partial evaluation... A runtime-only enum failure is an accepted, designed outcome -- not a "
        "defect.' expected.value models the plan validating successfully with the interpolated literal "
        "left unresolved (plan time performs no GIS evaluation of it); whether the eventual runtime value "
        "of ${strategy} is a member is exercised separately by ENUM-RUNTIME."
    ),
)

add(
    "TV-ENUM-PLAN-004", "ENUM-PLAN",
    "An enum-constrained decl on an unreachable step (when: always false) is still validated: well-formedness and literal-member checks are unconditional on execution path.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller='["graceful", "force"]',
        strategy_enum_sub='["graceful", "force"]',
        strategy_value='"aggressive"',
        node_value='"node-1"',
    ) | {
        "runbook.yaml": (
            "apiVersion: runbook/v1\nid: r\nname: r\n"
            "requires:\n  - package: acme.enum-tools\n    version: \"^1.0.0\"\n"
            "    path: ./vendor/acme-enum-tools\n"
            "toolRefs:\n  - name: kubectl\n    package: acme.enum-tools\n"
            "flow:\n  - step:\n      id: drain\n      type: tool\n"
            "      when: \"false\"\n      tool:\n        name: kubectl\n"
            "        action: drain-node\n        args:\n          node: \"node-1\"\n"
            "          strategy: \"aggressive\"\n      capture:\n        drained: outputs.drained\n"
            "  - step:\n      id: end\n      type: end\n"
            "      outcome: {category: success, code: ok}\n"
        ),
    },
    err("ENUM-007"),
    note=(
        "AR-ENUM-3 preamble ('validity is not conditional on execution path') and AR-ENUM-7 P1/P4: "
        "well-formedness and literal-member checks fire for EVERY declaration in the closure, "
        "'reachable or not'. The 'when: \"false\"' guard on the drain step does not suppress the "
        "ENUM-007 plan-time failure for its statically-known off-enum literal."
    ),
)

add(
    "TV-ENUM-PLAN-005", "ENUM-PLAN",
    "enum well-formedness (ENUM-002) of a lazily materialised include is validated when that file is materialised, per existing lazy-include semantics: not deferred, not skipped.",
    "runbook.yaml",
    {
        "runbook.yaml": (
            "apiVersion: runbook/v1\nid: r\nname: r\nexpand: lazy\n"
            "imports:\n  child: ./child.runbook.yaml\n"
            "flow:\n  - step:\n      id: go\n      type: include\n"
            "      include:\n        runbook: child\n"
            "        expand: lazy\n"
            "  - step:\n      id: end\n      type: end\n"
            "      outcome: {category: success, code: ok}\n"
        ),
        "child.runbook.yaml": runbook_input(
            "  env_name:\n    type: string\n    required: false\n    enum: [yes, no]\n"),
    },
    err("ENUM-002"),
    note=(
        "AR-ENUM-9: 'Lazy includes: enum well-formedness of a lazily materialised file is validated "
        "when that file is materialised, consistent with existing lazy-include semantics. This creates "
        "no late-binding hole... an enum is inert file-local data with no global namespace.' This vector "
        "pins that materialisation (whenever the planner performs it for a lazy include) still surfaces "
        "the child's ENUM-002, not a silent pass."
    ),
)

add(
    "TV-ENUM-PLAN-006", "ENUM-PLAN",
    "A parent and included child declaring the SAME variable name with DIFFERENT enum sets is not a conflict: enum is declaration-local, never inherited/merged across an include.",
    "runbook.yaml",
    {
        "runbook.yaml": (
            "apiVersion: runbook/v1\nid: r\nname: r\n"
            "inputs:\n  strategy:\n    type: string\n    required: false\n"
            "    default: fast\n    enum: [\"fast\", \"slow\"]\n"
            "imports:\n  child: ./child.runbook.yaml\n"
            "flow:\n  - step:\n      id: go\n      type: include\n"
            "      include:\n        runbook: child\n"
            "  - step:\n      id: end\n      type: end\n"
            "      outcome: {category: success, code: ok}\n"
        ),
        "child.runbook.yaml": (
            "apiVersion: runbook/v1\nid: child\nname: child\n"
            "inputs:\n  strategy:\n    type: string\n    required: false\n"
            "    default: graceful\n    enum: [\"graceful\", \"force\"]\n"
            "flow:\n  - step:\n      id: end\n      type: end\n"
            "      outcome: {category: success, code: ok}\n"
        ),
    },
    {"value": "fast"},
    note=(
        "AR-ENUM-9: each declaration constrains only its own binding site. The parent's own 'strategy' "
        "input resolves to its own default 'fast' under its own enum ['fast','slow']; the child's "
        "differently-enum-constrained 'strategy' is a wholly separate declaration. No PKG-013, no "
        "ENUM-*, no cross-file intersection error of any kind."
    ),
)

add(
    "TV-ENUM-PLAN-007", "ENUM-PLAN",
    "Dry-run mode validates all plan-time enum checks (well-formedness, defaults, static literals) plus any binding it actually performs, without executing side-effecting steps.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller='["graceful", "force"]',
        strategy_enum_sub='["graceful", "force"]',
        strategy_value='"aggressive"',
    ),
    err("ENUM-007"),
    note=(
        "AR-ENUM-7: 'Dry-run validates everything plan-time plus any binding it actually performs; it "
        "does not execute side-effecting steps, so runtime checks fire only for bindings that occur.' "
        "The ENUM-007 static-literal check is plan-time and therefore fires identically whether the "
        "eventual run mode is real, dry-run, or replay -- this vector pins that dry-run does not weaken "
        "or skip it."
    ),
    tags=["dry-run"],
)


# ===========================================================================
# ENUM-RUNTIME (min 8)
# ===========================================================================

add(
    "TV-ENUM-RUNTIME-001", "ENUM-RUNTIME",
    "A materialised value bound from from: env that is not a declared member is ENUM-008 at the moment of binding.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: true\n    from: env\n    enum: [\"prod\", \"stage\"]\n",
        flow_extra="",
    )},
    err("ENUM-008"),
    note="AR-ENUM-7: from: env is one of the enumerated binding sites for ENUM-008.",
)

add(
    "TV-ENUM-RUNTIME-002", "ENUM-RUNTIME",
    "A materialised value bound from: prompt that is not a declared member is ENUM-008; the operator is re-prompted per the existing collector retry contract where one applies, otherwise the run fails.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: true\n    from: prompt\n    enum: [\"prod\", \"stage\"]\n")},
    err("ENUM-008"),
    note="AR-ENUM-7: from: prompt binding site.",
)

add(
    "TV-ENUM-RUNTIME-003", "ENUM-RUNTIME",
    "A materialised value bound from a provider field (from: <provider>.<field>) that is not a declared member is ENUM-008.",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: true\n    from: myprovider.region\n    enum: [\"prod\", \"stage\"]\n")},
    err("ENUM-008"),
    note=(
        "AR-ENUM-7: 'from: <provider>.<field>' is one of the explicitly enumerated ENUM-008 binding "
        "sites, and 03-schema-vnext.tex §Input Declarations documents this 'from:' form as normative "
        "prose. Pre-existing, already-ratified schema drift D3 "
        "(.squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md §16, 'separate "
        "ticket, same reasoning as D2') notes that runbook.v1.schema.json's current $defs.Input.from "
        "enum is structurally narrower (['prompt','env','context']) than the spec prose "
        "(env.<NAME>, file.<path>, <provider>.<field>, var.<name>); this vector uses the spec-normative "
        "form per the schema's own documented boundary ('spec sections are updated first... schemas "
        "follow', design/gert/schemas/README.md) and will only validate against runbook.v1.schema.json "
        "once D3 is separately closed -- it already validates against this corpus's own "
        "vector.schema.json, which is the actual conformance gate."
    ),
)

add(
    "TV-ENUM-RUNTIME-004", "ENUM-RUNTIME",
    "A caller/context-bound value at a substituted action's input (S1->substitute's S3) that is not a declared member is ENUM-008.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller='["graceful", "force"]',
        strategy_enum_sub='["graceful", "force", "aggressive"]',
        strategy_value='"${strategy}"',
    ),
    err("ENUM-008"),
    note=(
        "AR-ENUM-7 + AR-ENUM-8: caller and substitute here deliberately declare UNEQUAL enum sets purely "
        "to model the runtime binding path in isolation (this is not an ENUM-SUBST/PKG-013 vector: the "
        "'from context' binding at the substitute's own 'strategy' input receives '${strategy}' at "
        "runtime; the point under test is single-site runtime membership at the moment of binding, "
        "using an interpolated value the caller resolves to a non-member of the SUBSTITUTE's declared "
        "set). Sites: tool action arg binding (S1), here observed at the substitute boundary."
    ),
    tags=["caller-binding"],
)

add(
    "TV-ENUM-RUNTIME-005", "ENUM-RUNTIME",
    "A substituted action's declared output (S2) resolving to a non-member value at production time is ENUM-009, before the value crosses the caller boundary or reaches outputs.<name> capture.",
    "runbook.yaml",
    pkg_scenario(
        strategy_enum_caller='["graceful", "force"]',
        strategy_enum_sub='["graceful", "force"]',
        extra_outputs="",
    ) | {
        "vendor/acme-enum-tools/tools/kubectl.tool.yaml": (
            "apiVersion: tool/v1\nmeta:\n  name: kubectl\n  version: \"1.0.0\"\n"
            "transport:\n  mode: stdio\ngovernance:\n  allowed-environments: [\"real\"]\n"
            "  requires-approval: false\nactions:\n  - name: get-nodes\n"
            "    argv: [\"get\", \"nodes\"]\n    args: {}\n  - name: drain-node\n"
            "    execute:\n      kind: runbook\n      path: ../runbooks/drain-node.yaml\n"
            "    args:\n      node: {type: string, required: true}\n"
            "      strategy:\n        type: string\n        required: false\n"
            "        default: graceful\n        enum: [\"graceful\", \"force\"]\n"
            "    outputs:\n      drain_state: {type: string, enum: [\"drained\", \"cordoned\"]}\n"
        ),
        "vendor/acme-enum-tools/runbooks/drain-node.yaml": (
            "apiVersion: runbook/v1\nid: acme.enum-tools/drain-node\nname: drain-node\n"
            "inputs:\n  node: {type: string, required: true, from: context}\n"
            "  strategy:\n    type: string\n    required: false\n    from: context\n"
            "    default: graceful\n    enum: [\"graceful\", \"force\"]\n"
            "outputs:\n  drain_state:\n    type: string\n    enum: [\"drained\", \"cordoned\"]\n"
            "    value: \"step.drain.json.drain_state\"\n"
            "flow:\n  - step:\n      id: drain\n      type: cli\n      command: bash\n"
            "      args: [\"-c\", \"echo '{\\\"drain_state\\\": \\\"pending\\\"}'\"]\n"
            "      capture:\n        drain: json\n"
        ),
        "runbook.yaml": (
            "apiVersion: runbook/v1\nid: r\nname: r\n"
            "requires:\n  - package: acme.enum-tools\n    version: \"^1.0.0\"\n"
            "    path: ./vendor/acme-enum-tools\n"
            "toolRefs:\n  - name: kubectl\n    package: acme.enum-tools\n"
            "flow:\n  - step:\n      id: drain\n      type: tool\n"
            "      tool:\n        name: kubectl\n        action: drain-node\n"
            "        args:\n          node: \"node-1\"\n"
            "      capture:\n        drain_state: outputs.drain_state\n"
            "  - step:\n      id: end\n      type: end\n"
            "      outcome: {category: success, code: ok}\n"
        ),
    },
    err("ENUM-009"),
    note="AR-ENUM-7/AR-ENUM-9: substitute's own declared output produces \"pending\", not a member of [\"drained\",\"cordoned\"] -- ENUM-009 at production time (S2).",
)

add(
    "TV-ENUM-RUNTIME-006", "ENUM-RUNTIME",
    "A runbook's own declared output (S4) resolving to a non-member value at completion is ENUM-009.",
    "runbook.yaml",
    {"runbook.yaml": (
        "apiVersion: runbook/v1\nid: r\nname: r\n"
        "outputs:\n  drain_state:\n    type: string\n    enum: [\"drained\", \"cordoned\"]\n"
        "    value: \"step.drain.json.drain_state\"\n"
        "flow:\n  - step:\n      id: drain\n      type: cli\n      command: bash\n"
        "      args: [\"-c\", \"echo '{\\\"drain_state\\\": \\\"pending\\\"}'\"]\n"
        "      capture:\n        drain: json\n"
        "  - step:\n      id: end\n      type: end\n"
        "      outcome: {category: success, code: ok}\n"
    )},
    err("ENUM-009"),
    note="AR-ENUM-9: runbook output semantics (S4), validated at runbook completion against outputs.<name>.value.",
)

add(
    "TV-ENUM-RUNTIME-007", "ENUM-RUNTIME",
    "A non-string value bound to an enum-constrained string declaration is reported as the TYPE error, never ENUM-008: single-cause reporting, one true root cause.",
    "runbook.yaml",
    {"runbook.yaml": (
        "apiVersion: runbook/v1\nid: r\nname: r\n"
        "outputs:\n  drain_state:\n    type: string\n    enum: [\"drained\", \"cordoned\"]\n"
        "    value: \"step.drain.json.drain_state\"\n"
        "flow:\n  - step:\n      id: drain\n      type: cli\n      command: bash\n"
        "      args: [\"-c\", \"echo '{\\\"drain_state\\\": 42}'\"]\n"
        "      capture:\n        drain: json\n"
        "  - step:\n      id: end\n      type: end\n"
        "      outcome: {category: success, code: ok}\n"
    )},
    {"error_class": "GCP-TYPE", "error_code": "GCP-TYPE-001"},
    note=(
        "AR-ENUM-7: 'A non-string value bound to an enum-constrained string declaration is reported as "
        "the type error, not ENUM-008. Single-cause reporting.' The declared output resolves to the "
        "number 42 (not a string) -- this is a GCP-level type error, never ENUM-009/ENUM-008, per the "
        "ordering in §7 (type resolution/coercion, then type check, THEN enum membership)."
    ),
)

add(
    "TV-ENUM-RUNTIME-008", "ENUM-RUNTIME",
    "${count} coercing to the PJVM string \"2\" and matching a declared member \"2\" is a PASS: enum operates on the post-coercion PJVM string, not on the pre-coercion number.",
    "runbook.yaml",
    {"runbook.yaml": (
        "apiVersion: runbook/v1\nid: r\nname: r\nvars:\n  count: 2\n"
        "inputs:\n  count_label:\n    type: string\n    required: false\n"
        "    default: \"${count}\"\n    enum: [\"1\", \"2\", \"3\"]\n"
        "flow:\n  - step:\n      id: end\n      type: end\n"
        "      outcome: {category: success, code: ok}\n"
    )},
    {"value": "2"},
    note=(
        "AR-ENUM-7: '${count} coercing to \"2\" and matching a member \"2\" is a pass. Enum operates on "
        "the post-coercion PJVM string.' This does not reopen GXL's no-implicit-coercion rule -- GIS "
        "coercion-to-string is an existing, separately ratified mechanism, applied before the enum check."
    ),
)


# ===========================================================================
# ENUM-SUBST (min 6) -- PKG-013 enum set-equality, no variance.
# ===========================================================================

add(
    "TV-ENUM-SUBST-001", "ENUM-SUBST",
    "Both the caller-visible action arg and its substitute's own input declare the identical enum member SET: OK, no PKG-013.",
    "runbook.yaml",
    pkg_scenario('["graceful", "force"]', '["graceful", "force"]'),
    {
        "catalog": {
            "kubectl": {"qualifiedName": "acme.enum-tools/kubectl", "tier": 1}
        }
    },
    note="AR-ENUM-8: equal member sets -> OK. catalog/success shape (AR-TP-5 opacity requirement).",
)

add(
    "TV-ENUM-SUBST-002", "ENUM-SUBST",
    "Neither the action arg nor its substitute's input declares an enum: OK, no PKG-013.",
    "runbook.yaml",
    (lambda: {
        "runbook.yaml": caller_runbook_yaml(),
        "vendor/acme-enum-tools/gert-package.yaml": PKG_YAML,
        "vendor/acme-enum-tools/tools/kubectl.tool.yaml": (
            "apiVersion: tool/v1\nmeta:\n  name: kubectl\n  version: \"1.0.0\"\n"
            "transport:\n  mode: stdio\ngovernance:\n  allowed-environments: [\"real\"]\n"
            "  requires-approval: false\nactions:\n  - name: get-nodes\n"
            "    argv: [\"get\", \"nodes\"]\n    args: {}\n  - name: drain-node\n"
            "    execute:\n      kind: runbook\n      path: ../runbooks/drain-node.yaml\n"
            "    args:\n      node: {type: string, required: true}\n"
            "      strategy: {type: string, required: false, default: graceful}\n"
            "    outputs:\n      drained: {type: boolean}\n"
        ),
        "vendor/acme-enum-tools/runbooks/drain-node.yaml": (
            "apiVersion: runbook/v1\nid: acme.enum-tools/drain-node\nname: drain-node\n"
            "inputs:\n  node: {type: string, required: true, from: context}\n"
            "  strategy: {type: string, required: false, from: context, default: graceful}\n"
            "outputs:\n  drained: {type: boolean, value: \"step.drain.json.drained\"}\n"
            "flow:\n  - step:\n      id: drain\n      type: cli\n      command: bash\n"
            "      args: [\"-c\", \"echo '{\\\"drained\\\": true}'\"]\n"
            "      capture:\n        drain: json\n"
        ),
    })(),
    {"catalog": {"kubectl": {"qualifiedName": "acme.enum-tools/kubectl", "tier": 1}}},
    note="AR-ENUM-8: both-absent passes.",
)

add(
    "TV-ENUM-SUBST-003", "ENUM-SUBST",
    "The caller-visible action arg declares an enum while the substitute's own input does not: PKG-013.",
    "runbook.yaml",
    pkg_scenario('["graceful", "force"]', None) | {
        "vendor/acme-enum-tools/runbooks/drain-node.yaml": (
            "apiVersion: runbook/v1\nid: acme.enum-tools/drain-node\nname: drain-node\n"
            "inputs:\n  node: {type: string, required: true, from: context}\n"
            "  strategy: {type: string, required: false, from: context, default: graceful}\n"
            "outputs:\n  drained: {type: boolean, value: \"step.drain.json.drained\"}\n"
            "flow:\n  - step:\n      id: drain\n      type: cli\n      command: bash\n"
            "      args: [\"-c\", \"echo '{\\\"drained\\\": true}'\"]\n"
            "      capture:\n        drain: json\n"
        ),
    },
    pkg013(),
    note="AR-ENUM-8: one side declares enum, the other does not -> PKG-013.",
)

add(
    "TV-ENUM-SUBST-004", "ENUM-SUBST",
    "The caller-visible action arg and its substitute's own input declare UNEQUAL enum sets (extra member on the caller side): PKG-013.",
    "runbook.yaml",
    pkg_scenario('["graceful", "force", "immediate"]', '["graceful", "force"]'),
    pkg013(),
    note="AR-ENUM-8: unequal sets -> PKG-013.",
)

add(
    "TV-ENUM-SUBST-005", "ENUM-SUBST",
    "A NARROWER substitute enum set (strict subset) is PKG-013: it would fail at runtime on values the caller's declared contract permits, turning a static contract into a landmine.",
    "runbook.yaml",
    pkg_scenario('["graceful", "force", "immediate"]', '["graceful", "force"]'),
    pkg013(),
    note="AR-ENUM-8: 'No variance. Not subset, not superset.' A narrower substitute is symmetric with SUBST-004: same rejection, called out separately because the direction matters to the rationale (§8, first bullet).",
)

add(
    "TV-ENUM-SUBST-006", "ENUM-SUBST",
    "A WIDER substitute enum set (strict superset) is PKG-013: silent contract erosion -- the caller's own plan-time ENUM-007 would reject a literal the substitute would accept.",
    "runbook.yaml",
    pkg_scenario('["graceful", "force"]', '["graceful", "force", "immediate"]'),
    pkg013(),
    note="AR-ENUM-8: wider substitute rejected symmetrically with the narrower case.",
)

add(
    "TV-ENUM-SUBST-007", "ENUM-SUBST",
    "Reordering enum members between caller and substitute (same SET, different declared order) MUST PASS: identity for contract comparison is SET equality, order-insensitive.",
    "runbook.yaml",
    pkg_scenario('["graceful", "force"]', '["force", "graceful"]'),
    {"catalog": {"kubectl": {"qualifiedName": "acme.enum-tools/kubectl", "tier": 1}}},
    note="AR-ENUM-5/AR-ENUM-8: order is presentation only; PKG-013 is SET, order-insensitive.",
)


# ===========================================================================
# ENUM-MOCK (min 5) -- the C2 obligation.
# ===========================================================================

REAL_PKG_YAML = """apiVersion: tool-package/v1
meta:
  name: acme.enum-tools
  version: "1.0.0"
exports:
  tools:
    - id: kubectl
      path: tools/kubectl.tool.yaml
"""


def mock_scenario(mock_strategy_enum, strategy_value, binding="real"):
    """Byte-identical caller runbook.yaml; only the bound package's tool
    definition (real vs mock) differs, per requires[].path pointing at
    either vendor/acme-enum-tools (real) or vendor-mock/acme-enum-tools
    (mock), matching AR-ENUM-8 §8: 'a mock package is an ordinary package
    bound via requires[].path or a --package-map override.'"""
    root = "vendor" if binding == "real" else "vendor-mock"
    files = {
        "runbook.yaml": (
            "apiVersion: runbook/v1\nid: r\nname: r\n"
            f"requires:\n  - package: acme.enum-tools\n    version: \"^1.0.0\"\n"
            f"    path: ./{root}/acme-enum-tools\n"
            "toolRefs:\n  - name: kubectl\n    package: acme.enum-tools\n"
            "flow:\n  - step:\n      id: drain\n      type: tool\n"
            "      tool:\n        name: kubectl\n        action: drain-node\n"
            f"        args:\n          node: \"node-1\"\n          strategy: {strategy_value}\n"
            "      capture:\n        drained: outputs.drained\n"
            "  - step:\n      id: end\n      type: end\n"
            "      outcome: {category: success, code: ok}\n"
        ),
        f"{root}/acme-enum-tools/gert-package.yaml": REAL_PKG_YAML,
        f"{root}/acme-enum-tools/tools/kubectl.tool.yaml": tool_yaml(mock_strategy_enum),
        f"{root}/acme-enum-tools/runbooks/drain-node.yaml": substitute_runbook_yaml(mock_strategy_enum),
    }
    return files


add(
    "TV-ENUM-MOCK-001", "ENUM-MOCK",
    "A byte-identical runbook under the REAL package binding: the literal strategy value passes because it is a declared member of the real package's enum set.",
    "runbook.yaml",
    mock_scenario('["graceful", "force"]', '"force"', binding="real"),
    {"value": "force"},
    note="C2 baseline (real binding, equal sets to be compared against MOCK-002).",
)

add(
    "TV-ENUM-MOCK-002", "ENUM-MOCK",
    "The SAME byte-identical runbook (same literal strategy value) under a MOCK package binding declaring the SAME enum set: also passes -- equal enum sets, both bindings OK.",
    "runbook.yaml",
    mock_scenario('["graceful", "force"]', '"force"', binding="mock"),
    {"value": "force"},
    note=(
        "AR-ENUM-8 §8 C2: 'Byte-identical runbook under real and mock binding: equal enum sets -> both "
        "pass.' The runbook itself is unchanged from MOCK-001 except requires[].path selecting the mock "
        "package root; the enum validation outcome (pass) is identical because the mock declares the "
        "identical enum set."
    ),
)

add(
    "TV-ENUM-MOCK-003", "ENUM-MOCK",
    "The same literal that PASSED under the real binding (MOCK-001) FAILS ENUM-007 under a mock declaring a DIVERGENT enum set: proves the check is binding-independent, not skipped.",
    "runbook.yaml",
    mock_scenario('["graceful", "immediate"]', '"force"', binding="mock"),
    err("ENUM-007"),
    note=(
        "AR-ENUM-8 §8 C2 (2): the mock's tool.yaml declares strategy enum [\"graceful\", \"immediate\"] -- "
        "'force' (the same literal that passed against the real package's [\"graceful\",\"force\"] set in "
        "MOCK-001) is not a member of the mock's set. This is a statically-known literal, so it is caught "
        "at PLAN time as ENUM-007 (the mock's own args.strategy declaration), demonstrating the check is "
        "binding-independent: it is the value's membership in WHICHEVER tool definition is bound, not a "
        "cached/real-only decision. 'A mock that declares a different enum set than the real package is "
        "an authoring error detectable only by running both bindings' -- this pair of vectors (MOCK-001, "
        "MOCK-003) IS that required detection, executed as the ratified conformance vector class rather "
        "than a runtime comparison (Barbara: 'I will not invent a lock field or a per-run signature "
        "registry to fake in-run detectability')."
    ),
)

add(
    "TV-ENUM-MOCK-004", "ENUM-MOCK",
    "A runtime-bound (interpolated) value that passed under the real binding fails ENUM-008 under a mock with a divergent set: the runtime check is also binding-independent.",
    "runbook.yaml",
    mock_scenario('["graceful", "immediate"]', '"${strategy_var}"', binding="mock") | (
        lambda d: (d.update({
            "runbook.yaml": d["runbook.yaml"].replace(
                "id: r\nname: r\n",
                "id: r\nname: r\nvars:\n  strategy_var: force\n",
            )
        }) or d)
    )(mock_scenario('["graceful", "immediate"]', '"${strategy_var}"', binding="mock")),
    err("ENUM-008"),
    note=(
        "AR-ENUM-8 §8 C2: the GIS-interpolated 'strategy_var' resolves to 'force' at runtime -- a member "
        "of the REAL package's set (['graceful','force']) but not of this vector's MOCK set "
        "(['graceful','immediate']). Because the value is interpolated (not a static literal), ENUM-007 "
        "does not apply (AR-ENUM-7); the failure surfaces at runtime binding as ENUM-008 against "
        "whichever tool definition is actually bound (here, the mock), which is the binding-independence "
        "property C2 requires for the runtime path specifically."
    ),
    tags=["gis-interpolated"],
)

add(
    "TV-ENUM-MOCK-005", "ENUM-MOCK",
    "Replay mode does not bypass enum validation: a scenario recording an off-enum bound value fails replay with the same ENUM-008 the live path would raise.",
    "runbook.yaml",
    {
        "runbook.yaml": (
            "apiVersion: runbook/v1\nid: r\nname: r\n"
            "inputs:\n  env_name:\n    type: string\n    required: true\n"
            "    from: prompt\n    enum: [\"prod\", \"stage\"]\n"
            "flow:\n  - step:\n      id: end\n      type: end\n"
            "      outcome: {category: success, code: ok}\n"
        ),
        "scenario.yaml": (
            "# design/gert/sections/13-evidence-tracing-resumption.tex §Scenario File Format /\n"
            "# §Replay Semantics: a scenario answering the enum-constrained prompt input with an\n"
            "# off-enum value. Replay does not bypass enum validation (AR-ENUM-7): this answer is\n"
            "# validated identically to a live prompt response and fails ENUM-008, not accepted\n"
            "# as a recorded/trusted value.\n"
            "inputs:\n  env_name: qa\n"
        ),
    },
    err("ENUM-008"),
    note=(
        "AR-ENUM-7: 'Replay mode MUST NOT bypass enum validation. Values answered from a scenario file "
        "are bound values and are validated identically to a live run... A scenario that records an "
        "off-enum value fails -- that is the scenario's bug.' This parallels the ratified 'replay does "
        "not bypass governance' rule."
    ),
    tags=["replay"],
)


# ===========================================================================
# ENUM-TRACE (min 3) -- metadata/trace/redaction.
# ===========================================================================

add(
    "TV-ENUM-TRACE-001", "ENUM-TRACE",
    "Enum metadata (declared-order member list) is carried once in the plan-validation trace record; never repeated in tool/invoked or tool/completed payloads (values only).",
    "runbook.yaml",
    {"runbook.yaml": runbook_input(
        "  env_name:\n    type: string\n    required: false\n    default: prod\n    enum: [\"prod\", \"stage\"]\n")},
    {"value": "prod"},
    note=(
        "design/gert/sections/03d-parse-time-enforcement.tex §ValidatedPlan Type Contract + "
        "design/gert/sections/07-runtime-events.tex §tool/invoked, §tool/completed: this is a "
        "contract-level assertion about WHERE enum metadata may appear across the run's event stream "
        "as a whole, not a single-expression evaluation; the corpus format has no per-event trace-shape "
        "assertion primitive, so the normative claim is pinned via 'note' at the abstraction the schema "
        "supports (successful plan validation of the declaration itself) while stating the full "
        "trace-carriage contract here for a conformance harness author to check directly against a real "
        "run's emitted trace file: exactly one 'enum' member-list occurrence in the plan/validation "
        "record, zero occurrences in any 'tool/invoked'/'tool/completed'/'step/started'/'step/completed' "
        "payload for this same run."
    ),
    tags=["untestable-in-corpus-format", "trace-shape"],
)

add(
    "TV-ENUM-TRACE-002", "ENUM-TRACE",
    "A tool arg declared enum with redact: true has its member list redacted to a member COUNT everywhere it would otherwise be emitted (trace, JSON, diagnostics) -- never the values themselves.",
    "runbook.yaml",
    pkg_scenario('["graceful", "force"]', '["graceful", "force"]') | {
        "vendor/acme-enum-tools/tools/kubectl.tool.yaml": (
            "apiVersion: tool/v1\nmeta:\n  name: kubectl\n  version: \"1.0.0\"\n"
            "transport:\n  mode: stdio\ngovernance:\n  allowed-environments: [\"real\"]\n"
            "  requires-approval: false\nactions:\n  - name: get-nodes\n"
            "    argv: [\"get\", \"nodes\"]\n    args: {}\n  - name: drain-node\n"
            "    execute:\n      kind: runbook\n      path: ../runbooks/drain-node.yaml\n"
            "    args:\n      node: {type: string, required: true}\n"
            "      strategy:\n        type: string\n        required: false\n"
            "        redact: true\n        default: graceful\n"
            "        enum: [\"graceful\", \"force\"]\n    outputs:\n      drained: {type: boolean}\n"
        ),
    },
    {"value": "force"},
    note=(
        "design/gert/sections/08-security-and-trust.tex §Redacted enum member lists (C1): a "
        "'redact: true' arg MAY declare enum, but every trace/JSON/diagnostic surface that would "
        "otherwise show the member list emits the literal string \"<redacted>\" plus a member count "
        "(here, 2), never the values [\"graceful\",\"force\"]. This vector exercises the PASSING binding "
        "path (value 'force' is a genuine member and the binding succeeds); redaction of the member list "
        "is a metadata-surface property parallel to ENUM-TRACE-001's note, stated here for the "
        "redact:true-specific surface."
    ),
    tags=["redact", "untestable-in-corpus-format"],
)

def trace_003_runbook():
    return caller_runbook_yaml(strategy_value='"${strategy_var}"').replace(
        "id: r\nname: r\n", "id: r\nname: r\nvars:\n  strategy_var: aggressive\n"
    )


add(
    "TV-ENUM-TRACE-003", "ENUM-TRACE",
    "An ENUM-008 failure for a redact: true tool arg (S1) MUST NOT enumerate permitted values and MUST NOT echo the rejected value: the error surfaces as the ordinary ENUM error class/code only.",
    "runbook.yaml",
    {
        "runbook.yaml": trace_003_runbook(),
        "vendor/acme-enum-tools/gert-package.yaml": PKG_YAML,
        "vendor/acme-enum-tools/tools/kubectl.tool.yaml": (
            "apiVersion: tool/v1\nmeta:\n  name: kubectl\n  version: \"1.0.0\"\n"
            "transport:\n  mode: stdio\ngovernance:\n  allowed-environments: [\"real\"]\n"
            "  requires-approval: false\nactions:\n  - name: get-nodes\n"
            "    argv: [\"get\", \"nodes\"]\n    args: {}\n  - name: drain-node\n"
            "    execute:\n      kind: runbook\n      path: ../runbooks/drain-node.yaml\n"
            "    args:\n      node: {type: string, required: true}\n"
            "      strategy:\n        type: string\n        required: false\n"
            "        redact: true\n        default: graceful\n"
            "        enum: [\"graceful\", \"force\"]\n    outputs:\n      drained: {type: boolean}\n"
        ),
        "vendor/acme-enum-tools/runbooks/drain-node.yaml": substitute_runbook_yaml('["graceful", "force"]'),
    },
    err("ENUM-008"),
    note=(
        "design/gert/sections/08-security-and-trust.tex §Redacted enum member lists (C1): 'An ENUM-008 "
        "message for a redacted declaration MUST NOT enumerate permitted values and MUST NOT echo the "
        "rejected value.' Uses the S1 site (tool action args.<name> with redact: true, "
        "design/gert/sections/06-tool-runtime.tex Tool Definition Schema), the one declaration site where "
        "a literal redact: true is schema-representable today (tool.yaml has no governing "
        "tool.v1.schema.json, C3, so the field is unconstrained; contrast the runbook Input/Output "
        "$defs, which have no comparable 'redact' property -- a schema-completeness question flagged "
        "alongside DEFAULT-006, not an enum-semantics ambiguity). The GIS-interpolated 'strategy_var' "
        "resolves to 'aggressive' at runtime, not a member of ['graceful','force'], raising ENUM-008 at "
        "the binding site. The vector's expected shape is the plain ErrorExpected envelope (error_class "
        "ENUM, error_code ENUM-008) with no 'permitted values' or 'rejected value' payload field "
        "anywhere -- vector.schema.json's ErrorExpected has no such field to populate in the first "
        "place, which is itself evidence the corpus format cannot accidentally leak one. A conformance "
        "harness additionally MUST assert, against the real error message text (outside this schema's "
        "structural fields), that neither the declared members nor the supplied value appear in it."
    ),
    tags=["redact", "gis-interpolated"],
)


print(len(OUT), "vectors generated")

header = """# Enum-Constrained Tool and Runbook Outputs MVP conformance corpus
# (AR-ENUM-1..15, .squad/decisions/inbox/barbara-enum-constraint-mvp-architecture-ruling.md,
# ratified 2026-08-10, §Handoff -> Tess).
#
# Generated by design/gert/scripts/gen_enum_vectors.py (scaffolding, safe to
# delete once this file is reviewed/stable), mirroring the tv-pkg-resolve.yaml
# fixture shape: 'variables' models a synthetic workspace filesystem as a flat
# map from workspace-relative POSIX path to file content string; 'input' is
# the entry-point file. Declaration-only vectors (well-formedness, defaults,
# most of ENUM-DECL/ENUM-UNICODE) use a single self-contained runbook.yaml;
# vectors exercising S1 (tool action args) / S2 (substituted action outputs) /
# substitution (PKG-013) / package binding (mock) use the same
# real-package + substitute-runbook shape as tv-pkg-resolve.yaml's PKG-SUBST
# vectors.
#
# Categories: ENUM-DECL, ENUM-UNICODE, ENUM-DEFAULT, ENUM-PLAN, ENUM-RUNTIME,
# ENUM-SUBST, ENUM-MOCK, ENUM-TRACE. Every ENUM-001..009 and ENUM-W001 is
# exercised at least once, with at least one negative vector each; PKG-013's
# enum set-equality extension (AR-ENUM-8) is exercised in ENUM-SUBST and
# ENUM-MOCK. Two vectors (ENUM-UNICODE-002 "invalid UTF-8 member",
# ENUM-TRACE-001/002 trace-shape/redaction-surface claims) are tagged
# 'untestable-in-corpus-format': they pin a normative contract at the
# abstraction this schema supports rather than inventing unsupported fields,
# per the corpus's determinism rules (no timestamps, no host paths, no
# locale dependence; assert on codepoints, not rendered text).
#
# See .squad/agents/tess/history.md for the full coverage matrix and
# validation report.
"""

import yaml as _yaml


class _LiteralStr(str):
    pass


def _literal_str_representer(dumper, data):
    style = "|" if "\n" in data else None
    return dumper.represent_scalar("tag:yaml.org,2002:str", data, style=style)


_yaml.add_representer(_LiteralStr, _literal_str_representer)


def _wrap_variables(v):
    out = {}
    for k, val in v.items():
        if isinstance(val, str) and "\n" in val:
            out[k] = _LiteralStr(val)
        else:
            out[k] = val
    return out


for vec in OUT:
    vec["variables"] = _wrap_variables(vec["variables"])

doc = {"vectors": OUT}

body = _yaml.dump(doc, sort_keys=False, allow_unicode=True, width=100, default_flow_style=False)

with open("design/gert/conformance/tv-enum.yaml", "w", encoding="utf-8", newline="\n") as f:
    f.write(header)
    f.write(body)

print("wrote design/gert/conformance/tv-enum.yaml")
