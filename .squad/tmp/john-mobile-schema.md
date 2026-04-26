# Mobile Execution: YAML/JSON Schema Specification

**Author:** John (YAML/Schema Specialist)  
**Date:** 2026-04-26  
**Status:** Specification Draft  
**Audience:** Schema Design, SDK Development, Kit Compilation

---

## Overview

This document specifies the complete schema additions needed to support mobile execution in gert v2. Mobile execution enables runbooks and tools to run on iOS and Android devices, in addition to traditional CLI and server execution. The specification covers:

1. **Per-platform `impl` block** — Tool definition schema for platform-specific handlers
2. **Kit bundle `manifest.json`** — Schema for compiled kit metadata
3. **Event tracing extensions** — `client` field for execution origin tracking
4. **Capability tokens** — Canonical format for device capabilities
5. **Platform kit catalog** — CDN-published kit index schema

All schemas are specified in JSON Schema Draft 2020-12, with canonical YAML examples for each component.

---

## 1. Per-Platform `impl` Block in `.tool.yaml`

### Use Case

A tool definition may support execution on multiple transports/platforms. The `impl` block declares platform-specific handler information, enabling the same tool to have different implementations for iOS, Android, CLI, and server environments.

### YAML Example

```yaml
$schema: "https://schemas.gert.dev/tool/v2.json"
apiVersion: tool/v2
name: camera.capture
version: "1.0.0"
description: Captures a photo or video using the device camera

requires-capabilities:
  - capability/camera

inputs:
  - name: mode
    type: string
    enum: [photo, video]
    default: photo
    description: Capture mode

outputs:
  - name: uri
    type: string
    description: File URI (file://, content://, or https://)
  
  - name: sha256
    type: string
    description: SHA256 hex digest of captured file
  
  - name: metadata
    type: object
    description: Platform-specific metadata (width, height, timestamp)

impl:
  ios:
    transport: native-sdk
    handler: GertSDK.Camera.capture
  
  android:
    transport: native-sdk
    handler: com.gert.platform.tools.CameraCaptureTool
  
  cli:
    transport: http
    handler: "http://localhost:9090/tools/camera-capture"
  
  server:
    transport: grpc
    handler: "gert.tools.Camera/Capture"
```

### JSON Schema for `impl` Property

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Tool Implementation Map",
  "description": "Platform-specific implementations of a tool. Keys are platform identifiers; values are PlatformImpl objects.",
  "type": "object",
  "minProperties": 0,
  "maxProperties": 100,
  "additionalProperties": {
    "$ref": "#/$defs/PlatformImpl"
  },
  "examples": [
    {
      "ios": {
        "transport": "native-sdk",
        "handler": "GertSDK.Camera.capture"
      },
      "android": {
        "transport": "native-sdk",
        "handler": "com.gert.platform.tools.CameraCaptureTool"
      }
    },
    {
      "cli": {
        "transport": "local-process",
        "handler": "/usr/local/bin/gert-camera"
      }
    }
  ],
  "$defs": {
    "PlatformImpl": {
      "type": "object",
      "title": "Platform Implementation Descriptor",
      "description": "Handler and transport for a single platform target.",
      "required": ["transport", "handler"],
      "additionalProperties": false,
      "properties": {
        "transport": {
          "type": "string",
          "title": "Transport Mechanism",
          "description": "How the handler is invoked: native SDK, HTTP, gRPC, or local process.",
          "enum": ["native-sdk", "http", "grpc", "local-process"],
          "examples": ["native-sdk", "http", "grpc", "local-process"]
        },
        "handler": {
          "type": "string",
          "title": "Handler Identifier",
          "description": "Opaque, platform-specific handler identifier. Format depends on transport:\n- native-sdk: Class/method path (e.g., 'GertSDK.Camera.capture')\n- http: URL (e.g., 'http://localhost:9090/tools/capture')\n- grpc: Service/method (e.g., 'gert.tools.Camera/Capture')\n- local-process: Executable path (e.g., '/usr/local/bin/camera-tool')",
          "minLength": 1,
          "maxLength": 500,
          "examples": [
            "GertSDK.Camera.capture",
            "com.gert.platform.tools.CameraCaptureTool",
            "http://localhost:9090/tools/camera-capture",
            "gert.tools.Camera/Capture",
            "/usr/local/bin/camera-tool"
          ]
        }
      }
    }
  }
}
```

### Integration with Tool Definition Schema

The `impl` property is **optional** at the tool definition root level:

```json
{
  "type": "object",
  "title": "Tool Definition",
  "properties": {
    "apiVersion": { "type": "string", "enum": ["tool/v2"] },
    "$schema": { "type": "string" },
    "name": { "type": "string" },
    "version": { "type": "string" },
    "description": { "type": "string" },
    "requires-capabilities": { "type": "array", "items": { "type": "string" } },
    "inputs": { "type": "array" },
    "outputs": { "type": "array" },
    "impl": {
      "type": "object",
      "description": "Optional platform-specific implementations. Absent = server-only tool.",
      "additionalProperties": { "$ref": "#/$defs/PlatformImpl" }
    }
  },
  "required": ["apiVersion", "name", "version", "description"],
  "examples": [
    {
      "apiVersion": "tool/v2",
      "$schema": "https://schemas.gert.dev/tool/v2.json",
      "name": "camera.capture",
      "version": "1.0.0",
      "description": "Captures a photo or video",
      "impl": {
        "ios": {
          "transport": "native-sdk",
          "handler": "GertSDK.Camera.capture"
        }
      }
    }
  ]
}
```

### Platform Names (Keys)

Platform names are **case-sensitive strings**. Builtin names include:

| Platform | Usage | Notes |
|----------|-------|-------|
| `ios` | Apple iOS device | Native SDK or HTTP |
| `android` | Google Android device | Native SDK or HTTP |
| `cli` | Command-line execution | Local process, HTTP, or gRPC |
| `server` | Server-side execution | HTTP or gRPC |
| `wasm` | WebAssembly runtime | Reserved for future use |

Custom platform names are allowed (e.g., `macos`, `windows`, `tvos`, `custom-device`).

### Validation Rules

1. **Transport + handler must be compatible:**
   - `native-sdk`: handler is class/method path (no slashes for packages, dots for method access)
   - `http`: handler must be valid URL starting with `http://` or `https://`
   - `grpc`: handler must be `package.Service/Method` format
   - `local-process`: handler should be absolute or relative executable path

2. **At least one platform:** If `impl` is present, it MUST have at least one entry.

3. **Handler uniqueness:** The same handler MAY NOT appear in multiple platforms within one tool (enforced at kit compilation time).

4. **Backward compatibility:** Omitting `impl` is allowed — indicates server-only tool (no mobile/CLI support).

---

## 2. Kit Bundle Manifest Schema (`manifest.json`)

### Use Case

When a kit (`.kit/` bundle) is compiled, it includes a `manifest.json` at the root. The manifest describes the kit's contents, target platform(s), dependencies, and checksums. The manifest is consumed by:

- Mobile SDK: Validates kit compatibility and downloads required kits
- Compiler: Embedded in compiled runbooks for audit trails
- Runtime: Enables offline verification of kit origins

### YAML Example (for reference)

```yaml
# Note: manifest.json is always JSON, not YAML. Shown in YAML for readability.
name: pool-maintenance
version: "1.2.0"
target: [ios, android]
description: Pool maintenance runbooks and tools
requires:
  - name: gert-mobile-platform
    version: ">=0.1.0"
runbooks:
  - id: pool-weekly-check
    name: Weekly Pool Maintenance Check
    version: "1.2.0"
  - id: filter-backwash
    name: Filter Backwash Procedure
    version: "1.1.0"
  - id: chemical-balance
    name: Chemical Balance Check
    version: "1.3.0"
provides-capabilities:
  - capability/camera
  - capability/location
compiled-at: "2026-04-26T15:00:00Z"
compiler-version: "2.0.0"
checksum: "sha256:abc123def456..."
```

### JSON Schema for `manifest.json`

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Kit Bundle Manifest",
  "description": "Metadata for a compiled kit bundle. Published at kit root as manifest.json.",
  "type": "object",
  "required": [
    "name",
    "version",
    "description",
    "compiled-at",
    "compiler-version",
    "checksum"
  ],
  "additionalProperties": false,
  "properties": {
    "name": {
      "type": "string",
      "title": "Kit Name",
      "description": "Unique kit identifier (kebab-case). Must match the directory name containing the kit.",
      "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
      "minLength": 1,
      "maxLength": 100,
      "examples": ["pool-maintenance", "gert-mobile-platform", "incident-response"]
    },
    "version": {
      "type": "string",
      "title": "Kit Version",
      "description": "Semantic version (major.minor.patch). See https://semver.org.",
      "pattern": "^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-((?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?$",
      "examples": ["1.0.0", "1.2.0", "0.1.0-alpha", "2.0.0-rc1+build.123"]
    },
    "target": {
      "type": "array",
      "title": "Target Platforms",
      "description": "List of platforms this kit supports.",
      "minItems": 1,
      "maxItems": 10,
      "uniqueItems": true,
      "items": {
        "type": "string",
        "enum": ["ios", "android", "cli", "server", "wasm"],
        "examples": ["ios", "android"]
      },
      "examples": [["ios", "android"], ["ios"], ["cli", "server"]]
    },
    "description": {
      "type": "string",
      "title": "Kit Description",
      "description": "Human-readable description of the kit's purpose.",
      "minLength": 10,
      "maxLength": 500,
      "examples": ["Pool maintenance runbooks", "Home automation incident response"]
    },
    "requires": {
      "type": "array",
      "title": "Kit Dependencies",
      "description": "Array of kits that this kit depends on.",
      "minItems": 0,
      "maxItems": 50,
      "items": {
        "$ref": "#/$defs/KitDependency"
      },
      "examples": [
        [
          {
            "name": "gert-mobile-platform",
            "version": ">=0.1.0"
          }
        ]
      ]
    },
    "runbooks": {
      "type": "array",
      "title": "Runbook Entries",
      "description": "List of runbooks bundled in this kit.",
      "minItems": 0,
      "maxItems": 1000,
      "items": {
        "$ref": "#/$defs/RunbookEntry"
      },
      "examples": [
        [
          {
            "id": "pool-weekly-check",
            "name": "Weekly Pool Maintenance Check",
            "version": "1.2.0"
          }
        ]
      ]
    },
    "provides-capabilities": {
      "type": "array",
      "title": "Provided Capabilities",
      "description": "List of device capabilities provided by tools in this kit.",
      "minItems": 0,
      "maxItems": 100,
      "uniqueItems": true,
      "items": {
        "type": "string",
        "pattern": "^capability/[a-z0-9]+(-[a-z0-9]+)*$"
      },
      "examples": [
        ["capability/camera", "capability/location", "capability/nfc"]
      ]
    },
    "compiled-at": {
      "type": "string",
      "title": "Compilation Timestamp",
      "description": "ISO 8601 timestamp when the kit was compiled.",
      "format": "date-time",
      "examples": ["2026-04-26T15:00:00Z", "2026-04-26T15:00:00+00:00"]
    },
    "compiler-version": {
      "type": "string",
      "title": "Compiler Version",
      "description": "Version of the gert compiler used to build this kit.",
      "pattern": "^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*).*$",
      "examples": ["2.0.0", "2.0.0-beta.1"]
    },
    "checksum": {
      "type": "string",
      "title": "Kit Archive Checksum",
      "description": "SHA256 hex digest of the entire kit .zip file (computed before manifest.json is written).",
      "pattern": "^sha256:[a-f0-9]{64}$",
      "examples": ["sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1"]
    }
  },
  "$defs": {
    "KitDependency": {
      "type": "object",
      "title": "Kit Dependency",
      "description": "Reference to another kit required by this kit.",
      "required": ["name", "version"],
      "additionalProperties": false,
      "properties": {
        "name": {
          "type": "string",
          "title": "Dependency Kit Name",
          "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
          "examples": ["gert-mobile-platform", "common-tools"]
        },
        "version": {
          "type": "string",
          "title": "Version Requirement",
          "description": "NPM-style version specifier: =1.0.0, >=1.0.0, ~1.2.0, ^1.0.0, 1.x, etc.",
          "examples": [">=0.1.0", "^1.0.0", "~2.1.0", "1.2.3"]
        }
      }
    },
    "RunbookEntry": {
      "type": "object",
      "title": "Runbook Metadata in Kit",
      "description": "Reference to a runbook included in this kit.",
      "required": ["id", "name", "version"],
      "additionalProperties": false,
      "properties": {
        "id": {
          "type": "string",
          "title": "Runbook ID",
          "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
          "examples": ["pool-weekly-check", "incident-response"]
        },
        "name": {
          "type": "string",
          "title": "Runbook Display Name",
          "minLength": 1,
          "maxLength": 200,
          "examples": ["Weekly Pool Maintenance Check"]
        },
        "version": {
          "type": "string",
          "title": "Runbook Version",
          "pattern": "^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*).*$",
          "examples": ["1.0.0", "1.2.0"]
        }
      }
    }
  },
  "examples": [
    {
      "name": "pool-maintenance",
      "version": "1.2.0",
      "target": ["ios", "android"],
      "description": "Pool maintenance runbooks and tools",
      "requires": [
        {
          "name": "gert-mobile-platform",
          "version": ">=0.1.0"
        }
      ],
      "runbooks": [
        {
          "id": "pool-weekly-check",
          "name": "Weekly Pool Maintenance Check",
          "version": "1.2.0"
        }
      ],
      "provides-capabilities": ["capability/camera", "capability/location"],
      "compiled-at": "2026-04-26T15:00:00Z",
      "compiler-version": "2.0.0",
      "checksum": "sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
    }
  ]
}
```

### Kit Manifest Validation Rules

1. **Name format:** Kebab-case, matching the directory containing the kit.
2. **Version:** Semantic versioning required. Pre-release versions allowed (e.g., `1.0.0-beta`).
3. **Target:** At least one platform. Must be from the enum.
4. **Checksum:** Must be `sha256:<64-hex-digits>`. Computed over the `.zip` before manifest is written.
5. **Compiled-at:** RFC 3339 timestamp (ISO 8601 with timezone).
6. **Runbooks:** Each runbook ID must be unique within the manifest.
7. **Capabilities:** Must be in `capability/<name>` format (kebab-case names).
8. **Dependencies:** Version specifiers follow npm semver syntax.

---

## 3. `client` Field in `run/started` Event

### Use Case

Execution traces (`run/started` JSONL events) need to record which client initiated the run. This enables audit trails, analytics, and client-specific behavior in the runtime.

### Trace Event Schema Extension

The existing `run/started` event gains a new optional field:

```json
{
  "event": {
    "kind": "run/started",
    "id": "run-uuid",
    "timestamp": "2026-04-26T15:30:00Z",
    "runbook": {
      "id": "pool-weekly-check",
      "name": "Weekly Pool Maintenance Check",
      "path": "runbooks/pool-weekly-check.yaml"
    },
    "client": "mobile-ios",
    "initiator": "user@example.com",
    "context": { }
  }
}
```

### JSON Schema Addition

```json
{
  "type": "object",
  "title": "RunStarted Event with Client Field",
  "properties": {
    "event": {
      "type": "object",
      "properties": {
        "kind": {
          "const": "run/started"
        },
        "client": {
          "type": "string",
          "title": "Execution Client",
          "description": "Client that initiated the run. Defaults to 'cli' if omitted (backward compatible).",
          "enum": ["cli", "server", "mobile-ios", "mobile-android"],
          "default": "cli",
          "examples": ["cli", "server", "mobile-ios", "mobile-android"]
        }
      },
      "required": ["kind"]
    }
  }
}
```

### Field Semantics

| Client | Context | Notes |
|--------|---------|-------|
| `cli` | Command-line execution | `gert run`, `gert validate` |
| `server` | Gert server API | `POST /api/runs`, webhook triggers |
| `mobile-ios` | Native iOS app | Swift/SwiftUI app on iOS 17+ |
| `mobile-android` | Native Android app | Kotlin app targeting Android 11+ |

### Backward Compatibility

- **Omitempty:** If `client` is absent in a trace event, the runtime defaults to `"cli"`.
- **Existing runs:** Runs recorded before this field was added will have no `client` field; queries assume `"cli"`.
- **No validation breakage:** Adding the field does not break existing runbooks or trace consumers.

### Trace Event Producer Rules

1. **CLI:** Always sets `client: "cli"` when writing traces locally.
2. **Server:** Always sets `client: "server"` when a runbook is triggered via HTTP API.
3. **Mobile SDK:** Always sets `client: "mobile-ios"` or `client: "mobile-android"` depending on OS.
4. **Callers of Engine.Run():** MUST pass a Context with execution client info. The engine propagates it into the event.

---

## 4. Capability Token Format

### Definition

Capability tokens are **canonical identifiers** for device capabilities that tools may require. Tokens are declared in:

- Tool definitions: `requires-capabilities` array
- Kit manifests: `provides-capabilities` array
- Runtime queries: capability checks before execution

### Token Format

```
capability/<name>
```

Where `<name>` is **kebab-case** (lowercase letters, digits, hyphens only).

### Builtin Capabilities

The following capabilities are reserved:

| Token | Platform | Description |
|-------|----------|-------------|
| `capability/camera` | iOS, Android | Photo/video capture (front or rear) |
| `capability/location` | iOS, Android | GPS/GNSS position data |
| `capability/nfc` | iOS, Android | NFC tag read/write |
| `capability/biometrics` | iOS, Android | Fingerprint or face recognition |
| `capability/bluetooth` | iOS, Android | Bluetooth Low Energy (BLE) |
| `capability/notifications` | iOS, Android | Push notifications |
| `capability/contacts` | iOS, Android | Contact list access |
| `capability/calendar` | iOS, Android | Calendar read/write |
| `capability/microphone` | iOS, Android | Audio recording |
| `capability/sensors` | iOS, Android | Accelerometer, gyro, magnetometer |
| `capability/network` | iOS, Android | HTTP/HTTPS network access |

### Custom Capabilities

Domain kits may define custom capabilities (vendor-specific):

```
capability/pool-automation
capability/hvac-control
capability/medical-device-interface
```

Custom names must be kebab-case and must not conflict with builtin capability names.

### YAML Examples

In a tool definition:

```yaml
name: weather-lookup
version: "1.0.0"
description: Fetch weather data

requires-capabilities:
  - capability/location
  - capability/network

impl:
  ios:
    transport: native-sdk
    handler: GertSDK.Weather.lookup
```

In a kit manifest:

```json
{
  "name": "pool-maintenance",
  "provides-capabilities": [
    "capability/camera",
    "capability/location",
    "capability/network"
  ]
}
```

### Runtime Capability Checking

Before executing a step that invokes a tool, the runtime checks:

```
tool.requires-capabilities ⊆ runtime.available-capabilities
```

If any required capability is unavailable, the run fails with `ErrCapabilityNotAvailable`.

---

## 5. Platform Kit Catalog Entry (CDN Schema)

### Use Case

Mobile SDKs fetch a catalog of available kits from a CDN. The catalog is a static JSON file published at a known URL, listing all available kits, versions, and download URLs.

### Catalog Endpoint

```
https://cdn.example.com/kits/catalog.json
```

### Catalog JSON Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Platform Kit Catalog",
  "description": "Master catalog of available kits, published by the kit distribution CDN.",
  "type": "object",
  "required": ["kits"],
  "additionalProperties": false,
  "properties": {
    "kits": {
      "type": "array",
      "title": "Kit Entries",
      "description": "List of all available kits.",
      "minItems": 0,
      "items": {
        "$ref": "#/$defs/CatalogKitEntry"
      }
    }
  },
  "$defs": {
    "CatalogKitEntry": {
      "type": "object",
      "title": "Kit Catalog Entry",
      "description": "Metadata for a single kit available on the CDN.",
      "required": [
        "name",
        "latest-version",
        "versions",
        "description",
        "target",
        "download-url",
        "sha256"
      ],
      "additionalProperties": false,
      "properties": {
        "name": {
          "type": "string",
          "title": "Kit Name",
          "description": "Unique kit identifier (kebab-case).",
          "pattern": "^[a-z0-9]+(-[a-z0-9]+)*$",
          "examples": ["pool-maintenance", "gert-mobile-platform"]
        },
        "latest-version": {
          "type": "string",
          "title": "Latest Version",
          "description": "The newest version of this kit available.",
          "pattern": "^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*).*$",
          "examples": ["1.2.0"]
        },
        "versions": {
          "type": "array",
          "title": "All Available Versions",
          "description": "Complete list of available versions for this kit (in any order).",
          "minItems": 1,
          "uniqueItems": true,
          "items": {
            "type": "string",
            "pattern": "^(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*).*$"
          },
          "examples": [["0.1.0", "1.0.0", "1.1.0", "1.2.0"]]
        },
        "description": {
          "type": "string",
          "title": "Kit Description",
          "description": "Human-readable description.",
          "minLength": 10,
          "maxLength": 500,
          "examples": ["Pool maintenance runbooks"]
        },
        "target": {
          "type": "array",
          "title": "Target Platforms",
          "description": "Platforms supported by this kit.",
          "minItems": 1,
          "uniqueItems": true,
          "items": {
            "type": "string",
            "enum": ["ios", "android", "cli", "server", "wasm"]
          },
          "examples": [["ios", "android"]]
        },
        "download-url": {
          "type": "string",
          "title": "Kit Download URL",
          "description": "HTTPS URL to download the latest kit .zip file.",
          "format": "uri",
          "pattern": "^https://.*\\.kit\\.zip$",
          "examples": ["https://cdn.example.com/kits/pool-maintenance-1.2.0.kit.zip"]
        },
        "sha256": {
          "type": "string",
          "title": "Latest Version Checksum",
          "description": "SHA256 hex digest of the latest-version kit .zip.",
          "pattern": "^[a-f0-9]{64}$",
          "examples": ["abc123def456abc123def456abc123def456abc123def456abc123def456abc1"]
        }
      },
      "examples": [
        {
          "name": "gert-mobile-platform",
          "latest-version": "0.1.0",
          "versions": ["0.1.0"],
          "description": "Shared mobile capability tools",
          "target": ["ios", "android"],
          "download-url": "https://cdn.example.com/kits/gert-mobile-platform-0.1.0.kit.zip",
          "sha256": "abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
        },
        {
          "name": "pool-maintenance",
          "latest-version": "1.2.0",
          "versions": ["1.0.0", "1.1.0", "1.2.0"],
          "description": "Pool maintenance runbooks",
          "target": ["ios"],
          "download-url": "https://cdn.example.com/kits/pool-maintenance-1.2.0.kit.zip",
          "sha256": "def456abc123def456abc123def456abc123def456abc123def456abc123def456"
        }
      ]
    }
  },
  "examples": [
    {
      "kits": [
        {
          "name": "gert-mobile-platform",
          "latest-version": "0.1.0",
          "versions": ["0.1.0"],
          "description": "Shared mobile capability tools",
          "target": ["ios", "android"],
          "download-url": "https://cdn.example.com/kits/gert-mobile-platform-0.1.0.kit.zip",
          "sha256": "abc123def456abc123def456abc123def456abc123def456abc123def456abc1"
        }
      ]
    }
  ]
}
```

### Catalog Consumption Rules

1. **SDK fetches once per session** — Catalog is fetched at app startup or on demand, cached in device storage.
2. **Version resolution:** Client uses semver range matching to resolve `requires` in kit manifests.
3. **Checksum validation:** Client compares downloaded kit's SHA256 against catalog entry before extraction.
4. **Fallback behavior:** If catalog fetch fails, offline kits (pre-bundled in app) are used.

### Catalog Example (Pretty-Printed)

```json
{
  "kits": [
    {
      "name": "gert-mobile-platform",
      "latest-version": "0.1.0",
      "versions": ["0.1.0"],
      "description": "Shared mobile capability tools",
      "target": ["ios", "android"],
      "download-url": "https://cdn.example.com/kits/gert-mobile-platform-0.1.0.kit.zip",
      "sha256": "3a1f2e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2d3c4b5a6f7e8d9c0b1a2"
    },
    {
      "name": "pool-maintenance",
      "latest-version": "1.2.0",
      "versions": ["1.0.0", "1.1.0", "1.2.0"],
      "description": "Pool maintenance runbooks and tools",
      "target": ["ios"],
      "download-url": "https://cdn.example.com/kits/pool-maintenance-1.2.0.kit.zip",
      "sha256": "5f4e3d2c1b0a9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f0e9d8c7b6a5"
    }
  ]
}
```

---

## Validation Rules and Edge Cases

### General Rules

1. **All strings case-sensitive:** Platform names, capability tokens, kit names are case-sensitive.
2. **No null values:** All fields use JSON `null` for optional absence; YAML `null` is mapped to `null`.
3. **ID uniqueness:** Within a kit manifest, runbook IDs must be unique. Duplicate IDs are a validation error.
4. **Circular dependencies:** Kit A requires Kit B, Kit B requires Kit A — detected and rejected at compile time.
5. **Missing dependencies:** If a kit requires a dependency not found in catalog, compilation fails with `ErrKitNotFound`.

### Platform-Specific Validation

- **iOS:** Requires `impl.ios` for tools usable on iOS. HTTP handlers must use `https://` (not `http://`).
- **Android:** Requires `impl.android` for tools usable on Android. May use `http://` for localhost debugging.
- **CLI:** Requires `impl.cli` for tools usable in CLI. May use `local-process` transport.
- **Server:** Requires `impl.server` for server-side tools. Typically `grpc` or `http`.

### Backward Compatibility Notes

1. **Omitting `impl`:** Tools without `impl` are **server-only**. They cannot be used in mobile kits.
2. **Omitting `client` field:** Trace events without `client` default to `"cli"` — old traces remain readable.
3. **Capability tokens:** New capabilities may be added without breaking existing runbooks (additive).
4. **Version changes:** Manifest version is independent of kit semver — manifest schema version is always `2.0`.

---

## Example: Complete Mobile Kit

### Directory Structure

```
home-automation.kit/
├── manifest.json
├── runbooks/
│   ├── daily-check.yaml
│   └── emergency-shutdown.yaml
├── tools/
│   ├── camera-monitor.tool.yaml
│   ├── sensor-reader.tool.yaml
│   └── notification.tool.yaml
└── lib/
    └── shared-defs.yaml
```

### Manifest Content

```json
{
  "name": "home-automation",
  "version": "1.0.0",
  "target": ["ios", "android"],
  "description": "Home automation monitoring and control runbooks",
  "requires": [
    {
      "name": "gert-mobile-platform",
      "version": ">=0.1.0"
    }
  ],
  "runbooks": [
    {
      "id": "daily-check",
      "name": "Daily System Check",
      "version": "1.0.0"
    },
    {
      "id": "emergency-shutdown",
      "name": "Emergency System Shutdown",
      "version": "1.0.0"
    }
  ],
  "provides-capabilities": [
    "capability/camera",
    "capability/location",
    "capability/notifications"
  ],
  "compiled-at": "2026-04-26T14:00:00Z",
  "compiler-version": "2.0.0",
  "checksum": "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
}
```

### Tool Example

```yaml
$schema: "https://schemas.gert.dev/tool/v2.json"
apiVersion: tool/v2
name: sensor.read
version: "1.0.0"
description: Read home automation sensors

requires-capabilities:
  - capability/network

inputs:
  - name: sensor_id
    type: string
    required: true
    description: Sensor unique ID

outputs:
  - name: temperature
    type: number
  - name: humidity
    type: number

impl:
  ios:
    transport: native-sdk
    handler: GertSDK.Sensors.read
  
  android:
    transport: native-sdk
    handler: com.gert.home.sensors.SensorReaderTool
  
  server:
    transport: http
    handler: "https://api.example.com/sensors/read"
```

---

## Summary of Schema Additions

| Component | Type | Required? | Notes |
|-----------|------|-----------|-------|
| `tool.impl` | object | No | Per-platform handler descriptors |
| `manifest.json` | file | Yes (in kits) | Kit metadata at bundle root |
| `run/started.client` | string | No | Execution client origin |
| Capability tokens | string | — | Format: `capability/<name>` |
| Catalog entry | JSON | Yes (on CDN) | Kit metadata from distribution |

---

## Appendix: JSON Schema Validation with Draft 2020-12

All schemas in this document are valid JSON Schema Draft 2020-12 and may be validated using standard tools:

```bash
# Validate using ajv CLI
npm install -g ajv-cli
ajv validate -s tool-impl-schema.json -d camera.tool.yaml

# Validate using check-jsonschema
pip install check-jsonschema
check-jsonschema --schemafile manifest-schema.json manifest.json

# Validate using jsonschema (Python)
python -m jsonschema -i manifest.json manifest-schema.json
```

---

**Specification complete.**  
John, YAML/Schema Specialist  
2026-04-26
