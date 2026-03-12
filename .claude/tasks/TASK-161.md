---
id: TASK-161
title: sys_scanner outbound adapter — port, CLI, and process probes
role: backend
planId: PLAN-023
status: todo
dependencies: [TASK-160]
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
This is the core scanning engine that detects AI providers installed or running on the local system. It implements the `SystemScanner` port using three probe strategies: TCP+HTTP port probing for API servers, `exec.LookPath` for CLI tools, and process enumeration for desktop apps. It runs as a fan-out with a concurrency limiter to avoid overwhelming the system.

## Files to Read
- `internal/core/ports/ports.go` — `SystemScanner` interface from TASK-160
- `internal/core/domain/provider.go` — `DiscoveredProvider`, `DiscoveryMethod`, `DiscoveryStatus`, `ProviderKind`
- `internal/adapters/outbound/llm_lmstudio/adapter.go` — reference for how Ping() probes work
- `internal/adapters/outbound/llm_ollama/adapter.go` — reference for Ollama API format

## Implementation Steps

1. Create `internal/adapters/outbound/sys_scanner/scanner.go` with struct:
   ```go
   type Scanner struct {
       httpClient *http.Client  // short timeout: 2s for probes
   }
   func New() *Scanner
   ```

2. Implement `Scan(ctx context.Context) ([]domain.DiscoveredProvider, error)`:
   - Launch all probe categories concurrently using an `errgroup.Group` with limit 8
   - Collect results into a thread-safe slice
   - Return deduplicated list sorted by name

3. **Port probes** — for each well-known endpoint, do TCP dial (2s timeout) then HTTP GET:
   | Name | Port | Health endpoint | Kind |
   |------|------|----------------|------|
   | LM Studio | 1234 | `GET /v1/models` | lmstudio |
   | Ollama | 11434 | `GET /api/tags` | ollama |
   | LocalAI | 8080 | `GET /v1/models` | localai |
   | vLLM | 8000 | `GET /v1/models` | vllm |
   | text-generation-webui | 5000 | `GET /v1/models` | textgenui |
   | Antigravity | 4315 | `GET /v1/models` (TBD) | desktopapp |
   
   For each reachable endpoint, try to parse model list from response. Set `Status: DiscoveryStatusReachable`.

4. **CLI probes** — use `exec.LookPath` to detect installed AI CLI tools:
   | Binary | Name | Kind |
   |--------|------|------|
   | `claude` | Claude CLI | cli |
   | `ollama` | Ollama CLI | ollama |
   | `lms` | LM Studio CLI | lmstudio |
   | `aichat` | aichat | cli |
   | `llm` | llm (Python) | cli |
   
   Set `CLIPath` to resolved path. Set `Status: DiscoveryStatusInstalled`.

5. **Process probes** — enumerate running processes to detect desktop AI apps:
   - macOS: use `exec.Command("pgrep", "-lf", pattern)` for each known process name
   - Known patterns: `Claude`, `Antigravity`, `ChatGPT`, `Copilot`
   - Set `ProcessName` and `Status: DiscoveryStatusRunning`
   - Use build tags or runtime.GOOS switch for platform-specific process enumeration

6. Deduplicate: if the same provider is found by both port probe AND CLI probe, merge into one entry with the highest status (reachable > running > installed).

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] Scanner probes at least 6 well-known ports
- [ ] Scanner detects at least 5 CLI tools via `exec.LookPath`
- [ ] Scanner detects running desktop AI apps via process enumeration
- [ ] All probes run concurrently with a semaphore limit of 8
- [ ] Individual probe failures don't crash the scan — errors are logged, scan continues
- [ ] Scan completes within 5s even if all probes time out (2s per probe)

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER block on a single unreachable endpoint — always use context + timeout
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER use goroutines inside `internal/core/services/` — the scanner adapter owns its own goroutines
- NEVER hard-fail the scan if one probe errors — collect what works, log failures
