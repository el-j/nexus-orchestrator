---
id: TASK-192
title: "README v0.9.x refresh: feature matrix, discovery, AI sessions, VS Code ext"
status: todo
priority: medium
role: devops
dependencies: none
estimated_effort: S 30min
---

## Goal

Rewrite the README `Features` section and the `Dogfooding` section to accurately document the v0.9.x feature set, removing stale PLAN-002 references and adding entries for the major capabilities shipped since v0.7.

## Context

- Current README `Features` bullet list (lines 9-17) omits: provider discovery scanner, persistent provider config with API-key masking, cloud providers (Anthropic, OpenAI, OpenAI-compat), AI session tracking, backlog/draft task management, SSE events stream, VS Code extension, GitHub Action
- The `Dogfooding` section (lines 112-163) is built around PLAN-002 tasks which are now complete — it should be updated to reference the general developer workflow
- `CHANGELOG.md` already has correct v0.8.0 / v0.9.0 / v0.9.1 entries — the README should cite the current version
- VS Code extension README is in `vscode-extension/README.md` — the main README should cross-link it
- The three-binary table (nexus-daemon, nexus-cli, nexus-submit) is correct — keep it

## Scope

### Files to modify
- `README.md` — Features section, badges, Dogfooding section, Quick Start section headers

## Implementation

1. **Features section** — replace the existing 5-bullet list with a comprehensive feature matrix grouped by area:
   - Core: hexagonal architecture, multi-turn session isolation, SQLite persistence
   - LLM Backends: LM Studio, Ollama, Anthropic, OpenAI, OpenAI-compat (cloud providers)
   - Provider Discovery: automatic port + process scanner for local LLM runtimes
   - Task Management: submit / queue / cancel, backlog/draft workflow, per-task context files, file writeback
   - Interfaces: HTTP API (:63987), MCP server (:63988), Desktop GUI (Wails), System Tray, VS Code Extension, GitHub Action
   - Observability: SSE events stream, AI session tracking, structured logs

2. **Dogfooding section** — replace PLAN-002-specific steps with a generic "use nexusOrchestrator for its own development" workflow: start daemon, submit a task from the `.claude/tasks/` directory using `nexus-submit`, track via `nexus-cli` or MCP

3. **Quick Start** — add a VS Code Extension subsection after MCP Integration, pointing to `vscode-extension/README.md`

4. **Version badge** — verify it points to the correct releases URL

## Acceptance Criteria
- [ ] Features section covers all major v0.9.x capabilities (at least 12 bullet points across 6 groups)
- [ ] No references to PLAN-002 tasks or specific task file names in the main user-facing Dogfooding section
- [ ] VS Code Extension quick-start is present with a link to `vscode-extension/README.md`
- [ ] All links in README are valid (no dead hrefs)
- [ ] `go vet ./...` still passes (doc-only change, but confirm no generated files regressed)
