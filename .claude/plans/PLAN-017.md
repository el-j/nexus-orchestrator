---
id: PLAN-017
title: Fix all broken downloads + macOS Gatekeeper UX
status: active
createdAt: 2026-03-11T01:00:00.000Z
---

# PLAN-017: Fix all broken downloads + macOS Gatekeeper UX

## Goal

Make every download link on the downloads page actually resolve to a real GitHub Release
artifact, and provide clear macOS Gatekeeper workaround instructions so users can actually
open the app after downloading.

## Root Cause Analysis

### 1. Archive naming mismatch — ALL download links 404
The `publish.yml` pipeline produces archives named `nexus-orchestrator-*` (hyphenated),
but `docs/downloads.md` links to `nexusOrchestrator-*` (camelCase). Every single download
link on the page results in a 404. This affects all platforms, not just macOS.

Pipeline produces:
- CLI: `nexus-orchestrator-{os}-{arch}.tar.gz` / `.zip`
- Desktop: `nexus-orchestrator-desktop-{os}-{arch}.tar.gz` / `.zip`

Downloads page links to:
- CLI: `nexusOrchestrator-{os}-{arch}.tar.gz` / `.zip`  (WRONG)
- Desktop: `nexusOrchestrator-desktop-{os}-{arch}.tar.gz` (WRONG)

### 2. macOS desktop links use .tar.gz but pipeline produces .zip
The `build-desktop` matrix has `archive_ext: zip` for macOS, but the downloads page
links to `.tar.gz`. Even after fixing the prefix, these would still 404.

### 3. macOS Gatekeeper blocks ad-hoc signed apps from the internet
Without `APPLE_CERTIFICATE` secrets, the pipeline falls back to ad-hoc codesigning
(`codesign --sign -`). macOS Gatekeeper always blocks these when downloaded from the
internet. Users MUST clear the quarantine attribute before opening.

The downloads page has zero mention of this. Users see "Apple could not verify" and
assume the app is broken.

## Solution

| # | Action | Effect |
|---|--------|--------|
| 1 | Fix ALL download hrefs in docs/downloads.md | Match actual pipeline artifact names |
| 2 | Fix macOS desktop links from .tar.gz to .zip | Match actual pipeline archive format |
| 3 | Add prominent macOS Gatekeeper instructions section | Users can actually open the app |
| 4 | Fix checksum verification examples | Reference correct file names |
| 5 | Update CHANGELOG.md | Document the fixes |

## Execution Waves

| Wave | Tasks (parallel) | Blocking on |
|------|------------------|-------------|
| 1 | TASK-122, TASK-123 | — |
| 2 | TASK-124 | TASK-122, TASK-123 |

## Task Index

| ID | Title | Role | Dependencies |
|----|-------|------|--------------|
| TASK-122 | Fix all download hrefs + add macOS Gatekeeper section in downloads.md | devops | — |
| TASK-123 | Add macOS first-run instructions to getting-started.md | docs | — |
| TASK-124 | Update CHANGELOG + orchestrator.json tracking | planning | TASK-122, TASK-123 |

## Definition of Done

- Every download link resolves to the correct artifact (prefix: `nexus-orchestrator-`)
- macOS desktop links end in `.zip` (not `.tar.gz`)
- Prominent macOS Gatekeeper instructions visible on downloads page
- getting-started page mentions the quarantine workaround
- CHANGELOG updated
- orchestrator.json tracking updated
