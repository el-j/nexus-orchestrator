---
id: TASK-492
plan: PLAN-063
status: done
wave: 4
priority: 2
---

# TASK-492: Fix `release.sh` - rebuild `github-action/dist` and extension before commit

## Problem

`scripts/release.sh` bumps version numbers but does NOT rebuild `github-action/dist/index.js` or the VS Code extension compiled output before committing. After a version bump, the committed `dist` bundle contains the previous version string, meaning the published GitHub Action runs old code until a manual re-build.

## Checklist

- [ ] After the version sync block in `release.sh`, add: `cd github-action && npm ci && npm run build && cd ..` to rebuild `dist/index.js` with the new version baked in
- [ ] Add: `cd vscode-extension && npm ci && npm run compile && cd ..` to recompile the extension; note this is not the `.vsix` package step — that runs in CI
- [ ] Ensure both build steps run before `git add` and `git commit` so the rebuilt artifacts are included in the version-bump commit
- [ ] If either `npm run build` fails, `release.sh` should exit non-zero and print a clear error message — add `set -e` at the top of the script if not already present, and wrap the build steps with an `|| { echo "Build failed in $PWD"; exit 1; }`
- [ ] Smoke test: run `bash -n scripts/release.sh` to check for syntax errors; document the manual dry-run steps in a comment at the top of the script
- [ ] Add a GitHub Actions CI job that runs `scripts/release.sh --dry-run` (or equivalent) on PRs that touch `release.sh` to prevent broken release scripts from merging

## Files to change

- `scripts/release.sh`
- `.github/workflows/` (add or update a lint-release-script job)

## Acceptance criteria

- [ ] After running `release.sh`, `git diff --stat HEAD` shows updated `github-action/dist/index.js` and `vscode-extension/` compiled output alongside the version bump commits
- [ ] `release.sh` exits non-zero and prints a diagnostic message if either build step fails
- [ ] `bash -n scripts/release.sh` exits 0 (no syntax errors)
