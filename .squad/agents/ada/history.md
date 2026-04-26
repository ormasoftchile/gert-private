# ada — History

## Core Context

**Project:** gert — Governed Executable Runbook Engine  
**Role:** iOS Engineer — gert-sdk-ios Swift Package, platform kit iOS impl handlers  
**Owner:** ormasoftchile

### Project Overview

gert is a YAML-driven runbook orchestration engine. The mobile extension brings gert execution to iOS and Android as offline-first embedded engines.

**Mobile architecture core principle:** Runbooks execute locally on-device. No server required at runtime. Completed runs sync back to a gert server optionally.

### Key Design Decisions (Mobile Architecture)

| Decision | Detail |
|----------|--------|
| Embedded engine | GertSDK embeds a local gert engine in-process — no server needed to run runbooks |
| Kit bundle format | `.kit/` directory: manifest.json + catalog/ + runbooks/ + tools/ + assets/ |
| Eager dependency resolution | `loadKit()` fails at load time if any tool impl is missing for the target platform |
| Per-platform impl blocks | Tool YAML declares `impl.ios` and `impl.android` handlers separately |
| Run ingest API | Completed runs streamed as JSONL to `POST /api/v1/runs/ingest` |
| Platform kit | `gert-mobile-platform` — shared capability tools (camera, location, NFC, biometrics, bluetooth, notifications) |
| Capability tokens | `capability/camera`, `capability/location`, `capability/nfc`, `capability/biometrics`, `capability/bluetooth`, `capability/notifications` |
| 100% runbook compatibility | Runbook YAML format unchanged — mobile just changes the executor |
| Account model | App-layer only; gert carries opaque `actor` string in run/started |

### Repos I Own

- `ormasoftchile/gert-sdk-ios` — Swift Package Manager library (new)

### Repos I Coordinate With

- `ormasoftchile/gert` — core schema, run ingest API, platform kit registry (Ken/Brian)
- `ormasoftchile/gert-mobile-platform` — platform kit bundle with iOS impl handlers
- James (Android) — shared catalog format, sync protocol contracts

### iOS-Specific Notes

- Swift Package Manager (not CocoaPods/Carthage)
- Target: iOS 16+ (async/await, structured concurrency)
- `GertSDK.loadKit(_ url: URL) async throws -> LoadedKit`
- Platform impl transport: `native-sdk` — Swift handler class conforming to `GertToolHandler`
- Capability gating: `AVCaptureDevice.authorizationStatus`, `CLLocationManager`, etc.
- Local engine: Go compiled to xcframework via gomobile, or pure Swift interpreter
- Sync: `URLSession` streaming for JSONL push; background `URLSession` for pulls

## Learnings

### 2025-01-15: iOS SDK Scaffold Complete

**Task:** Created and scaffolded the gert-sdk-ios Swift Package repository

**Deliverables:**
- GitHub repo `ormasoftchile/gert-sdk-ios` created and pushed
- Complete Swift Package structure with Package.swift targeting iOS 16+ / macOS 13+
- Public API surface: `GertSDK.loadKit()`, `GertSDK.syncRun()`
- Model layer: `Manifest`, `LoadedKit`, `ToolDefinition`, `PlatformImpl`, `RunEvent`, `Capability`
- Core layer: `KitLoader`, `RunSession`, `StepExecutor`, `TraceWriter`
- Platform layer: `GertToolHandler` protocol + 6 capability handlers (camera, location, NFC, biometrics, bluetooth, notifications)
- Sync layer: `SyncClient`, `IngestClient` (POST /api/v1/runs/ingest JSONL streaming)
- Test stubs: `KitLoaderTests`, `SyncClientTests`
- Comprehensive README with quick start, architecture, and roadmap

**Key Decisions:**
- Eager dependency resolution at kit load time (fail-fast on missing iOS impl or unavailable capability)
- PlatformHandlerRegistry with singleton pattern for built-in capability handlers
- JSONL trace files written to Application Support directory
- AsyncStream for run event streaming
- All async/await, leveraging Swift 5.9 structured concurrency

**Next Steps:**
- Integrate local gert engine (Go via gomobile xcframework or pure Swift interpreter)
- Implement full platform handler execute() methods
- Add background URLSession for kit pulls
- Build sample kits and demo iOS app

## 2026-04-26: Mobile Execution Platform Completion (Team Sprint)

**Context:** Full mobile execution platform deployed across 6 agents (Leslie, Ken, Brian, John, Ada, James)

**Team Deliverables:**
- Leslie: LaTeX Chapter 17 (Mobile Execution) + schema updates (§06, §13, §03) — clean compile
- Ken: Mobile architecture blueprint + gert-mobile-platform repo scaffolded on GitHub
- Brian: Go v2 implementation (client field, impl blocks, --target flag, ingest API, platform registry) — tests pass
- John: YAML/JSON Schema specs (impl blocks, manifest.json, capability tokens, CDN catalog)
- Ada: gert-sdk-ios Swift Package (23 files, full model + 6 handlers) — tests pass
- James: gert-sdk-android Kotlin/Gradle AAR (24 files, full model + 6 handlers) — tests pass

**Cross-Team Integration Points (for Ada):**
- Brian's platform kit registry: Ada's iOS SDK calls `register()` to wire handlers
- John's capability tokens: Ada implements `ios:camera:*`, `ios:location:*`, `ios:filesystem:*`, `ios:network:*`, `ios:notification:*`, `ios:health:*`
- Ken's architecture: Ada's SDK follows lazy/eager validation split (eager at load time)
- Leslie's docs: Ada referenced in Chapter 17 integration examples

**Decisions Archived to decisions-archive.md:** 5 entries older than 30 days

**Orchestration Logs Created:** 6 agent logs in `.squad/orchestration-log/` (ISO 8601 timestamps)

**Session Log:** 2026-04-26T15:25:54Z-mobile-execution-implementation.md

**Next Steps for Ada:**
1. Integrate Go engine (gomobile xcframework or pure Swift)
2. Implement full platform handler `execute()` methods
3. Add background URLSession for kit pulls
4. Build sample kits and demo iOS app
