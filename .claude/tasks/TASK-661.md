---
id: TASK-661
planId: PLAN-080
title: 'A real multi-task plan through the three subscription providers'
role: dogfood
status: done
createdAt: 2026-11-24
---

# TASK-661 — A real multi-task plan through the three subscription providers

With the gateway in place, run a plan of at least six real tasks through claude, agy and Copilot (bridge) from the daemon, record routing, quota, WAITING and verification behaviour, and fix what breaks.

Done (2026-11-24, the daemon built from this stack of branches): six real agentic tasks in a fresh Go module, each verified by `go vet` and `go test` of its package. Two ran through the Claude subscription (stack, set), two through agy (ring-buffer queue, token-bucket limiter), two through the VS Code bridge (LRU cache on `vscode-copilot-claude-sonnet-5.5`, a demo program that reads the other packages on `vscode-copilot-gpt-5.3-codex`). All six ended COMPLETED within two minutes, concurrently; the limiter and the demo needed a correction turn (wrong import path, then a read of the real API), which the agent loop and the verification gate handled themselves. A chat sent to `/v1/chat/completions` with `X-Nexus-Project` and the model `Claude (subscription)` answered from the project's stored convention, and the calls appear in the usage ledger of the project (8 calls). No provider hit a limit, so WAITING was not exercised here (it is covered by the fake-claude e2e). Nothing broke; nothing to fix.
