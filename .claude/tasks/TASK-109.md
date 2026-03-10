---
id: TASK-109
title: Update downloads.md desktop section links to nexusOrchestrator-desktop-* names
role: ui
planId: PLAN-014
status: todo
dependencies: [TASK-108]
createdAt: 2026-03-10T22:00:00.000Z
---

## Context

`docs/downloads.md` has two sections:
1. **Desktop App** — links currently point to `nexusOrchestrator-darwin-arm64.tar.gz` etc.
2. **CLI + Daemon** — links also point to the same `nexusOrchestrator-darwin-arm64.tar.gz` etc.

After TASK-108, desktop archives are renamed to `nexusOrchestrator-desktop-{os}-{arch}.tar.gz` (or `.zip`). CLI archives keep their original name `nexusOrchestrator-{os}-{arch}.tar.gz`.

## Changes Required

In `docs/downloads.md`, update ONLY the Desktop App section `<a>` href attributes:
- `nexusOrchestrator-darwin-arm64.tar.gz` → `nexusOrchestrator-desktop-darwin-arm64.tar.gz`
- `nexusOrchestrator-darwin-amd64.tar.gz` → `nexusOrchestrator-desktop-darwin-amd64.tar.gz`
- `nexusOrchestrator-windows-amd64.zip` → `nexusOrchestrator-desktop-windows-amd64.zip`
- `nexusOrchestrator-linux-amd64.tar.gz` → `nexusOrchestrator-desktop-linux-amd64.tar.gz` (desktop section only)

The CLI + Daemon section links keep their original names unchanged (they correctly map to CLI archives).

Also update the verify section to mention both `SHA256SUMS.txt` (CLI) and `SHA256SUMS-desktop.txt` (desktop) is already documented — verify it's accurate, no change needed if already correct.

## Definition of Done

- Desktop App section links use `nexusOrchestrator-desktop-*` names
- CLI + Daemon section links unchanged (still `nexusOrchestrator-{os}-{arch}.*`)
- No broken links in either section
