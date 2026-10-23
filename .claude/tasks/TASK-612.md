---
id: TASK-612
planId: PLAN-075
title: 'Persist AI session PID in SQLite for TerminateAISession and fix GetDiscoveredAgents cache'
role: coder
wave: 2
status: done
createdAt: 2026-10-19
---

# TASK-612 — Persist AI session PID in SQLite for TerminateAISession and fix GetDiscoveredAgents cache

1. **`internal/adapters/outbound/repo_sqlite/migrations.go` & `ai_session_repo.go`**:
   - Add migration adding `pid INTEGER NOT NULL DEFAULT 0` to `ai_sessions` and update `SaveAISession` / `scanAISession` to persist and read `domain.AISession.PID`.
2. **`internal/adapters/outbound/sys_scanner/agent_scanner.go`**:
   - Preserve `a.PID` and `a.ProcessName` when merging duplicate agent entries across discovery probes.
3. **`internal/core/services/session_service.go`**:
   - Protect `o.agentScanner`, `o.agentRepo`, and cached `o.lastScannedAgents` under `o.mu.Lock()` in `GetDiscoveredAgents` so repeated calls within the 30-second cache window return cached results even when `o.agentRepo == nil`.

Done when: the failing test came first, `make test` and `make nice` pass, docs and changelog updated.
