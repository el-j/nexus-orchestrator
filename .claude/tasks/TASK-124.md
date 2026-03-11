---
id: TASK-124
title: Update CHANGELOG.md and orchestrator.json for PLAN-017
role: planning
planId: PLAN-017
status: todo
dependencies: [TASK-122, TASK-123]
createdAt: 2026-03-11T01:00:00.000Z
---

## Changes Required

### CHANGELOG.md
Add to `[Unreleased]` section:
- Fixed: All download links on docs page now resolve correctly (prefix `nexus-orchestrator-`)
- Fixed: macOS desktop download links now use `.zip` format (matching pipeline output)
- Added: macOS Gatekeeper/quarantine workaround instructions on downloads and getting-started pages

### orchestrator.json
- Add PLAN-017 entry with status `completed`
- Add TASK-122, TASK-123, TASK-124 entries with status `done`
- Update counters: nextTaskId → 125, nextPlanId → 18
- Update notes

## Definition of Done
- CHANGELOG.md [Unreleased] section has the fix entries
- orchestrator.json valid JSON with PLAN-017 + tasks
