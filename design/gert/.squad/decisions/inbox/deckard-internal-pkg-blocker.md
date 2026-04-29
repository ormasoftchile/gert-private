# Internal Package Access Blocker — Architectural Resolution

**Date**: 2026-04-29  
**Lead**: Deckard (Architectural Lead, gert-tui)  
**Status**: DECISION APPROVED  

---

## Executive Summary

Roy's gert-tui scaffold compiled but identified two blockers preventing engine integration:
1. **Blocker 1**: go:embed path traversal — `//go:embed ../../runbooks/*.yaml` violates Go embed rules
2. **Blocker 2**: Internal package access — gert-tui cannot import `internal/parser`, `internal/planner`, `internal/engine`

Both are resolved. Decision recorded below.

---

## Blocker 1: go:embed Path Traversal

### Problem
`internal/embed/embed.go` cannot use `//go:embed ../../runbooks/*.yaml` — Go's embed directive forbids `..` path traversal. The runbooks directory must be within the package tree that embeds it.

### Decision
**Move runbooks into cmd/gert-tui/**

```
gert-tui/
├── cmd/gert-tui/
│   ├── main.go
│   └── runbooks/              ← MOVE HERE
│       └── example.yaml
├── internal/embed/
│   ├── embed.go              ← simplify to: //go:embed runbooks
│   └── embed_test.go
```

**Rationale**: 
- Simple file move; no architectural impact
- Runbooks are TUI-specific assets
- Embedding from local directory avoids external dependency

**Action**: Roy moves `/gert-tui/runbooks/` → `/gert-tui/cmd/gert-tui/runbooks/` and updates embed.go comment.

---

## Blocker 2: Internal Package Access (The Hard Blocker)

### Problem Statement

gert-tui (separate module: `github.com/ormasoftchile/gert-tui`) cannot import:
- `github.com/ormasoftchile/gert/internal/parser`
- `github.com/ormasoftchile/gert/internal/planner`
- `github.com/ormasoftchile/gert/internal/engine`

Go module system enforces: `internal/` packages are module-private. External modules cannot access them.

Roy's scaffold currently notes this blocker in main.go and does not wire the engine.

### Options Considered

#### **Option A: Add pkg/run to gert (✅ CHOSEN)**

Create a new public package `pkg/run` in gert that bundles the complete workflow:

```go
// github.com/ormasoftchile/gert/pkg/run

type Runner interface {
	Run(ctx context.Context, runbookPath string, opts RunOptions) (RunHandle, error)
}

func NewRunner(cfg RunnerConfig) (Runner, error) {
	// Wire parser + planner + engine internally (using internal/ packages)
	// Return a single simple interface to gert-tui
}
```

**gert-tui imports only:**
```go
import "github.com/ormasoftchile/gert/pkg/run"
import "github.com/ormasoftchile/gert/pkg/input"
import "github.com/ormasoftchile/gert/pkg/engine"  // for RunHandle, Event types
```

**Pros:**
- ✅ Spec principle: "grow gert, not gert-tui"
- ✅ Clean separation: TUI does not touch gert internals
- ✅ gert-tui remains separate, distributable binary
- ✅ Extensible: New runner modes (batch, dry-run) live in pkg/run
- ✅ Backward compatible: Doesn't change existing public APIs
- ✅ Long-term maintainable: Central place for parser/planner/engine wiring

**Cons:**
- Requires gert team to add pkg/run (not blocking TUI, but sequential)

#### **Option B: Move gert-tui into gert repo as cmd/gert-tui/**

Make gert-tui a command in the gert repo. Same module → internal access solved.

**Pros:**
- ✅ Unblocks immediately; no new package required
- ✅ Single build pipeline for gert + gert-tui

**Cons:**
- ❌ Loses separate binary distribution story
- ❌ gert-tui binary versioning tied to gert releases
- ❌ Violates stated spec principle
- ❌ Couples TUI development to gert core

#### **Option C: Promote internal packages to public**

Move `internal/parser`, `internal/planner`, `internal/engine` → `pkg/parser_impl`, `pkg/planner_impl`, etc. Expose them publicly.

**Pros:**
- ✅ Unblocks immediately

**Cons:**
- ❌ Exposes unstable API surface
- ❌ Internal packages have unclear contracts
- ❌ Violates separation of concerns
- ❌ Not recommended for long-term maintenance

### Recommendation & Decision

**✅ OPTION A APPROVED**

**Rationale:**
1. Aligns with stated spec principle: "if you need anything new from gert, grow gert, not gert-tui"
2. gert-tui remains a separate, independently-versioned binary
3. Cleanest architectural boundary: TUI is a consumer, not a modifier of internals
4. Extensible: batch mode, dry-run mode, replay mode all live in pkg/run
5. Future-proofed: New TUI variants (CLI, web, mobile) can also use pkg/run

---

## Action Required

### Phase 1: Blocker 1 (go:embed) — Roy, immediate
- [ ] Move `/gert-tui/runbooks/` to `/gert-tui/cmd/gert-tui/runbooks/`
- [ ] Update `internal/embed/embed.go` comment to reflect new structure
- [ ] Verify build succeeds with updated embed path
- [ ] Commit: "refactor: relocate runbooks for go:embed compatibility"

### Phase 2: Blocker 2 (internal packages) — Gert team, sequential
- [ ] Add `pkg/run/run.go` to gert with `Runner` interface and `NewRunner` constructor
- [ ] Wire parser (from internal/ or public pkg/parser), planner, engine
- [ ] Document `pkg/run` contract and supported RunOptions (mode, vars, scenario dir, etc.)
- [ ] Add tests for pkg/run with example runbooks
- [ ] Merge to gert main

### Phase 3: TUI engine integration — Roy, after Phase 2
- [ ] Update gert-tui/cmd/gert-tui/main.go to call `pkg/run.NewRunner()`
- [ ] Wire TUI app to engine.RunHandle (via pkg/run)
- [ ] Implement step execution loop
- [ ] Commit: "feat: integrate gert v2 engine via pkg/run"

---

## Timeline

- **Blocker 1 (go:embed)**: ~30 min (Roy)
- **Blocker 2 pkg/run design**: ~1 hr (gert team review)
- **Blocker 2 pkg/run implementation**: ~2 hr (gert team)
- **Phase 3 TUI integration**: ~3 hr (Roy)

**Critical Path**: Blocker 1 → pkg/run implementation → TUI integration

---

## Document Version

- **Created**: 2026-04-29
- **Lead**: Deckard
- **Next Review**: After Phase 2 completion
