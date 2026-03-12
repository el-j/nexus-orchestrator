---
id: PLAN-019
title: VSIX release pipeline + docs site mention
status: active
createdAt: 2026-03-11T18:00:00.000Z
---

## Goal

Build the VS Code extension `.vsix` in the GitHub Actions release pipeline and attach it
as a downloadable release asset. Mention it prominently on the GitHub Pages documentation
site (downloads page + home page feature list).

## Context

PLAN-018 delivered a complete VS Code extension at `vscode-extension/`. However:
- `vscode-extension/package.json` is missing `publisher` field and `@vscode/vsce` devDependency,
  so `vsce package` cannot run.
- `publish.yml` has no job to build the VSIX or upload it as a release artifact.
- `ci.yml` has no job to typecheck / build the extension on PRs.
- `docs/src/views/DownloadsView.vue` has no VS Code extension download section.
- `docs/src/views/HomeView.vue` features list does not mention the extension.
- `docs/index.md` landing page markdown does not mention the extension.

## Tasks

| ID       | Role     | Title                                           | Deps |
|----------|----------|-------------------------------------------------|------|
| TASK-137 | devops   | Prepare vscode-extension for vsce packaging     | —    |
| TASK-138 | devops   | Add VSIX build job to publish.yml               | —    |
| TASK-139 | devops   | Add vscode-extension CI check to ci.yml         | —    |
| TASK-140 | frontend | Add VS Code extension section to DownloadsView  | —    |
| TASK-141 | frontend | Add VS Code Extension to HomeView features      | —    |
| TASK-142 | docs     | Update docs/index.md with VS Code extension     | —    |

All six tasks are independent — execute in a single parallel wave.

## Success Criteria

- `cd vscode-extension && npm ci && npx @vscode/vsce package --no-dependencies` succeeds
- `publish.yml` has a `build-vscode` job that uploads `nexus-orchestrator-vscode.vsix`
- The release job includes the `.vsix` in GitHub Release assets
- `ci.yml` validates the extension on PRs with `vscode-extension/**` changes
- DownloadsView.vue has a VS Code extension download section and "What's Included" card
- HomeView.vue features list includes a VS Code Extension entry
- `docs/index.md` mentions the extension
