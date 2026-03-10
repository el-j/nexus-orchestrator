---
id: TASK-101
title: Update release.yml + desktop.yml for GitVersion + zig fix
role: devops
planId: PLAN-012
status: todo
dependencies: [TASK-100]
createdAt: 2026-03-10T20:00:00.000Z
---

## Context
The zig install via `apt-get` fails on Ubuntu runners (not in repos). Need to install zig from official tarball. Also ensure workflows work correctly with GitVersion-generated tags.

## Files to Read
- `.github/workflows/release.yml`
- `.github/workflows/desktop.yml`
- `.github/workflows/version.yml` (created in TASK-100)

## Implementation Steps
1. In `release.yml` — zig install already fixed (direct download from ziglang.org). Verify both test job and build job use the tarball approach.
2. Verify `VERSION="${GITHUB_REF_NAME:-dev}"` still works correctly with GitVersion-generated `v*.*.*` tags (it should — GITHUB_REF_NAME will be the tag name).
3. Ensure `workflow_dispatch` trigger is preserved for manual builds.
4. No changes expected in desktop.yml (it doesn't use zig).

## Acceptance Criteria
- [ ] Zig installed from official tarball, not apt-get
- [ ] `VERSION` extraction compatible with GitVersion tags
- [ ] `workflow_dispatch` preserved for manual builds
- [ ] YAML is valid

## Anti-patterns to Avoid
- Do NOT remove the `v*.*.*` tag trigger — it's how version.yml triggers builds
- Do NOT hardcode version numbers in workflows
