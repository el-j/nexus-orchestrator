---
id: TASK-119
title: Fix docs/downloads.md Desktop section href links
role: devops
planId: PLAN-016
status: todo
dependencies: []
createdAt: 2026-03-11T00:00:00.000Z
---

## Context

`publish.yml` (`build-desktop` job) produces archives named:
- `nexusOrchestrator-desktop-darwin-arm64.zip`
- `nexusOrchestrator-desktop-darwin-amd64.zip`
- `nexusOrchestrator-desktop-windows-amd64.zip`
- `nexusOrchestrator-desktop-linux-amd64.tar.gz`

The **Desktop App** section of `docs/downloads.md` still links to
`nexusOrchestrator-{os}-{arch}.*` (without the `-desktop-` infix), which would 404 on the
GitHub Release after a new release is published.

## Changes

In `docs/downloads.md`, update **only** the Desktop App section `<a href="...">` target URLs:
- `nexusOrchestrator-darwin-arm64.zip` → `nexusOrchestrator-desktop-darwin-arm64.zip`
- `nexusOrchestrator-darwin-amd64.zip` → `nexusOrchestrator-desktop-darwin-amd64.zip`  
  (or `.tar.gz` → `.zip` if the current link uses .tar.gz)
- `nexusOrchestrator-windows-amd64.zip` → `nexusOrchestrator-desktop-windows-amd64.zip`
- `nexusOrchestrator-linux-amd64.tar.gz` → `nexusOrchestrator-desktop-linux-amd64.tar.gz`
  (desktop section only)

The CLI+Daemon section links keep their original `nexusOrchestrator-{os}-{arch}.*` names unchanged.

## Definition of Done

- Desktop App section hrefs all include the `-desktop-` infix
- CLI + Daemon section hrefs unchanged
- No other content changes
