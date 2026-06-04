# Cross-Agent Note: Platform Infrastructure Implications

**To:** John (Azure Platform Engineer)  
**From:** Scribe  
**Session:** 2026-06-03 Declaration-Before-Process Brainstorm  
**Date:** 2026-06-04

## Infrastructure Review Points

Declaration-before-process scenario family introduces the following infrastructure considerations:

### 1. **Cosmos DB New Container: `user_inputs`**
- **Required for:** A6 MVP IUserInputGate implementation
- **Partition key:** `/runId`
- **Capacity:** Serverless (existing baseline)
- **Action:** Add to Cosmos DB schema before A6 sprint

### 2. **Blob Storage — Long Retention Tier**
- **Scenario context:** Declaration/consent workflows often have 7–30 year legal holds (healthcare, finance)
- **Trace immutability:** JSONL traces must survive in append blob for audit
- **Decision needed:** Archive blobs to cold/long-term storage after compliance retention window?
- **Note:** Current A6 cost baseline assumes hot tier; long-retention scenarios may prefer tiered strategy

### 3. **Declaration Registry Pattern (Future)**
- **Brainstorm finding:** Some scenarios require external "declaration registry" (government, regulatory)
- **Infrastructure question:** If GERT becomes a registration service, would need:
  - Cosmos DB container for declaration metadata (indexed by tenant + declaration_id)
  - Webhook delivery SLA: at-least-once + idempotency (Service Bus topic subscription in v2.0?)
- **Status:** Proposal only; no immediate action required

### 4. **Service Bus — Session Capacity**
- **Current decision:** Sessions required for at-most-once on declaration runs
- **Note:** Each active `run_id` = one session lock (5 min timeout)
- **Scaling question:** If declaration runs span hours (approval gates), will accumulated session count cause backpressure?
- **Recommendation:** Monitor session lock utilization post-MVP

---

**Team decision pending:** Infrastructure choices above require ratification before architecture review.
