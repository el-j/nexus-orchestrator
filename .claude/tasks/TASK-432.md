---
id: TASK-432
plan: PLAN-057
status: done
role: testing
wave: 7
---

# TASK-432: Go tests — ProviderInfo enrichment, agent dedup, DurationMs

Write Go tests:

1. **ProviderInfo enrichment** (`discovery_test.go` or new): `ListProviders()` returns `ContextLimit`, `TimeoutSec`, `LastChecked`, `ConsecutiveFails` fields populated correctly
2. **Agent dedup** (`agent_scanner_test.go`): `probeClaudeSubAgents` with multiple session files in same project dir returns single DiscoveredAgent (not N duplicates)
3. **DurationMs** (`task_test.go` or handler test): Task with CreatedAt+CompletedAt returns correct DurationMs in JSON

**Files:** test files in `internal/core/services/`, `internal/adapters/outbound/sys_scanner/`, `internal/core/domain/`
**Depends on:** TASK-407, TASK-408, TASK-409
