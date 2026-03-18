### 2026-03-18T14:30:00Z: TreeNode Structure Authority
**By:** Killua (Backend / Schema)
**What:** Go schema is the authority. TreeNode = {step?, iterate?, branches?}. Branches are structural containers, NOT a step type. Step.Branches (legacy) should not be used; all branching goes via TreeNode.Branches.
**Why:** TypeScript form was treating branch as a step type, causing schema mismatch. Clarifying the structure prevents further divergence and enables safe form design.
**Output:** SCHEMA_CLARIFICATION.md (complete ref for Kurapika's form fix).
**Blocking:** None — clarification doc is informational, unblocks Kurapika's form redesign.
