---
id: TASK-562
planId: PLAN-072
title: 'Smart Provider Fallback Chain & Role-Driven Router'
role: backend
status: done
createdAt: 2026-09-25T10:05:00Z
completedAt: 2026-09-25T10:32:00Z
---

# TASK-562 — Smart Provider Fallback Chain & Role-Driven Router

## Context

When an agent task is executed, hard-binding to a single provider fails if that provider is rate-limited (HTTP 429), down, or missing. A resilient orchestrator needs:

1. Role-driven routing hints (`task.Role`: "architect", "developer", "tester", "linter").
2. Automated fallback chains (`Frontier Cloud` $\to$ `Fallback Cloud` $\to$ `Local Ollama`).

## Completed Work

1. **Domain**:
   - Added `Role string` (`json:"role,omitempty"`) to `domain.Task`.
2. **Services (`execution_engine.go` & `discovery.go`)**:
   - Added `GetAllClients() []ports.LLMClient` on `DiscoveryService`.
   - Implemented `resolveProviderWithFallback(task domain.Task) ([]ports.LLMClient, error)` returning an ordered chain of available providers with role-driven prioritization and model compatibility filtering.
   - Implemented `executeWithFallbackChain` with pre-flight token headroom checks per provider, transient/429 error failover, and `[failover: ...]` log preservation.
   - Added `appendTaskLog` helper to prevent log overwrites during multi-stage execution and self-healing turns.
3. **Tests**:
   - Added unit tests `TestProviderFallback_PrimaryFails_SecondarySucceeds`, `TestRoleDrivenRouter_ArchitectRoutesToFrontierFirst`, and `TestRoleDrivenRouter_LinterRoutesToLocalFirst` in `internal/core/services/execution_engine_test.go`.
   - Verified 100% pass across all tests with `-race -count=1`.
