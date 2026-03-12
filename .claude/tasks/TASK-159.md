---
id: TASK-159
title: Domain types for system-wide discovery and log streaming
role: architecture
planId: PLAN-023
status: todo
dependencies: []
createdAt: 2026-03-11T21:00:00.000Z
---

## Context
nexusOrchestrator currently only knows about providers explicitly wired at startup. Users have Claude CLI, VS Code Copilot, Antigravity, and other AI tools installed locally that the app cannot see. We need domain types to represent providers discovered by scanning the system (port probes, CLI lookups, process enumeration). Additionally, the user wants an in-app log console instead of a separate terminal window, requiring a `LogEntry` domain type.

## Files to Read
- `internal/core/domain/provider.go` — existing `ProviderConfig`, `ProviderKind`, `ProviderInfo`
- `internal/core/domain/task.go` — domain pattern reference
- `internal/core/domain/session.go` — domain pattern reference

## Implementation Steps

1. In `internal/core/domain/provider.go`, add new `ProviderKind` constants:
   ```go
   ProviderKindLocalAI    ProviderKind = "localai"
   ProviderKindVLLM       ProviderKind = "vllm"
   ProviderKindTextGenUI  ProviderKind = "textgenui"
   ProviderKindCLI        ProviderKind = "cli"
   ProviderKindDesktopApp ProviderKind = "desktopapp"
   ```

2. Add `DiscoveryMethod` enum:
   ```go
   type DiscoveryMethod string
   const (
       DiscoveryMethodPort    DiscoveryMethod = "port"
       DiscoveryMethodCLI     DiscoveryMethod = "cli"
       DiscoveryMethodProcess DiscoveryMethod = "process"
   )
   ```

3. Add `DiscoveryStatus` enum:
   ```go
   type DiscoveryStatus string
   const (
       DiscoveryStatusReachable DiscoveryStatus = "reachable"  // API responding
       DiscoveryStatusInstalled DiscoveryStatus = "installed"  // CLI binary found
       DiscoveryStatusRunning   DiscoveryStatus = "running"    // Process detected
   )
   ```

4. Add `DiscoveredProvider` struct:
   ```go
   type DiscoveredProvider struct {
       ID          string          `json:"id"`
       Name        string          `json:"name"`
       Kind        ProviderKind    `json:"kind"`
       Method      DiscoveryMethod `json:"method"`
       Status      DiscoveryStatus `json:"status"`
       BaseURL     string          `json:"baseUrl,omitempty"`
       CLIPath     string          `json:"cliPath,omitempty"`
       ProcessName string          `json:"processName,omitempty"`
       Models      []string        `json:"models,omitempty"`
       LastSeen    time.Time       `json:"lastSeen"`
   }
   ```

5. Create `internal/core/domain/log_entry.go` with `LogEntry` struct:
   ```go
   type LogLevel string
   const (
       LogLevelInfo  LogLevel = "info"
       LogLevelWarn  LogLevel = "warn"
       LogLevelError LogLevel = "error"
       LogLevelDebug LogLevel = "debug"
   )

   type LogEntry struct {
       Timestamp time.Time `json:"timestamp"`
       Level     LogLevel  `json:"level"`
       Source    string    `json:"source"`
       Message   string    `json:"message"`
   }
   ```

## Acceptance Criteria
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `DiscoveredProvider` struct has all fields: ID, Name, Kind, Method, Status, BaseURL, CLIPath, ProcessName, Models, LastSeen
- [ ] `LogEntry` struct has all fields: Timestamp, Level, Source, Message
- [ ] New `ProviderKind` values are valid constants alongside existing ones

## Anti-patterns to Avoid
- NEVER import adapters from core services (hexagonal dependency rule)
- NEVER use goroutines inside `internal/core/services/`
- NEVER skip `fmt.Errorf("package: operation: %w", err)` error wrapping
- NEVER add framework imports to domain types — they must be pure Go
