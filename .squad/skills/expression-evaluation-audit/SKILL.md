# Expression Evaluation Audit Skill

Use this pattern when auditing a DSL or runbook system for non-portable expression behavior.

## Steps

1. **Separate implementation language from authored semantics.** A Go/C#/TS implementation is fine; author-visible Go/C#/TS syntax is the portability concern.
2. **Inventory every authored evaluator surface.** Include guards (`when`, `condition`, `until`), branching, loops, computed fields, interpolation, assertions, capture paths, filters, templates, and examples.
3. **Record mechanism and grammar.** For each site, identify the evaluator/library, exposed operators, literals, identifiers, function calls, traversal syntax, and error behavior.
4. **Compare docs to fixtures.** Fixtures often reveal non-normative syntax that the prose says should not be used.
5. **Flag conflicted contracts.** Tables, examples, and architecture docs must agree on whether a field uses a template, expression evaluator, path language, or literal map.
6. **Assess runtime parity risk.** Ask whether a second runtime could evaluate the same runbook without embedding the original evaluator.
7. **Offer options without premature recommendation.** Present migration directions such as CEL, a native subset, allowlisted current evaluator, or structured predicates.

## Output Checklist

- Inventory table with file:line references.
- Go-style / host-language-style feature table.
- Test and example coverage table.
- Portability risk assessment.
- Explicit distinction between host implementation and authored language.
- Decision questions for the team.