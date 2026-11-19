---
id: TASK-616
planId: PLAN-075
title: 'Use FTS-ranked focused brain context in task prompts, make GetProjectBrief byte-stable, and add Anthropic prompt caching'
role: coder
wave: 3
status: done
createdAt: 2026-10-19
---

# TASK-616 — Use FTS-ranked focused brain context in task prompts, make GetProjectBrief byte-stable, and add Anthropic prompt caching

1. **`internal/core/services/prompts.go` & `policy_engine.go`**:
   - When `o.brain != nil` and `task.Instruction != ""`, query `o.brain.GetFocusedContext` (FTS-ranked brain entries matching the task instruction within a token budget) and include matching sections in the task prompt (`PLAN-074` D1).
2. **`internal/core/services/brain_service.go` (`GetProjectBrief`)**:
   - Remove the live task queue counters (`%d queued, %d processing...`) and `Current Task` from the static project brief so the system prefix is byte-stable across tasks for the same project (`PLAN-074` D3).
3. **`internal/adapters/outbound/llm_anthropic`**:
   - Send `system` as a content block with `cache_control: {"type": "ephemeral"}` when non-empty so Anthropic caches the stable system/rules/briefing prefix across calls (`PLAN-074` D3).

Done when: the failing test came first, `make test` and `make nice` pass, docs and changelog updated.
