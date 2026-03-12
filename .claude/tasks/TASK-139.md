---
id: TASK-139
title: Add vscode-extension CI check to ci.yml
role: devops
planId: PLAN-019
status: todo
dependencies: []
createdAt: 2026-03-11T18:00:00.000Z
---

## Context

`ci.yml` validates Go code on every push/PR, but there is no job to typecheck or package
the VS Code extension. Adding a `build-vscode` job ensures extension regressions are caught
on every PR that touches `vscode-extension/**`.

## Files to Read

- `.github/workflows/ci.yml` (full file)

## Implementation Steps

1. Open `.github/workflows/ci.yml` and add a new job named `build-vscode` at the end.

   The job should:
   - `name: Build VS Code Extension`
   - `runs-on: ubuntu-latest`
   - `timeout-minutes: 15`
   - Steps:
     a. `actions/checkout@v6`
     b. `actions/setup-node@v6` with `node-version: "20"`, `cache: "npm"`,
        `cache-dependency-path: vscode-extension/package-lock.json`
     c. **Install dependencies** (`working-directory: vscode-extension`): `npm ci`
     d. **Build extension** (`working-directory: vscode-extension`): `npm run build`
     e. **Package VSIX** (`working-directory: vscode-extension`, shell: bash):
        ```bash
        npx @vscode/vsce package --no-dependencies
        ```
        (This validates the extension can be fully packaged — acts as a smoke test.)

2. No changes are needed to the `on:` trigger — it already covers all pushes/PRs.
   The job runs unconditionally (no tag-skip condition needed for CI).

## Acceptance Criteria

- [ ] `ci.yml` has a `build-vscode` job
- [ ] The job installs Node 20, runs `npm ci`, `npm run build`, and `vsce package`
- [ ] The job does NOT upload any artifact (CI only — no release publishing)
- [ ] No existing ci.yml jobs are modified

## Anti-patterns to Avoid

- Do not add `if: startsWith(github.ref, 'refs/tags/')` or similar — this should run on every push
- Do not add `needs:` dependencies — let it run in parallel with the Go jobs
- Do not upload the artifact from CI (only publish.yml should upload release artifacts)
