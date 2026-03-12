---
id: TASK-135
title: VS Code status bar and provider picker
role: devops
planId: PLAN-018
status: todo
dependencies: [TASK-133]
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Add a status bar item showing the daemon connection status and active task count, plus a provider/model quick-pick command.

## Files to Read
- `vscode-extension/src/extension.ts`
- `vscode-extension/src/nexusClient.ts`

## Implementation Steps
1. Create a status bar item (left side) showing: "$(zap) Nexus: 3 tasks" or "$(warning) Nexus: offline".
2. Poll daemon health endpoint (`GET /health` or `/api/providers`) every 10 seconds.
3. Click on the status bar item → show quick-pick with options: "Submit Task", "View Queue", "Select Provider", "Open Dashboard".
4. Implement `nexus.selectProvider` command — fetch providers, show quick-pick of active providers and their models, save selection as default for subsequent task submissions.
5. Store selected provider/model in workspace settings.

## Acceptance Criteria
- [ ] Status bar shows daemon connection state and task count
- [ ] Clicking status bar opens action quick-pick
- [ ] Provider/model selection persists in workspace settings
- [ ] Status bar updates reflect real-time state

## Anti-patterns to Avoid
- NEVER poll health more than every 10 seconds from the status bar
- NEVER store API keys in VS Code settings
