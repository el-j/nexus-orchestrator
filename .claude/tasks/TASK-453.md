---
id: TASK-453
plan: PLAN-059
status: todo
wave: 5
priority: 5
---

# TASK-453: Replace critical log.Printf anti-patterns

**Problem:** 38+ instances in Go backend where recoverable errors are logged with `log.Printf` instead of being returned. Spans orchestrator.go, execution_engine.go, activity_service.go, httpapi handlers.

**Fix:** Audit the most critical log.Printf calls in core services and convert to error returns where the caller can handle them. Keep log.Printf for truly best-effort operations (background cleanup, telemetry). Focus on:

- orchestrator.go: errors during task processing should propagate
- execution_engine.go: LLM call failures should be returned
- activity_service.go: reader errors that affect data completeness

**Files:** `internal/core/services/orchestrator*.go`, `internal/core/services/execution_engine.go`, `internal/core/services/activity_service.go`
