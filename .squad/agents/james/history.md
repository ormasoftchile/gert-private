# james — History

## Core Context

**Project:** gert — Governed Executable Runbook Engine  
**Role:** Android Engineer — gert-sdk-android Kotlin library, platform kit Android impl handlers  
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

- `ormasoftchile/gert-sdk-android` — Kotlin/Gradle library, AAR artifact (new)

### Repos I Coordinate With

- `ormasoftchile/gert` — core schema, run ingest API, platform kit registry (Ken/Brian)
- `ormasoftchile/gert-mobile-platform` — platform kit bundle with Android impl handlers
- Ada (iOS) — shared catalog format, sync protocol contracts

### Android-Specific Notes

- Kotlin + Gradle (Kotlin DSL), published as AAR
- Target: Android API 26+ (Oreo, 2017) — covers 95%+ of active devices
- `GertSDK.loadKit(context: Context, url: Uri): LoadedKit` (suspend fun)
- Platform impl transport: `native-sdk` — Kotlin class implementing `GertToolHandler`
- Capability gating: `PackageManager.hasSystemFeature()`, runtime permissions via `ActivityResultContracts`
- Local engine: Go compiled via gomobile to AAR, or pure Kotlin interpreter of gert primitives
- Coroutines + Flow for async tool execution and run event streaming
- Sync: OkHttp streaming for JSONL push; WorkManager for background kit pulls

## Learnings

### 2026-04-19: Android SDK Scaffold Complete

**Task:** Create and scaffold the gert-sdk-android Kotlin library repository.

**What I did:**
1. Created `ormasoftchile/gert-sdk-android` GitHub repository (public)
2. Scaffolded complete Gradle Kotlin DSL project structure:
   - Root build config with version catalog (libs.versions.toml)
   - `gertsdk` library module with Android API 26+ (Oreo) target
   - Kotlin 1.9.22, Coroutines, OkHttp, Moshi dependencies
3. Implemented public API surface in `GertSDK.kt`:
   - `loadKit(context, uri)` — load from local bundle
   - `loadKit(context, name, version)` — load from registry
   - `syncRun(run, serverUrl, authToken)` — push completed run
4. Created core architecture modules (all stubs):
   - `KitLoader` — manifest parsing, dependency resolution, capability validation
   - `RunSession` — runbook execution orchestration
   - `StepExecutor` — tool dispatch logic
   - `TraceWriter` — JSONL trace event persistence
5. Defined model classes:
   - `Manifest`, `LoadedKit`, `ToolDefinition`, `PlatformImpl`
   - `RunEvent` sealed class hierarchy (8 event types)
   - `Capability` object with 6 token constants
6. Implemented `GertToolHandler` interface + 6 platform handler stubs:
   - `CameraHandler` — Camera2 API, FEATURE_CAMERA_ANY
   - `LocationHandler` — FusedLocationProvider, GPS/Network feature checks
   - `NFCHandler` — NfcManager, FEATURE_NFC + adapter enabled check
   - `BiometricsHandler` — BiometricManager, BIOMETRIC_STRONG|WEAK check
   - `BluetoothHandler` — BluetoothManager, FEATURE_BLUETOOTH + adapter check
   - `NotificationsHandler` — NotificationManager, areNotificationsEnabled() check
7. Created sync clients:
   - `SyncClient` — kit pull/push orchestration
   - `IngestClient` — JSONL streaming to POST /api/v1/runs/ingest
8. Added test stubs for KitLoaderTest and SyncClientTest
9. Wrote comprehensive README with quick start, capabilities table, architecture diagram
10. Committed and pushed to main branch

**Key decisions:**
- **Minimum API level 26 (Oreo, 2017)** — covers 95%+ of active devices, allows modern APIs
- **Kotlin Coroutines + Flow** — async tool execution, run event streaming
- **Eager dependency resolution** — fail at `loadKit()` time, not runtime
- **`checkAvailability()`** — each handler validates capability before execution
- **Platform impl transport `native-sdk`** — tool YAML declares handler class name
- **JSONL trace format** — matches gert server ingest API contract

**What's working:**
- Complete project structure compiles (all TODOs marked for future impl)
- Platform handlers reference correct Android APIs for each capability
- Model classes align with kit bundle manifest.json schema
- Public API surface matches mobile architecture design from history.md

**Next steps:**
1. Implement KitLoader manifest parsing (JSON → Manifest data class)
2. Implement capability validation logic (PackageManager feature checks)
3. Implement StepExecutor tool dispatch (type → handler routing)
4. Implement each of the 6 platform handlers' `execute()` methods
5. Implement SyncClient HTTP pull/push with OkHttp
6. Add integration tests with mock kit bundles
7. Create sample app module demonstrating end-to-end usage

**Repository:** https://github.com/ormasoftchile/gert-sdk-android  
**Commit:** 981f785 ("initial scaffold: GertSDK Android Kotlin library v0.1.0")
