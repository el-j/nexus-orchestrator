---
id: TASK-574
planId: PLAN-073
title: 'macOS signing, notarization and deployment target'
role: polish
status: todo
createdAt: 2026-10-15
---

# TASK-574 — macOS signing, notarization and deployment target

The desktop app is ad-hoc signed (Open Nexus.command clears quarantine) and links for macOS 11 with objects built for 13. Needs an Apple Developer ID certificate in the repo secrets; set MACOSX_DEPLOYMENT_TARGET.

Progress: `MACOSX_DEPLOYMENT_TARGET=13.0` is now exported on Darwin in `Makefile` and set in `publish.yml`. Developer ID signing and notarization remain blocked on adding an Apple Developer ID certificate and notary credentials to repository secrets.
