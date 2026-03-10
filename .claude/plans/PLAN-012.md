---
id: PLAN-012
title: Semantic versioning + MIT license + zig CI fix
status: active
createdAt: 2026-03-10T20:00:00.000Z
---

# PLAN-012: Semantic Versioning, MIT License, and CI Fixes

## Goal

Add GitVersion-based semantic versioning, an automated release workflow driven by conventional commits, MIT license, and fix the zig installation in CI (zig is not in Ubuntu's apt repos).

## Context

- Both `release.yml` and `desktop.yml` already inject version with `-X main.version=${VERSION}` using `GITHUB_REF_NAME`
- All doc download URLs use `releases/latest/download/` — no hardcoded versions to update
- All Go dependencies are MIT/Apache/BSD compatible — safe for MIT licensing
- Zig install via `apt-get` fails on GitHub Actions Ubuntu runners (not in repos)

## Tech Decisions

- **GitVersion**: Use `gittools/actions/gitversion` in CI for SemVer calculation from git history + conventional commits
- **Versioning workflow**: New `version.yml` runs on push to main, calculates version, creates git tag → triggers existing release.yml and desktop.yml
- **License**: MIT with copyright 2025 el-j
- **Zig install**: Direct download from ziglang.org instead of apt

## Execution Waves

| Wave | Tasks | Description |
|------|-------|-------------|
| 1 | TASK-098, TASK-099 (parallel) | MIT LICENSE + GitVersion config |
| 2 | TASK-100 (sequential) | Semantic-release CI workflow |
| 3 | TASK-101 (sequential) | Update existing workflows for GitVersion |
| 4 | TASK-102 (sequential) | README badges |
| 5 | TASK-103 (sequential) | QA verification |

## Task Index

| ID | Title | Deps |
|----|-------|------|
| TASK-098 | Create MIT LICENSE file | — |
| TASK-099 | Add GitVersion.yml config | — |
| TASK-100 | Create semantic-release CI workflow | TASK-099 |
| TASK-101 | Update release.yml + desktop.yml for GitVersion + zig fix | TASK-100 |
| TASK-102 | Add license + version badges to README | TASK-098 |
| TASK-103 | QA verify all changes | TASK-101, TASK-102 |
