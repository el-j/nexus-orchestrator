---
id: PLAN-013
title: Update CI to latest GitHub Action versions
status: active
createdAt: 2026-03-10T21:00:00.000Z
---

# PLAN-013: Update CI to Latest GitHub Action Versions

## Goal

Fix the broken GitVersion CI workflow and update all outdated tool versions across GitHub Actions workflows to their latest stable releases.

## Root Cause

`gittools/actions/gitversion/setup@v3.1.11` constrains GitVersion to `>=5.2.0 <6.1.0`, but `versionSpec: '6.x'` resolves to 6.6.0 which violates that range. The v4 series (latest: v4.3.3, released 2 weeks ago) properly supports GitVersion 6.6.0.

## Changes Required

| Component | Current | Target | File |
|-----------|---------|--------|------|
| gittools/actions setup+execute | v3.1.11 | v4.3.3 | version.yml |
| GitVersion.yml config | v5-era schema | v6-compatible schema | GitVersion.yml |
| Zig | 0.13.0 (Jun 2024) | 0.14.0 (Mar 2025) | release.yml |

## NOT Changing (and why)

- **Wails CLI**: v2.11.0 stays — Wails v3 is alpha (v3.0.0-alpha.74), not production-ready
- **Standard GH Actions**: checkout v4, setup-go v5, cache v4, etc. — all already at latest majors
- **softprops/action-gh-release**: v2 — already latest

## Execution Waves

| Wave | Tasks | Description |
|------|-------|-------------|
| 1 | TASK-104, TASK-105, TASK-106 (parallel) | Update version.yml, GitVersion.yml, release.yml |
| 2 | TASK-107 (sequential) | QA verification |

## Task Index

| ID | Title | Deps |
|----|-------|------|
| TASK-104 | Update gittools/actions from v3.1.11 to v4.3.3 | — |
| TASK-105 | Update GitVersion.yml for v6.x compatibility | — |
| TASK-106 | Update zig from 0.13.0 to 0.14.0 in release.yml | — |
| TASK-107 | QA verify all workflow changes | TASK-104, TASK-105, TASK-106 |
