---
id: TASK-614
planId: PLAN-075
title: 'Enforce daily token budgets for explicit providerName and multi-turn loops, and step up the model ladder in agent_loop.go'
role: coder
wave: 3
status: done
createdAt: 2026-10-19
dependsOn: [TASK-613]
---

# TASK-614 — Enforce daily token budgets for explicit providerName and multi-turn loops, and step up the model ladder in agent_loop.go

1. **`internal/core/services/ladder.go`**:
   - Load project policy and call `o.enforceBudgets(chain, policy)` in `routeChain` _before_ returning early on `task.ProviderName != ""`, so daily token budgets apply even when a task names its provider.
2. **`internal/core/services/agent_loop.go` & `verification.go`**:
   - Re-check `o.enforceBudgets` after each turn records usage, and if the current provider is now on cooldown, switch to the next available provider (or hold the task in `WAITING` when all candidates are cooling down).
   - In `agent_loop.go`, when the `"verify"` or `"finish"` tool fails `a.verify(run)`, call `current = a.orch.stepUp(run, current)` so agentic tasks step up the model ladder after a failed verification just like non-agentic tasks.

Done when: the failing test came first, `make test` and `make nice` pass, docs and changelog updated.
