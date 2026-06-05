---
name: "go-conformance-scaffold"
description: "Scaffold a Go runtime against YAML conformance corpus before parser implementation"
domain: "go-runtime"
confidence: "medium"
source: "Don Phase 2 Day 1 scaffold for GXL/GIS/GCP runtime"
---

## Context

Use this pattern when a Go implementation must conform to a normative YAML vector corpus, but implementation should not begin until package boundaries and shared value types are locked.

## Pattern

1. Read the normative grammar/spec files before choosing package boundaries.
2. Put shared cross-grammar contracts in a small `core` package; keep grammar-specific parser/evaluator packages separate.
3. Define the portable value model before writing evaluators.
4. Add a `TestConformance` harness immediately that discovers all `tv-*.yaml` files, loads vectors, counts them, and creates one skipped subtest per vector ID.
5. Make the scaffold compile with `go build ./...`, `go test ./... -run TestConformance`, and `go mod tidy` before implementing parser logic.

## Anti-Patterns

- Do not hand-code behavior that exists only in the current Go implementation idea; derive it from grammar/spec/corpus.
- Do not let grammar-specific extensions leak across packages.
- Do not wait until after evaluator implementation to wire the corpus harness.
