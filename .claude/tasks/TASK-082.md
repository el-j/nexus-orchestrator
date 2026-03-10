---
id: TASK-082
title: "GitHub Actions workflow for Pages deployment"
role: devops
planId: PLAN-009
status: todo
dependencies: [TASK-079, TASK-080, TASK-081]
createdAt: 2026-03-10T15:00:00.000Z
---

## Context
A GitHub Actions workflow is needed to automatically build and deploy the Jekyll-based documentation site to GitHub Pages whenever changes are pushed to the main branch.

## Files to Read
- `docs/_config.yml` (from TASK-078)
- `docs/Gemfile` (from TASK-078)

## Implementation Steps
1. Create `.github/workflows/pages.yml`:
   ```yaml
   name: Deploy GitHub Pages
   
   on:
     push:
       branches: ["main"]
       paths: ["docs/**"]
     workflow_dispatch:
   
   permissions:
     contents: read
     pages: write
     id-token: write
   
   concurrency:
     group: "pages"
     cancel-in-progress: false
   
   jobs:
     build:
       runs-on: ubuntu-latest
       steps:
         - name: Checkout
           uses: actions/checkout@v4
         - name: Setup Pages
           uses: actions/configure-pages@v5
         - name: Build with Jekyll
           uses: actions/jekyll-build-pages@v1
           with:
             source: ./docs
             destination: ./_site
         - name: Upload artifact
           uses: actions/upload-pages-artifact@v3
   
     deploy:
       environment:
         name: github-pages
         url: ${{ steps.deployment.outputs.page_url }}
       runs-on: ubuntu-latest
       needs: build
       steps:
         - name: Deploy to GitHub Pages
           id: deployment
           uses: actions/deploy-pages@v4
   ```
2. Verify the workflow syntax is valid YAML.
3. Ensure the `source: ./docs` path matches the actual docs directory.

## Acceptance Criteria
- [ ] `.github/workflows/pages.yml` exists with correct workflow
- [ ] Workflow triggers on push to main with docs/** changes
- [ ] Workflow also has manual trigger (workflow_dispatch)
- [ ] Correct permissions for GitHub Pages deployment
- [ ] Uses official GitHub Actions for Jekyll build and Pages deployment
- [ ] No Go source files modified

## Anti-patterns to Avoid
- NEVER modify any Go source files
- NEVER use third-party actions when official GitHub ones exist
