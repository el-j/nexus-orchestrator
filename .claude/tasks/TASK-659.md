---
id: TASK-659
planId: PLAN-080
title: 'ESLint for the VS Code extension and the GitHub Action'
role: quality
status: done
createdAt: 2026-11-24
---

# TASK-659 — ESLint for the VS Code extension and the GitHub Action

Flat config and a CI step for vscode-extension and github-action, as for frontend and docs; fix the findings.

Done: flat ESLint configs (`eslint.config.mjs`) for both, `npm run lint`, CI steps in `ci.yml` and `action-ci.yml`; findings fixed (unused imports and helpers, `catch (_)`), the extension-host and jest suites get their own globals.
