# Batch Inference Engine — Design Document

> Design for a backend service that ingests batches of AI prompts, processes them concurrently against a mock rate-limited inference endpoint, and aggregates the results. This document covers **what** the system does and **why** it is shaped this way. Implementation-level detail (data structures, interfaces, packages) lives in [codeExplainability.md](codeExplainability.md).

---

## Table of Contents

- [Problem Statement](#problem-statement)
- [Assumptions & Scope](#assumptions--scope)
- [API Contract (Asynchronous)](#api-contract-asynchronous)
- [API Request Formats](#api-request-formats)
- [API Response Formats](#api-response-formats)
- [Status Codes](#status-codes)
- [Concurrency Model](#concurrency-model)
- [Retry Mechanism](#retry-mechanism)
- [State Management & Storage](#state-management--storage)
- [Trade-Offs & Future Recommendations](#trade-offs--future-recommendations)

---

## Problem Statement

Build a backend service that:

1. **Ingests** a batch of AI prompts (a JSON array) via API upload or file read.
2. **Acknowledges immediately** and processes the batch **in the background**.
3. **Processes prompts concurrently** through a bounded pool of workers against a mock inference endpoint.
4. **Handles rate limiting** — the inference endpoint periodically returns `429 Too Many Requests`; workers back off and retry without dropping prompts.
5. **Aggregates** all results once the batch completes, and exposes **progress** while it runs.

The core tension: the inference endpoint is **rate-limited**, so the service must extract throughput from concurrency while never exceeding the allowed number of in-flight requests, and must degrade gracefully (retry/back-off) when throttled.

---

## Assumptions & Scope

| Area | Assumption (in scope) |
| --- | --- |
| **Prompt content** | Prompts are strings of English Unicode characters, digits, and common English punctuation. |
| **Size limits** | Configurable constants. Defaults: ~**1,000 prompts** per batch, **1,000 characters** per prompt, **1,000 characters** per response. |
| **Inference server** | A **single** mock inference server, reached through one shared client. |
| **Rate limit** | The inference server allows **4 concurrent requests**, enforced globally across all batches. |
| **Retries** | Up to **3 retries** per prompt and **2 retries** per batch (see [Retry Mechanism](#retry-mechanism)). |
| **Identifiers** | Each prompt carries a client-supplied `id` that is unique within its batch; it keys the result mapping. |

Items deliberately left out are listed under [Trade-Offs & Future Recommendations](#trade-offs--future-recommendations).

---

## API Contract (Asynchronous)

The service never blocks the client while a batch runs. The lifecycle is:

```
1. Client submits a batch        → POST /batches        → 202 Accepted + job_id   (returns instantly)
2. Service processes in the background (bounded workers, retries, aggregation)
3. Client polls progress         → GET /batches/{job_id} → 200 OK (completed / total)
4. When finished, the same poll  → GET /batches/{job_id} → 200 OK (responses + errors)
```

- Submission validates the request envelope and registers the job **before** returning, so a follow-up poll can never miss the job.
- Progress (`completed / total`) is available throughout processing.
- The final aggregated result is retrieved from the **same** status endpoint once `status` becomes `done`.

---

## API Request Formats

### Submit a batch — `POST /batches`

A JSON array of prompts. Each prompt has a unique `id` and the `prompt` text.

```json
[
  { "id": 1, "prompt": "Summarize the theory of relativity." },
  { "id": 2, "prompt": "Translate 'good morning' to French." },
  { "id": 3, "prompt": "List three prime numbers." }
]
```

Clients send **only** `id` and `prompt`. Internal tracking metadata (such as retry counts) is attached by the service after ingestion and is never supplied by the client.

### Poll a batch — `GET /batches/{job_id}`

No body; the `job_id` returned at submission identifies the batch.

---

## API Response Formats

### Submission acknowledgement — `202 Accepted`

```json
{
  "job_id": "b1f2c3d4",
  "status": "queued",
  "total": 1000
}
```

### Progress (while processing) — `200 OK`

```json
{
  "job_id": "b1f2c3d4",
  "status": "processing",
  "completed": 400,
  "total": 1000
}
```

### Final result (when done) — `200 OK`

```json
{
  "job_id": "b1f2c3d4",
  "status": "done",
  "responses": [
    { "id": 1, "response": "Relativity describes..." },
    { "id": 3, "response": "2, 3, 5" }
  ],
  "errors": [
    { "id": 2, "status_code": 503, "error_message": "inference unavailable after retries" }
  ]
}
```

- **`responses`** — one entry per successful prompt: `id` + `response`.
- **`errors`** — one entry per failed prompt: `id` + `status_code` + `error_message`. Empty when all prompts succeed.
- Both lists are **sorted by `id`** for deterministic output, since prompts complete out of order under concurrency.

### Validation failure — `400 Bad Request`

Returned at submission when the envelope is invalid; lists every issue at once.

```json
{
  "status": "rejected",
  "errors": [
    { "reason": "batch exceeds max prompt count", "detail": "1200 > 1000" },
    { "reason": "duplicate id", "detail": "id 7 appears twice" }
  ]
}
```

---

## Status Codes

### HTTP codes

| Code | When |
| --- | --- |
| `202 Accepted` | Batch envelope valid; job created and queued for background processing. |
| `200 OK` | Status/progress poll, or final result retrieval. |
| `400 Bad Request` | Validation failed (too many prompts, oversized prompt, missing/duplicate `id`). |
| `404 Not Found` | Unknown `job_id`. |

### Per-prompt outcomes (within a completed batch)

| Outcome | Handling |
| --- | --- |
| **Success** | Added to `responses`. |
| **Response too long** (> limit) | Recorded as a **per-prompt error** — it does not fail the whole batch. |
| **Retries exhausted** (`429` / `5xx`) | Recorded as a per-prompt error; the batch still completes with partial results. |

### Inference-server status handling & logging

- **Rate limited (`429`)** → **logged as an error** by the service for observability, then backed off and retried. It is *not* masked or silently swallowed.
- **`5xx` after retries exhausted** → logged and recorded as a per-prompt error (with a mapped code such as `502` / `503`); raw inference internals are not leaked to the client.

---

## Concurrency Model

The service uses a **fan-out / fan-in** pipeline:

```
ingest → validate ─┐
                   ▼
            [ jobs queue ]
                   ▼
        ┌──── N workers (N ≤ 4) ────┐   ← shared global rate limiter
        ▼          ▼          ▼
            [ results queue ]
                   ▼
        single collector → aggregated result → SQLite (on completion)
```

- **Bounded worker pool.** A fixed pool of N workers (N ≤ the rate limit) drains the jobs queue. The worker count is constant regardless of batch size — this is what bounds concurrency and memory, so the service never spawns unbounded goroutines.
- **Global rate limiter.** The 4-request limit belongs to the inference server, so it is enforced **once, globally**, across all batches. Two concurrent batches share the same 4 slots and cannot collectively exceed them.
- **Overlapping stages.** Validation and processing run as separate stages connected by queues; a validated prompt is streamed straight to processing to minimize latency.
- **Single collector (fan-in).** One collector aggregates results and updates a progress counter, so polling clients never contend with the workers.

---

## Retry Mechanism

| Level | Retry when… | Do **not** retry when… | Scope of retry |
| --- | --- | --- | --- |
| **Per prompt** (max 3) | Rate limited (`429`) or transient `5xx` from the inference server. | Validation / invalid-prompt errors. | The single prompt. |
| **Per batch** (max 2) | **Total** inference-server failure (unreachable / wholesale `5xx`). | Partial failures — handled per prompt. | **Only the failed subset** — succeeded prompts are never re-run. |

- **Back-off:** exponential back-off with **jitter**, capped; honor a `Retry-After` header if present. Jitter prevents all workers from waking simultaneously after a shared `429` (thundering herd).
- **Timeouts:** every inference call runs under a deadline; cancellation propagates on shutdown.
- **No dropped prompts:** a prompt either ends in `responses` (success) or `errors` (retries exhausted) — never silently lost.

---

## State Management & Storage

State is split into two planes with different lifetimes and durability needs:

| Plane | Holds | Backing store | Durability |
| --- | --- | --- | --- |
| **Control plane** | Job registry: status, progress counter, retry counts, outstanding prompts | **In-memory**, fronted by a storage abstraction | Volatile |
| **Data plane** | Final aggregated `responses` + `errors` of a completed batch | **SQLite** (single-file local DB), written **once on completion** | Durable |

- The job is **registered synchronously** at submission, before the `202` is returned, so the first status poll always finds it.
- The control-plane store sits **behind an abstraction**, so the in-memory implementation can be swapped for an external store later without touching request handling or workers.
- **TTL eviction** removes completed jobs from memory (after persistence and read) to bound memory growth.

> ### Durability trade-off (accepted)
> Only **completed** results are durable (persisted to SQLite). **In-flight jobs are lost on crash or restart** — there is no per-prompt progress persistence and interrupted batches are not resumed. This is an accepted trade-off for a single-node service: per-prompt durability would add significant write amplification for little benefit here.

---

## Trade-Offs & Future Recommendations

| Area | Current (in scope) | Future recommendation |
| --- | --- | --- |
| **Durability** | Completed results durable; in-flight jobs lost on crash. | Persist progress to an external store (Redis / Postgres) behind the same abstraction for crash-resumable jobs. |
| **Scale** | Single node; job state pinned to one process. | Shared external state + multiple service instances for horizontal scale. |
| **Inference backend** | One mock inference server. | Pool of inference servers with per-server usage tracking and rerouting for maximum utilization. |
| **Limits** | Fixed config constants (counts, sizes, retries). | Per-client / per-request configurable limits. |
| **Prompt content** | English text only, single type. | Multiple prompt types and Unicode families; attachment support. |
| **Result delivery** | Client polls for status/results. | Optional webhook / callback on completion to avoid polling. |

---

*Implementation details — data structures, the storage interface, the worker/collector internals, and package layout — are documented separately in [codeExplainability.md](codeExplainability.md).*
