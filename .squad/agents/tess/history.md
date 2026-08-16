# tess

Current role: Implementation engineer.

Session: Dynamic Runbook Includes (2026-08-15)
- Feature complete and approved
- Ready for merge

Session: MCP HTTP Stream D (2026-08-16) — **FINAL**
- Adversarial test suite complete: 31 pass / 0 skip / 0 fail (27 Stream D vectors + 4 Group I AzureCLI vectors)
- DEF-007/DEF-008/DEF-009/DEF-010 resolved concurrently before tests ran
- DEF-011/DEF-012/DEF-013 initially reported as open; all three were stale — fixes had already landed in HEAD at time of first read
- Stale defect reports corrected after parallel-build-hazard note from Cristián; all 6 skipped vectors unskipped and passing
- B-27 (mcp/authAttached wiring) independently verified: engine installs emitter in emitterCtx before every exec.Execute call, flows through full transport chain
- End-to-end token-leak sweep (sentinel through real TokenGate + trace emitter): CLEAN — no token in errors, result fields, or trace event payloads
- AUTH-003 tightened: conditional `if err != nil` replaced with unconditional `if err == nil { t.Fatal }` — assertion holds after tightening (DEF-013 confirmed fatal)
- Stale `// BLOCKED: DEF-013` annotations and file header defect commentary cleaned up
- Stream C extension: 4 Group I adversarial vectors for AzureCLIAuthProvider — sentinel sweep, no-retry-loop, Invalidate-clears-stale, 401-driven-Invalidate — all PASS
- David's `knownToken = "******"` sentinel weakness noted (filed in tess-mcp-http-streamd-final.md) — complementary sentinels used, no contradiction
- Process note recorded: re-read production code before filing if result contradicts stream's own report
- Reports filed: .squad/decisions/inbox/tess-mcp-http-streamd.md (updated), .squad/decisions/inbox/tess-mcp-http-streamd-final.md

Detailed history: .squad/agents/tess/history-archive.md

## Learnings

### From Stream D / Stream C final pass (2026-08-16)

**PowerShell string.Replace() is unreliable on Unicode test files.**
[System.IO.File]::ReadAllText().Replace(old, new) silently fails to match when the file contains non-ASCII characters (em-dashes, arrow characters) or has tab/newline encoding that doesn't round-trip through PowerShell literals. Workaround: use Get-Content (line array) + index-based surgery + [System.IO.File]::WriteAllLines(..., UTF8Encoding::new(False)).

**Off-by-one errors in loop skip ranges.**
When removing a block with $i -ge  -and  -le , the loop's own $i++ fires after any $i += N body adjustment — accounting for this is non-trivial. Always Write-Host the content at boundary indices before committing a range to confirm what will be dropped.

**David's knownToken = "******" is a weak sentinel.**
Asterisks appear in truncated Go error strings ("..."), making a redaction test using them collision-prone. Use high-entropy sentinels (e.g. TESS_SENTINEL_TOKEN_D42E9B1C) that cannot appear in normal output.

**Assertion shape: unconditional error required after a fatal fix.**
When a function was formerly warn-and-continue (nil return) and is now fatal (error return), conditional if err != nil { check error } tests silently pass on regression. The correct shape is if err == nil { t.Fatal(...) } — this fails immediately if the fatal behavior is lost.

**Complement, don't duplicate, existing test coverage.**
David's redaction tests and Group I's sentinel sweep are complementary — different sentinel strengths, different code paths exercised. Noting which is stronger is a useful observation (worth a decision file) but is not a contradiction.

## 2026-08-16T02:33:36Z — MCP HTTP Transport Feature — Team Session Orchestration

**Feature:** Native Streamable HTTP MCP transport (transport.mode: mcp-http)
**Outcome:** ✅ COMPLETE — All requirements met, final review gate APPROVED (unconditional)

- Completed assigned stream work
- Collaborated on binding architecture contract
- Participated in team orchestration
- All deliverables verified and tested
- Ready for production merge

