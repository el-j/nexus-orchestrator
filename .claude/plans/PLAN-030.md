# PLAN-030: Comprehensive Hardening

**Status:** Completed  
**Completed:** 2026-03-12T16:50:00.000Z

## Tasks

| ID | Title | Role | Status |
|----|-------|------|--------|
| TASK-208 | Fix TaskEvent type safety - AISessionStatus cast | architecture | done |
| TASK-209 | Fix error handling consistency in core services | backend | done |
| TASK-210 | Add constructor nil-validation in NewOrchestrator | backend | done |
| TASK-211 | Add HeartbeatAISession to Wails + CLI PromoteProvider | backend | done |
| TASK-212 | Fix HTTP API error response format consistency | api | done |
| TASK-213 | Fix frontend circular dep + incomplete components | backend | done |
| TASK-214 | Fix VSCode ext memory leak + hardcoded URLs | backend | done |
| TASK-215 | Fix race conditions in VSCode session monitor | backend | done |
| TASK-216 | Decompose processNext god method | backend | done |
| TASK-217 | Fix silent error handling in frontend composables | backend | done |
| TASK-218 | Add LLM adapter unit tests - all 4 providers | qa | done |
| TASK-219 | Add provider_config_repo + Wails binding tests | qa | done |
| TASK-220 | Security hardening - CSP, permissions, XSS | backend | done |
| TASK-221 | Add Vue global error boundary + rejection handler | backend | done |
| TASK-222 | Add GitHub Action tests - daemon, installer, index | qa | done |
| TASK-223 | Frontend accessibility audit fixes | backend | done |
| TASK-224 | Add golangci-lint + eslint to CI pipeline | devops | done |
| TASK-225 | Fix discovery silent error + configurable constants | backend | done |

## Summary

Comprehensive hardening sprint addressing type safety, error handling, test coverage, security, accessibility, and CI infrastructure.

**Key changes:**
- `internal/core/domain/`: TaskEvent AISessionStatus cast made type-safe
- `internal/core/services/orchestrator.go`: `processNext` decomposed into 4 helpers; constructor nil-validation added; error wrapping `fmt.Errorf("orchestrator: ...: %w")` applied consistently
- `internal/adapters/outbound/`: Unit tests added for all 4 LLM providers (lmstudio, ollama, anthropic, openaicompat) and provider_config_repo
- `internal/adapters/inbound/httpapi/`: Unified JSON error response format `{"error":"..."}` with correct HTTP status codes
- `vscode-extension/src/`: Memory leaks fixed (disposable arrays, event subscriptions), hardcoded localhost URLs removed, session monitor mutex added
- `frontend/src/`: Circular dependency resolved, silent `catch {}` blocks replaced with `console.error`, Vue global error boundary added, accessibility ARIA labels added, CSP nonce support
- `.github/workflows/ci.yml`: golangci-lint + eslint steps added
- `internal/adapters/outbound/sys_scanner/`: Error logging added; configurable discovery constants extracted
