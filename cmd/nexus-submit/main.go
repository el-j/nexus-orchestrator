// Command nexus-submit submits task files to a running nexusOrchestrator daemon for LLM code generation.
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

var version = "dev"

func main() {
	var (
		taskFile = flag.String("task-file", "", "path to .claude/tasks/TASK-NNN.md (required)")
		project  = flag.String("project", "", "project root path (default: $PWD)")
		target   = flag.String("target", "", "relative target file path for LLM output (e.g. internal/foo/bar.go)")
		context  = flag.String("context", "", "comma-separated relative file paths to include as context")
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

	// Read task file content as the instruction
	content, err := os.ReadFile(*taskFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: read task file:", err)
		os.Exit(1)
	}

	// Parse optional comma-separated context files
	var contextFiles []string
	if *context != "" {
		for _, f := range strings.Split(*context, ",") {
			if f = strings.TrimSpace(f); f != "" {
				contextFiles = append(contextFiles, f)
			}
		}
	}

	// Build request body using camelCase field names (aligns with domain.Task json tags from TASK-032)
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
		fmt.Fprintf(os.Stderr, "error: daemon returned HTTP %d\n", resp.StatusCode)
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
	fmt.Printf("ui:    %s/ui\n", *addr)

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
			if t.Logs != "" {
				fmt.Println("logs:", t.Logs)
			}
			if t.Status == "FAILED" {
				os.Exit(1)
			}
			return
		}
	}
	fmt.Fprintln(os.Stderr, "error: timed out waiting for task completion")
	os.Exit(1)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
