---
id: TASK-104
title: Update gittools/actions from v3.1.11 to v4.3.3
role: devops
planId: PLAN-013
status: todo
dependencies: []
createdAt: 2026-03-10T21:00:00.000Z
---

## Context
`gittools/actions/gitversion/setup@v3.1.11` constrains GitVersion to `>=5.2.0 <6.1.0` but `versionSpec: '6.x'` resolves to 6.6.0, causing CI failure. The v4 series (latest v4.3.3) supports GitVersion 6.6.0 natively. v4.0.0 release notes: "Update docs to use v4 and GitVersion.Tool v6.1.x". v4.3.0 adds "GitVersion 6.6.0 output variables".

## Files to Read
- `.github/workflows/version.yml`

## Implementation Steps
1. In `.github/workflows/version.yml`, replace `gittools/actions/gitversion/setup@v3.1.11` with `gittools/actions/gitversion/setup@v4.3.3`
2. Replace `gittools/actions/gitversion/execute@v3.1.11` with `gittools/actions/gitversion/execute@v4.3.3`
3. Keep `versionSpec: '6.x'` and `useConfigFile: true` — these are correct for v4
4. Verify the output variable reference `steps.gitversion.outputs.semVer` is still valid in v4 (it is — v4 preserves semVer output)

## Acceptance Criteria
- [ ] version.yml uses `gittools/actions/gitversion/setup@v4.3.3`
- [ ] version.yml uses `gittools/actions/gitversion/execute@v4.3.3`
- [ ] `versionSpec: '6.x'` is preserved
- [ ] YAML is valid

## Anti-patterns to Avoid
- Do NOT use floating `@v4` tag — pin to exact `v4.3.3` for reproducible builds
- Do NOT change the output variable name `semVer` — it's valid in v4
