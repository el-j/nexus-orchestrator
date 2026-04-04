# TASK-334: Network probe activity reader

**Plan:** PLAN-050 · Wave 2
**Status:** DONE
**Agent:** Senior Developer

## Description

Create `internal/adapters/outbound/activity_network/reader.go` implementing `ports.ActivityReader`.

Probes local AI API endpoints to detect active model state:

- LM Studio: GET http://127.0.0.1:1234/v1/models → list loaded models
- Ollama: GET http://127.0.0.1:11434/api/ps → running models with context size
- Antigravity: GET http://127.0.0.1:4315/v1/models → loaded models

Generate AIActivity with type "generation" when models are actively loaded, summary = "Model <name> loaded", metadata includes model details.

Use configurable base URLs from NEXUS\_\*\_URL env vars.
Short timeout (2s) per probe to avoid blocking.

## Acceptance

- Implements ActivityReader
- Probes all 3 endpoints with timeout
- Uses env var URLs
- Handles unreachable gracefully (skip, don't error)

## Completed

Implemented `activity_network` reader probing LM Studio, Ollama, and Antigravity with 2s timeout and configurable env-var base URLs.
