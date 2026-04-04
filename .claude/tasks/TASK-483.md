---
id: TASK-483
plan: PLAN-063
status: done
wave: 1
priority: 2
---

# TASK-483: Implement nexus.showProviders command

Completed. Implemented `showProvidersCommand(client: NexusClient)` in `commands/index.ts` — fetches providers via `getProviders()`, shows a QuickPick with name/active status/model, displays detail on selection. Updated `extension.ts` to call the real implementation instead of the stub toast.
