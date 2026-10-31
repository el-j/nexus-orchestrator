---
layout: default
title: Releasing
nav_order: 9
---

# Releasing

Every stable release has a clean `vX.Y.Z` tag, never `v0.10.0-48`. Stable releases are made by
[release-please](https://github.com/googleapis/release-please); betas come from `develop`.

| Where     | Version                          | Made by                                          |
| --------- | -------------------------------- | ------------------------------------------------ |
| `develop` | `X.(Y+1).0-beta.N` (pre-release) | Publish workflow, `scripts/next-version.sh`      |
| `main`    | `X.Y.Z` (stable, "Latest")       | release-please, then Publish attaches the builds |

`N` counts the commits since the last stable tag and never goes below an existing beta tag.

## Making a release

1. Open a pull request `develop` → `main` and merge it with a **merge commit** (`main` allows it; do not squash), so the
   conventional commits stay visible.
2. The Release workflow (`release-please.yml`) opens or updates **`chore(main): release X.Y.Z`**.
   The version follows the commits: `feat:` is a minor bump, `fix:` a patch, `!` or
   `BREAKING CHANGE:` a major bump (while the major is 0, breaking changes bump the minor).
3. Merge that pull request. release-please creates the tag `vX.Y.Z` and the GitHub release.
4. The Publish workflow is called with that tag, builds the CLI, desktop app and VS Code extension and
   attaches them to the release.
5. A `main` → `develop` pull request is opened; squash-merge it (`develop` requires linear history, so merge commits are blocked there) so betas continue from the new version.

`CHANGELOG.md` is written by hand (Keep a Changelog); add the section for the version in the release
pull request branch before merging. Configuration: `release-please-config.json` and
`.release-please-manifest.json` (the last stable version).

## Setup

- Settings → Actions → General: allow "Read and write permissions" and "Allow GitHub Actions to create
  and approve pull requests".
- Optional secret `RELEASE_PLEASE_TOKEN` (a personal access token with contents and pull-requests
  write). Pull requests opened with the default token do not start CI; with the token they do.
- The first release is forced to `0.10.0` by `release-as` in `release-please-config.json`; remove that
  line (and `bootstrap-sha`) after it ships.

## Signing the macOS app

Without secrets the desktop app is ad-hoc signed and users clear the quarantine flag themselves
(`Open Nexus.command` does it). To ship a signed and notarized app, add these repository secrets; the
**Sign macOS app** step in `publish.yml` switches to Developer ID signing as soon as
`APPLE_CERTIFICATE` is set:

| Secret                                            | Value                                                                     |
| ------------------------------------------------- | ------------------------------------------------------------------------- |
| `APPLE_CERTIFICATE`                               | the Developer ID Application certificate as a `.p12`, base64              |
| `APPLE_CERTIFICATE_PASSWORD`                      | the password of that `.p12`                                               |
| `APPLE_DEVELOPER_NAME`                            | the name in the certificate, as in `Developer ID Application: <name>`     |
| `APPLE_ID`, `APPLE_APP_PASSWORD`, `APPLE_TEAM_ID` | the Apple account, an app-specific password and the team for `notarytool` |

The hardened-runtime entitlements are in `packaging/macos/entitlements.plist`. The app targets
macOS 13 (`MACOSX_DEPLOYMENT_TARGET`).

## Rebuilding a release

Run **Publish** manually (Actions → Publish → Run workflow) and enter the tag, for example `v0.10.0`.
