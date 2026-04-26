# Mobile Execution Support — Implementation Decisions

**Author:** Brian (Go Programmer)  
**Date:** 2026-04-26  
**Status:** Implemented  

## Context

The gert v2 design team identified mobile execution (iOS/Android) as a key expansion surface. Mobile apps need to:
1. Execute runbooks locally with native SDK capabilities (camera, location, NFC, biometrics)
2. Upload completed runs to central server for audit/compliance
3. Validate platform compatibility at compile-time to avoid runtime failures

This document records the Go implementation decisions for mobile execution support.

---

## Decision 1: Client Field in run/started Event

**Problem:** Trace analysis and debugging require knowing which runtime surface initiated a run (CLI, server, mobile-ios, mobile-android).

**Decision:**
- Add `Client string` field to `pkg/engine.RunOptions` and `pkg/engine.Run`
- Emit `client` field in `run/started` trace event payload
- Set default values: "cli" for CLI commands, "server" for RPC server
- Mobile SDKs will set "mobile-ios" or "mobile-android"

**Rationale:**
- Simple string field, no enum enforcement (extensible for future surfaces)
- Threaded through existing RunOptions pattern (no new plumbing)
- Enables mobile-specific analytics and debugging workflows

**Implementation:**
- `pkg/engine/run.go`: Add Client field to RunOptions and Run structs
- `internal/engine/engine.go`: Include client in run/started payload
- `cmd/gert/run.go`: Set Client: "cli"
- `internal/serve/rpc.go`: Set Client: "server"

---

## Decision 2: Per-Platform impl Blocks in Tool Definitions

**Problem:** Tools need platform-specific invocation descriptors. A tool available on iOS might use native SDK calls, while the same tool on server uses HTTP. Mobile needs to know *how* to invoke each tool on each platform.

**Decision:**
- Add `Impl map[string]*PlatformImpl` to `schema.ToolDef`
- PlatformImpl has `Transport` (e.g., "native-sdk") and `Handler` (e.g., "GertSDK.Camera.capture")
- Absent platform key = capability unavailable on that platform
- Empty/nil Impl map = pure server-side tool (no mobile support)

**Rationale:**
- Explicit platform support declaration in tool metadata
- Avoids runtime errors when mobile tries to invoke unavailable tools
- Allows same logical tool to have different invocation patterns per platform

**Implementation:**
- `pkg/schema/tool.go`: Add Impl field and PlatformImpl struct
- YAML/JSON tags for serialization

---

## Decision 3: Compiler --target Flag for Platform Validation

**Problem:** Mobile builds need compile-time assurance that all tools are available on the target platform. Runtime discovery is too late (user already installed the app).

**Decision:**
- Add `gert compile` command with `--target ios|android|mobile` flag
- Scans all .tool.yaml files in current directory
- For each tool with impl blocks: validate target platform has an impl entry
- Pure server-side tools (no impl blocks) are allowed (ignored on mobile)
- Emits `manifest.json` with kit name, target, compiled-at timestamp
- "mobile" shorthand validates both ios and android

**Rationale:**
- Fail-fast at SDK build time, not app runtime
- Manifest.json enables mobile SDK to verify compatibility before execution
- Single command validates entire platform kit

**Implementation:**
- `cmd/gert/compile.go`: New command with flag parsing, tool scanning, validation, manifest generation
- `cmd/gert/main.go`: Register compile command in CLI router

**Example Usage:**
```bash
gert compile --target ios --kit-name gert-mobile-platform --output ./dist
# Validates all tools, emits dist/manifest.json
```

---

## Decision 4: Run Ingest API for Mobile Trace Upload

**Problem:** Mobile apps execute runbooks offline (no real-time server connection). After completion, the full trace must be uploaded to the central server for audit/compliance.

**Decision:**
- Add `POST /api/v1/runs/ingest` endpoint accepting JSONL stream
- Validates first event is `run/started` (ensures well-formed trace)
- Creates new run directory, writes events to trace.jsonl
- Returns `{"run_id": "<uuid>", "events_received": N}`
- Add `POST /api/v1/runs/{run-id}/attachments/{sha256}` for evidence uploads
- Validates SHA-256 hash matches body content
- Returns 201 on success, 400 on hash mismatch, 404 if run not found

**Rationale:**
- Mobile-first offline execution pattern (record locally, upload later)
- Atomic batch upload (not real-time streaming)
- Content-addressed attachments prevent corruption/tampering
- Server owns run ID generation (avoids mobile-generated ID collisions)

**Implementation:**
- `internal/serve/ingest.go`: Two new handlers with JSONL parsing, file I/O, SHA validation
- `internal/serve/server.go`: Register routes in router

**Wire Format:**
```http
POST /api/v1/runs/ingest
Content-Type: application/x-ndjson

{"event_id":"...","kind":"run/started",...}
{"event_id":"...","kind":"step/started",...}
{"event_id":"...","kind":"step/completed",...}
```

---

## Decision 5: Platform Kit Registry

**Problem:** Mobile SDKs and server need a canonical list of platform kits, their capabilities, and where to download them.

**Decision:**
- Create `pkg/platformkit` package
- `PlatformKitEntry` struct: Name, Version, Capabilities, URL
- `BuiltinPlatformKits` registry with initial entry: gert-mobile-platform v0.1.0
- Capabilities: camera, location, nfc, biometrics, bluetooth, notifications

**Rationale:**
- Centralized capability discovery (what can mobile do?)
- Version tracking for SDK compatibility
- URL field enables dynamic kit download (future: marketplace)

**Implementation:**
- `pkg/platformkit/registry.go`: Registry data structure and builtin entry
- `pkg/platformkit/doc.go`: Package documentation

---

## Cross-Cutting Concerns

### Backward Compatibility
- All changes are additive (new fields, new command, new endpoints)
- Existing runs without `client` field: omitted from trace (backward compatible)
- Existing tools without `impl` blocks: remain server-only (no regression)

### Security
- Attachment SHA-256 validation prevents content tampering
- Run ingest creates new UUID (mobile can't force a specific run ID)
- No authentication changes (uses existing JWT/bearer token middleware)

### Testing Strategy
- Existing engine tests pass (no regression)
- Manual testing: `gert compile --target mobile` with sample tools
- Integration testing deferred to mobile SDK repository

---

## Open Questions & Future Work

1. **Capability resolution at planning time:** Should the planner fail early if a tool requires a capability not available on the target platform? (Currently: validation only at compile-time)

2. **Dynamic kit download:** Should the server expose `GET /api/v1/platform-kits` to list available kits? (Currently: builtin registry only)

3. **Tool impl validation:** Should we validate Transport/Handler format at compile-time? (Currently: opaque strings)

4. **Run ingest idempotency:** Should the server reject duplicate ingests of the same trace? (Currently: creates new run ID every time)

---

## References

- `pkg/engine/run.go` — Run lifecycle and options
- `pkg/schema/tool.go` — Tool definition schema
- `cmd/gert/compile.go` — Platform kit compiler
- `internal/serve/ingest.go` — Run ingest API handlers
- `pkg/platformkit/registry.go` — Platform kit registry
