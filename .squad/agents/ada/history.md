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
