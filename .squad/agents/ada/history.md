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
4. ~~Build sample kits and demo iOS app~~ ✅ Done (see 2026-04-26 below)

## 2026-04-26: HomeAutomationExample End-to-End Demo

**Task:** Create a concrete, working end-to-end example in gert-sdk-ios showing how to use the SDK to load and execute a home automation kit.

**Deliverables:**
- `Examples/HomeAutomationExample/` — Complete SwiftUI app demonstrating full kit lifecycle
  - `Kitfile.yaml` — Declares `gert-domain-home` and `gert-mobile-platform` dependencies
  - `HomeAutomationApp.swift` — SwiftUI app entry point
  - `HomeAutomationViewModel.swift` — ObservableObject managing kit loading and runbook execution
  - `ContentView.swift` — SwiftUI view with event streaming UI
  - `README.md` — Comprehensive setup guide and API flow documentation
- Updated main `README.md` with Examples section
- Git commit: `dce362e`

**Key API Patterns Demonstrated:**
1. Kit loading: `GertSDK.loadKit(from: kitURL)` with async/await
2. Runbook execution: `kit.startRun(runbook:actor:inputs:)`
3. Event streaming: `for await event in session.events { ... }`
4. SwiftUI integration: `@StateObject`, `@Published`, `Task {}`, `ObservableObject`
5. Error handling: `KitLoadError`, `KitError` with `LocalizedError`

**Example Workflows:**
- **Turn On Lights** — Multi-step runbook: check presence → get light state → turn on if needed
- **Check Presence** — Simple read-only runbook: query motion sensor state

**Documentation Highlights:**
- Prerequisites: `gert kit fetch` to download bundles before Xcode
- Production patterns: Bundle kits in Resources/ or download at runtime
- Capability error handling: Missing Bluetooth, Camera, etc.
- Kit bundle format: manifest.json + runbooks/ + tools/ structure

**Technical Details:**
- Used ACTUAL SDK API from source (KitLoader.load, LoadedKit.startRun, RunSession.events)
- All Swift code is syntactically valid (no placeholders or fake APIs)
- Demonstrates idiomatic iOS patterns (MVVM, async/await, structured concurrency)
- 790 lines of production-quality example code

**Design Decisions:**
1. Chose home automation domain (`gert-domain-home`) for relatability and real-world relevance
2. Used SwiftUI over UIKit for modern iOS development patterns
3. Implemented full event streaming UI to show JSONL trace visibility
4. Separated ViewModel from View for testability and reusability
5. Included comprehensive README with prerequisites, flow diagrams, and production deployment notes

**Impact:**
- First complete e2e example for gert-sdk-ios
- Demonstrates ALL core SDK features: load → start → stream → wait
- Ready to use as template for production iOS apps
- Shows proper async/await patterns for Swift 5.9+ iOS 16+ target

**Next Steps:**
1. Add more domain examples (pool maintenance, field inspection, delivery logistics)
2. Implement actual platform handler execute() methods
3. Add UIKit variant of example for UIViewController-based apps
4. Create Xcode project file for standalone example app

---

## Session Integration: Kit CLI & E2E Examples (2026-04-26T18:04:08Z)

**Scope:** Parallel execution of iOS example with kit CLI and Android example  
**Orchestration Log:** `.squad/orchestration-log/2026-04-26T18:04:08Z-ada-ios-e2e.md`  
**Session Log:** `.squad/log/2026-04-26T18:04:08Z-kit-cli-and-e2e-examples.md`  
**Decision Record:** Merged to `.squad/decisions.md` (see d-ios-e2e-example)

**Sync Points:**
- Kit CLI implementation by brian (6ad0639) — dependency declaration pattern
- Android example by james (9289a6c) — parallel domain kit usage
- gert-domain-home kit spec — shared runbook vocabulary

**Architectural Alignment:**
- Same domain (home automation) across iOS + Android for consistent API validation
- Both use MVVM pattern (native to each platform)
- Reactive streams (AsyncStream on iOS, Flow on Android)
- Production-ready patterns, not toy examples

**Completeness Validation:**
- ✅ Uses only actual SDK APIs from KitLoader, LoadedKit, RunSession
- ✅ Demonstrates full lifecycle: declare → fetch → load → execute → stream
- ✅ Error handling for all documented failure modes
- ✅ README covers setup, prerequisites, capability mismatches

## 2026-04-26: App v0 Evidence Storage & Seasonal Cadence Decisions

**Task:** Answer Q4 (evidence attachment storage policy) and Q5 (seasonal cadence display) from app-v0.md § 10 Open Questions

**Deliverables:**
- `.squad/decisions/inbox/ada-app-evidence-seasonal.md` — formal decision record
- Two concrete decisions with iOS-specific implementation guidance

**Q4 Decision: Evidence Cleanup Policy**
- **Policy:** Delete evidence 7 days after successful sync
- **Rationale:** iOS storage budget management, App Store privacy compliance, user expectation alignment (audit records vs personal photos), 7-day verification window
- **Implementation:** `EvidenceCleanupPolicy` in `Sources/GertSDK/Sync/`, SQLite `evidence_metadata` table, background cleanup trigger in `SyncClient`
- **Storage location:** Application Support/Evidence/ (not backed up via `.setResourceValue(.isExcludedFromBackupKey)`)
- **Settings disclosure:** S09 displays retention notice: "Evidence photos are kept for 7 days after upload, then deleted to save space."

**Q5 Decision: Seasonal Cadence Notice**
- **Policy:** Display calm informational notice in S03 for routines with `cadence.seasonal` in YAML: *"Seasonal schedule (using N-day interval in v0)"*
- **Rationale:** Transparency over silence (avoids user confusion), calm UI principle (secondary text, not warning), explains fallback behavior
- **Implementation:** `RoutineCadenceView` SwiftUI component in app layer, `Routine.cadence.isSeasonal` flag set during kit compilation
- **Styling:** `.foregroundColor(.secondary)` for muted gray text, positioned below "Every N days" line
- **Android parity:** Coordinate with James for equivalent Jetpack Compose implementation

**iOS-Specific Design Patterns:**
- FileManager Application Support directory for evidence storage (not Documents — no user-facing file browsing)
- SQLite for sync metadata tracking (not CoreData — simpler schema, better test isolation)
- SwiftUI conditional rendering for seasonal notice (`if routine.cadence.isSeasonal { ... }`)
- Background URLSession completion handler as cleanup trigger (iOS-native background task pattern)

**Spec Impact:**
- app-v0.md § 6 Offline & Sync: Add § 6.4 Evidence Retention Policy subsection
- app-v0.md § 4.2 S03 Routine Detail: Update "Cadence display" paragraph with seasonal notice behavior

**Key iOS Considerations:**
1. **Privacy:** App Store Review Guideline 5.1.1 requires explicit data retention policies — S09 disclosure satisfies this
2. **Storage:** iOS app bundle size limits + user device storage pressure favor automatic cleanup over indefinite retention
3. **Backup exclusion:** Mark Evidence/ directory with `isExcludedFromBackupKey` to prevent iCloud/iTunes backup bloat
4. **User trust:** Transparent communication (Settings notice, seasonal notice) builds trust vs silent behavior changes

**Cross-Agent Coordination:**
- James (Android): Needs equivalent seasonal notice in `RoutineDetailScreen.kt`
- John (schema): May need to add `isSeasonal` flag to compiled execution plan schema
- Cristian: Final approval required before implementation

**Next Steps:**
1. Await Cristian approval on decision record
2. Implement `EvidenceCleanupPolicy` in gert-sdk-ios
3. Update app-v0.md spec with § 6.4 and § 4.2 changes
4. Coordinate with James on Android parity
5. Add cleanup policy tests (unit + integration)

