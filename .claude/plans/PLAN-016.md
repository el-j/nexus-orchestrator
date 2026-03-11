---
id: PLAN-016
title: Release pipeline finalization + CHANGELOG
status: active
createdAt: 2026-03-11T00:00:00.000Z
---

# PLAN-016: Release pipeline finalization + CHANGELOG

## Goal

Make the next `git push origin main` atomically trigger the **complete** build-and-release
pipeline (test → multi-platform CLI/daemon builds → multi-platform Desktop builds → GitHub
Release with all artifacts and SHA256 checksums). Also ship a comprehensive `CHANGELOG.md`
following Keep-a-Changelog convention.

## Root Cause of Current Failure

1. **Dual-trigger conflict** — `version.yml` and `publish.yml` both fire on `push: branches: [main]`.
   `version.yml` creates the semver tag first (via `GITHUB_TOKEN`). By the time `publish.yml`
   checks `exists`, the tag already exists → `if: exists == 'false'` guards skip every build
   job → **zero artifacts, zero release**.
2. **Double-release risk** — `release.yml` triggers on `push: tags: v*.*.*`. If it fires after
   `publish.yml` creates the tag, a second GitHub Release is created for the same tag.
3. **Missing CHANGELOG.md** — no changelog file exists; `softprops/action-gh-release@v2` uses
   `generate_release_notes: true` but a CHANGELOG provides richer human-authored notes.
4. **downloads.md Desktop links wrong** — Desktop section hrefs still point to
   `nexusOrchestrator-{os}-{arch}.*` instead of `nexusOrchestrator-desktop-{os}-{arch}.*`.

## Solution

| # | Action | Effect |
|---|--------|--------|
| 1 | Delete `.github/workflows/version.yml` | Removes conflicting tag creator |
| 2 | Delete `.github/workflows/release.yml` | Removes duplicate release trigger |
| 3 | Fix `docs/downloads.md` Desktop links | Aligns with publish.yml artifact names |
| 4 | Create `CHANGELOG.md` | Comprehensive history in Keep-a-Changelog format |
| 5 | Mark TASK-108..111 done in orchestrator.json | Close out PLAN-014 |

After this plan, `push: branches: [main]` triggers **only** `publish.yml`, which owns the
full lifecycle: version → test → build-cli → build-desktop → release.

## Execution Waves

| Wave | Tasks (parallel) | Blocking on |
|------|------------------|-------------|
| 1 | TASK-118, TASK-119, TASK-120 | — |
| 2 | TASK-121 | TASK-118, TASK-119, TASK-120 done |

## Task Index

| ID | Title | Role | Dependencies |
|----|-------|------|--------------|
| TASK-118 | Delete version.yml + release.yml | devops | — |
| TASK-119 | Fix docs/downloads.md Desktop section links | devops | — |
| TASK-120 | Create CHANGELOG.md | devops | — |
| TASK-121 | Update orchestrator.json + orchestrator-index.md | planning | TASK-118, TASK-119, TASK-120 |

## Definition of Done

- `.github/workflows/` contains only: `ci.yml`, `publish.yml`, `pages.yml`, `action-ci.yml`
- `docs/downloads.md` Desktop links use `nexusOrchestrator-desktop-*` prefix
- `CHANGELOG.md` exists at repo root with entries for all 15 completed plans
- `orchestrator.json` counters updated: `nextTaskId=123`, `nextPlanId=17`, PLAN-016 marked `completed`
- `go vet ./...` and `CGO_ENABLED=1 go build ./...` pass
