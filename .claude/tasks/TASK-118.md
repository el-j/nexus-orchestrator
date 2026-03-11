---
id: TASK-118
title: Delete superseded workflows version.yml and release.yml
role: devops
planId: PLAN-016
status: todo
dependencies: []
createdAt: 2026-03-11T00:00:00.000Z
---

## Context

Both `version.yml` and `release.yml` were superseded by the unified `publish.yml` created in
PLAN-014/TASK-108. They currently conflict with `publish.yml`:

- `version.yml` fires on `push: branches: [main]`, calculates version via GitVersion, creates the
  semver tag. Because it runs concurrently with `publish.yml`, the tag is often created before
  `publish.yml`'s `check_tag` step runs — causing `exists=true` → all build jobs skipped.
- `release.yml` fires on `push: tags: v*.*.*`. When `publish.yml` eventually succeeds and pushes
  a tag, `release.yml` wakes up and creates a **second** GitHub Release for the same tag.

## Action

Delete both files:
- `.github/workflows/version.yml`
- `.github/workflows/release.yml`

## Definition of Done

- Neither file exists in `.github/workflows/`
- `.github/workflows/` contains only: `ci.yml`, `publish.yml`, `pages.yml`, `action-ci.yml`
