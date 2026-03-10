---
id: TASK-110
title: Remove superseded workflows version.yml release.yml desktop.yml
role: devops
planId: PLAN-014
status: todo
dependencies: [TASK-108]
createdAt: 2026-03-10T22:00:00.000Z
---

## Context

After TASK-108 creates the unified `publish.yml`, the three old workflows are superseded and must be deleted to prevent double-triggering:
- `.github/workflows/version.yml`
- `.github/workflows/release.yml`
- `.github/workflows/desktop.yml`

## Definition of Done

- All three files deleted
- `.github/workflows/` only contains `publish.yml` + `pages.yml` (doc site)
