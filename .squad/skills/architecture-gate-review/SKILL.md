# Skill: Architecture Gate Review

## When to use
When evaluating an inbound implementation request or architecture proposal from another team/repo that proposes new abstractions, schema changes, or multi-phase work against the Gert core.

## Pattern

1. **Verify the seam.** Before evaluating the proposal's abstractions, verify what actually exists in the source. Read the interfaces/types the proposal claims to build on. Confirm the seam exists or doesn't.

2. **Check for internal contradictions.** Proposals often declare "non-goals" that contradict their own requirements. The most common: "no schema changes" when the feature requires new declarative metadata. Call these out immediately.

3. **Separate static from dynamic.** Any preflight/validation design must distinguish:
   - Static: can be checked offline from config/declarations alone (does a binding EXIST?)
   - Dynamic: requires live resources (can we reach endpoint? can we acquire token?)
   These deserve different error codes and different operator guidance.

4. **Challenge estimates that depend on unanswered questions.** If a phase's architecture depends on an open question listed elsewhere in the same document, the estimate is unreliable. Require the question answered in a prior phase.

5. **Structure the verdict.** Always deliver:
   - Clear verdict (Accept / Accept-with-modifications / Reject / Defer)
   - What to push back on (specific items with justification)
   - What you'd restructure (concrete alternative phasing if applicable)
   - Schema fragments for any recommendation (make it tangible)

## Anti-patterns
- Restating the proposal back without judgment
- Accepting "no schema changes" at face value without checking if the feature needs new declarations
- Treating all preflight failures as the same error class
- Accepting phase estimates without checking their dependencies on open questions
