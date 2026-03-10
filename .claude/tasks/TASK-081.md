---
id: TASK-081
title: "Pages: Getting Started + MCP Integration guide"
role: devops
planId: PLAN-009
status: todo
dependencies: [TASK-078]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
A step-by-step getting started guide and MCP integration page for users who want to set up nexusOrchestrator quickly and connect it to Claude Desktop or other MCP clients.

## Files to Read
- `README.md`
- `cmd/nexus-daemon/main.go`
- `cmd/nexus-cli/main.go`
- `internal/adapters/inbound/mcp/server.go`
- `.github/copilot-instructions.md`

## Implementation Steps
1. Create `docs/getting-started.md`:
   ```yaml
   ---
   layout: default
   title: Getting Started
   nav_order: 4
   ---
   ```
   Content:
   - **Prerequisites**: Go 1.24+, CGO_ENABLED=1, C compiler (gcc/clang), at least one LLM provider
   - **Installation**: Clone, build daemon, build CLI
   - **Starting the Daemon**: `./nexus-daemon` with env var config
   - **Submitting Your First Task**: curl example with POST /api/tasks
   - **Checking Task Status**: curl GET /api/tasks/{id}
   - **Using the CLI**: `nexus-cli` commands
   - **Using the Dashboard**: Open http://localhost:9999/ui
   - **Configuration**: Environment variables table (NEXUS_DB_PATH, NEXUS_LISTEN_ADDR, NEXUS_MCP_ADDR)
   - **Provider Setup**: How to configure LM Studio, Ollama, cloud providers

2. Create `docs/mcp-integration.md`:
   ```yaml
   ---
   layout: default
   title: MCP Integration
   nav_order: 5
   ---
   ```
   Content:
   - **What is MCP**: Brief explanation of Model Context Protocol
   - **Claude Desktop Setup**: JSON config snippet for `claude_desktop_config.json`
   - **Available Tools**: Table of all 6 MCP tools with descriptions
   - **Usage Examples**: JSON-RPC 2.0 request/response for each tool
   - **Troubleshooting**: Common issues (daemon not running, port conflicts)

## Acceptance Criteria
- [ ] `docs/getting-started.md` exists with step-by-step guide
- [ ] `docs/mcp-integration.md` exists with Claude Desktop config
- [ ] Curl examples for HTTP API
- [ ] MCP tool usage examples with JSON-RPC format
- [ ] No Go source files modified

## Anti-patterns to Avoid
- NEVER modify any Go source files
- NEVER include placeholder "TODO" content — every section must be complete
