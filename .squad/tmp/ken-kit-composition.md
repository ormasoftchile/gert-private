# Domain Kit Layered Composition Model

**Author:** Ken (Software Architect)  
**Date:** 2026-04-25  
**Status:** Design Sketch — Not Final Doc  
**Context:** Can a foundational Domain Kit be a dependency of a specialized Domain Kit?

---

## The Short Answer

Yes. A specialized kit imports a foundational kit as a versioned Go module. The composition point is a shared `CompilerRegistry` that is populated at process startup (load time) by each kit contributing its step type compilers. The output invariant — a flat gert core `ExecutionPlan` — is never broken because every step compiler, at every layer, ultimately emits `[]schema.FlowNode` using only core primitive types.

---

## 1. The Composition Model

### What a Kit Exposes

A foundational kit exposes three things:

1. **Model types** — domain value objects (`Task`, `Person`, `Approval`, `Chore`) as plain Go structs. These are the "nouns" of the domain vocabulary. They carry YAML tags and validation constraints.

2. **Step compiler functions** — `StepCompilerFunc` implementations, one per step type the kit defines. A step compiler takes a raw `yaml.Node` (the step's content from the authored runbook) and returns `([]schema.FlowNode, error)`. The returned nodes are core gert primitives only.

3. **A `KitBundle`** — a named collection of (step-type → compiler-function) pairs, ready to be merged into a `CompilerRegistry`. This is the integration surface for downstream kits.

```go
// Package kitruntime — shared infrastructure, lives in gert core or a shared module.

// StepCompilerFunc compiles one Kit-level step into core FlowNodes.
// Input: raw YAML node for the step's body (after type dispatch).
// Output: core schema nodes (cli, manual, tool, approve, branch, iterate, etc.)
type StepCompilerFunc func(ctx context.Context, node *yaml.Node, scope CompileScope) ([]schema.FlowNode, error)

// CompileScope carries context the compiler may need: variable bindings,
// governance config, run metadata. Read-only during compilation.
type CompileScope struct {
    Vars       map[string]string
    Governance *schema.GovernanceConfig
    KitMeta    KitMeta
}

// KitBundle is what a foundational kit exports.
type KitBundle struct {
    Name     string                       // e.g. "household"
    Version  string                       // semver
    Compilers map[string]StepCompilerFunc // key: qualified step type e.g. "household.chore"
}

// CompilerRegistry holds all registered step compilers, keyed by qualified type.
// Multiple kits contribute to the same registry; the specialized kit's main()
// (or its exported Bundle() function) merges all bundles before compilation begins.
type CompilerRegistry struct {
    compilers map[string]StepCompilerFunc
}

func NewRegistry() *CompilerRegistry { ... }

// Merge adds all compilers from a KitBundle into the registry.
// Panics on duplicate key — collision = programming error, not runtime error.
func (r *CompilerRegistry) Merge(b KitBundle) { ... }

// Dispatch looks up and calls the compiler for a given qualified step type.
func (r *CompilerRegistry) Dispatch(ctx context.Context, stepType string, node *yaml.Node, scope CompileScope) ([]schema.FlowNode, error) { ... }
```

### How the Specialized Kit References the Foundational Kit

The specialized kit (`gert-kit-home`) imports the foundational kit (`gert-kit-household`) as a Go module dependency:

```go
// gert-kit-home/go.mod
module github.com/ormasoftchile/gert-kit-home

require (
    github.com/ormasoftchile/gert-kit-household v0.3.0
    github.com/ormasoftchile/gert/v2             v2.1.0
)
```

At startup, the specialized kit builds its registry by merging the foundational kit's bundle, then adding its own compilers:

```go
// gert-kit-home/pkg/compiler/registry.go

func BuildRegistry() *kitruntime.CompilerRegistry {
    r := kitruntime.NewRegistry()
    r.Merge(household.Bundle()) // foundational kit contributes its compilers
    r.Merge(home.Bundle())      // specialized kit adds its own
    return r
}
```

`household.Bundle()` is exported from `gert-kit-household/pkg/compiler/bundle.go`:

```go
// gert-kit-household/pkg/compiler/bundle.go
func Bundle() kitruntime.KitBundle {
    return kitruntime.KitBundle{
        Name:    "household",
        Version: "0.3.0",
        Compilers: map[string]kitruntime.StepCompilerFunc{
            "household.chore":   compileChore,
            "household.approve": compileApprove,
            "household.notify":  compileNotify,
        },
    }
}
```

### Composition Point: Load Time + Compile Time

- **Go module dependency = compile time.** The type system enforces the contract. If `household.Bundle()` changes its signature, the specialized kit fails to build.
- **Registry merge = load time.** When the kit compiler process starts (or when the `BuildRegistry()` call executes in a test), the registry is assembled. Step type dispatch happens at this point.
- **Step lowering = compile time (authoring phase).** When the specialized kit's compiler processes a `.home.yaml` input, it runs step compilers from the registry. This is "compile time" in the sense that it precedes gert runtime execution.

---

## 2. Step Type Namespacing

**Decision: Prefixed step types.** Format: `<kit-name>.<step-type>`.

Examples:
```yaml
- type: household.chore
  ...
- type: household.approve
  ...
- type: home.morning-routine
  ...
```

### Why Not Implicit (Disjoint by Convention)

Implicit disjoint sets rely on kit authors not colliding. With two kits in scope, `approve` could mean `household.approve` or `core.approve`. The conflict is silent until runtime. The registry `Merge()` panic on duplicate key would catch it at load time — but only if both kits use the same bare type name. Still fragile. Convention doesn't scale to a public kit ecosystem.

### Why Not a `kit:` Field

```yaml
# Option rejected
- kit: household
  type: chore
```

This is more verbose, splits what is conceptually one thing into two fields, and makes grep/search harder. "What type is this step?" becomes a two-field lookup. The prefix form `household.chore` is self-contained, unambiguous, and survives copy-paste.

### Why Prefix Wins

- **Unambiguous in YAML.** A single `type:` field carries full identity.
- **Registry key = YAML type value.** No translation needed — the dispatcher uses the `type` field directly as the map key.
- **Readable for authors.** `household.chore` communicates both domain and action. Domain context is part of the name.
- **grep-friendly.** `grep 'type: household\.'` finds all household steps in a runbook corpus.
- **Collision-safe at registry level.** The `Merge()` call panics on duplicate — collision is caught at kit build time, not at author time.

### The Core Kit

The gert core step types (cli, manual, tool, approve, branch, iterate, etc.) do not get prefixed — they remain unqualified. They are not Kit-managed. Kit-managed types must always be qualified. This creates a clear boundary: unqualified = core primitive; qualified = Kit-authored.

---

## 3. Compiler Delegation

**Decision: Option B — shared `CompilerRegistry` with dispatch by step type.**

### Why Not Option A (Direct Function Calls)

The specialized kit would have to enumerate every foundational kit step type it might encounter:

```go
// home compiler — Option A (rejected)
switch step.Type {
case "household.chore":
    nodes, err = household.CompileChore(ctx, node, scope)
case "household.approve":
    nodes, err = household.CompileApprove(ctx, node, scope)
// ...must list every household type explicitly
```

This creates tight coupling. Adding a new step type to `household` requires updating `home`'s switch statement. The specialized kit must have full knowledge of the foundational kit's internals. Violates the open/closed principle.

### Why Not Option C (BaseCompiler Embedding)

```go
// Option C (rejected)
type HomeCompiler struct {
    household.BaseCompiler  // embedding
}
```

Go embedding works for single inheritance but breaks down with multiple foundational kits. If `home` depends on both `household` and `schedule`, there is no clean embedding. Also, `BaseCompiler` as a struct embeds behavior in a way that makes it hard to test in isolation — you can't inject a test double for the foundational compiler.

### Why Option B Wins

The `CompilerRegistry` is a simple map with a dispatch function. Each kit contributes its entries at startup. The specialized kit's compiler calls `registry.Dispatch(stepType, ...)` for every step, regardless of which kit owns it:

```go
// home compiler — Option B
func (c *HomeCompiler) CompileStep(ctx context.Context, step RawStep, scope Scope) ([]schema.FlowNode, error) {
    return c.registry.Dispatch(ctx, step.Type, step.Body, scope)
}
```

The specialized kit doesn't need to know which kit owns a given step type. The registry knows. New step types in the foundational kit are automatically available — the specialized kit just calls `r.Merge(household.Bundle())` and gets them all.

**Tradeoff acknowledged:** The registry approach requires that all bundles are merged before the first compile call. This is a startup invariant. It is enforced by making `BuildRegistry()` a required setup step (not lazy).

---

## 4. Model Composition

**Decision: Go embedding for model types, plus shared type imports.**

### The Three Options

**Go embedding** means specialized types include foundational types structurally:
```go
// home model embeds household model
type MorningRoutine struct {
    household.Routine                // embedded: ID, Label, Cadence
    Steps    []household.ChoreRef   // references foundational types
}
```

**Interface** means the foundational kit defines abstract behavior; the specialized kit implements it. Good for polymorphism, overkill for value objects.

**Separate** means no sharing — the kits share only compiler function signatures. Each kit defines its own model types from scratch.

### Why Embedding Wins

Model types in Domain Kits are **value objects** (data carriers), not behavior objects. They represent authoring-time domain concepts: `Chore`, `Approval`, `Person`. Go embedding is the idiomatic way to extend value types without losing type safety.

The specialized kit (home) uses foundational kit types (household) as building blocks:

```go
// gert-kit-household/pkg/model/types.go
type Chore struct {
    ID          string   `yaml:"id"`
    Label       string   `yaml:"label"`
    Description string   `yaml:"description"`
    Zone        string   `yaml:"zone"`
    Frequency   Cadence  `yaml:"cadence"`
    Assignee    string   `yaml:"assignee,omitempty"`
}

type Approval struct {
    ID       string   `yaml:"id"`
    Roles    []string `yaml:"roles"`
    Required int      `yaml:"required"`
    Timeout  string   `yaml:"timeout,omitempty"`
}

// gert-kit-home/pkg/model/types.go
import household "github.com/ormasoftchile/gert-kit-household/pkg/model"

type MorningRoutine struct {
    ID    string           `yaml:"id"`
    Label string           `yaml:"label"`
    Steps []RoutineStep    `yaml:"steps"`   // ordered sequence
}

type RoutineStep struct {
    Type  string            `yaml:"type"`   // "chore" | "approve" | "notify"
    Chore *household.Chore  `yaml:"chore,omitempty"`
    // ... other embedded step kinds
}
```

**Why not interfaces here:** The compiler doesn't need polymorphic dispatch over model types — it needs to read concrete field values from the YAML AST. Interfaces add indirection without benefit at the model layer. Reserve interfaces for the compiler functions (`StepCompilerFunc`), not the data types.

---

## 5. The Invariant: All Roads Lead to ExecutionPlan

**Invariant:** No matter how many Kit layers are involved, the output of the full compilation chain is always a flat gert core `ExecutionPlan`. Kit-specific step types never reach the gert runtime.

### How the Layered Model Preserves It

**Every `StepCompilerFunc` has the same return type:**
```go
type StepCompilerFunc func(ctx context.Context, node *yaml.Node, scope CompileScope) ([]schema.FlowNode, error)
```

`schema.FlowNode` is gert core's union type — it can only be `Step`, `IterateNode`, or `ParallelNode` using core primitive step types (cli, manual, tool, approve, etc.). A `StepCompilerFunc` cannot return Kit-level nodes because `schema.FlowNode` doesn't have a Kit-level variant. The type system enforces this.

**Composite step types expand recursively, then flatten:**

When the specialized kit's `home.morning-routine` compiler runs, it returns a sequence of `schema.FlowNode` that may include steps like `household.chore`. Wait — does that mean Kit-level nodes propagate?

No. The `morning-routine` compiler does NOT emit `household.chore` as a FlowNode. It calls `registry.Dispatch("household.chore", ...)` for each sub-step and concatenates the results. The foundational kit's `compileChore()` returns `[]schema.FlowNode` with core primitives. The specialized kit concatenates those core nodes.

```go
// home compiler for morning-routine
func compileMorningRoutine(ctx context.Context, node *yaml.Node, scope CompileScope) ([]schema.FlowNode, error) {
    var routine model.MorningRoutine
    if err := node.Decode(&routine); err != nil { return nil, err }

    var allNodes []schema.FlowNode
    for _, step := range routine.Steps {
        // Dispatch to the foundational kit's compiler for each sub-step.
        // Returns []schema.FlowNode with CORE primitives only.
        nodes, err := scope.Registry.Dispatch(ctx, step.QualifiedType(), step.Body, scope)
        if err != nil { return nil, err }
        allNodes = append(allNodes, nodes...)
    }
    return allNodes, nil  // flat list of core nodes
}
```

The key is that `Dispatch()` always returns core nodes. The composite step expands in-place. The final `[]schema.FlowNode` slice contains no Kit references — only cli, manual, tool, approve, branch, iterate.

**Then the planner flattens everything:**

The gert core planner receives the compiled `runbook/v2` YAML (after the Kit compiler writes it to disk or to a byte buffer). The planner calls its own `Plan()` which produces an `ExecutionPlan` with a flat `[]ResolvedStep`. The planner is unaware that any Kit was involved.

**Three-level guarantee:**
1. `StepCompilerFunc` return type (`[]schema.FlowNode`) prevents Kit-level nodes.
2. Composite compilers call `Dispatch()` recursively, not by emitting raw Kit steps.
3. The gert core planner only accepts `runbook/v2` YAML — Kit YAML is a pre-processing artifact.

---

## 6. Worked Example

### The Kits

`gert-kit-household` (foundational): defines `household.chore`, `household.approve`, `household.notify`  
`gert-kit-home` (specialized): imports household, adds `home.morning-routine` as a composite step

### YAML: A Home Runbook Using Both Kits

```yaml
# casa-santiago.home.yaml
# Compiled by gert-kit-home compiler (which includes gert-kit-household compilers)
apiVersion: runbook/v2+home/v1
id: morning-routine-weekday
name: Weekday Morning Routine
kind: reference

vars:
  day: monday

flow:
  - type: home.morning-routine   # specialized kit step
    id: morning_core
    label: Core Morning Tasks
    steps:
      - type: household.chore    # foundational kit step, used inline
        id: make_coffee
        label: Make coffee
        zone: kitchen
        cadence: { frequency: daily }

      - type: household.chore
        id: feed_pets
        label: Feed cats
        zone: utility
        cadence: { frequency: daily }

      - type: household.approve  # foundational kit approval gate
        id: morning_sign_off
        label: Morning checklist sign-off
        roles: [resident]
        required: 1
        timeout: 30m

      - type: household.notify
        id: done_notify
        label: Notify family
        channel: push
        message: "Morning routine complete"
```

After compilation by `gert-kit-home`, the output is standard `runbook/v2`:

```yaml
# Compiled output: runbook/v2 YAML (no kit types remain)
apiVersion: runbook/v2
id: morning-routine-weekday
name: Weekday Morning Routine
kind: reference

flow:
  - type: manual         # from household.chore → manual step
    id: make_coffee
    title: Make coffee
    subtitle: "[kitchen] daily"

  - type: manual
    id: feed_pets
    title: Feed cats
    subtitle: "[utility] daily"

  - type: approve        # from household.approve → approve gate
    id: morning_sign_off
    title: Morning checklist sign-off
    roles: [resident]
    required: 1
    timeout: 30m

  - type: tool           # from household.notify → tool step
    id: done_notify
    title: Notify family
    tool: notify
    action: push
    args:
      message: Morning routine complete
```

### Go Interface Sketch

```go
// --- gert-kit-household/pkg/compiler/bundle.go ---

package compiler

import (
    "github.com/ormasoftchile/gert/v2/pkg/schema"
    "github.com/ormasoftchile/gert-kit-household/pkg/model"
    kitruntime "github.com/ormasoftchile/gert/kitruntime"
)

func Bundle() kitruntime.KitBundle {
    return kitruntime.KitBundle{
        Name:    "household",
        Version: "0.3.0",
        Compilers: map[string]kitruntime.StepCompilerFunc{
            "household.chore":   compileChore,
            "household.approve": compileApprove,
            "household.notify":  compileNotify,
        },
    }
}

func compileChore(ctx context.Context, node *yaml.Node, scope kitruntime.CompileScope) ([]schema.FlowNode, error) {
    var chore model.Chore
    if err := node.Decode(&chore); err != nil {
        return nil, fmt.Errorf("household.chore: %w", err)
    }
    step := schema.Step{
        ID:    chore.ID,
        Type:  schema.StepTypeManual,
        Title: chore.Label,
        Subtitle: fmt.Sprintf("[%s] %s", chore.Zone, chore.Frequency.Human()),
    }
    return []schema.FlowNode{{Step: &step}}, nil
}

// compileApprove and compileNotify follow the same pattern.


// --- gert-kit-home/pkg/compiler/registry.go ---

package compiler

import (
    household "github.com/ormasoftchile/gert-kit-household/pkg/compiler"
    kitruntime "github.com/ormasoftchile/gert/kitruntime"
)

func BuildRegistry() *kitruntime.CompilerRegistry {
    r := kitruntime.NewRegistry()
    r.Merge(household.Bundle())  // foundational kit
    r.Merge(homeBundle())        // specialized kit
    return r
}

func homeBundle() kitruntime.KitBundle {
    return kitruntime.KitBundle{
        Name:    "home",
        Version: "0.1.0",
        Compilers: map[string]kitruntime.StepCompilerFunc{
            "home.morning-routine": compileMorningRoutine,
        },
    }
}

func compileMorningRoutine(ctx context.Context, node *yaml.Node, scope kitruntime.CompileScope) ([]schema.FlowNode, error) {
    var routine model.MorningRoutine
    if err := node.Decode(&routine); err != nil {
        return nil, fmt.Errorf("home.morning-routine: %w", err)
    }

    var allNodes []schema.FlowNode
    for _, step := range routine.Steps {
        nodes, err := scope.Registry.Dispatch(ctx, step.QualifiedType(), step.Body, scope)
        if err != nil {
            return nil, fmt.Errorf("home.morning-routine step %s: %w", step.ID, err)
        }
        allNodes = append(allNodes, nodes...)
    }
    return allNodes, nil
}


// --- gert-kit-home/pkg/compiler/compiler.go ---

// Compiler is the top-level entry point for the home kit.
type Compiler struct {
    registry *kitruntime.CompilerRegistry
}

func New() *Compiler {
    return &Compiler{registry: BuildRegistry()}
}

// Compile transforms a .home.yaml document into a core runbook/v2 YAML document.
func (c *Compiler) Compile(ctx context.Context, input []byte) ([]byte, error) {
    // 1. Parse input as Home Kit AST.
    // 2. For each flow step, call c.registry.Dispatch(stepType, node, scope).
    // 3. Assemble schema.Runbook with the returned []schema.FlowNode.
    // 4. Marshal to YAML → return []byte (valid runbook/v2 YAML).
}
```

---

## Summary of Decisions

| Question | Decision | Rationale |
|----------|----------|-----------|
| Composition mechanism | `KitBundle` + `CompilerRegistry.Merge()` | Loose coupling; each kit registers its own compilers; no inheritance hierarchy |
| Step type namespacing | Prefixed: `<kit>.<type>` | Unambiguous, self-documenting, grep-friendly, collision-detected at startup |
| Compiler delegation | Option B: shared registry dispatch | No tight coupling; new foundational types available automatically |
| Model composition | Go embedding + direct import of foundational types | Idiomatic for value objects; no polymorphism needed at model layer |
| Core invariant | `StepCompilerFunc` return type = `[]schema.FlowNode` (core only) + composite compilers call `Dispatch()` for sub-steps | Type system + architectural pattern together prevent Kit leakage |

---

## 7. Border Cases & Error Handling

### 7.1 Unknown Step Type at Dispatch

**Scenario:** `Dispatch()` is called with a `stepType` key that has no entry in the registry (e.g., a typo, a kit that was never merged, or a version skew).

**Specified behavior:** `Dispatch()` returns a structured error — never panics. The caller (a composite compiler or the top-level compilation loop) receives an error it can wrap with positional context.

**Invariant:** `Dispatch()` must not panic on missing key. The registry is a closed map after `BuildRegistry()` completes; any lookup miss is a configuration error, not a programmer error.

```go
func (r *CompilerRegistry) Dispatch(ctx context.Context, stepType string, node *yaml.Node, scope CompileScope) ([]schema.FlowNode, error) {
    fn, ok := r.compilers[stepType]
    if !ok {
        return nil, fmt.Errorf("kitruntime: unknown step type %q (registry contains %d types)", stepType, len(r.compilers))
    }
    return fn(ctx, node, scope)
}
```

Error message format: `kitruntime: unknown step type "<qualified-type>" (registry contains N types)`. The count aids diagnostics when the registry was accidentally built empty or with the wrong bundles.

---

### 7.2 Version Conflicts Between Kits

**Scenario:** Two kits both call `r.Merge(household.Bundle())` in the same registry build, or two different versions of `gert-kit-household` are transitively resolved by Go MVS, each exporting a `Bundle()` that registers the same step keys.

**Specified behavior:** If the same key is merged twice, `Merge()` panics (programming error, caught in tests). Go MVS resolves the module version conflict before the binary is built — there is exactly one `household.Bundle()` in the final binary. The registry-level panic is a safety net for the rare case where two kits independently call `Merge` for the same bundle.

**Invariant — merge ownership:** A foundational kit **exports `Bundle()` but never calls `r.Merge()` itself**. Only the top-level kit (or the application's `BuildRegistry()` function) calls `r.Merge()`. This means there is exactly one site in the codebase that merges each bundle, and it is always visible in the specialized kit's `BuildRegistry()`.

**Convention:** If kit A depends on foundational kit F, and kit B also depends on F, and a composite registry uses both A and B, the `BuildRegistry()` for that composite must merge F exactly once, then A's own compilers, then B's own compilers. F must never appear in both A's and B's exported `Bundle()`.

---

### 7.3 Overlapping Prefixes / Namespace Collision

**Scenario:** Two kits register step types under the same prefix — e.g., both `gert-kit-home` and `gert-kit-homepro` define `home.routine`. The `Merge()` panic catches this at startup, but only if the key is an exact match.

**Specified behavior:** An exact key collision causes a startup panic (programming error). There is no partial overlap detection (e.g., one kit's prefix being a substring of another's type name) — this is a documentation/convention problem, not a runtime problem.

**Invariant — prefix reservation:** Each kit owns its prefix exclusively. The `gert-kit.yaml` manifest **must** declare a `prefix:` field; this prefix is the kit author's namespace claim. The registry of published prefixes is maintained in the `gert` project's `docs/kit-prefixes.md` (or equivalent community registry). Kit authors must not use a prefix they do not own.

**Guard:** The `Merge()` implementation panics with a message naming both the incoming bundle and the existing registration, so the offending kit is immediately identifiable:

```go
func (r *CompilerRegistry) Merge(b KitBundle) {
    for key := range b.Compilers {
        if _, exists := r.compilers[key]; exists {
            panic(fmt.Sprintf("kitruntime: duplicate step type %q — registered by %s, also claimed by %s", key, r.owners[key], b.Name))
        }
        r.compilers[key] = b.Compilers[key]
        r.owners[key] = b.Name
    }
}
```

The registry tracks an `owners map[string]string` (step key → kit name) for diagnostic-quality panic messages.

---

### 7.4 Partial Registration / Startup Order

**Scenario:** A composite step compiler in kit B calls `scope.Registry.Dispatch("household.chore", ...)` but `household.Bundle()` has not yet been merged into the registry (e.g., bundles merged in the wrong order, or `BuildRegistry()` called lazily after compilation begins).

**Specified behavior:** This is a programming error. `Dispatch()` returns an "unknown step type" error (§7.1). If this error propagates unhandled, the compile call fails with a clear message. There is no silent fallback.

**Invariant — startup order:** `BuildRegistry()` must complete and return a fully-populated `*CompilerRegistry` **before** any `Dispatch()` call is possible. The constructor pattern enforces this:

```go
// CORRECT — registry is fully built before the Compiler is usable
func New() *Compiler {
    return &Compiler{registry: BuildRegistry()}
}
```

`BuildRegistry()` must never be deferred, lazy-initialized, or called conditionally. The `Compiler` struct must not expose a zero-value that allows `Dispatch()` without a registry (guard with a nil check that panics on first use if registry is nil). There is no "register later" API on `CompilerRegistry` after construction.

---

### 7.5 Nil or Empty Step Body

**Scenario:** A step in the authored YAML has a null body (`node:` with no value, or an explicit `null`), and `Dispatch()` is called with `node == nil` or with a `*yaml.Node` whose `Kind == yaml.ScalarNode` and `Value == "null"`.

**Specified behavior — contract split into two layers:**

1. **Loader validates before dispatch.** The kit's input loader (the layer that walks the raw YAML AST and extracts step nodes) is responsible for rejecting null step bodies before calling `Dispatch()`. If a step's body is null and the step type requires a body, the loader returns a structured validation error naming the step `id` and `type`. `Dispatch()` is never called with a nil node.

2. **`StepCompilerFunc` receives a non-nil node as a contract.** Kit authors may assume `node` is non-nil. If a step type legitimately allows an empty body (e.g., a no-args step), the loader passes the empty mapping node, not nil.

**Invariant:** `Dispatch()` passes `node` to the compiler unchanged; it does not inject a synthetic empty node. The loader is the validation boundary, not the dispatcher.

```go
// In the loader, before calling Dispatch:
if node == nil || (node.Kind == yaml.ScalarNode && node.Value == "null") {
    return nil, fmt.Errorf("step %q (type %q): body is required but was null", stepID, stepType)
}
```

---

## Open Questions (Not Blocking)

1. **`kitruntime` module location.** Does it live in `gert/v2/pkg/kitruntime`, in a separate `gert-kit-sdk` module, or is each kit expected to define its own `StepCompilerFunc` type with compatible signatures? Recommend: `gert/v2/pkg/kitruntime` — shared SDK type in core, kits depend on it.

2. **Circular expansion detection.** If `home.morning-routine` expands into a step that somehow re-emits `home.morning-routine` (misconfigured), the registry would recurse infinitely. Add a `depth` counter to `CompileScope` and return an error at depth > N (suggest 10).

3. **Kit manifest (`gert-kit.yaml`) for layered kits.** The `gert-kit.yaml` manifest format (§5.5 of the Domain Kit chapter) should add a `depends:` block listing foundational kits, analogous to `go.mod`. This makes dependencies declarative and enables tooling to validate that the registry is built correctly.
