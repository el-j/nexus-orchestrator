---
id: TASK-106
title: Update zig from 0.13.0 to 0.14.0 in release.yml
role: devops
planId: PLAN-013
status: todo
dependencies: []
createdAt: 2026-03-10T21:00:00.000Z
---

## Context
Zig 0.13.0 (June 2024) is used for cross-compilation. Zig 0.14.0 (March 2025) is the latest stable release that maintains the same tarball naming convention (`zig-linux-x86_64-{VERSION}.tar.xz`). Note: 0.14.1+ changed naming to `zig-{arch}-{os}-{VERSION}.tar.xz`, so 0.14.0 is the safest upgrade that requires minimal URL changes.

## Files to Read
- `.github/workflows/release.yml`

## Implementation Steps
1. In release.yml, change `ZIG_VERSION="0.13.0"` to `ZIG_VERSION="0.14.0"` in BOTH locations (test job and build job)
2. Update the download URL pattern: note that 0.14.0 uses `zig-linux-x86_64-0.14.0.tar.xz` (same pattern as 0.13.0)
3. Update the extracted directory name from `zig-linux-x86_64-0.13.0` to `zig-linux-x86_64-0.14.0`
4. Verify `zig cc` and `zig c++` cross-compilation targets still work with 0.14.0

## Acceptance Criteria
- [ ] release.yml uses `ZIG_VERSION="0.14.0"` in both test and build jobs
- [ ] Download URL is correct for zig 0.14.0
- [ ] YAML is valid

## Anti-patterns to Avoid
- Do NOT upgrade to 0.14.1+ — the tarball naming convention changed
- Do NOT change the zig cross-compilation targets
