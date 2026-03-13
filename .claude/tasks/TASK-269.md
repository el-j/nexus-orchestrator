# TASK-269 — Extend DiscoveredProvider domain type

**Plan**: PLAN-041  
**Status**: done

## What
Add to `domain.DiscoveredProvider` struct in `internal/core/domain/provider.go`:
- `ActiveModels []string` — models currently loaded in memory (Ollama /api/ps)
- `Generating   bool`    — true if an active generation is in progress
- `PID          int`     — process ID when detected via process pattern (optional, 0 if unknown)
All fields are json-tagged and optional (zero values = unknown/not applicable).
