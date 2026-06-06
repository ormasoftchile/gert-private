---
name: "go-ast-evaluator"
description: "Implement a strict PJVM AST evaluator from normative conformance vectors"
domain: "go-runtime"
confidence: "medium"
source: "Don Phase 2 Day 4 GXL evaluator"
---

## Context

Use this pattern when a parser already produces a closed AST, values are a typed JSON/PJVM sum, and conformance vectors pin both successful values and structured error codes.

## Pattern

1. Keep the evaluator entry small: `Eval(ast, bindings, clock)` constructs an internal walker with injected dependencies.
2. Return typed diagnostics with stable codes and make the harness compare codes, not message text.
3. Evaluate call arity before argument expressions so wrong-arity vectors cannot be masked by missing-variable or type errors in unused arguments.
4. Implement short-circuit operators as AST-control flow, not as eager boolean helpers; only evaluate the right side when the left side permits it.
5. Use PJVM constructors for every produced value so non-finite numbers and host-native types cannot cross the runtime boundary.
6. Keep stdlib functions in a separate file by namespace; shared helpers should validate arity, scalar/list/string types, and return PJVM values.
7. Inject time via a `Clock` and format timestamps at the boundary; never read wall-clock time inside tests without an injector.

## Anti-Patterns

- Do not coerce PJVM types to make comparisons or arithmetic succeed.
- Do not implement future namespaces because they are reserved; match only the grammar/vector surface.
- Do not let unresolved `TBD` vector semantics silently become permanent behavior; pass the corpus only with an explicit flagged diagnostic.
