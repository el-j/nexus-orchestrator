---
id: TASK-078
title: "Scaffold docs/ with Jekyll config + theme"
role: devops
planId: PLAN-009
status: todo
dependencies: []
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
Create the GitHub Pages documentation site scaffold using Jekyll with the "just-the-docs" theme. This provides a clean, professional, and searchable documentation site that builds automatically via GitHub Pages.

## Files to Read
- `README.md` (for content to reference)
- `.github/copilot-instructions.md` (for project details)

## Implementation Steps
1. Create `docs/` directory at the project root.
2. Create `docs/_config.yml` with:
   ```yaml
   title: nexusOrchestrator
   description: A local AI orchestrator that routes code-generation tasks to LM Studio, Ollama, OpenAI, Anthropic — with per-project session memory, MCP server, and a full dashboard
   baseurl: "/nexusOrchestrator"
   url: ""
   remote_theme: just-the-docs/just-the-docs
   plugins:
     - jekyll-remote-theme
   color_scheme: dark
   search_enabled: true
   nav_sort: case_insensitive
   aux_links:
     "GitHub":
       - "https://github.com/YOUR_USERNAME/nexusOrchestrator"
   footer_content: "nexusOrchestrator — Local AI task orchestration"
   ```
3. Create `docs/Gemfile`:
   ```ruby
   source "https://rubygems.org"
   gem "jekyll", "~> 4.3"
   gem "just-the-docs"
   ```
4. Create `docs/.gitignore` with `_site/`, `.jekyll-cache/`, `.jekyll-metadata`, `Gemfile.lock`, `.bundle/`.
5. Create `docs/assets/css/custom.scss` with custom overrides for branding colors and code block styling.
6. Create a placeholder `docs/index.md` with just the frontmatter (content filled in TASK-079).

## Acceptance Criteria
- [ ] `docs/_config.yml` exists with just-the-docs theme
- [ ] `docs/Gemfile` exists with correct gems
- [ ] `docs/.gitignore` excludes build artifacts
- [ ] `docs/assets/css/custom.scss` exists with basic overrides
- [ ] `docs/index.md` exists as placeholder
- [ ] No Go source files modified

## Anti-patterns to Avoid
- NEVER modify any Go source files in this task
- NEVER use npm/npx — Jekyll is Ruby-based
- NEVER commit Gemfile.lock (it's gitignored)
