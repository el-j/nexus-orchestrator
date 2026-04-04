# PLAN-061 — Go Backend — Pre-Launch Critical Fixes

**Status:** active
**Created:** 2026-04-04
**Author:** copilot

## Summary

Pre-launch audit of the Go backend identified 10 issues requiring fixes before a production release. Six are critical (data loss, race conditions, silent failures) and four are high-priority (type safety, dead code, missing test coverage). All must be resolved before the v1.0 release gate.

## Waves

### Wave 1 — Data Loss & Correctness

| Task     | Description                                              |
| -------- | -------------------------------------------------------- |
| TASK-460 | Fix httpapi_client — replace http.DefaultClient bypasses |
| TASK-461 | Fix main.go (Wails) — wire WithRuntimeConfigRepo         |

### Wave 2 — Race Conditions & Concurrency Safety

| Task     | Description                                                      |
| -------- | ---------------------------------------------------------------- |
| TASK-462 | Fix ActivityService.Stop() — add WaitGroup for goroutine cleanup |
| TASK-463 | Fix llm_openaicompat sync.Once permanent error caching           |
| TASK-464 | Fix SetAgentScanner/SetDiscoveredAgentRepo — missing mutex       |

### Wave 3 — Dead Code & Architecture Gaps

| Task     | Description                                    |
| -------- | ---------------------------------------------- |
| TASK-465 | Investigate and implement fs_writeback adapter |

### Wave 4 — High: Type Safety & Context Limits

| Task     | Description                                                       |
| -------- | ----------------------------------------------------------------- |
| TASK-466 | Implement ContextLimit() for Anthropic and OpenAI-compat adapters |
| TASK-467 | Replace map[string]interface{} with typed structs in LLM adapters |

### Wave 5 — High: Dead Code & Test Coverage

| Task     | Description                   |
| -------- | ----------------------------- |
| TASK-468 | Remove dead wailsbind package |
| TASK-469 | Add httpapi_client test suite |

## Validation

- `CGO_ENABLED=1 go test -race ./...` passes with zero race detector findings
- `go vet ./...` reports zero issues
- `go build ./cmd/nexus-daemon/... ./cmd/nexus-cli/...` succeeds
- Desktop Wails build (`go build -tags desktop .`) compiles clean
- Runtime config changes (API tokens, queue cap) survive daemon restart
- No regressions in existing test suite
