# Batch Inference Engine — REST API Service

> **Coding Interview Project** — Build a backend service that reads a batch of AI prompts, processes them concurrently against a mock rate-limited inference endpoint, and aggregates the results.

---

## Table of Contents

- [Overview](#-overview)
- [Environment Rules](#-environment-rules)
- [Logistics & Tooling](#-logistics--tooling)
- [Project Objective](#-project-objective)
- [Functional Requirements](#-functional-requirements)
- [Engineering Requirements](#-engineering-requirements)
- [Extensions & Next Steps](#-extensions--next-steps)

---

## Overview

| | |
| --- | --- |
| **Goal** | A backend service that ingests a batch of AI prompts, processes them concurrently against a mock rate-limited inference endpoint, and aggregates the results. |
| **Deliverables** | Source code + architecture diagram, pushed to a personal GitHub repository. |

---

## Environment Rules

> **Mandatory** — Failure to follow these rules may result in permanent loss of work.

- **Stay in the Workspace** — All development must occur inside the `/workspaces` directory within your IDE.
- **No Host Saving** — Do **not** save code or assets to the host machine's **Desktop** or **Downloads** folder. These areas are not bridged to the container; data saved there will be permanently lost when the session ends.
- **Final Submission** — Push your code **and** architecture diagram to a personal GitHub repository **before the timer expires**.
- **Session Cleanup** — Sign out of all personal accounts (GitHub, Cursor, Browser) before handing back the workstation.

---

## Logistics & Tooling

### Supported Languages

`Go` · `Python` · `Java` · `TypeScript / Node.js`

> Standard editor extensions for YAML, Python, Go, and Java are highly recommended.

### IDE Environment

VS Code or Cursor, bridged directly to an **Ubuntu 24** Dockerized container.

### AI Assistance

GitHub Copilot, Claude Code, and Cursor are permitted. You are encouraged to use Chrome, documentation, and any AI tools to assist your development.

> You must sign in using your **personal accounts** and remain fully responsible for the review, architecture, and correctness of all generated output.

### Permissions

You have **passwordless `sudo`** access to install additional dependencies via `apt` or `brew`.

### Pre-Installed Utilities

| Tool | Purpose |
| --- | --- |
| `gh` | GitHub CLI |
| `doctl` | DigitalOcean CLI |
| `s3cmd` | S3-compatible object storage |
| `jq` | JSON processing |
| `yq` | YAML processing |
| `neovim` | Terminal editor |
| — | Core image processing libraries |

---

## Project Objective

Build a backend service that:

1. Reads a **batch of AI prompts**.
2. Processes them **concurrently** against a mock rate-limited inference endpoint.
3. **Aggregates** the results into a final output.

---

## Functional Requirements

### 1. Batch Ingestion
Accept a JSON array of prompts (e.g. **1,000 items**) via file read or API upload. Return an acknowledgment **immediately** while processing continues in the background.

### 2. Concurrent Processing
Distribute the processing of prompts across a **bounded pool of concurrent workers** rather than executing them strictly sequentially.

### 3. Rate Limit Handling
The mock external API will periodically return **`HTTP 429 Too Many Requests`**. Workers must implement **retry logic with back-off (sleep)** to safely recover without dropping prompts.

### 4. Result Aggregation
Compile all successful inferences into a **final JSON output structure**, or persist them to a local database, once the batch completes.

---

## Engineering Requirements

| Area | Requirement |
| --- | --- |
| **Architecture Diagram** | Map the worker pool, retry loop, and aggregation logic. |
| **Concurrency Discipline** | Cap the worker pool size so the application does not exhaust system memory or spawn unbounded threads. |
| **Testing** | Unit tests verifying that the 429 back-off logic triggers appropriately **without failing the batch**. |
| **CI/CD** | A basic GitHub Actions pipeline configuration. |
| **Documentation** | A README with setup instructions explaining how the concurrency model was designed. |

---

## Extensions & Next Steps

- **Job Status API** — Add an endpoint to query the real-time progress of an ongoing batch (e.g. `400/1000 completed`).
