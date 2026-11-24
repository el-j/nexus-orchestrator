---
id: PLAN-080
title: 'Nexus as the model endpoint, brain and orchestrator for every tool'
status: active
createdAt: 2026-11-24
tasks: [TASK-657, TASK-658, TASK-659, TASK-660, TASK-661]
---

# PLAN-080 — Nexus as the model endpoint, brain and orchestrator for every tool

Audit of develop on 2026-11-24 (v0.15.0-beta.4 installed and run): Go coverage 100 % (11 244 statements), every
frontend, docs, extension and action suite at 100 % with type checks, docs checks and lint clean, every REST
route in an e2e test. What is missing is reach: the OpenAI-compatible gateway runs only as a separate binary
without the brain, so tools that speak the OpenAI API cannot use Nexus's providers or project memory; the new
binaries have no e2e; two TypeScript packages have no linter.

| Task     | Title                                                | Role         | Status |
| -------- | ---------------------------------------------------- | ------------ | ------ |
| TASK-657 | e2e for the standalone binaries and the pipeline CLI | testing      | todo   |
| TASK-658 | The OpenAI-compatible gateway inside the daemon, wit | architecture | done   |
| TASK-659 | ESLint for the VS Code extension and the GitHub Acti | quality      | done   |
| TASK-660 | Close TASK-578 and make the MCP shutdown test determ | polish       | done   |
| TASK-661 | A real multi-task plan through the three subscriptio | dogfood      | todo   |
