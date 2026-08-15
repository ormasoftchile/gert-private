# Session Log: Dynamic Runbook Includes — Phase 4 Wrap-up

**Date:** 2026-08-15T23-21-47Z  
**Feature:** Runtime-resolved (dynamic) runbook includes for `gert` Go repo  
**Outcome:** Feature complete, approved, ready for merge

## Session Defining Lesson

**The feature appeared complete four separate times and wasn't.**

This is the defining insight to carry forward: real verification requires exercising the running system, not reading the code or trusting status reports.

1. **First gap**: Code looked complete (schema, parser, executor all implemented). Dead code — no callers wired. Found by real CLI exercise (Tess e2e).

2. **Second gap**: Feature "completed" with CLI preflight crash on every dynamic-include runbook. All internal APIs looked correct. Bug was in two visitors, not one. Found only by running the real CLI (Tess e2e, then Don fixed).

3. **Third gap**: Governance was composed at execution time but never enforced. All composition logic present, all error codes defined. Enforcement path missing. Found by adding real enforcement code (Ken implementation), then testing (Tess corpus).

4. **Fourth gap**: Entry runbook governance seeded as nil, so composition ran against null and silently dropped all parent deny rules. All child governance enforcement present, all seeding logic written elsewhere. Integration point missed. Found by code review (Don fixed).

Every one of these was discovered by verifying against the running system rather than reading the code or trusting a status report.

## Secondary Finding

**Plausible descriptions of the code can be materially wrong.**

Three times during this session, the coordinator (myself) described what the code does and was corrected by the agent writing or reviewing it:

1. Pin recording scope (described as "identity only"; actually records both identity and digest)
2. Resume drift detection (described incorrectly; David clarified the actual algorithm)
3. Governance composition operator precedence (described incorrectly; David clarified intersection vs. union ordering)

These were not clarifications — they were material misunderstandings of the implementation. Each correction would have changed the design if it had remained wrong. This is important: a senior coordinator can still misunderstand code. Only the implementers reading the source know for sure. Rely on code review, not on narrative summaries.

## Conformance Corpus as Truth Amplifier

Tess's 29-vector conformance corpus, authored before implementation existed, proved invaluable. Every defect found during real CLI exercise was in a vector. The corpus didn't exhaustively specify the feature (too large for a designed corpus), but it was authoritative on the cases it covered. By running vectors against the real CLI repeatedly during implementation, the team found gaps the code itself obscured.

## Architecture Approval Gate

Barbara's review gate (APPROVE WITH CONDITIONS) was correct on substance. Three conditions were documentation-only (schema descriptions + example warnings), all satisfied before ship. All 11 user requirements verified met. The feature is architecturally sound.

## Deferred Work Properly Captured

Three defects found during implementation were deferred (B-18, B-19/B-20, B-21), not hidden. All preserved as formal open work items in the decision ledger. This is the right pattern: surface discovery (don't bury it), make the deferral decision explicit (don't pretend it's solved), and document the reason (production risk, out-of-scope, acceptable-as-designed).

## Team Dynamics

Five agents over 24 hours, one lockout (procedural), one correction of coordinator's understanding (three times), all work captured in decision ledger. No rework loops. No uncaught assumptions. Architecture held through four implementation gaps. This is how the protocol should work.

## Recommendation

Before declaring a feature "done," exercise it via the real user-facing CLI, not just internal APIs. Make test-driven discovery part of the gate. Trust conformance corpus more than code-reading. Expect descriptions (even senior coordinator's) to be incomplete. Get implementers to review each other's mental models, not just code. Capture deferred work formally; don't bury it.
