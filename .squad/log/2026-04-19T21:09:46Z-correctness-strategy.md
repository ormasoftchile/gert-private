# Session Log — Correctness Strategy

**Phase:** Foundation (Phase 0)  
**Outcome:** Barbara defined spec-driven correctness verification.

**Key Points:**
- Spec rules (MUST/MUST NOT/SHALL) have tagged tests with `// spec:` comments
- Coverage tool `gert dev spec-coverage` enforces ≥95% by Phase 13
- Golden traces use normalized JSONL with HMAC verification
- Ken reviews each phase for spec alignment and interface correctness
- Brian implements via TDD: spec → failing tests → code → pass

**Deliverable:** `internal/dev/speccoverage/` + `gert dev spec-coverage` subcommand
