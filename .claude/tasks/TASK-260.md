# TASK-260 — Fix vscode-extension package-lock.json out of sync

**Plan**: PLAN-039  
**Status**: done  
**Agent**: Senior Developer

## Problem
`npm ci` in `vscode-extension/` fails with:
```
npm error Missing: esbuild@0.27.4 from lock file
```
`vitest@^4.1.0` transitively requires `esbuild@^0.27.0`. The existing `package-lock.json`
was generated before this transitive constraint existed.

## Fix
1. Update `esbuild` range in `package.json` from `^0.20.0` to `^0.27.0` to make the
   direct dep align with what the transitive resolution requires
2. Run `npm install` in `vscode-extension/` to regenerate `package-lock.json`

## Files Changed
- `vscode-extension/package.json` — esbuild `^0.20.0` → `^0.27.0`
- `vscode-extension/package-lock.json` — regenerated

## Result
`npm ci` completes without errors. All extension build + test commands work.
