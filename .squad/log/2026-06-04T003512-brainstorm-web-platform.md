# Session Log: GERT Web Platform Architecture Brainstorm

**Date:** 2026-06-04T00:35:12Z  
**Type:** Architecture & Integration Brainstorm  
**Agents:** Barbara, John, Don, David, Leslie (in progress)

## Scope

Foundational architecture decisions for GERT Web Execution Platform on Azure. Five parallel agents investigating orthogonal problem domains: runtime topology, service stack, governance constraints, webhook delivery, and white-label frontend options.

## Explored Topics

### 1. Execution Topology (Barbara)
Five candidate patterns evaluated for runtime hosting. Thin Relay (Functions + Container Apps Jobs) selected as MVP pattern — per-run isolation, governance preservation, acceptable cold start latency. Sidecar topology identified as endgame for enterprise tenants.

### 2. Azure Service Constraints (John)
Hard constraints identified across five service dimensions: compute (Container Apps chosen), queue (Service Bus Standard), auth (Entra + Managed Identity), frontend (Static Web Apps Standard), webhooks (Event Grid primary + Service Bus fallback). Cost/performance trade-offs documented per layer.

### 3. Governance Red Lines (Don)
Five non-negotiable constraints formulated that eliminate per-step Durable Functions, direct tool invocation, external trace writers, concurrent multi-worker patterns, and worker patterns incompatible with approval gate pause/resume semantics. Queue-triggered worker recommended as baseline pattern satisfying all constraints.

### 4. Webhook & Event Delivery (David)
Baseline v1.0 established: per-execution webhook with retry + exponential backoff (5 attempts, 24h window), summary events only, HMAC-SHA256 signing, DLQ handling. Upgrade path: tenant-level registration (v1.1), Service Bus topics (v2.0), Event Grid integration (v2.0).

### 5. White-Label Frontend (Leslie)
In progress — researching custom domain strategy, multi-tenant isolation patterns, and Static Web Apps deployment constraints.

## Decisions Merged to decisions.md

- Architecture Options: 5 patterns, recommendation, open questions
- Azure Service Stack: Hard constraints, recommended services, open questions
- Governance Red Lines: 5 red lines, pattern evaluation matrix
- Webhook Baseline: V1–V2 roadmap, retry model, security model

## Key Recommendations

1. **Start with Thin Relay topology** (Barbara)
2. **Use Container Apps + Service Bus Standard** (John)
3. **Enforce governance red lines** (Don) — especially Red Line 1, 2, 3 for multi-tenant safety
4. **Implement per-execution webhook + DLQ in sprint 2** (David)

## Status

Four of five agents completed. Leslie (white-label frontend) still running. Session log complete; orchestration records created per agent; decisions.md merged and inbox cleared.
