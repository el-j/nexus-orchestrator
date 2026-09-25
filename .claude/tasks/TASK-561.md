---
id: TASK-561
planId: PLAN-072
title: 'Native Google Gemini LLM Outbound Adapter'
role: backend
status: done
createdAt: 2026-09-25T10:05:00Z
completedAt: 2026-09-25T10:23:35Z
---

# TASK-561 — Native Google Gemini LLM Outbound Adapter

## Context

Google Gemini models (`gemini-2.5-pro`, `gemini-2.5-flash`, etc.) are frontier models with 1M+ token context windows and low latency. Currently, Nexus only supports Anthropic, OpenAI, LM Studio, and Ollama natively. Gemini requires an outbound adapter that satisfies `ports.LLMClient`.

## Work Required

1. Create `internal/adapters/outbound/llm_gemini/adapter.go`:
   - Implement `ports.LLMClient` (`ProviderName`, `GenerateCode`, `Chat`, `ActiveModel`, `Models`, `Liveness`, `ContextLimit`).
   - Use Google AI Studio REST API / Google GenAI endpoint with `GEMINI_API_KEY`.
   - Support standard model list: `gemini-2.5-pro`, `gemini-2.5-flash`, `gemini-1.5-pro`, `gemini-1.5-flash`.
2. Register Gemini provider in `internal/bootstrap/providers.go` and `cmd/nexus-daemon/main.go`.
3. Add unit tests in `internal/adapters/outbound/llm_gemini/adapter_test.go` using `httptest.NewServer`.
