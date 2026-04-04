---
id: TASK-485
plan: PLAN-063
status: done
wave: 2
priority: 1
---

# TASK-485: Implement Marketplace publishing in CI pipeline

## Problem

`.github/workflows/publish.yml` only uploads the compiled `.vsix` as a GitHub Release artifact — no `vsce publish` step exists. Users searching the VS Code Marketplace or Open VSX cannot find the extension. The extension is functionally invisible.

## Checklist

- [ ] In `publish.yml`, add a `vsce publish` step after the existing artifact upload step, gated on `secrets.VSCE_PAT` being present; use `npx @vscode/vsce publish --pat ${{ secrets.VSCE_PAT }}` from the `vscode-extension/` working directory
- [ ] Add an `ovsx publish` step for Open VSX Registry using `npx ovsx publish *.vsix --pat ${{ secrets.OVSX_PAT }}`; gate this step similarly on the secret being present so the workflow degrades gracefully if only one secret is set
- [ ] Ensure both publish steps run only on tag pushes matching `v*` (not every commit)
- [ ] Add both `VSCE_PAT` and `OVSX_PAT` to the repository secret names documented in `vscode-extension/README.md` under a new "Release" section
- [ ] In `scripts/release.sh`, add a step that bumps `vscode-extension/package.json` `version` field to match the root `package.json` version being cut (use `jq` or `npm version` with `--no-git-tag-version`); place this step before the final `git commit`
- [ ] Verify the `publish.yml` trigger also fires on the `release` event (`types: [published]`) as a belt-and-suspenders alongside `push: tags`

## Files to change

- `.github/workflows/publish.yml`
- `scripts/release.sh`
- `vscode-extension/README.md`

## Acceptance criteria

- [ ] A release tag push triggers `vsce publish` and `ovsx publish` steps in CI
- [ ] Workflow succeeds with secrets set; degrades to a warning (not a failure) when secrets are absent
- [ ] `vscode-extension/package.json` version matches root version after running `release.sh`
