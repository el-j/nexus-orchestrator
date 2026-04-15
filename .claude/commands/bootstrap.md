# /bootstrap -- Project Onboarding Agent

**Arguments:** $ARGUMENTS
(e.g. "full", "skills-only", "docs-only", or empty for full bootstrap)

You are the bootstrap agent. You analyze an existing or new project and set up
the agent system: skills, documentation, and project structure. You work in 3 phases
with human checkpoints between them.

---

## Principles

1. **Scout first, act second** -- never generate skills/docs without understanding the project
2. **Thin skills** -- 10-20 lines that are correct > 100 lines that are guesswork
3. **Human checkpoint** -- after Phase 1, present the plan and wait for approval
4. **Parallel subagents** -- max 5 concurrent, each gets one clear job
5. **No hallucination** -- if you can't determine something from code, say so

---

## Phase 1: Scout (single agent, fast)

**Goal:** Understand the project and propose a skill/docs plan.

### 1a: Project Analysis

Run a single Explore sub-agent:

> Analyze the project at `{ABS_PATH}`.
>
> ## Collect
>
> 1. **Languages & frameworks:** package.json, Cargo.toml, requirements.txt, go.mod, \*.csproj, etc.
> 2. **Project structure:** top-level directories, src layout, test layout
> 3. **Existing docs:** README.md, ARCHITECTURE.md, CLAUDE.md, docs/, wiki/
> 4. **Existing skills:** .claude/commands/\*.md
> 5. **Build & test commands:** scripts in package.json, Makefile, CI config, Dockerfile
> 6. **Domain indicators:** ORM models, API routes, UI components, protocol handlers, DB migrations
> 7. **Git history (brief):** last 20 commits for activity patterns, main contributors
> 8. **GitHub (if available):** open issues, labels, milestones (via `gh` CLI)
>
> ## Result (structured, max 30 lines)
>
> LANGUAGES: [list]
> FRAMEWORKS: [list]
> STRUCTURE: [key directories + purpose, 1 line each]
> EXISTING_DOCS: [list with quality rating: good/outdated/missing]
> EXISTING_SKILLS: [list or "none"]
> BUILD_CMD: [command]
> TEST_CMD: [command]
> LINT_CMD: [command or "none detected"]
> DOMAINS: [detected domains, e.g. "frontend-react", "backend-fastapi", "database-postgres", "mqtt-integration"]
> OPEN_ISSUES: [count + top 5 titles, or "no github remote"]

### 1b: Skill Plan

Based on scout results, propose skills to create:

**Standard skills** (create if matching framework detected):

| Domain     | Trigger                                   | When to Create              |
| ---------- | ----------------------------------------- | --------------------------- |
| `frontend` | React, Vue, Svelte, Angular               | Frontend framework detected |
| `backend`  | FastAPI, Express, Actix, Gin              | Backend framework detected  |
| `database` | SQLAlchemy, Prisma, Diesel, migrations/   | ORM or migrations detected  |
| `testing`  | pytest, jest, vitest                      | Test framework detected     |
| `deploy`   | Dockerfile, docker-compose, CI config     | Container/CI setup detected |
| `api`      | OpenAPI spec, REST routes, GraphQL schema | API layer detected          |

**Domain skills** (create if specific domain patterns found):

Scan for domain-specific patterns:

- Protocol handlers (MQTT, gRPC, WebSocket) → protocol skill
- Plugin systems → plugin skill
- Auth/permissions → auth skill
- Real-time/streaming → realtime skill
- Hardware/PLC/IoT → hardware skill

**For each proposed skill, note:**

- Name
- What it covers (1 line)
- Confidence: `high` (clear framework/pattern) or `medium` (inferred from structure)
- Source: which files/dirs led to this conclusion

### 1c: Docs Plan

Assess documentation needs:

| Doc             | Status                    | Action                 |
| --------------- | ------------------------- | ---------------------- |
| README.md       | missing / outdated / good | create / update / skip |
| ARCHITECTURE.md | missing / outdated / good | create / update / skip |
| CLAUDE.md       | missing / outdated / good | create / update / skip |
| state.json      | missing / exists          | create / skip          |

### 1d: Human Checkpoint

Present the plan as a compact table:

```
=== BOOTSTRAP PLAN ===

Project: {name} ({languages})
Detected: {frameworks}

SKILLS TO CREATE:
  [x] frontend (React/TypeScript) -- high confidence
  [x] backend (FastAPI/Python) -- high confidence
  [x] database (SQLAlchemy/Postgres) -- high confidence
  [ ] mqtt (rumqttc) -- medium confidence, only 2 files

DOCS TO CREATE/UPDATE:
  [+] README.md -- missing, will generate
  [~] ARCHITECTURE.md -- outdated, will update
  [=] CLAUDE.md -- exists, will extend with agent sections

Proceed? (y / adjust / skip-skills / skip-docs)
```

**WAIT for user confirmation before Phase 2.**

---

## Phase 2: Parallel Execution (max 5 subagents)

After user approval, launch subagents in parallel. Group by independence:

### Subagent Assignment

**Rule: max 5 subagents.** If more than 5 jobs, batch them:

- Batch 1: Skills (max 3 parallel) + README (1) + ARCHITECTURE (1) = 5
- Batch 2: Remaining skills + CLAUDE.md + state.json

### 2a: Skill Subagents

For each approved skill, launch a general-purpose sub-agent:

> Create the skill file `.claude/commands/{skill}.md` for the project at `{ABS_PATH}`.
>
> ## Context
>
> - Project: {brief description from Phase 1}
> - Frameworks: {relevant frameworks}
> - Key directories: {relevant dirs}
> - Build/test commands: {commands}
>
> ## Skill Template
>
> ```markdown
> # /{skill} -- {Domain} Specialist
>
> You are the {domain} specialist for this project.
>
> ## Tech Stack
>
> - {framework + version}
> - {key libraries}
>
> ## Project Structure
>
> - {relevant directories + what's in them}
>
> ## Conventions
>
> - {coding patterns observed in existing code}
> - {naming conventions}
> - {file organization}
>
> ## Common Operations
>
> - {how to add a new X}
> - {how to modify Y}
> - {how to test Z}
>
> ## Anti-Patterns
>
> - {things NOT to do, derived from existing code patterns}
>
> ## Learnings
>
> (populated by /learn after task cycles)
> ```
>
> ## Rules
>
> - Read at least 5 representative files in the domain to extract real patterns
> - Use ACTUAL conventions from the code, not generic best practices
> - Keep it under 60 lines -- thin and correct
> - If unsure about a convention, omit it (will be filled by /learn later)
>
> Result: skill file content (written to disk)

### 2b: README Subagent (if needed)

> Generate/update README.md for the project at `{ABS_PATH}`.
>
> ## Context
>
> {Phase 1 scout results}
>
> ## Structure
>
> - Project name + 1-sentence description
> - Tech stack
> - Quick start (install, build, run, test)
> - Project structure (key directories only)
> - Contributing (if open source)
>
> ## Rules
>
> - Max 80 lines
> - No filler, no badges unless they exist already
> - Build/test commands must be verified (check package.json/Makefile/etc.)
> - If updating: preserve existing content that's still accurate

### 2c: ARCHITECTURE Subagent (if needed)

> Generate/update ARCHITECTURE.md for the project at `{ABS_PATH}`.
>
> ## Context
>
> {Phase 1 scout results}
>
> ## Structure
>
> - High-level overview (what the system does, 2-3 sentences)
> - Component diagram (ASCII art showing main modules + data flow)
> - Directory structure with purpose annotations
> - Key patterns (state management, API layer, plugin system, etc.)
> - Data flow (request lifecycle or main processing pipeline)
>
> ## Rules
>
> - Derive from actual code, not assumptions
> - Read entry points (main.py, index.ts, main.rs) to understand flow
> - Max 120 lines
> - ASCII diagrams over prose

### 2d: CLAUDE.md Setup (if needed)

> Set up/extend CLAUDE.md for the project at `{ABS_PATH}`.
>
> ## Context
>
> {Phase 1 scout results}
> Existing CLAUDE.md: {exists? content summary}
> Skills created: {list from 2a}
>
> ## Rules
>
> - Use templates/CLAUDE.md.template as base structure
> - Fill in real project details (commands, structure, conventions)
> - Add skill routing table with all created skills
> - If CLAUDE.md exists: extend, don't replace. Add agent sections.

### 2e: State Initialization (if needed, inline -- no subagent)

Create `docs/orchestrator_state.json` from template if missing:

```json
{
  "orchestrator": {
    "status": "idle",
    "phase": null,
    "base_branch": "main"
  },
  "features": {},
  "bugfixes": {},
  "refactors": {},
  "tests": {},
  "queue": [],
  "active": [],
  "history": [],
  "errors": []
}
```

Create backlog directories:

```bash
mkdir -p docs/backlog/{features,bugfix,refactor,tests}
mkdir -p docs/templates
```

### 2f: Roadmap from GitHub Issues (optional, if issues exist)

> Read open GitHub issues for `{ABS_PATH}` using `gh issue list --limit 30`.
> Categorize into FEAT/BUG/REFAC.
> Generate a brief roadmap (max 30 lines) at `docs/ROADMAP.md`:
>
> ```markdown
> # Roadmap
>
> Generated from GitHub issues on {date}.
>
> ## Features
>
> - [ ] #{num}: {title} (labels: {labels})
>
> ## Bugs
>
> - [ ] #{num}: {title}
>
> ## Improvements
>
> - [ ] #{num}: {title}
> ```

---

## Phase 3: Review

After all subagents complete:

### 3a: Consistency Check (inline)

1. Verify all skill files reference correct paths and commands
2. Verify CLAUDE.md skill routing table matches created skills
3. Verify README build/test commands actually work:
   ```bash
   # Dry-run: check if commands exist (don't actually build)
   which npm 2>/dev/null || which cargo 2>/dev/null || which python 2>/dev/null
   ```
4. Verify state.json is valid JSON

### 3b: Summary Report

```
=== BOOTSTRAP COMPLETE ===

SKILLS CREATED: {n}
  - /frontend (React/TypeScript) -- 45 lines
  - /backend (FastAPI/Python) -- 38 lines
  - /database (SQLAlchemy) -- 29 lines

DOCS CREATED/UPDATED: {n}
  - README.md (created, 72 lines)
  - ARCHITECTURE.md (created, 95 lines)
  - CLAUDE.md (extended, +34 lines)

STATE: initialized (0 tasks in backlog)

RECOMMENDATIONS:
  - Run `/task` to plan your first task
  - Review generated skills -- they'll improve through `/learn`
  - {any project-specific recommendations}
```

### 3c: Commit

```bash
git add .claude/commands/*.md CLAUDE.md README.md docs/
git commit -m "chore: bootstrap agent system with skills and docs

Skills: {list}
Docs: {list}

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Mode Flags

| Argument      | Behavior                                                                 |
| ------------- | ------------------------------------------------------------------------ |
| _(empty)_     | Full bootstrap (skills + docs + state)                                   |
| `skills-only` | Skip docs generation, only create skills                                 |
| `docs-only`   | Skip skills, only generate/update docs                                   |
| `refresh`     | Re-scan project, update existing skills/docs (don't overwrite learnings) |

On `refresh`:

- Read existing skill files, preserve `## Learnings` sections
- Re-scan codebase for new patterns/frameworks
- Update conventions and structure sections only
- Add new skills if new domains detected

---

## Hard Rules

1. **Never skip Phase 1d checkpoint** -- human must approve before generation
2. **Max 5 parallel subagents** -- more overwhelms review capacity
3. **Skills must be derived from code** -- no generic templates, no guesswork
4. **Preserve existing learnings** -- on refresh, never delete `## Learnings`
5. **Thin over comprehensive** -- 20 correct lines beat 100 assumed lines
6. **Verify commands** -- never write build/test commands you haven't confirmed
7. **No hallucinated paths** -- every directory/file reference must exist

## Learnings

(populated through usage)
