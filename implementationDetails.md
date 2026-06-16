# Code Explainability

> Implementation-level companion to the [Design Document](initialUnderstanding.md). This is where data structures, the storage interface, concurrency internals, and package layout live. It is expanded as the code is written; the design doc stays conceptual.

---

## Table of Contents

- [Package Layout](#package-layout)
- [Data Structures](#data-structures)
- [Storage Interface](#storage-interface)
- [Concurrency Internals](#concurrency-internals)
- [Inference Client & Test Seam](#inference-client--test-seam)
- [Configuration](#configuration)
- [Component ↔ Design Map](#component--design-map)

---

## Package Layout

> Provisional — refine as packages are created.

```
/cmd/server          → entrypoint: wires config, store, inference client, HTTP server
/internal/api        → HTTP handlers (submit, status), request validation, response shaping
/internal/batch      → orchestration: worker pool, collector, retry/back-off
/internal/inference  → inference client interface + mock/real implementations
/internal/store      → JobStore interface + in-memory and SQLite implementations
/internal/config     → limits and tunables (counts, sizes, retries, worker count)
```

---

## Data Structures

> Field types and names are indicative; finalize against the code.

### Job (control plane)

Held in the registry for the life of the job. Includes the **input prompts** so the failed subset can be retried.

```
Job {
  ID        string      // unique job identifier
  Status    string      // queued | processing | done | failed
  Total     int         // total prompts in the batch
  Completed atomic int   // bumped as prompts finish (read by status polls)
  Retry     int         // batch-level retry count, default 0
  Prompts   []Prompt    // outstanding (not-yet-succeeded) prompts
  Results   []Result    // successful inferences
  Errors    []PromptError
}
```

### Prompt (internal)

```
Prompt {
  ID    int     // unique within the batch
  Text  string  // the prompt text
  Retry int     // internal per-prompt retry count, default 0 — NOT from the client
}
```

> The client payload is only `{ id, prompt }`. `Retry` is attached server-side after ingestion.

### Result & error (mirror the external contract)

```
Result      { ID int; Response string }
PromptError { ID int; StatusCode int; Message string }
```

---

## Storage Interface

The control-plane store is fronted by an interface so the in-memory implementation can be swapped (SQLite already used for the data plane; Redis/Postgres later) without touching handlers or workers.

```
JobStore {
  CreateJob(job) error          // synchronous on the submit path, before 202
  GetJob(id) (Job, error)       // backs status polls
  RecordResult(id, result)      // append a response/error, bump progress
  MarkDone(id) error            // finalize; triggers SQLite persistence
  Evict(id) error               // TTL cleanup of completed jobs
}
```

- **In-memory implementation:** `map[string]*Job` guarded by a mutex (or `sync.Map`); `Completed` via `atomic`.
- **SQLite implementation (data plane):** the aggregated result is written **once** on `MarkDone`.

---

## Concurrency Internals

Maps the [Concurrency Model](initialUnderstanding.md#concurrency-model) onto Go primitives.

- **Jobs queue / results queue:** buffered channels carrying prompts and completed results.
- **Worker pool:** a fixed number of worker goroutines (≤ rate limit) range over the jobs channel.
- **Global rate limiter:** a single shared semaphore (buffered channel of size 4) acquired around every inference call, shared across all batches.
- **Collector:** one goroutine is the sole writer of `Results`/`Errors` and increments the atomic `Completed` counter; workers only send over the results channel — no shared-slice locking on the hot path.
- **Lifecycle:** a `sync.WaitGroup` tracks in-flight prompts; when drained, the results channel is closed and the job is marked done.
- **Cancellation:** a `context.Context` per request enforces per-call timeouts and propagates graceful shutdown.

---

## Inference Client & Test Seam

The inference client is an **interface** so tests can inject a fake — this is the seam for the required `429` back-off test.

```
InferenceClient {
  Infer(ctx, prompt) (response, error)
}
```

- **Real/mock client:** calls the mock rate-limited endpoint.
- **Fake client (tests):** returns `429` a configurable N times, then `200`, to assert that:
  - back-off triggers and the prompt eventually succeeds,
  - a prompt whose retries are exhausted lands in `Errors` while the **batch still completes** (no batch failure),
  - concurrent in-flight calls never exceed the rate limit.

---

## Configuration

Centralize tunables (no magic numbers) in one config surface:

| Setting | Default | Notes |
| --- | --- | --- |
| Max prompts per batch | 1000 | Validation limit |
| Max prompt characters | 1000 | Validation limit |
| Max response characters | 1000 | Per-prompt error if exceeded |
| Worker pool size | 4 | ≤ rate limit |
| Rate limit (concurrent inference calls) | 4 | Global semaphore size |
| Per-prompt retries | 3 | |
| Per-batch retries | 2 | Failed subset only |
| Completed-job TTL | TBD | In-memory eviction |

---

## Component ↔ Design Map

| Design concept | Implementation home |
| --- | --- |
| Async submit / status endpoints | `/internal/api` handlers |
| Bounded worker pool & collector | `/internal/batch` |
| Global rate limiter | shared semaphore in `/internal/batch` |
| Retry & back-off | `/internal/batch` retry loop |
| Control-plane registry | `JobStore` (in-memory) in `/internal/store` |
| Data-plane durable results | `JobStore` (SQLite) in `/internal/store` |
| Inference calls + 429 test seam | `InferenceClient` in `/internal/inference` |
