---
id: TASK-463
plan: PLAN-061
status: done
wave: 2
priority: 2
---

# TASK-463: Fix llm_openaicompat sync.Once permanent error caching

**Problem:** `internal/adapters/outbound/llm_openaicompat/adapter.go` `GetAvailableModels()` uses a `sync.Once` to cache the models list. If the first call fails (e.g., the provider is not yet running), the `Once` is consumed, the error is silently swallowed, and every subsequent call returns `nil, nil` with no models — permanently. This means a temporarily unavailable provider is treated as permanently empty for the lifetime of the process.

**Fix:** Replace `sync.Once` with a time-based TTL cache. On success, cache the result for a configurable TTL (e.g. 5 minutes). On failure, do not cache — allow the next call to retry. Use a `sync.Mutex` to protect the cached value and its timestamp.

**Files:**

- `internal/adapters/outbound/llm_openaicompat/adapter.go`
- `internal/adapters/outbound/llm_openaicompat/adapter_test.go` (add retry test)

## Checklist

- [x] Remove `availableOnce sync.Once` and its associated cached result field from the `Adapter` struct
- [x] Add fields: `modelsMu sync.Mutex`, `cachedModels []string`, `modelsCachedAt time.Time`, `modelsTTL time.Duration`
- [x] Initialize `modelsTTL` to 5 minutes in the adapter constructor (make it configurable via a functional option if other adapters already use that pattern)
- [x] In `GetAvailableModels()`: lock `modelsMu`, check if cache is valid (`time.Since(modelsCachedAt) < modelsTTL`), return cached value if so; otherwise unlock, fetch from API, re-lock, store on success only, return result
- [x] Ensure the function returns the actual error to the caller on failure (not `nil, nil`)
- [x] Add a test that simulates a failing first call followed by a successful second call, confirming that the second call succeeds and is not blocked by the failed first attempt
- [x] Run `go test -race ./internal/adapters/outbound/llm_openaicompat/...` clean

## Acceptance Criteria

- A failed `GetAvailableModels()` call does not permanently poison subsequent calls
- Success results are cached for ~5 minutes to avoid redundant API calls
- Error from the provider is returned to the caller (not silently discarded)
- Race detector passes
