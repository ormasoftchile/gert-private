---
name: "go-recursive-descent-parser"
description: "Build a safe Go recursive-descent parser from a normative EBNF grammar and YAML conformance vectors"
domain: "go-runtime"
confidence: "medium"
source: "Don Phase 2 Day 3 GXL lexer/parser"
---

## Context

Use this pattern when a grammar is normative, conformance vectors pin error codes, and later evaluators must walk an AST without bypassing governance boundaries.

## Pattern

1. Read the EBNF, decision log, and vector notes before coding; vectors may ratify ambiguities that differ from a task brief.
2. Put lexer, AST, parser, and package tests in the grammar package; expose only small entry points such as `Lex` and `Parse`.
3. Make tokens carry source positions from the lexer. Preserve those positions on AST nodes and parse diagnostics.
4. Model parse diagnostics as typed errors with stable codes; have the conformance harness compare codes, not message text.
5. Implement recursive-descent functions exactly in precedence order. Use iterative loops for left-associative levels and recursion for right-associative unary levels.
6. Keep syntax-only parsing separate from evaluation: parse may validate closed namespaces/method names, but it must not infer runtime operand types.
7. Wire one corpus runner at a time so future grammar families remain intentionally `not implemented` until their day of work.

## Anti-Patterns

- Do not let evaluator semantics leak into parse vectors.
- Do not broaden grammar acceptance to make fixtures pass; unknown syntax should become a structured parse error.
- Do not collapse all parse failures into one generic code when the corpus distinguishes lexer, forbidden-syntax, and syntactic failures.
