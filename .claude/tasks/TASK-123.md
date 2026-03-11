---
id: TASK-123
title: Add macOS first-run instructions to getting-started.md
role: docs
planId: PLAN-017
status: todo
dependencies: []
createdAt: 2026-03-11T01:00:00.000Z
---

## Context

Users downloading the macOS desktop .app will hit the Gatekeeper "Apple could not verify"
dialog because the app is ad-hoc signed (no Apple Developer ID certificate). The
getting-started guide should mention this upfront so users don't assume the app is malware.

## Changes Required

In `docs/getting-started.md`, add a section after the initial download/install steps titled
"macOS: Allow nexusOrchestrator to Run" with:

1. Right-click the .app → "Open" → click "Open" in the dialog (bypasses Gatekeeper once)
2. Or terminal: `xattr -dr com.apple.quarantine /path/to/nexusOrchestrator.app`
3. Brief explanation: this is normal for open-source apps not (yet) notarized by Apple

## Definition of Done
- getting-started.md contains macOS Gatekeeper workaround
- Instructions are clear and accurate
