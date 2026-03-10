---
id: TASK-111
title: QA validate artifact names and workflow consistency
role: qa
planId: PLAN-014
status: todo
dependencies: [TASK-109, TASK-110]
createdAt: 2026-03-10T22:00:00.000Z
---

## Checks

1. `publish.yml` YAML is syntactically valid (run `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/publish.yml'))"`)
2. All upload-artifact `name:` values in `build-cli` job match exactly the download-artifact expectations in `release` job
3. All upload-artifact `name:` values in `build-desktop` job match exactly the download-artifact expectations in `release` job
4. `SHA256SUMS.txt` covers only CLI archives (`nexusOrchestrator-{os}-{arch}.*`)
5. `SHA256SUMS-desktop.txt` covers only desktop archives (`nexusOrchestrator-desktop-{os}-{arch}.*`)
6. `scripts/install.sh` still references `nexusOrchestrator-${OS}-${ARCH}.tar.gz` (CLI naming — unchanged, should be fine)
7. `docs/downloads.md` Desktop section uses `nexusOrchestrator-desktop-*` links
8. `docs/downloads.md` CLI section uses `nexusOrchestrator-{os}-{arch}.*` links (no `-desktop-` in CLI section)
9. Old workflow files (`version.yml`, `release.yml`, `desktop.yml`) no longer exist

## Definition of Done

All 9 checks pass. Report any inconsistency found.
