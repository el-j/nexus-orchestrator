---
id: TASK-122
title: Fix all download hrefs and add macOS Gatekeeper section in downloads.md
role: devops
planId: PLAN-017
status: todo
dependencies: []
createdAt: 2026-03-11T01:00:00.000Z
---

## Problem

ALL download links on docs/downloads.md are broken (404) because:
1. Links use `nexusOrchestrator-*` (camelCase) but pipeline produces `nexus-orchestrator-*` (hyphenated)
2. macOS desktop links use `.tar.gz` but pipeline produces `.zip` for macOS

## Changes Required

### Desktop section — fix prefix AND macOS extension
- `nexusOrchestrator-desktop-darwin-arm64.tar.gz` → `nexus-orchestrator-desktop-darwin-arm64.zip`
- `nexusOrchestrator-desktop-darwin-amd64.tar.gz` → `nexus-orchestrator-desktop-darwin-amd64.zip`
- `nexusOrchestrator-desktop-windows-amd64.zip` → `nexus-orchestrator-desktop-windows-amd64.zip`
- `nexusOrchestrator-desktop-linux-amd64.tar.gz` → `nexus-orchestrator-desktop-linux-amd64.tar.gz`
- Also update button text: macOS cards should say "Download .zip" not "Download .tar.gz"

### CLI section — fix prefix only
- `nexusOrchestrator-darwin-arm64.tar.gz` → `nexus-orchestrator-darwin-arm64.tar.gz`
- `nexusOrchestrator-darwin-amd64.tar.gz` → `nexus-orchestrator-darwin-amd64.tar.gz`
- `nexusOrchestrator-windows-amd64.zip` → `nexus-orchestrator-windows-amd64.zip`
- `nexusOrchestrator-linux-amd64.tar.gz` → `nexus-orchestrator-linux-amd64.tar.gz`
- `nexusOrchestrator-linux-arm64.tar.gz` → `nexus-orchestrator-linux-arm64.tar.gz`

### Verify section — fix checksum file names
- Fix example file references to use `nexus-orchestrator-*` prefix

### Add macOS Gatekeeper section
After the "Verify Your Download" section, add a prominent macOS first-run section:
- Title: "🍎 macOS: First Run"
- Explain that the app is ad-hoc signed (not notarized) so Gatekeeper will block it
- Provide two solutions: (1) Right-click → Open, (2) `xattr -dr com.apple.quarantine nexusOrchestrator.app`
- Style it prominently with a warning color

## Definition of Done
- All 9 + 4 download links use correct `nexus-orchestrator-*` prefix
- macOS desktop links end in `.zip`
- macOS desktop button text says "Download .zip"
- Gatekeeper instructions section added
- Checksum examples use correct file names
