### 2026-04-07: User directive — web feature parity is non-negotiable

**By:** ormasoftchile (via Copilot)
**What:** Chain views, prune, iterate pass selection, annotation badges, and minimap are REQUIRED features for the web version — not optional. The shared renderer must include ALL VS Code graph features. Missing features must be implemented, not left as web-only gaps.
**Why:** User requirement — web must be functionally equivalent to VS Code. Gaps are bugs, not deferred scope.
**Impact on plan:** Phase 1 of the shared renderer refactor must implement missing graph features DIRECTLY in the shared package (not in web first and then refactor). Implement once, both platforms consume.
