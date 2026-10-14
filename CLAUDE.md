# nexus-orchestrator

Go daemon (plus Wails desktop app, Vue frontend, VS Code extension and GitHub Action) that queues
coding tasks, routes them to LLM providers, verifies the result and keeps per-project knowledge and
history. This repository is developed with itself: use the orchestrator for the workflow below.

## Use nexus first

The daemon must be running (`make dogfood` starts it and loads this file; `make status` checks it).
Project MCP server `nexus` is declared in `.mcp.json` (http://localhost:63988/mcp).

1. Call `get_onboarding_context` for this project path before reading code, then
   `get_focused_context` / `search_knowledge` for the area you touch instead of scanning the tree.
2. Plans live in `.claude/plans/PLAN-NNN.md`, tasks in `.claude/tasks/TASK-NNN.md`, counters in
   `.claude/orchestrator.json`. Write the plan and its tasks before code; one task = one small change.
3. Hand code-generation tasks to the orchestrator with `submit_task` (set `verificationCommand`),
   or `create_draft` then `promote_task`. `/execute-via-nexus` pushes a whole plan.
4. Record what was learned with `ingest_knowledge` (kind `learning`) so the next session starts with it.

## File map (what is where)

- `internal/core/domain`, `internal/core/ports`, `internal/core/services` — pure core, no framework imports.
- `internal/adapters/inbound/{httpapi,mcp,cli}` and `internal/adapters/outbound/*` — adapters.
- `internal/bootstrap` — wiring shared by the daemon and the desktop app; `cmd/*` — binaries.
- `internal/e2e` — black-box tests against the real daemon binaries (`-tags e2e`).
- `internal/docscheck/surface_matrix_test.go` — every capability must exist on REST, MCP, CLI, desktop, frontend, extension.
- `frontend`, `docs` (Vue sites), `vscode-extension`, `github-action` — each with its own tests and 100% coverage gate.
- `docs/using-nexus.md` how to run it; `docs/architecture.md`, `docs/api-reference.md` reference.

## Rules (enforced where noted)

- Write the failing test first, then the code. Black-box behaviour goes in `internal/e2e` or `frontend/e2e`.
- Small files: at most 500 lines (`scripts/check-file-size.sh`, CI). Split instead of growing.
- Document everything exported: Go doc comments (revive `exported`), TSDoc and a description comment on every Vue component (`npm run check:docs`).
- Do not widen an interface without updating the surface matrix, the API reference and the changelog.
- Mutation testing is local only (`make mutation`); never add it to a workflow.

## Commands

- `make` lists targets; `make test`, `make nice` (format, vet, lint, type-check), `make build`.
- Go needs `CGO_CFLAGS=-DSQLITE_ENABLE_FTS5` (the Makefile sets it); Node >= 22 (`.nvmrc`).
- Coverage: `scripts/go-coverage.sh` (merged Go coverage, CI gate in `.github/workflows/ci.yml`).
