---
id: TASK-147
title: Update CHANGELOG.md with PLAN-018/019/020 as v0.8.0
role: planning
planId: PLAN-021
status: todo
dependencies: []
createdAt: 2026-03-12T10:00:00.000Z
---

## Context
CHANGELOG.md has not been updated since PLAN-017. Three substantial feature releases have landed:
- PLAN-018: Provider visibility (configurable URLs, SQLite-persisted CRUD, VS Code extension)
- PLAN-019: VS Code extension VSIX added to release pipeline + docs updates
- PLAN-020: GUI provider display fixed (TypeScript PascalCase→camelCase mismatch, routing wired, ProvidersView created)

We follow "Keep a Changelog" format (https://keepachangelog.com/en/1.1.0/) and semantic versioning.

## Current top of CHANGELOG.md (for reference only — do NOT duplicate)
```markdown
## [Unreleased]

### Fixed

- All download links on docs site now resolve correctly — prefix corrected from
  `nexusOrchestrator-*` to `nexus-orchestrator-*` to match actual GitHub Release artifact names
- macOS Desktop download links now point to `.zip` format (matching pipeline output);
  previously linked to `.tar.gz` which would 404
- Checksum verification examples on downloads page now reference correct file names

### Added

- macOS Gatekeeper / quarantine workaround instructions on Downloads page
  (prominent warning section explaining "Apple could not verify" is expected for
  unsigned open-source apps, with right-click and `xattr` solutions)
- macOS first-run setup instructions added to Getting Started guide

## [0.2.0] - 2026-03-11
```

## Implementation Steps
1. Add a new `## [0.8.0] - 2026-03-12` section **immediately after** the `## [Unreleased]` block
   (between `[Unreleased]` and `[0.2.0]`).
2. The new section must contain exactly these subsections in this order:

   ### Added

   #### Provider Management (PLAN-018)
   - Configurable Ollama and LM Studio base URLs — override defaults via `NEXUS_OLLAMA_BASE_URL` and `NEXUS_LMSTUDIO_BASE_URL` environment variables; also configurable in the GUI
   - `BaseURL` and `Error` fields added to `ProviderInfo` so the frontend can display the live endpoint and error state
   - SQLite-persisted provider CRUD (`ProviderConfigRepository` port + `repo_sqlite` implementation)
   - Providers management panel in the Wails GUI: add, edit, delete configured providers with preset URLs for LM Studio, Ollama, OpenAI, Anthropic, and custom OpenAI-compatible endpoints
   - API key field masked (`***`) in all HTTP responses to prevent credential leakage
   - VS Code extension (`vscode-extension/`) with task submit, task queue view, and status bar provider indicator

   #### Release Pipeline (PLAN-019)
   - `@vscode/vsce` packaging for the VS Code extension added to `publish.yml` — VSIX artifact included in every GitHub Release
   - `build-vscode` job added to `ci.yml` for smoke-test validation of the extension on every PR
   - VS Code extension download section added to the docs Downloads page and Homepage feature list

   ### Fixed

   #### GUI Provider Display (PLAN-020)
   - `ProviderInfo` TypeScript fields corrected from PascalCase (`Name`, `Active`, `ActiveModel`, `Models`) to camelCase (`name`, `active`, `activeModel`, `models`) to match Go JSON serialisation — provider cards were rendering blank before this fix
   - `AppSidebar.vue` navigation wired with `defineEmits` — clicking "Providers" now correctly switches the view
   - `ProvidersView.vue` created as a full-page provider management view (discovered providers grid + configured provider CRUD)

3. Output the **complete CHANGELOG.md file** — all existing sections must remain intact below the new `[0.8.0]` entry.
   - Do NOT alter the `[Unreleased]` block
   - Do NOT alter existing `[0.2.0]` or older sections

## Acceptance Criteria
- [ ] `## [0.8.0] - 2026-03-12` section exists between `[Unreleased]` and `[0.2.0]`
- [ ] Section has ### Added and ### Fixed subsections covering all three plans
- [ ] Existing CHANGELOG content is preserved verbatim
- [ ] Output is the complete file

## Anti-patterns to Avoid
- Do NOT remove or alter the `[Unreleased]` section
- Do NOT invent features that were not listed above
- Do NOT use bullet styles other than `-` (no `*` no numbered lists in changelog entries)
