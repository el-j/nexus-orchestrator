---
id: TASK-623
planId: PLAN-075
title: 'Complete VS Code extension (cooldowns, quota, usage, WAITING tasks, lmBridge usage) and GitHub Action (NO_PROVIDER, waitUntil, api_token)'
role: coder
wave: 5
status: todo
createdAt: 2026-10-19
---

# TASK-623 — Complete VS Code extension (cooldowns, quota, usage, WAITING tasks, lmBridge usage) and GitHub Action (NO_PROVIDER, waitUntil, api_token)

1. **`vscode-extension/src/*`**:
   - Add `waitUntil?: string` and `retryCount?: number` to `Task` in `nexusTypes.ts`, include `WAITING` tasks in active counts (`statusBar.ts`) and active tree sections (`taskQueueProvider.ts`, `workspaceOrchView.ts`), show `waitUntil` in `TaskItem` tooltips, allow cancelling `WAITING` tasks in `extension.ts` (`nexus.cancelTask`), and log `task.waiting` in `liveOutput.ts`.
   - Add `getUsage(since)` to `NexusClient` and show 24h token usage, provider cooldowns (`rateLimitedUntil`, `cooldownReason`), and plan quota (`utilization`) in `NexusStatusBar` tooltip and `showProvidersCommand`.
   - Include estimated `usage` (`prompt_tokens`, `completion_tokens`, `total_tokens`) in `lmBridge.ts` responses so `vscode.lm` calls populate the daemon's usage ledger.
2. **`github-action/src/*`**:
   - Handle `NO_PROVIDER` explicitly in `index.ts` terminal status switch, log `waitUntil` when a task is `WAITING` in `submit.ts`, and support optional `api_token` (`Authorization: Bearer`) in `types.ts`, `submit.ts`, and `action.yml`.

Done when: the failing test came first, 100% coverage in `vscode-extension` and `github-action` passes, docs and changelog updated.
