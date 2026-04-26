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
