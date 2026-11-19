---
id: TASK-610
planId: PLAN-075
title: 'Unblock v0.11.0 in release-please-config.json, sync all 5 package manifests in version-sync.sh, and fix publish.yml stamping/tagging'
role: devops
wave: 1
status: done
createdAt: 2026-10-19
---

# TASK-610 — Unblock v0.11.0 in release-please-config.json, sync all 5 package manifests in version-sync.sh, and fix publish.yml stamping/tagging

1. **`release-please-config.json`**:
   - Remove `"release-as": "0.10.0"` and `"bootstrap-sha"` so `release-please` computes `0.11.0` from `.release-please-manifest.json` (`0.10.0`), and add `extra-files` for `frontend/package.json`, `vscode-extension/package.json`, `github-action/package.json`, `docs/package.json`, and `wails.json`.
2. **`scripts/version-sync.sh` & manifests**:
   - Extend `scripts/version-sync.sh` to also stamp `github-action/package.json` and `docs/package.json`, and run `bash scripts/version-sync.sh 0.10.0` so all 5 manifests match `.release-please-manifest.json`.
3. **`.github/workflows/publish.yml`**:
   - Run `bash scripts/version-sync.sh "${{ needs.version.outputs.semVer }}"` and `actions/setup-node@v6` in `build-desktop` (plus copy `build/frontend/.` into `internal/adapters/inbound/webui/dist/`) before `wails build`.
   - Run `scripts/version-sync.sh` _before_ `npm run build` in `build-vscode`.
   - Tag `HEAD` directly in the `release` job instead of creating an unpushed local commit whose SHA differs from the binary `COMMIT` SHA.

Done when: the failing test came first, `make test` and `make nice` pass, docs and changelog updated.
