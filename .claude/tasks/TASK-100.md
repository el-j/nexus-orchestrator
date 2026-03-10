---
id: TASK-100
title: Create semantic-release CI workflow
role: devops
planId: PLAN-012
status: todo
dependencies: [TASK-099]
createdAt: 2026-03-10T20:00:00.000Z
---

## Context
Need a GitHub Actions workflow that runs on push to main, uses GitVersion to calculate the next semantic version, and creates a git tag. The tag then triggers the existing release.yml and desktop.yml workflows.

## Files to Read
- `GitVersion.yml` (created in TASK-099)
- `.github/workflows/release.yml` (triggered by `v*.*.*` tags)
- `.github/workflows/desktop.yml` (triggered by `v*.*.*` tags)

## Implementation Steps
1. Create `.github/workflows/version.yml`:
   - Trigger: `push` to `main` branch (not on tag pushes to avoid loops)
   - Permissions: `contents: write`
   - Steps:
     a. Checkout with `fetch-depth: 0` (GitVersion needs full history)
     b. Install GitVersion via `gittools/actions/gitversion/setup@v3.1.11` (version 6.x)
     c. Execute GitVersion via `gittools/actions/gitversion/execute@v3.1.11`
     d. Check if tag `v{semver}` already exists — skip if so
     e. Create and push annotated tag `v{semver}` using `github.token`
2. Add concurrency group to prevent parallel version runs
3. Add timeout-minutes: 10

## Acceptance Criteria
- [ ] `.github/workflows/version.yml` exists
- [ ] Triggers on push to main only
- [ ] Uses `gittools/actions/gitversion/setup` and `execute`
- [ ] Creates tag with `v` prefix matching existing workflow triggers
- [ ] Has guard against duplicate tag creation
- [ ] Has concurrency group

## Anti-patterns to Avoid
- Do NOT trigger release.yml directly — let the tag push event trigger it naturally
- Do NOT use `actions/create-release` — only create the tag, not the release
- Do NOT use `[skip ci]` in tag push — the whole point is to trigger downstream workflows
