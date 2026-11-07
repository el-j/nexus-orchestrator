---
id: TASK-636
planId: PLAN-077
title: 'Go coverage to 100 %: core services'
role: testing
status: done
createdAt: 2026-10-31
---

# TASK-636 — Go coverage to 100 %: core services

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

policy_admin 6, agent_loop 4, execution_engine 4, orchestrator_config 4, planfiles 4, policy 4,
policy_engine 4, verification 4, and the rest of services listed by the merged profile.

Done: `internal/core/services` is at 100 % statement coverage on its own tests (`-race`, 4 repeats). New tests: `coverage_gaps_test.go`, `engine_gaps_test.go`, `planfiles_gaps_test.go`, `waiting_internal_test.go` and additions to `policy_test.go`, `discovery_cache_test.go`. Test seams added: `HoldReleased`, `RecordOutcome`, `HoldUntilReset`, `NoteProviderError`, `SetWaitGrace`, `SetFindProcess`, `SetMaxPlanWalkEntries`. Dead or duplicate code removed instead of tested: the nil-discovery guard and the empty-result check of `resolveProviderWithFallback`, the top-of-turn cancel check of the agent loop (the next model call fails at once and is checked), the `os.FindProcess`-less provider name fallback in `writeAndVerifyTaskOutput`, the token-generation error (`crypto/rand.Read` cannot fail), the three failable temp-file steps of the policy writer (now one `writePolicyFile` shared by `SetProjectPolicy` and `EnsureProjectPolicy`), the Stat error of `resolveIngestPath` (folded into the regular-file check). The plan-file search limits moved to `planfiles_limits.go` to keep `planfiles.go` under 500 lines. The cancel-failure test now walks all six stages of `CancelTask`.
