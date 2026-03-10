---
id: TASK-107
title: QA verify all PLAN-013 workflow changes
role: verify
planId: PLAN-013
status: todo
dependencies: [TASK-104, TASK-105, TASK-106]
createdAt: 2026-03-10T21:00:00.000Z
---

## Context
Verify all PLAN-013 changes are correct and consistent.

## Files to Read
- `.github/workflows/version.yml`
- `.github/workflows/release.yml`
- `GitVersion.yml`

## Verification Steps
1. Run `go vet ./...` — must exit 0
2. Run `CGO_ENABLED=1 go test -race -count=1 ./...` — all tests pass
3. Verify version.yml uses `gittools/actions/gitversion/setup@v4.3.3` and `execute@v4.3.3`
4. Verify version.yml has `versionSpec: '6.x'`
5. Verify GitVersion.yml has valid v6 config keys
6. Verify release.yml has `ZIG_VERSION="0.14.0"` in both locations
7. Verify no remaining `v3.1.11` references in any workflow
8. Verify no remaining `0.13.0` zig references

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All action versions are up to date
- [ ] Zero `v3.1.11` references remain
- [ ] Zero zig `0.13.0` references remain

## Anti-patterns to Avoid
- Do NOT modify files — this is a verify-only task
