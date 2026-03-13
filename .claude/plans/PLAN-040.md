# PLAN-040 — Google Antigravity LLM Provider Integration

**Status**: active  
**Created**: 2026-03-13

## Problem Statement

The user has a Google Antigravity account. Antigravity is a Google-backed AI desktop app that:
- Runs locally on port `4315` and exposes an **OpenAI-compatible** REST API at `http://127.0.0.1:4315/v1`
- Requires a Google account for activation; the local app handles auth transparently
- Is already detected by the `sys_scanner` (port target + process pattern) as `ProviderKindDesktopApp`

**Gap**: The orchestrator already *discovers* Antigravity but can't *use* it because:
1. `buildProviderFromConfig` has no `ProviderKindDesktopApp` case → promotion fails silently
2. `buildProviders()` doesn't include Antigravity → it's never registered on startup
3. `ProviderKindLocalAI`, `ProviderKindVLLM`, `ProviderKindTextGenUI` have the same gap (also OpenAI-compat)

## Solution

All of `desktopapp / localai / vllm / textgenui` expose OpenAI-compatible APIs.
Wire them through the existing `llm_openaicompat` adapter.

| Task | What |
|------|------|
| TASK-261 | Wire Antigravity in `buildProviders()` (both entry points) |
| TASK-262 | Add `ProviderKindDesktopApp` + co. to `buildProviderFromConfig()` (both entry points) |
| TASK-263 | Document `NEXUS_ANTIGRAVITY_URL` in copilot-instructions.md |

## Env Var
- `NEXUS_ANTIGRAVITY_URL` — default `http://127.0.0.1:4315/v1`

## Success Criteria
- `go vet ./...` + `go build` pass
- `buildProviderFromConfig({Kind: "desktopapp", BaseURL: "http://127.0.0.1:4315"})` returns an `llm_openaicompat` adapter
- Antigravity appears in `buildProviders()` always (like LM Studio / Ollama)
