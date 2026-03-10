---
id: TASK-105
title: Update GitVersion.yml for v6.x compatibility
role: devops
planId: PLAN-013
status: todo
dependencies: []
createdAt: 2026-03-10T21:00:00.000Z
---

## Context
GitVersion.yml was written with v5-era schema. GitVersion 6.x changed some config keys. Need to verify and update for full compatibility. Key changes in GitVersion 6: `mode` now uses `ContinuousDelivery`/`ContinuousDeployment`/`Mainline` (same as before), but `assembly-versioning-scheme` is renamed to `assembly-versioning-format` in some contexts. Branch config structure also changed: `prevent-increment-of-merged-branch-version` → `prevent-increment-when-branch-merged`. Also `source-branches` may use different naming.

## Files to Read
- `GitVersion.yml`

## Implementation Steps
1. Read current GitVersion.yml
2. Verify all configuration keys are valid for GitVersion 6.x
3. Update any deprecated keys to their v6 equivalents
4. Ensure branch regex patterns are correct
5. Keep `mode: ContinuousDelivery`, `tag-prefix: v`, `next-version: 0.2.0`

## Acceptance Criteria
- [ ] GitVersion.yml uses only v6-compatible configuration keys
- [ ] YAML is valid
- [ ] Branch strategies remain functionally equivalent

## Anti-patterns to Avoid
- Do NOT change the versioning strategy — only update deprecated key names
- Do NOT remove the `next-version: 0.2.0` baseline
