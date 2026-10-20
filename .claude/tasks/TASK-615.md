---
id: TASK-615
planId: PLAN-075
title: 'Keep self-healing correction turns out of persistent sessionRepo and bound verification output'
role: coder
wave: 3
status: done
createdAt: 2026-10-19
dependsOn: [TASK-614]
---

# TASK-615 — Keep self-healing correction turns out of persistent sessionRepo and bound verification output

1. **`internal/core/services/verification.go`**:
   - Do not persist intermediate failed self-healing correction prompts and broken outputs into the project-wide `sessionRepo` (only persist the final successful turn or keep correction turns ephemeral to the task attempt), preventing failed test dumps from inflating subsequent tasks' session history.
   - Bound `verifyOut` in `correctionPrompt` (e.g., via `truncateObservation` / failing-line extraction) so multi-megabyte test logs do not exhaust the stepped-up frontier model's token budget (`PLAN-074` D4).

Done when: the failing test came first, `make test` and `make nice` pass, docs and changelog updated.
