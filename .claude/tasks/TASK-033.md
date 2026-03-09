---
id: TASK-033
title: cmd/nexus-submit — feed TASK-NNN.md files to daemon via HTTP
role: devops
planId: PLAN-003
status: todo
dependencies: [TASK-032]
createdAt: 2026-03-09T15:00:00.000Z
---

## Context

To dogfood PLAN-002, we need a way to submit a `.claude/tasks/TASK-NNN.md` file to the running nexus daemon as a code-generation task. `nexus-submit` is a standalone Go CLI binary that reads a task file, constructs a properly-shaped POST request, and returns the assigned task ID.

This is the primary tool for the dogfood loop described in PLAN-003.

## Files to Create

- `cmd/nexus-submit/main.go`

## Files to Read

- `internal/core/domain/task.go` — Task struct (after TASK-032, fields have json tags)
- `internal/adapters/inbound/httpapi/server.go` — POST /api/tasks format
- `.claude/tasks/TASK-013.md` — example input file format (YAML frontmatter + body)

## Implementation: cmd/nexus-submit/main.go

```go
package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "net/http"
    "os"
    "strings"
    "time"
)

func main() {
    var (
        taskFile = flag.String("task-file", "", "path to .claude/tasks/TASK-NNN.md (required)")
        project  = flag.String("project", "", "project root path (default: $PWD)")
        target   = flag.String("target", "", "relative target file path for output (e.g. internal/foo/bar.go)")
        context  = flag.String("context", "", "comma-separated relative file paths for context")
        addr     = flag.String("addr", getEnv("NEXUS_ADDR", "http://127.0.0.1:9999"), "daemon base URL")
        wait     = flag.Bool("wait", false, "poll until task completes and print result")
        timeout  = flag.Duration("timeout", 5*time.Minute, "max wait time when --wait is set")
    )
    flag.Parse()

    if *taskFile == "" {
        fmt.Fprintln(os.Stderr, "error: --task-file is required")
        flag.Usage()
        os.Exit(1)
    }

    // Resolve project path
    projectPath := *project
    if projectPath == "" {
        var err error
        projectPath, err = os.Getwd()
        if err != nil {
            fmt.Fprintln(os.Stderr, "error: get cwd:", err)
            os.Exit(1)
        }
    }

    // Read task file
    content, err := os.ReadFile(*taskFile)
    if err != nil {
        fmt.Fprintln(os.Stderr, "error: read task file:", err)
        os.Exit(1)
    }

    // Parse optional context files
    var contextFiles []string
    if *context != "" {
        for _, f := range strings.Split(*context, ",") {
            if f = strings.TrimSpace(f); f != "" {
                contextFiles = append(contextFiles, f)
            }
        }
    }

    // Build request body (matches domain.Task json tags after TASK-032)
    body := map[string]interface{}{
        "projectPath":  projectPath,
        "targetFile":   *target,
        "instruction":  string(content),
        "contextFiles": contextFiles,
    }
    reqJSON, err := json.Marshal(body)
    if err != nil {
        fmt.Fprintln(os.Stderr, "error: marshal request:", err)
        os.Exit(1)
    }

    // POST to daemon
    resp, err := http.Post(*addr+"/api/tasks", "application/json", bytes.NewReader(reqJSON))
    if err != nil {
        fmt.Fprintln(os.Stderr, "error: POST /api/tasks:", err)
        os.Exit(1)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        fmt.Fprintf(os.Stderr, "error: daemon returned %d\n", resp.StatusCode)
        os.Exit(1)
    }

    var result struct {
        TaskID string `json:"task_id"`
        Status string `json:"status"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        fmt.Fprintln(os.Stderr, "error: decode response:", err)
        os.Exit(1)
    }

    fmt.Printf("submitted: task_id=%s status=%s\n", result.TaskID, result.Status)
    fmt.Printf("track: %s/api/tasks/%s\n", *addr, result.TaskID)
    fmt.Printf("ui: %s/ui\n", *addr)

    if *wait {
        waitForCompletion(*addr, result.TaskID, *timeout)
    }
}

func waitForCompletion(addr, taskID string, timeout time.Duration) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        time.Sleep(3 * time.Second)
        resp, err := http.Get(addr + "/api/tasks/" + taskID)
        if err != nil {
            fmt.Fprintln(os.Stderr, "poll error:", err)
            continue
        }
        var t struct {
            Status string `json:"status"`
            Logs   string `json:"logs"`
        }
        _ = json.NewDecoder(resp.Body).Decode(&t)
        resp.Body.Close()

        fmt.Printf("  [%s] status=%s\n", time.Now().Format("15:04:05"), t.Status)
        if t.Status == "COMPLETED" || t.Status == "FAILED" {
            fmt.Println("logs:", t.Logs)
            if t.Status == "FAILED" {
                os.Exit(1)
            }
            return
        }
    }
    fmt.Fprintln(os.Stderr, "timed out waiting for completion")
    os.Exit(1)
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

## Usage Examples

```bash
# Submit TASK-013 to implement orchestrator hardening
nexus-submit \
  --task-file .claude/tasks/TASK-013.md \
  --project /path/to/nexusOrchestrator \
  --target internal/core/services/orchestrator.go \
  --context internal/core/services/orchestrator.go,internal/core/ports/ports.go \
  --wait

# Submit TASK-015 (HTTP API history endpoint)
nexus-submit \
  --task-file .claude/tasks/TASK-015.md \
  --target internal/adapters/inbound/httpapi/server.go \
  --context internal/adapters/inbound/httpapi/server.go,internal/core/ports/ports.go

# Submit with custom daemon address
NEXUS_ADDR=http://192.168.1.10:9999 nexus-submit --task-file .claude/tasks/TASK-013.md ...
```

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-submit/...` exits 0
- [ ] `nexus-submit --help` prints usage with all flags
- [ ] Running without `--task-file` exits 1 with error
- [ ] Submitting an existing `.claude/tasks/TASK-013.md` to a running daemon returns a task ID
- [ ] `--wait` flag polls until status is COMPLETED or FAILED
- [ ] `--context` flag correctly parses comma-separated file list
- [ ] `NEXUS_ADDR` env var overrides default daemon address

## Anti-patterns to Avoid

- NEVER import anything from `nexus-ai/internal/` — nexus-submit is a pure HTTP client
- NEVER require the daemon source code to build — use only stdlib + http
- NEVER hardcode project paths in the binary
- NEVER ignore the HTTP status code from POST /api/tasks
