---
id: TASK-016
title: CLI — queue submit command + sessions subcommand
role: cli
planId: PLAN-002
status: todo
dependencies: [TASK-015]
createdAt: 2026-03-09T12:00:00.000Z
---

## Context

The CLI (`cmd/nexus-cli`) is a remote HTTP client that talks to the nexusOrchestrator daemon at `127.0.0.1:9999`. It currently has `queue list`, `queue get <id>`, `queue cancel <id>`, and `providers` commands — but is **missing the primary use case**: `queue submit`. It also has no commands for managing session history, which is now required for external projects using the writeback workflow.

## Files to Read

- `internal/adapters/inbound/cli/root.go` — all existing commands
- `cmd/nexus-cli/main.go` — entry point
- `internal/adapters/inbound/httpapi/server.go` — API routes to call (especially new ones from TASK-015)

## Implementation Steps

1. **Add `queue submit` command**:
   ```
   nexus-cli queue submit --project <projectPath> [--model <model>] <prompt text...>
   ```
   - `--project` flag: path to the project directory (required; defaults to `$PWD` if omitted)
   - `<prompt text...>` positional args joined with spaces form the prompt
   - Makes `POST /api/tasks` with body `{"projectPath": "<path>", "prompt": "<text>"}`
   - On success: prints `Submitted task <id>`
   - On error: prints to stderr, exits 1
   
   Implementation in `root.go`:
   ```go
   var submitCmd = &cobra.Command{
       Use:   "submit [prompt...]",
       Short: "Submit a new task to the orchestrator",
       RunE:  runSubmit,
   }
   // flag: --project
   submitCmd.Flags().StringP("project", "p", "", "project directory path (default: $PWD)")
   queueCmd.AddCommand(submitCmd)
   ```

2. **Add `sessions` top-level command** with three subcommands:

   ```
   nexus-cli sessions list                     → GET /api/sessions — NOT YET SUPPORTED (returns stub)
   nexus-cli sessions get <projectPath>        → GET /api/sessions/<encoded-path>
   nexus-cli sessions clear <projectPath>      → DELETE /api/sessions/<encoded-path>
   ```
   
   - `sessions get <path>`: prints each message as `[role] content` on separate lines
   - `sessions clear <path>`: prints `Session cleared for <path>` on success
   - `projectPath` must be URL-encoded when placed in the URL path: use `url.PathEscape(projectPath)`

3. **Helper: `baseURL()` function** — reads `NEXUS_ADDR` env var (default `http://127.0.0.1:9999`) so the CLI can point at remote daemons.

4. **Consistent error formatting**: all HTTP errors should print `Error: <status> <message>` to stderr and exit with code 1. Reuse whatever pattern already exists in root.go.

5. **Update `cmd/nexus-cli/main.go`** if needed to ensure `sessionsCmd` is registered on the root command.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `nexus-cli queue submit --help` shows `--project` flag and prompt argument
- [ ] `nexus-cli sessions --help` lists `get`, `clear` subcommands
- [ ] `nexus-cli queue submit --project /tmp/foo "write a hello world"` makes correct POST request (verified with httptest in new CLI test)
- [ ] `nexus-cli sessions clear /tmp/foo` makes correct DELETE request
- [ ] `NEXUS_ADDR` env var overrides the default daemon address

## Anti-patterns to Avoid

- NEVER import `internal/core/services` or any adapter directly — CLI is a pure HTTP client
- NEVER store credentials or tokens — this is a local daemon client
- NEVER use `os.Exit` inside cobra `RunE` — return errors instead; cobra handles the exit
- NEVER skip URL-encoding of project paths in URL segments
