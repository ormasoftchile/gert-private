# Aspire Local Development Reality Check for A6

**Author:** John (Azure Platform Engineer)  
**Date:** 2026-06-04T12:28:39.293-04:00  
**Status:** Reference draft

---

## Executive Summary

Short version: **no, A6 cannot be worked 100% locally with Aspire in a clean “everything emulated” way**.

Aspire is good today at orchestrating the app and a subset of Azure dependencies, but it is **not** a full local Azure. For this stack, Aspire is strong for **Service Bus**, **Blob Storage/Azurite**, and **Cosmos DB emulator** (with caveats). It is weak or nonexistent for **Entra External ID/B2C**, **Event Grid**, and **Static Web Apps**, and its **SignalR** story only helps if you are using **Azure SignalR in serverless mode**—which is **not** the same thing as A6’s current in-process hub model.

If the expectation is “Aspire should let us run the whole Azure topology locally as if Azure existed on my laptop,” that expectation is the problem. **Use Aspire as the conductor, not as a fantasy Azure region.**

My recommendation for A6 is a **hybrid local strategy**:

- Run the **API + BackgroundService worker** locally under Aspire.
- Use Aspire emulators for **Service Bus**, **Blob Storage**, and optionally **Cosmos DB**.
- Use **real Azure dev resources** for **Entra External ID**, **Event Grid**, and optionally **SignalR Service** if we ever move beyond in-process SignalR.
- For the current A6 MVP, keep **SignalR local as plain ASP.NET Core SignalR**, not Azure SignalR.
- Treat full-cloud validation as a separate smoke path, not something Aspire replaces.

---

## 1. What .NET Aspire CAN do for this stack today

## 1.1 Services with solid Aspire support

### Azure Service Bus

**Works well in Aspire.**

- **Hosting package:** `Aspire.Hosting.Azure.ServiceBus` — **stable** (`13.4.2` on NuGet)
- **Client package:** `Aspire.Azure.Messaging.ServiceBus` — **stable** (`13.4.2`)
- **Local option:** `RunAsEmulator()`
- **Emulator backing:** Azure Service Bus Emulator container + companion SQL container

What you get:

- AppHost modeling of namespace, queues, topics, subscriptions
- Local emulator orchestration from Aspire
- Automatic connection injection into the app
- Health checks and telemetry in C# client apps
- Ability to model queue-first local development without hand-wiring connection strings

For A6 specifically, this is the **best Aspire fit** in the stack.

### Azure Cosmos DB

**Good Aspire support, but with caveats.**

- **Hosting package:** `Aspire.Hosting.Azure.CosmosDB` — **stable** (`13.4.2`)
- **Client package:** `Aspire.Microsoft.Azure.Cosmos` — **stable** (`13.4.2`)
- **Local options:**
  - `RunAsEmulator()`
  - `RunAsPreviewEmulator()` for the newer Linux-based emulator

What you get:

- AppHost modeling of account, databases, and containers
- Containerized local emulator support
- Injected connection info
- C# DI/health/telemetry integration for `CosmosClient`
- A clean way to define the `runs`, `approvals`, and `user_inputs` containers in local orchestration

For A6 this is useful, but you must understand that **local Cosmos is not production Cosmos**—especially because production is **serverless** and the emulator is **not serverless**.

### Azure Blob Storage

**Very good Aspire support.**

- **Hosting package:** `Aspire.Hosting.Azure.Storage` — **stable** (`13.4.2`)
- **Client package:** `Aspire.Azure.Storage.Blobs` — **stable** (`13.4.2`)
- **Local option:** `RunAsEmulator()`
- **Emulator backing:** Azurite

What you get:

- AppHost modeling of storage account + blob resources/containers
- Local Azurite orchestration
- Injected connection strings / endpoints
- C# DI/health/telemetry support for `BlobServiceClient`

For A6 this is a strong fit for:

- JSONL trace output
- artifact storage
- append-blob style audit traces

Important: **append blob is a Blob Storage concern, not a Cosmos concern**. If someone says “Cosmos emulator doesn’t support append blobs,” they are mixing services.

### Azure Functions (useful for future A8, not required for current A6)

**Supported in Aspire.**

- **Hosting package:** `Aspire.Hosting.Azure.Functions` — **stable** (`13.4.2`)
- Local experience: starts local Functions host and wires triggers/bindings

This matters mostly if we later build A8 / Durable Functions or any webhook/event adapters as Functions. It does **not** solve A6’s current App Service worker topology, but it does mean Aspire is a better fit for our future A8 path than people often assume.

### Azure App Service (publish target only)

**Supported, but not for the inner loop.**

- **Hosting package:** `Aspire.Hosting.Azure.AppService` — NuGet latest is stable `13.4.2`, but the docs still mark the integration **Preview**

What it actually does:

- Models an App Service environment as a deployment target
- Helps when publishing
- Generates Bicep
- Supports slots and deployment customization

What it does **not** do:

- It does not “run App Service locally”
- It does not reproduce App Service hosting semantics on your laptop
- It does not make managed identity, auth, networking, or platform behavior magically local

For A6, this package is useful for **deployment plumbing**, not for solving local development.

### Azure SignalR Service

**Supported in Aspire, but only partly useful for us.**

- **Hosting package:** `Aspire.Hosting.Azure.SignalR` — **stable** (`13.4.2`)
- **No Aspire-specific client package found on NuGet**; consuming apps use the regular SignalR packages:
  - `Microsoft.Azure.SignalR` for **Default mode** hub servers
  - `Microsoft.Azure.SignalR.Management` for **Serverless mode**
- **Local option:** `RunAsEmulator()`

This is where people get burned.

Aspire’s SignalR integration is real, but the **emulator only works for Azure SignalR Service in Serverless mode**. It does **not** emulate the **Default mode** hub-server setup that typical ASP.NET Core SignalR apps use.

For current A6, where the MVP direction is **in-process hub in App Service**, Aspire’s Azure SignalR emulator is **not the right local abstraction**. The right local abstraction is simply:

- run the ASP.NET Core app locally
- map the hub locally
- use plain local SignalR

So yes, Aspire has Azure SignalR support. No, that does **not** mean it cleanly solves A6 local development.

---

## 2. What .NET Aspire CANNOT do or does poorly today

## 2.1 No official Aspire integration packages found

As of 2026-06-04T12:28:39.293-04:00, I did **not** find public official Aspire packages for:

- **Azure Event Grid**
- **Azure Static Web Apps**
- **Microsoft Entra External ID / Azure AD B2C / Entra External ID**

Specifically, there is no public NuGet package found for:

- `Aspire.Hosting.Azure.EventGrid`
- `Aspire.Azure.EventGrid`
- `Aspire.Hosting.Azure.StaticWebApps`
- `Aspire.Hosting.Azure.Entra`

And these do not appear in the public Aspire integration gallery the way Service Bus / Cosmos / Storage / SignalR / Functions / App Service do.

That means for this stack, the following remain **manual or external** from an Aspire standpoint:

- Event Grid topics/subscriptions
- Static Web Apps hosting/runtime behavior
- Entra External ID / B2C tenant, policies, redirect URIs, user flows, federation

## 2.2 Cloud-only or effectively cloud-only pieces

### Entra External ID / B2C

There is **no local emulator**. Full stop.

If your app depends on:

- real OIDC redirects
- user flows / custom policies
- external tenant setup
- redirect URI validation
- token issuance semantics

then you need a **real Azure tenant**.

Any attempt to make Aspire “fully support auth locally” usually collapses into one of these:

- fake local auth
- bypass auth in dev
- use a dev External ID/B2C tenant

That is not Aspire’s fault. Identity is just cloud-only here.

### Event Grid

There is **no real local Event Grid emulator** story for the topology we care about.

You can locally simulate webhook/event payloads by POSTing HTTP payloads to handlers, but that is **not** the same as local Event Grid infrastructure. If you need subscription validation, delivery behavior, retries, dead-letter semantics, or actual Azure Event Grid wiring, use **real Azure**.

### Static Web Apps

Static Web Apps local development is handled by the **SWA CLI**, not by Aspire.

The SWA CLI is actually useful because it gives you:

- local static site hosting/proxying
- local routes/config enforcement
- mock auth/authz server
- local API proxying

But that is a **parallel toolchain**, not a first-class Aspire Azure resource.

So if you are expecting Aspire to be the one local host for the frontend + SWA semantics + auth emulation, that expectation is wrong. Use SWA CLI for SWA behavior.

## 2.3 Emulator gaps that matter to A6

### Service Bus emulator limitations

The emulator is useful, but it is still a dev emulator.

Known limitations called out by Microsoft include:

- no partitioned entities
- no AMQP over WebSockets
- no JMS
- no VNet / Entra ID / portal / activity logs
- no large messages
- no data persistence after container restart unless you rebuild state yourself
- sequential testing intent, not production-like scale testing

For A6, the key issue is not “can it enqueue and dequeue?” — it can. The real issue is whether you trust it for:

- session lock behavior under concurrency
- recovery behavior after worker crashes
- operational behavior under load
- exact parity with Standard-tier Service Bus semantics

I would trust it for **functional dev**. I would **not** trust it as proof that session-based at-most-once behavior is production-safe.

### Cosmos DB emulator limitations

This is the other big one.

Important gaps:

- emulator supports **provisioned throughput**, not **serverless**
- classic emulator has architecture pain; on Mac/Apple Silicon this is exactly where people suffer
- newer Linux-based emulator is still a **preview path** in Aspire (`RunAsPreviewEmulator()`)
- Linux-based emulator is **gateway mode only** with a **subset of features**
- emulator only flags consistency levels; it does not reproduce cloud scalability behavior
- emulator can lag cloud features

For A6, this means local dev is fine for CRUD and basic container semantics, but not for validating:

- serverless-specific behavior/cost assumptions
- exact consistency/performance behavior
- production RU patterns
- “this will behave exactly like Cosmos serverless in Azure” assumptions

### SignalR limitations for our topology

This is likely the most misunderstood piece.

- Azure SignalR emulator only supports **Serverless** mode
- It does **not** help for the common **hub-server / Default mode** local dev path
- The official guidance for Default mode is effectively: just use **self-hosted SignalR locally**

For A6 MVP, that means:

- if we are using **in-process hub in the ASP.NET Core app**, local dev should stay local and simple
- if we later move to **Azure SignalR Service**, then Aspire can model it, but the emulator still only helps in serverless mode

### Blob/Azurite limitations

Azurite is the least painful of the emulators here, but it is still not Azure.

Key limits:

- Blob/Queue/Table only
- no Files / no Data Lake Gen2
- different local endpoints and URL forms
- no production-scale behavior guarantees
- error behavior is aligned, not identical

For A6 JSONL trace writes, Azurite is usually good enough. I would not make Blob the reason to avoid Aspire.

---

## 3. The specific failure modes the user has likely hit

Given the comment _“I haven’t been able to fully support with Aspire any project”_, the probable failure pattern is very familiar.

## 3.1 Expecting Aspire to emulate the whole Azure architecture

This is the number-one mistake.

Aspire is an **orchestrator and wiring layer**. It is not a local Azure control plane.

If you add `AddAzure*` resources and do nothing else, Aspire will often try to **provision real Azure resources locally** using your Azure credentials unless you explicitly choose:

- `RunAsEmulator()`
- `RunAsPreviewEmulator()`
- `RunAsExisting()`
- `AsExisting()`

So a lot of failed Aspire attempts are actually this:

- developer thought they were building a “pure local” stack
- Aspire started talking to Azure
- credentials / region / permissions / quotas / first-run delays became the problem

That is not a bug. That is how Aspire Azure integrations are designed.

## 3.2 SignalR mismatch: hub server routing vs Azure SignalR serverless assumptions

Very likely pain point.

If someone tried to put A6’s SignalR push path behind Aspire’s Azure SignalR integration and expected a smooth local loop, they probably hit one of these:

- trying to use Azure SignalR emulator with a **Default mode** hub app
- negotiate endpoint assumptions not matching serverless mode
- local hub routes working before Azure SignalR is introduced, then breaking after introduction
- confusion between `MapHub(...)` local behavior and Azure SignalR-backed negotiation behavior

For A6 MVP, the local answer is simpler: **do not put Azure SignalR in the loop unless the architecture actually requires it**.

## 3.3 B2C / External ID redirect loops and localhost auth pain

Another common wall.

Symptoms usually look like:

- login succeeds then bounces back to login
- callback URL mismatch
- localhost port mismatch between frontend and backend
- SameSite/cookie confusion
- HTTPS mismatch between app registrations and dev servers
- portal running on one local port today and another tomorrow

This is not something Aspire fixes. If your dev ports move around, your identity config becomes brittle.

## 3.4 Cosmos emulator pain on Mac, TLS/certs, or serverless mismatch

Especially relevant on macOS.

Common failure modes:

- classic emulator image doesn’t behave well on Apple Silicon
- certificate trust problems
- HTTPS vs HTTP mismatch in SDK configuration
- code works locally with emulator but diverges from serverless production account behavior
- indexing / consistency / RU assumptions do not match cloud

If the user is on a Mac and tried Cosmos with Aspire before, I would strongly suspect this is one of the reasons they concluded Aspire “doesn’t fully support” the project.

## 3.5 Service Bus sessions “work locally” but give false confidence

Another trap.

Yes, Aspire can stand up the Service Bus emulator.

No, that does not mean you have truly validated:

- session affinity under real contention
- duplicate avoidance under process crashes
- lock renewal behavior under long-running handlers
- dead-letter and recovery patterns at scale

If the project depends on Service Bus sessions for governance integrity, the emulator is necessary but insufficient.

## 3.6 Static Web Apps assumptions leaking into local frontend development

If the team is using Azure Static Web Apps in Azure but running only the SPA dev server locally, they can hit:

- route mismatches
- auth endpoint mismatches
- missing SWA config behavior
- “works in Azure SWA, fails locally” differences

That is why SWA CLI exists. Aspire does not replace it.

---

## 4. A realistic local dev strategy for A6

## Recommendation: hybrid, not ideological

For this architecture, the sane answer is:

> **Use Aspire where it genuinely helps, and stop forcing it onto services that are cloud-only or badly emulated.**

## 4.1 What should run locally under Aspire

### Run locally in Aspire

- **API + BackgroundService worker** (the actual A6 application)
- **Azure Service Bus** via emulator
- **Blob Storage** via Azurite
- **Cosmos DB** via emulator **only if the team accepts emulator caveats**
- **Portal frontend dev server** if we want Aspire to launch it, but not as “Static Web Apps”

### Keep local but not Azure-specific

- **SignalR user input push** as **plain ASP.NET Core SignalR** inside the API app

This matches the current A6 MVP note: **in-process Hub for MVP**. That is the right local story.

## 4.2 What should stay in Azure even during local development

Use a **dev/sandbox subscription** for:

- **Entra External ID / B2C**
- **Event Grid**
- optionally **real Cosmos DB serverless** if emulator friction becomes too high
- optionally **real Azure SignalR Service** only if/when we choose to depend on it beyond local self-hosted SignalR

This is not surrender. It is disciplined engineering.

## 4.3 Suggested local-dev modes

### Mode A — fast inner loop

Use for day-to-day coding.

- API + worker local
- Service Bus emulator
- Azurite
- local SignalR hub
- fake/dev auth toggle
- optional Cosmos emulator

Best for:

- runtime logic
- queue consumption logic
- user input gate behavior
- trace writing
- approval polling logic

### Mode B — auth/integration loop

Use when touching identity, webhook ingress/egress, or cloud routing.

- API + worker local
- Service Bus emulator or real Service Bus
- Azurite or real Blob
- **real Entra External ID/B2C**
- **real Event Grid**
- local or real Cosmos depending the change

Best for:

- login/logout
- token validation
- customer portal access flows
- webhook dispatch/validation
- anything involving cloud callbacks

### Mode C — pre-merge smoke

Use for “prove the architecture still works.”

- everything against real Azure dev resources
- either deployed or local app pointed at dev Azure resources

Best for:

- confidence before merge
- emulator parity gaps
- session/concurrency sanity checks
- auth + webhook + storage + queue end-to-end validation

## 4.4 Auth strategy in dev

Be pragmatic.

Use one of these, explicitly:

1. **Real External ID/B2C tenant** for auth work
2. **Dev auth bypass / fake principal** for non-auth work
3. **SWA CLI mock auth** if the frontend needs a fake identity profile locally

Do **not** force every local developer flow through real External ID all the time if they are not actively working on identity.

## 4.5 My opinionated recommendation for A6

If this were my platform call, I would standardize on this:

- **Aspire:** API/worker + Service Bus emulator + Azurite
- **Cosmos:** start with emulator if x64/Linux path is clean; on Apple Silicon prefer either `RunAsPreviewEmulator()` or just use a cheap real dev Cosmos account
- **SignalR:** local in-process hub only
- **External ID/B2C:** real dev tenant only
- **Event Grid:** real dev topic/subscription only
- **Static Web Apps behavior:** SWA CLI, not Aspire pretending to be SWA

That gets you a real inner loop without wasting days chasing emulator fiction.

---

## 5. What a good Aspire AppHost for this project would look like

## 5.1 Packages I would actually use

### AppHost packages

- `Aspire.Hosting.Azure.ServiceBus`
- `Aspire.Hosting.Azure.CosmosDB`
- `Aspire.Hosting.Azure.Storage`
- `Aspire.Hosting.Azure.SignalR` **only if we intentionally model Azure SignalR later**
- `Aspire.Hosting.Azure.Functions` **only for future A8 / adapter work**
- `Aspire.Hosting.Azure.AppService` **for publish target, not for local inner loop**

### App/project packages

- `Aspire.Azure.Messaging.ServiceBus`
- `Aspire.Microsoft.Azure.Cosmos`
- `Aspire.Azure.Storage.Blobs`
- `Microsoft.Azure.SignalR` or `Microsoft.Azure.SignalR.Management` only if/when Azure SignalR is truly part of the app path

## 5.2 What I would declare as Aspire resources

### Declare directly in AppHost

- Service Bus namespace
  - `runs` queue
  - `webhooks` queue
  - `dlq` behavior via queue config / Service Bus semantics
- Storage account
  - blob service
  - trace/artifact containers
- Cosmos DB account
  - database
  - `runs` container
  - `approvals` container
  - `user_inputs` container
- API/worker project
- optional frontend dev server project

### Do NOT try to model as local Aspire-backed resources

- Entra External ID / B2C
- Static Web Apps
- Event Grid

These should be fed in as:

- parameters
- secrets
- configuration
- environment variables
- or external references handled outside the AppHost

## 5.3 What I would mark as external / existing

If using real Azure in dev:

- Cosmos DB can be `AsExisting(...)`
- Service Bus can be `AsExisting(...)`
- SignalR can be `AsExisting(...)` if ever needed

For unsupported services:

- Entra config via `WithEnvironment(...)`
- Event Grid topic endpoint / keys / webhook URLs via config
- SWA-specific local behavior outside Aspire via SWA CLI

## 5.4 Service discovery shape

Service discovery should be boring.

- use `WithReference(...)` from API/worker to Service Bus queue resources
- use `WithReference(...)` to Cosmos database/container resources
- use `WithReference(...)` to blob resource/container
- inject unsupported cloud-only settings as environment variables/secrets

The important thing is that the **application code should not care** whether Service Bus / Blob / Cosmos are emulated or real. That switch belongs in AppHost and environment config.

## 5.5 Illustrative AppHost sketch

```csharp
using Aspire.Hosting.Azure;

var builder = DistributedApplication.CreateBuilder(args);

// Messaging
var bus = builder.AddAzureServiceBus("messaging")
    .RunAsEmulator();

var runsQueue = bus.AddServiceBusQueue("runs");
var webhooksQueue = bus.AddServiceBusQueue("webhooks");

// For A6 we care about session-based dispatch.
// Configure session requirement and queue policies in the queue resource config.

// Storage
var storage = builder.AddAzureStorage("storage")
    .RunAsEmulator();

var blobs = storage.AddBlobs("blobs");
var traces = blobs.AddBlobContainer("traces");
var artifacts = blobs.AddBlobContainer("artifacts");

// Cosmos
#pragma warning disable ASPIRECOSMOSDB001
var cosmos = builder.AddAzureCosmosDB("cosmos")
    .RunAsPreviewEmulator();
#pragma warning restore ASPIRECOSMOSDB001

var appDb = cosmos.AddCosmosDatabase("gert");
var runs = appDb.AddContainer("runs", "/runId");
var approvals = appDb.AddContainer("approvals", "/runId");
var userInputs = appDb.AddContainer("user_inputs", "/runId");

// API + worker
var api = builder.AddProject<Projects.GertWebApi>("api")
    .WithReference(runsQueue)
    .WithReference(webhooksQueue)
    .WithReference(traces)
    .WithReference(artifacts)
    .WithReference(runs)
    .WithReference(approvals)
    .WithReference(userInputs)
    .WithEnvironment("Auth__Mode", "DevBypass")
    .WithEnvironment("Auth__Authority", builder.Configuration["Auth:Authority"])
    .WithEnvironment("Auth__ClientId", builder.Configuration["Auth:ClientId"])
    .WithEnvironment("EventGrid__TopicEndpoint", builder.Configuration["EventGrid:TopicEndpoint"]);

// Optional frontend dev server: run locally, but use SWA CLI separately when SWA behavior matters.

builder.Build().Run();
```

## 5.6 What this AppHost intentionally does NOT do

It does **not** pretend to model:

- External ID tenant provisioning
- B2C user flows / custom policies
- Static Web Apps runtime/auth/routing semantics
- Event Grid subscriptions and delivery pipeline

That omission is deliberate. An honest AppHost is better than a magical one that lies.

---

## 6. Gotchas and advice

## 6.1 The biggest gotcha: Aspire defaults to real Azure unless you say otherwise

This catches experienced people because the marketing story sounds “local-first,” but the Azure story is often actually “local provisioning into your subscription.”

If you do not explicitly choose emulator/existing-resource modes, Aspire can and will try to provision real Azure resources.

That means:

- credentials matter
- permissions matter
- region matters
- quota matters
- cost matters
- first-run latency matters

If your prior Aspire attempts felt heavy or flaky, this is probably one reason.

## 6.2 App Service integration does not solve local App Service development

For A6, App Service is a **publish target**. Your real local development model is just:

- run the ASP.NET Core app locally
- run the `BackgroundService` locally inside that app
- wire dependencies through Aspire

Do not overcomplicate this by trying to force App Service platform behavior into the inner loop.

## 6.3 SignalR: keep local dev simple

If the MVP is **in-process hub**, keep it that way locally.

Do **not** introduce Azure SignalR into local development unless there is a proven requirement. It adds complexity without helping the inner loop for current A6.

## 6.4 Cosmos on Mac: pick your pain consciously

If the team is on Apple Silicon, do not casually assume `RunAsEmulator()` will be happy.

Pick one:

- `RunAsPreviewEmulator()` and accept preview caveats
- real dev Cosmos serverless account and accept cloud dependency

I would rather pay a few dollars for a real dev Cosmos account than lose a week to emulator nonsense.

## 6.5 Use emulators for logic, not for confidence theater

Good uses of emulators:

- schema/contract validation
- local CRUD and queue flows
- basic integration wiring
- developer productivity

Bad uses:

- proving production semantics
- proving concurrency behavior
- proving scale behavior
- proving identity behavior

This is the line teams often cross without admitting it.

## 6.6 Split “developer convenience” from “architecture truth”

For this project, the architecture truth is:

- some services are fundamentally Azure-only
- some services are locally emulatable
- some services are better kept local with a non-Azure equivalent in dev

That means a healthy developer setup is **multi-mode by design**. That is not a failure. That is maturity.

---

## Final Answer

If the question is:

> Can all this be worked locally with Aspire while developing?

My answer is:

**Not fully, and trying to force it will waste time.**

For A6, Aspire is worth using for:

- local orchestration
- Service Bus
- Blob Storage
- maybe Cosmos DB
- wiring the API/worker together cleanly

It is **not** the answer for:

- Entra External ID / B2C
- Event Grid
- Static Web Apps semantics
- Azure SignalR in the way A6 currently uses SignalR

The winning move is a **hybrid dev topology**:

- local app + local emulators where they are good
- real Azure where Azure is the product
- explicit dev bypasses where auth would otherwise slow everybody down

That is the first Aspire setup for this project I would actually trust.
