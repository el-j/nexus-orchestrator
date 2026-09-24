---
id: PLAN-071
title: 'Self-Healing Execution Gates & Role-Driven Gateway Architecture'
goal: 'Implement deterministic execution gating and turn-based self-healing in Nexus Orchestrator, replacing blind single-shot completion with ground-truth verification (compilers, linters, tests) and automated error correction.'
status: completed
createdAt: 2026-09-24T15:58:00Z
completedAt: 2026-09-24T16:10:00Z
---

# PLAN-071 — Self-Healing Execution Gates & Role-Driven Gateway Architecture

## Background

Nexus Orchestrator previously operated on a single-shot execution model:
$$\text{Task Prompt} + \text{Context Files} \longrightarrow \text{LLM Call} \longrightarrow \text{Write TargetFile}$$

Code was written to disk and immediately marked `COMPLETED` without verifying whether it compiled, passed tests, or satisfied syntax rules. Rather than adding complex, hallucination-prone neural scoring engines (such as OpenJev), this plan introduces **deterministic execution gates**:

1. Execution of real verification commands (`go test`, `npm test`, `npx tsc`, `ruff check`, etc.) in the project directory after writing generated code.
2. If verification fails, Nexus automatically captures the error output, enters a turn-based correction loop (up to `MaxCorrectionTurns`), and asks the model to repair its own mistakes with the exact compiler/test trace.
3. Integration with role-driven model hints and off-the-shelf OpenAI-compatible gateways (LiteLLM / OpenRouter).

## Architecture & Layers

```
internal/core/domain/      -> Task { VerificationCommand, MaxCorrectionTurns, VerificationOutput }
internal/core/ports/       -> CommandRunner { Run(ctx, dir, cmd) (output, error) }
internal/adapters/outbound/ -> cmd_runner (os/exec with context timeout & stderr/stdout capture)
internal/core/services/    -> execution_engine (writeAndVerifyTaskOutput self-healing loop)
internal/adapters/inbound/ -> HTTP API, MCP tools, CLI flag support
```

## Tasks

- **TASK-554**: Domain & Ports Extension (Task fields + CommandRunner interface) — DONE
- **TASK-555**: SQLite Persistence & Additive Migrations (tasks table schema update + queries) — DONE
- **TASK-556**: Outbound Command Runner Adapter (`cmd_runner` package + tests) — DONE
- **TASK-557**: Execution Engine Self-Healing Loop (`writeAndVerifyTaskOutput` in `execution_engine.go` + tests) — DONE
- **TASK-558**: Inbound Adapters & Wiring (MCP, HTTP, CLI, main entry points + full test validation) — DONE

## Validation

- `go vet ./...`: clean (exited 0)
- `CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5" go test -race -count=1 ./...`: 100% passing across all 15+ packages
