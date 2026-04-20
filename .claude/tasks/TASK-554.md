---
id: TASK-554
title: Fix CI release pipeline — broken artifact action, CGO flag, marketplace publish
role: devops
planId: PLAN-071
status: todo
dependencies: []
createdAt: 2026-04-15T00:00:00Z
---

## Context

The `publish.yml` workflow references `actions/download-artifact@v8` which does not exist (current version is v4), causing every release to fail at the download step. Additionally, the CI build job uses `CGO_ENABLED=0` for `nexus-daemon` which silently produces a broken binary (sqlite3 requires CGO). VS Code Marketplace and Open VSX publish steps are permanently commented out, so the extension never reaches users via automation.

## Files to Read

- `.github/workflows/publish.yml`
- `.github/workflows/ci.yml`
- `Makefile`

## Implementation Steps

1. In `publish.yml`: replace every `actions/download-artifact@v8` reference with `actions/download-artifact@v4` (and matching `actions/upload-artifact@v4` if any v8 uploads exist)
2. In `ci.yml` build job: change the `nexus-daemon` build step from `CGO_ENABLED=0` to `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"` (match the Makefile `build-daemon` target)
3. In `publish.yml`: uncomment the VS Code Marketplace publish step and the Open VSX publish step; ensure they run only when a `VSCE_PAT` / `OVSX_TOKEN` secret is present (use `if: env.VSCE_PAT != ''` guard so CI does not fail on forks)
4. Verify all `upload-artifact` / `download-artifact` version pairs are consistent (v4 throughout)
5. Run a dry-run syntax check: `act -n` or validate YAML schema manually

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] No `actions/download-artifact@v8` or `actions/upload-artifact@v8` references remain in any workflow file
- [ ] `ci.yml` nexus-daemon build step has `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"`
- [ ] VS Code publish steps in `publish.yml` are uncommented and gated with secret-presence `if:` condition

## Anti-patterns to Avoid

- NEVER change version numbers of actions that are already on v4 (only fix the v8 ones)
- NEVER remove the secret-presence guard from publish steps — CI must not fail on forks without secrets
