# Phase 17 Design Decisions — Ken

**Date:** 2026-04-21  
**Author:** Ken (Software Architect)  
**Phase:** 17  
**Status:** PROPOSED

---

## D-17-01: Constant-Time Token Comparison is Mandatory

**Decision:** All token comparisons in `gert serve` MUST use `subtle.ConstantTimeCompare`.

**Context:**
- Phase 16 shipped bearer token auth using direct string `!=` comparison
- Review identified this as NBI-16-08 (renamed NBI-16-01 in carry-forward)
- While bearer auth is currently "dev-only" (D-16-04), the pattern is wrong

**Impact:**
- One-line fix in `internal/serve/middleware.go`
- Adds `crypto/subtle` import
- No behavioral change for correct tokens
- Blocks timing side-channel attacks

**Rationale:**
1. **Security hygiene:** Timing attacks are real and well-documented
2. **Pattern establishment:** Future auth work (Phase 18+) will copy this pattern
3. **Low cost:** Trivial fix, no performance impact
4. **Best practice:** Industry standard for credential comparison

**Alternatives considered:**
- Do nothing (rejected: security risk, wrong pattern)
- Wait for production auth (rejected: establishes bad precedent)

---

## D-17-02: Token Expiry is Opt-In

**Decision:** Token expiry validation is enabled only when `--auth-token-expiry` is set. Default behavior (zero duration) remains opaque shared-secret comparison.

**Context:**
- NBI-16-02 requested "full auth hardening"
- Full OAuth2/OIDC is large scope for a single phase
- Token expiry is a useful stepping stone

**Impact:**
- New CLI flag: `--auth-token-expiry`
- When set, tokens must be base64-encoded JSON with `iat`/`exp`/`secret` claims
- When unset, tokens are opaque strings (current behavior)

**Rationale:**
1. **Backward compatibility:** Existing deployments continue to work
2. **Incremental hardening:** Adds expiry without full JWT complexity
3. **Clear scope:** Part B work is bounded, not a full auth system
4. **Phase 18 foundation:** Expiry logic can be extended to full JWT validation

**Alternatives considered:**
- Full JWT validation (rejected: too large for Phase 17)
- Skip expiry entirely (rejected: NBI-16-02 has merit)
- Require expiry always (rejected: breaks existing deployments)

---

## D-17-03: SSE Synchronization via WaitForSubscriber

**Decision:** Fix the `TestSSE_ConnectReceivesEvents` flake by adding `WaitForSubscriber(ctx, timeout)` to `EventBridge` rather than increasing sleep durations.

**Context:**
- Test was `t.Skip`'d in Phase 16 (NBI-15-01 carry-forward)
- Root cause: race between SSE client connection and event broadcast
- Sleep-based fixes are inherently flaky

**Impact:**
- New method on `EventBridge`: `WaitForSubscriber(ctx, timeout) error`
- Test uses explicit synchronization instead of sleep
- `t.Skip` removed — test runs in CI

**Rationale:**
1. **Deterministic tests:** Explicit sync > magic sleep values
2. **Reusable API:** `WaitForSubscriber` is useful for integration tests
3. **Root cause fix:** Addresses the actual race condition
4. **CI reliability:** No more flaky test failures

**Alternatives considered:**
- Increase sleep duration (rejected: still flaky, just less frequent)
- Use `testutil.Eventually` with retry (rejected: polling is wasteful)
- Keep `t.Skip` (rejected: test provides value, should run)

---

## D-17-04: run.delete Deferred to Phase 19

**Decision:** Do not add `run.delete` RPC in Phase 17. Defer to Phase 19 (post-auth hardening).

**Context:**
- `run.list` and `run.get` are implemented
- Natural CRUD pattern suggests `run.delete`
- Delete has complex implications

**Impact:**
- No new RPC method this phase
- Current cleanup: `run.cancel` + manual/eventual expiry
- Phase 19 will design delete with auth context

**Rationale:**
1. **Phase focus:** Phase 17 is security hardening, not feature expansion
2. **Auth dependency:** Delete permissions need auth layer to be complete
3. **Complexity:** Delete raises questions (disk cleanup, concurrent access, audit)
4. **MVP sufficient:** Cancel + manual cleanup works for initial users

**Alternatives considered:**
- Add delete now (rejected: auth dependency, scope creep)
- Add soft-delete now (rejected: same complexity, less value)

---

## Summary

| Decision | Status | Priority | Implementation |
|----------|--------|----------|----------------|
| D-17-01 | APPROVED | High | Required for Phase 17 seal |
| D-17-02 | APPROVED | Medium | Part B work |
| D-17-03 | APPROVED | Medium | Part B work |
| D-17-04 | APPROVED | N/A | Deferral decision |

---

## Related Documents

- Phase 17 design: `.squad/tmp/ken-phase17-design.md`
- Phase 16 review: `.squad/decisions/inbox/ken-phase16-review.md`
- Phase 16 design: `.squad/tmp/ken-phase16-design.md`

---

*Ken, Software Architect*
