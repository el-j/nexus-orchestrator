---
id: TASK-108
title: Create unified publish.yml CI/CD pipeline
role: devops
planId: PLAN-014
status: todo
dependencies: []
createdAt: 2026-03-10T22:00:00.000Z
---

## Context

Three separate GitHub Actions workflows exist today:
- `.github/workflows/version.yml` — calculates version via GitVersion, creates a `v*.*.*` git tag using `GITHUB_TOKEN`
- `.github/workflows/release.yml` — triggers on `push: tags: v*.*.*`, builds CLI+daemon for 5 platforms, publishes GitHub Release
- `.github/workflows/desktop.yml` — triggers on `push: tags: v*.*.*`, builds Wails desktop app for 4 platforms, attaches to GitHub Release

**Root problem**: `version.yml` uses `GITHUB_TOKEN` to push the tag. GitHub does NOT propagate workflow-triggered events (tags pushed by `GITHUB_TOKEN`) to other workflows — so `release.yml` and `desktop.yml` never fire.

## Goal

Create `.github/workflows/publish.yml` — a single unified workflow that:

1. Triggers on `push` to `main` (same as `version.yml`) with the same `paths-ignore`
2. **Job: version** — GitVersion calculate + check if tag already exists (outputs: `semVer`, `tag`, `exists`)
3. **Job: test** — needs `version`, only runs if `exists == false`. Uses CGO+zig, same as current release.yml test job.
4. **Job: build-cli** — needs `[version, test]`, matrix of 5 platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64), builds nexus-cli + nexus-daemon + nexus-submit. Archives named `nexusOrchestrator-{os}-{arch}.tar.gz` (or `.zip` for windows). Uses zig cross-compiler on ubuntu-latest for linux+windows targets, native macos-latest for darwin targets.
5. **Job: build-desktop** — needs `[version, test]`, matrix of 4 platforms (darwin/arm64, darwin/amd64, windows/amd64, linux/amd64), builds Wails GUI app. Archives named `nexusOrchestrator-desktop-{os}-{arch}.tar.gz` (or `.zip` for windows).
6. **Job: release** — needs `[version, build-cli, build-desktop]`, creates git tag atomically after all builds succeed, downloads all artifacts, generates `SHA256SUMS.txt` (CLI archives) + `SHA256SUMS-desktop.txt` (desktop archives), creates GitHub Release via `softprops/action-gh-release@v2`.

## Requirements

- `permissions: contents: write` at workflow level
- `concurrency: group: publish-${{ github.ref }}, cancel-in-progress: true`
- All jobs have `if: needs.version.outputs.exists == 'false'` (except `version` job itself)  
- `version` job outputs: `semVer`, `tag` (e.g. `v0.2.1`), `exists` (true/false)
- Go version: `1.24`, Wails version: `v2.11.0`, Zig version: `0.14.0`
- Upload artifacts via `actions/upload-artifact@v4`, download via `actions/download-artifact@v4`
- CLI artifact names: `nexusOrchestrator-{os}-{arch}.tar.gz` / `.zip` (preserves `install.sh` compatibility)
- Desktop artifact names: `nexusOrchestrator-desktop-{os}-{arch}.tar.gz` / `.zip`
- `LDFLAGS: -s -w -X main.version=${VERSION}` for both CLI and desktop builds
- Desktop linux build needs: `libgtk-3-dev libwebkit2gtk-4.0-dev libayatana-appindicator3-dev pkg-config`
- Tag is created in the `release` job (NOT in `version` job) — tag creation is the final atomic step after all builds pass
- Linux builds (zig cross-compile) for CLI: use `ubuntu-latest`, `GOOS={linux|windows}`, `GOARCH={amd64|arm64}`, `CC/CXX = zig cc/c++ -target ...`
- Darwin CLI builds: use `macos-latest`, no zig needed, `CGO_ENABLED=1`
- Desktop macOS builds: `macos-latest`
- Desktop Windows build: `windows-latest`, `CGO_ENABLED=1`
- Desktop Linux build: `ubuntu-latest`, `CGO_ENABLED=1`
- Also keep `workflow_dispatch:` as a secondary trigger for manual runs
- On `workflow_dispatch`, the `exists` check should still prevent duplicate tags

## File to create

`.github/workflows/publish.yml`

## Definition of Done

- File created with valid YAML
- All 6 jobs defined with correct `needs`, `if`, and `matrix` entries
- All artifact upload/download steps consistent with exact file names
- SHA256SUMS step covers CLI archives; SHA256SUMS-desktop.txt covers desktop archives  
- `softprops/action-gh-release@v2` attaches all archives + both checksum files
- `go vet` and YAML lint would pass
