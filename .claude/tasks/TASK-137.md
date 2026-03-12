---
id: TASK-137
title: Prepare vscode-extension for vsce packaging
role: devops
planId: PLAN-019
status: todo
dependencies: []
createdAt: 2026-03-11T18:00:00.000Z
---

## Context

`vsce package` requires a `publisher` field in `package.json` and needs `@vscode/vsce`
available in devDependencies. Without these, the CI build job (TASK-138) cannot run.
The extension also needs a `repository` field and a `LICENSE` file for vsce compliance.

## Files to Read

- `vscode-extension/package.json`
- `LICENSE` (root — to copy MIT text)

## Implementation Steps

1. Open `vscode-extension/package.json` and make these changes:
   a. Add `"publisher": "el-j"` after the `"version"` field.
   b. Add `"repository": { "type": "git", "url": "https://github.com/el-j/nexus-orchestrator" }` after `"categories"`.
   c. In `"devDependencies"`, add `"@vscode/vsce": "^3.4.0"`.
   d. Update the `"package"` script to:
      `"package": "vsce package --no-dependencies"`
      (avoids bundling the full `node_modules`, keeping the VSIX lean).

2. Run `npm install` inside `vscode-extension/` to regenerate `package-lock.json`.

3. Create `vscode-extension/LICENSE` containing the standard MIT license text
   matching the root `LICENSE` file (same copyright holder `el-j`).

4. Verify packaging works locally:
   ```
   cd vscode-extension
   npm ci
   npm run build
   npx @vscode/vsce package --no-dependencies
   ```
   Confirm a `.vsix` file is produced.

## Acceptance Criteria

- [ ] `vscode-extension/package.json` has `publisher`, `repository`, and `@vscode/vsce` devDependency
- [ ] `npm run package` in `vscode-extension/` succeeds and outputs a `.vsix` file
- [ ] `vscode-extension/LICENSE` exists with MIT text
- [ ] No other files are modified

## Anti-patterns to Avoid

- Do not bump the extension version — leave it at `0.1.0`
- Do not add unnecessary fields to `package.json`
- Do not run `npm install` in any directory other than `vscode-extension/`
