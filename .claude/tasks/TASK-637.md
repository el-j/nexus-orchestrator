---
id: TASK-637
planId: PLAN-077
title: 'Go coverage to 100 %: scanners and LLM adapters'
role: testing
status: done
createdAt: 2026-10-31
---

# TASK-637 — Go coverage to 100 %: scanners and LLM adapters

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

sys_scanner plan_scanner 8, scanner 7, agent_scanner 6, claude_subagents 5; activity_claude 6,
activity_network 4; llm_cli adapter 5, agy_driver 3, llm_openaicompat 5, llm_gemini 3, llm_lmstudio
3, llm_ollama 3. Then raise the CI gate in ci.yml from 96 to 100.

Done: the merged Go profile measured by `scripts/go-coverage.sh` is at 100.0 % (9 299 statements). `cmd/*`: tests for the daemon's clean exit after SIGTERM and the submit tool's working-directory failure; the one-line `os.Exit(run(...))` bodies of `func main` in `cmd/*/main.go` are excluded by the script (documented in its header; the binaries are run by `internal/e2e`). The five MCP tools that encoded JSON by hand now share `jsonResult`. The CI gate in `ci.yml`, `publish.yml` and its guard test in `internal/docscheck` moved from 96 to 100. The gate has only been measured on macOS: if Linux or Windows CI shows a platform-only gap, fix the test, do not lower the gate.
