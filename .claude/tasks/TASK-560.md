---
id: TASK-560
planId: PLAN-072
title: 'Token-Budgeted Project Onboarding Handshake (get_onboarding_context)'
role: backend
status: done
createdAt: 2026-09-25T10:05:00Z
completedAt: 2026-09-25T10:20:15Z
---

# TASK-560 — Token-Budgeted Project Onboarding Handshake

## Context

When an agent starts a session or switches projects, injecting massive code dumps degrades model performance and wastes budget. Agents need an atomic, token-budgeted (< 800 tokens) Tier 0/1 summary containing:

- Project root, language, primary tech stack.
- Mandatory verification commands (compiler/test flags).
- Active development state (current plan, active task, locked files).
- Key architectural invariants and recent decisions.

## Work Required

1. **Domain & Service**:
   - Add `GetOnboardingContext(ctx context.Context, projectPath string, maxTokens int) (string, error)` to `ports.BrainService` and `BrainService`.
   - Synthesizer reads active project status, CLAUDE.md / README summary, active plan/task from orchestrator, and formats a markdown payload budgeted to `maxTokens` (default 800).
2. **MCP & HTTP**:
   - Add `get_onboarding_context` MCP tool (parameters: `projectPath`, optional `maxTokens`).
   - Add `GET /api/brain/onboarding?projectPath=<path>&maxTokens=<int>` HTTP endpoint.
3. **Tests**:
   - Unit tests in `brain_service_test.go`, `brain_handlers_test.go`, and `brain_tools_test.go`.
