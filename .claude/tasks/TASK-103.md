---
id: TASK-103
title: QA verify all PLAN-012 changes
role: verify
planId: PLAN-012
status: todo
dependencies: [TASK-101, TASK-102]
createdAt: 2026-03-10T20:00:00.000Z
---

## Context
Final verification that all PLAN-012 changes are correct: zig fix works in YAML, GitVersion config is valid, version.yml workflow is correct, LICENSE exists, badges render.

## Files to Read
- `.github/workflows/release.yml`
- `.github/workflows/desktop.yml`
- `.github/workflows/version.yml`
- `GitVersion.yml`
- `LICENSE`
- `README.md`

## Verification Steps
1. Run `go vet ./...` — must exit 0
2. Run `CGO_ENABLED=1 go test -race -count=1 ./...` — all tests pass
3. Validate all YAML files: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/version.yml'))"` (repeat for release.yml, desktop.yml)
4. Verify LICENSE contains MIT text
5. Verify GitVersion.yml has valid structure
6. Verify version.yml has correct trigger (push to main, not tags)
7. Verify release.yml zig install uses tarball (no apt-get for zig)
8. Verify README badges use `el-j/nexusOrchestrator`
9. Grep for any remaining `apt-get install -y zig` — should be zero

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All workflow YAML files are valid
- [ ] No `apt-get install -y zig` remains
- [ ] LICENSE, GitVersion.yml, version.yml all exist
- [ ] README badges are correct

## Anti-patterns to Avoid
- Do NOT skip test execution
- Do NOT modify files — this is a verify-only task
