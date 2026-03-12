---
id: TASK-136
title: VS Code extension README and docs
role: planning
planId: PLAN-018
status: todo
dependencies: [TASK-134, TASK-135]
createdAt: 2026-03-11T16:10:00.000Z
---

## Context
Write the VS Code extension README with installation instructions, configuration guide, usage examples, and screenshots placeholders. Update the main project docs to reference the extension.

## Files to Read
- `vscode-extension/package.json`
- `docs/getting-started.md`
- `README.md`

## Implementation Steps
1. Create `vscode-extension/README.md` with:
   - Overview (what it does, architecture diagram)
   - Prerequisites (running nexus daemon)
   - Installation (from VSIX or marketplace)
   - Configuration (`nexus.daemonUrl` setting)
   - Usage: submit task, view queue, pick provider
   - Troubleshooting (daemon not running, wrong port)
2. Add a "VS Code Extension" section to the main `README.md`.
3. Add a VS Code integration page to `docs/` or update `getting-started.md`.

## Acceptance Criteria
- [ ] `vscode-extension/README.md` exists with complete documentation
- [ ] Main README references the extension
- [ ] Getting-started docs mention VS Code workflow

## Anti-patterns to Avoid
- Don't over-document — keep it concise and scannable
- Don't include screenshots that don't exist yet — use placeholders
