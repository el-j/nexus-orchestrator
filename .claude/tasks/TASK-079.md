---
id: TASK-079
title: "Pages: Landing page with hero + features"
role: devops
planId: PLAN-009
status: todo
dependencies: [TASK-078]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
The landing page is the first thing visitors see. It must clearly communicate what nexusOrchestrator does, its key features, and how to get started — all in a compelling, well-structured layout.

## Files to Read
- `README.md`
- `docs/_config.yml` (from TASK-078)
- `.github/copilot-instructions.md`

## Implementation Steps
1. Write `docs/index.md` with Jekyll frontmatter:
   ```yaml
   ---
   layout: default
   title: Home
   nav_order: 1
   ---
   ```
2. Content sections:
   - **Hero**: "nexusOrchestrator" title with tagline: "Route AI code-generation tasks to any LLM backend — locally or in the cloud"
   - **What it does**: 3-4 sentences explaining the core value proposition (local AI orchestration, multi-provider routing, session isolation, MCP integration)
   - **Key Features** grid (use markdown with bold headers):
     - Multi-Backend LLM Routing (LM Studio, Ollama, OpenAI, Anthropic)
     - Per-Project Session Memory (SQLite-backed conversation isolation)
     - MCP Server (JSON-RPC 2.0 for Claude Desktop and compatible clients)
     - HTTP REST API (Full task management on `:9999`)
     - Context-Window Guard (Pre-flight token check prevents overflow)
     - Smart Provider Discovery (Auto-detect providers with failover)
     - Command-Aware Routing (Plan vs Execute task classification)
     - Desktop GUI + System Tray (Wails-powered native app)
   - **Quick Start** code block showing daemon build & run
   - **Navigation**: Links to Architecture, API Reference, Getting Started pages
3. Ensure all markdown renders cleanly with just-the-docs theme.

## Acceptance Criteria
- [ ] `docs/index.md` has complete landing page content
- [ ] Hero section with clear tagline
- [ ] At least 8 feature highlights
- [ ] Quick start code block
- [ ] Navigation to other pages
- [ ] No Go source files modified

## Anti-patterns to Avoid
- NEVER modify any Go source files
- NEVER use HTML unless absolutely needed — prefer clean markdown
