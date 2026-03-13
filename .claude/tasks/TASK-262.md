# TASK-262 — Add DesktopApp/LocalAI/VLLM/TextGenUI to buildProviderFromConfig()

**Plan**: PLAN-040  
**Status**: done

Add `ProviderKindDesktopApp`, `ProviderKindLocalAI`, `ProviderKindVLLM`, `ProviderKindTextGenUI` cases
to `buildProviderFromConfig()` in both entry points. All route through `llm_openaicompat`.
BaseURL is normalized to include `/v1` suffix if missing.
