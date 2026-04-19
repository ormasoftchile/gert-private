# SKILL: Architecture Review

## Purpose
A reusable pattern for conducting architectural reviews of design documents in the gert v2 project context. Apply when reviewing any design section before implementation begins.

## Review Axes

For each section, evaluate across four axes:

| Axis | Key Question |
|---|---|
| **Completeness** | Is there enough here to build from? Could an engineer implement this without asking follow-up questions? |
| **Coherence** | Is it consistent with adjacent sections? Do the interfaces align? |
| **Architectural correctness** | Does the design reflect sound principles (separation of concerns, dependency direction, explicit contracts)? |
| **v1 coverage** | Does it account for what v1 does well (preserve it), and what v1 does poorly (fix it)? |

## Red Flags — Incomplete Design

A section is a placeholder, not a design, when it contains only:
- Principle statements without implementation guidance ("deny by default")
- Category names without member definitions ("planning events, step lifecycle events")
- Reference to a spec file that doesn't exist
- Bullet lists of requirements with no contracts, schemas, or interfaces
- A component name without an interface definition

## Red Flags — Incoherence

Watch for these cross-section inconsistencies:
- Two sections share a concept (e.g. "capabilities") but each defines it differently
- A section refers to a component that another section contradicts (e.g. pure event model vs. direct API calls)
- The decisions log documents already-implemented behavior that the design sections haven't caught up to
- A section describes a breaking change without any migration path

## Severity Classification

| Severity | Meaning |
|---|---|
| **Critical** | Cannot build without this. Implementation would be guessing. |
| **Important** | Build is possible but likely to produce inconsistent results. Decisions will be made ad hoc. |
| **Nice-to-have** | Adds clarity but engineers can reasonably resolve via conventions. |

## Gap Analysis Output Structure

```
## Section-by-Section Review
For each §N:
- Current state (1-2 sentences)
- Gaps (bulleted, specific, actionable)
- Severity: Critical / Important / Nice-to-have

## Cross-Cutting Gaps
- Missing sections / entirely absent topics
- Inconsistencies between sections
- Architectural concerns no section addresses

## Priority Recommendations
Top 5-7 items that must be resolved before the document is a build reference

## New Section Proposals
Each with: name, 2-3 sentence description of required content
```

## Handoff Protocol

After gap analysis:
1. Write analysis to `.squad/tmp/<author>-gap-analysis.md`
2. Write implied-but-unstated decisions to `.squad/decisions/inbox/<author>-<topic>.md`
3. Update `history.md` under `## Learnings`
4. Notify Leslie (document structure), Dennis (research gaps), domain experts (schema: John, integrations: Barbara, Go: Brian)

## Calibration Notes

- A design document is ready to build from when a new engineer can read a section and write an interface definition without ambiguity.
- A "skeleton" document (correct structure, thin content) is normal at early stages — the review's job is to make the gaps visible, not to criticize the work.
- Cross-reference the decisions log: implemented decisions that aren't in the design represent design debt that must be reconciled.
- Always check for dangling spec references (files cited that don't exist).
