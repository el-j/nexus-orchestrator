# PLAN-028: Health Throttle + Workspace Orchestration Awareness

**Status:** Completed  
**Completed:** 2026-03-12T00:00:00Z

## Tasks

| ID | Title | Role | Completed |
|----|-------|------|-----------|
| TASK-199 | DiscoveryService: TTL health cache + circuit breaker backoff | backend | 2026-03-12T00:00:00Z |
| TASK-200 | Reduce frontend provider polling interval | frontend | 2026-03-12T00:00:00Z |
| TASK-201 | VS Code: WorkspaceScanner + WorkspaceOrchView + extension wiring | vscode | 2026-03-12T00:00:00Z |
| TASK-202 | QA: verify health throttle, tree view, tests | qa | 2026-03-12T00:00:00Z |

## Summary

Addressed two systemic performance concerns and added workspace-level orchestration visibility to the VS Code extension:

- **TTL health cache in DiscoveryService**: provider health checks were firing on every request and hitting LM Studio / Ollama synchronously. Added a 30-second TTL cache so repeated `/api/providers` calls return the cached state, and introduced a circuit-breaker backoff that doubles the check interval (up to 5 min) when a provider is repeatedly unreachable.
- **Frontend polling reduction**: `useProviders` composable was polling every 5 s; aligned it to the 30 s backend cache TTL, cutting pointless HTTP traffic by 6×.
- **WorkspaceScanner + WorkspaceOrchView**: new VS Code tree-view data provider that lists currently open workspaces alongside their active/queued task counts fetched from the daemon. Wired into the extension's ActivityBar view container so developers can see orchestration status without switching to the GUI dashboard.
- All changes verified with race-free `go test -race` across the full test suite.
