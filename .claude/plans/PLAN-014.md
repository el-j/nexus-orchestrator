# PLAN-014: Fix unified release pipeline

**Goal**: Eliminate the GitHub Actions cross-trigger limitation that prevents builds from running when a tag is created by the version workflow. Consolidate `version.yml`, `release.yml`, and `desktop.yml` into one `publish.yml` pipeline. Fix the artifact naming collision between CLI and desktop builds. Align `downloads.md` with correct artifact names.

**Status**: in-progress  
**CreatedAt**: 2026-03-10T22:00:00.000Z

## Problem Summary

1. `version.yml` pushes a tag via `GITHUB_TOKEN`. GitHub intentionally does NOT trigger other workflows on events caused by `GITHUB_TOKEN` — so `release.yml` and `desktop.yml` never fire and no release artifacts are ever published.  
2. Both `release.yml` and `desktop.yml` produce archives named `nexusOrchestrator-{os}-{arch}.tar.gz` — identical filenames, so one overwrites the other in the GitHub Release asset list.  
3. `docs/downloads.md` Desktop section and CLI section both link to the same URLs — one section always serves the wrong binary.

## Solution Architecture

- Replace three separate workflows with a single `publish.yml` that calculates version, runs tests, builds CLI+daemon and Desktop app in parallel, then creates the tag and GitHub Release with all artifacts atomically.
- Rename CLI archives to `nexusOrchestrator-{os}-{arch}.tar.gz` (backward-compatible with `install.sh`).
- Rename desktop archives to `nexusOrchestrator-desktop-{os}-{arch}.tar.gz`.
- Update `docs/downloads.md` Desktop section links to use `nexusOrchestrator-desktop-*` names.
- Delete `version.yml`, `release.yml`, `desktop.yml` (superseded).

## Tasks

| ID | Title | Role | Dependencies |
|----|-------|------|--------------|
| TASK-108 | Create unified publish.yml CI/CD pipeline | devops | — |
| TASK-109 | Fix desktop artifact naming in publish.yml + update downloads.md | devops+ui | TASK-108 |
| TASK-110 | Remove superseded workflows | devops | TASK-108 |
| TASK-111 | QA — validate YAML syntax + artifact consistency | qa | TASK-109, TASK-110 |
