---
id: TASK-138
title: Add VSIX build job to publish.yml and release job
role: devops
planId: PLAN-019
status: todo
dependencies: []
createdAt: 2026-03-11T18:00:00.000Z
---

## Context

`publish.yml` currently builds CLI binaries (5 platforms) and the Wails desktop app
(4 platforms) and uploads them to GitHub Releases. The VS Code extension VSIX needs an
equivalent job that packages the extension and attaches the `.vsix` to the same release.

## Files to Read

- `.github/workflows/publish.yml` (full file)

## Implementation Steps

1. Add a new `build-vscode` job to `publish.yml`, positioned **after** the `build-desktop`
   job and **before** the `release` job. The job should:
   - `name: Build VS Code Extension`
   - `needs: [version, test]`
   - `if: needs.version.outputs.exists == 'false'`
   - `runs-on: ubuntu-latest`
   - `timeout-minutes: 15`
   - Steps:
     a. `actions/checkout@v6`
     b. `actions/setup-node@v6` with `node-version: "20"`, `cache: "npm"`,
        `cache-dependency-path: vscode-extension/package-lock.json`
     c. **Install dependencies**: `working-directory: vscode-extension`, `run: npm ci`
     d. **Build extension**: `working-directory: vscode-extension`, `run: npm run build`
     e. **Package VSIX** (shell: bash):
        ```bash
        cd vscode-extension
        npx @vscode/vsce package --no-dependencies --out nexus-orchestrator-vscode.vsix
        mv nexus-orchestrator-vscode.vsix ../nexus-orchestrator-vscode.vsix
        ```
     f. `actions/upload-artifact@v6` with:
        - `name: nexus-orchestrator-vscode.vsix`
        - `path: nexus-orchestrator-vscode.vsix`

2. Update the `release` job:
   a. In `needs`, add `build-vscode`:
      `needs: [version, build-cli, build-desktop, build-vscode]`
   b. In the `Generate checksums` step, add a third find command to produce
      `SHA256SUMS-vscode.txt`:
      ```bash
      find . -name 'nexus-orchestrator-vscode*.vsix' -exec sha256sum {} + > ../SHA256SUMS-vscode.txt
      ```
   c. In the `softprops/action-gh-release` step, add the following lines to the `files`
      block:
      ```
      nexus-orchestrator-vscode*.vsix
      SHA256SUMS-vscode.txt
      ```

## Acceptance Criteria

- [ ] `build-vscode` job exists in `publish.yml` after `build-desktop`
- [ ] The job uses Node 20, installs via `npm ci`, builds, then calls `vsce package`
- [ ] Output artifact is named `nexus-orchestrator-vscode.vsix`
- [ ] `release` job `needs` includes `build-vscode`
- [ ] GitHub Release `files` glob includes `nexus-orchestrator-vscode*.vsix` and `SHA256SUMS-vscode.txt`

## Anti-patterns to Avoid

- Do not use `npm install -g vsce` — use `npx @vscode/vsce` which respects the local devDependency
- Do not change the desktop or CLI build jobs
- Do not merge the VSIX checksum into `SHA256SUMS.txt` — keep a separate `SHA256SUMS-vscode.txt`
