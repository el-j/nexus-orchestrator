# TASK-289 — Go tests for PLAN-044 components

**Plan:** PLAN-044  
**Status:** TODO  
**Layer:** Go · tests  
**Depends on:** TASK-278, TASK-280  

## Objective

Unit and integration tests for the two new Go layers: agent scanner probes and orchestrator delegation methods.

## `internal/adapters/outbound/sys_scanner/agent_scanner_test.go`

### Test: `probeVSCodeExtensions` with fake extension dir

```go
func TestProbeVSCodeExtensions(t *testing.T) {
    dir := t.TempDir()
    // Create fake extension directories
    for _, d := range []string{
        "saoudrizwan.claude-dev-1.2.3",
        "continue.continue-0.9.0",
        "unknown.extension-1.0.0",
    } {
        os.MkdirAll(filepath.Join(dir, d), 0755)
    }
    // Inject test dir via a package-level var or test hook
    agents := probeVSCodeExtensionsDir(context.Background(), dir)
    if len(agents) != 2 {
        t.Errorf("expected 2 agents, got %d", len(agents))
    }
    names := map[string]bool{}
    for _, a := range agents { names[a.Name] = true }
    if !names["Cline"] || !names["Continue"] {
        t.Errorf("expected Cline and Continue, got %v", names)
    }
}
```

The internal `probeVSCodeExtensionsDir(ctx, dir string)` helper takes the directory as a parameter so it is testable without touching `~/.vscode/extensions`. The production `probeVSCodeExtensions(ctx)` calls it with the real path.

### Test: `probeMCPPorts` with local test server

```go
func TestProbeMCPPorts(t *testing.T) {
    // Start a mock MCP server responding to initialize
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"test-mcp","version":"1.0"},"capabilities":{}}}`)
    }))
    defer srv.Close()
    port := srv.Listener.Addr().(*net.TCPAddr).Port
    // Inject test port
    agents := probeMCPPortList(context.Background(), s, []int{port})
    if len(agents) != 1 || agents[0].Name != "test-mcp" {
        t.Errorf("expected test-mcp agent, got %v", agents)
    }
}
```

Again uses an internal helper `probeMCPPortList(ctx, scanner, ports)`.

### Test: `probeProcessFlags` skipped on Windows

```go
func TestProbeProcessFlagsSkippedOnWindows(t *testing.T) {
    if runtime.GOOS == "windows" {
        t.Skip("pgrep not available on windows")
    }
    // Just verify it returns []DiscoveredAgent without panic on the current platform
    agents, err := probeProcessFlags(context.Background())
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    _ = agents
}
```

## `internal/core/services/orchestrator_delegate_test.go`

Uses the existing test helper pattern (in-memory SQLite + aiSessionTestLLM stub).

### Test: `DelegateToNexus` happy path

```go
func TestDelegateToNexus_Success(t *testing.T) {
    // Setup: register a session
    // Call DelegateToNexus
    // Assert: instruction contains "nexusOrchestrator coordination"
    // Assert: session.DelegatedToNexus == true
    // Assert: session.DelegationTimestamp != nil
}
```

### Test: `DelegateToNexus` nonexistent session

```go
func TestDelegateToNexus_NotFound(t *testing.T) {
    _, err := orch.DelegateToNexus(ctx, "nonexistent-id")
    if !errors.Is(err, domain.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

### Test: `GetDiscoveredAgents` with mock scanner

```go
type mockAgentScanner struct{ agents []domain.DiscoveredAgent }
func (m *mockAgentScanner) ScanAgents(_ context.Context) ([]domain.DiscoveredAgent, error) {
    return m.agents, nil
}

func TestGetDiscoveredAgents_WithScanner(t *testing.T) {
    // Set mock scanner returning 2 agents
    // Call GetDiscoveredAgents
    // Assert 2 agents returned
    // Assert both upserted into repo (list from repo also returns 2)
}
```

### Test: `GetDiscoveredAgents` with nil scanner (repo fallback)

```go
func TestGetDiscoveredAgents_NilScannerFallback(t *testing.T) {
    // No SetAgentScanner called
    // Pre-seed discoveredAgentRepo with 1 agent
    // Call GetDiscoveredAgents → should return 1 from repo
}
```

## Run command

```sh
CGO_ENABLED=1 go test -race ./internal/adapters/outbound/sys_scanner/... ./internal/core/services/...
```

## Acceptance Criteria

- All 6 tests pass under `-race`
- No test modifies real filesystem (uses `t.TempDir()` and mocks)
