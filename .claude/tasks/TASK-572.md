---
id: TASK-572
planId: PLAN-073
title: 'Contract checks of the streaming adapters against the real services'
role: polish
status: done
createdAt: 2026-10-15
---

# TASK-572 — Contract checks of the streaming adapters against the real services

Anthropic, Gemini and LM Studio streaming is tested against fake servers only. Add opt-in tests (skipped without NEXUS_TEST_ANTHROPIC_KEY etc.) that stream one tiny prompt for real, and run them manually before a release.
