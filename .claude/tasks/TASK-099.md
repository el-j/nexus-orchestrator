---
id: TASK-099
title: Add GitVersion.yml configuration
role: devops
planId: PLAN-012
status: todo
dependencies: []
createdAt: 2026-03-10T20:00:00.000Z
---

## Context
GitVersion calculates semantic versions from git history and conventional commits. A `GitVersion.yml` config file tells it how to derive versions from branches.

## Files to Read
- `.github/workflows/release.yml` (current tag-based triggers)
- `.github/workflows/desktop.yml` (current tag-based triggers)

## Implementation Steps
1. Create `GitVersion.yml` at repo root with:
   - `mode: ContinuousDelivery`
   - `assembly-versioning-scheme: MajorMinorPatch`
   - Branch config: `main` (release increment: Patch), `feature` branches (pre-release tag: alpha), `hotfix` (increment: Patch)
   - `tag-prefix: v` to work with existing `v*.*.*` tag pattern
   - `next-version: 0.2.0` as baseline (since 0.1.0 was the initial)

## Acceptance Criteria
- [ ] `GitVersion.yml` exists at repo root
- [ ] Config uses `mode: ContinuousDelivery`
- [ ] `tag-prefix` set to `v`
- [ ] Branch strategies defined for main, feature, hotfix

## Anti-patterns to Avoid
- Do not use `mode: Mainline` — ContinuousDelivery is better for projects with explicit releases
- Do not set initial version to 1.0.0 — project is pre-1.0
