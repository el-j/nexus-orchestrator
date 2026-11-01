---
id: TASK-637
planId: PLAN-077
title: 'Go coverage to 100 %: scanners and LLM adapters'
role: testing
status: todo
createdAt: 2026-10-31
---

# TASK-637 — Go coverage to 100 %: scanners and LLM adapters

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

sys_scanner plan_scanner 8, scanner 7, agent_scanner 6, claude_subagents 5; activity_claude 6,
activity_network 4; llm_cli adapter 5, agy_driver 3, llm_openaicompat 5, llm_gemini 3, llm_lmstudio
3, llm_ollama 3. Then raise the CI gate in ci.yml from 96 to 100.
