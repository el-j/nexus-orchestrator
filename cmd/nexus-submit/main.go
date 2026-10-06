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

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	var (
		taskFile = flag.String("task-file", "", "path to .claude/tasks/TASK-NNN.md (required)")
		project  = flag.String("project", "", "project root path (default: $PWD)")
		target   = flag.String("target", "", "relative target file path for LLM output (e.g. internal/foo/bar.go)")
		context  = flag.String("context", "", "comma-separated relative file paths to include as context")
		addr     = flag.String("addr", getEnv("NEXUS_ADDR", "http://127.0.0.1:63987"), "daemon base URL")
		wait     = flag.Bool("wait", false, "poll until task completes and print result")
		timeout  = flag.Duration("timeout", 5*time.Minute, "max wait time when --wait is set")
		verify   = flag.String("verify", "", "verification command (e.g. 'go test ./...', 'npm test') to trigger self-healing verification gates")
		turns    = flag.Int("turns", 2, "max self-healing correction turns if verification fails")
		showVer  = flag.Bool("version", false, "print version information and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("nexus-submit %s (%s %s)\n", version, commit, buildDate)
		return
	}

	if *taskFile == "" {
		fmt.Fprintln(os.Stderr, "error: --task-file is required")
		flag.Usage()
		os.Exit(1)
	}

	body, err := BuildRequestBody(*taskFile, *project, *target, *context, *verify, *turns)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	taskID, status, err := SubmitTask(*addr, body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("submitted: task_id=%s status=%s\n", taskID, status)
	fmt.Printf("track: %s/api/tasks/%s\n", *addr, taskID)
	fmt.Printf("ui:    %s/ui\n", *addr)

	if *wait {
		waitForCompletion(*addr, taskID, *timeout)
	}
}

// BuildRequestBody prepares the JSON payload for submitting a task to the daemon.
func BuildRequestBody(taskFilePath, projectPath, targetFile, contextFilesStr, verify string, turns int) (map[string]interface{}, error) {
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get cwd: %w", err)
		}
	}

	content, err := os.ReadFile(taskFilePath)
	if err != nil {
		return nil, fmt.Errorf("read task file: %w", err)
	}

	var contextFiles []string
	if contextFilesStr != "" {
		for _, f := range strings.Split(contextFilesStr, ",") {
			if f = strings.TrimSpace(f); f != "" {
				contextFiles = append(contextFiles, f)
			}
		}
	}

	body := map[string]interface{}{
		"projectPath":  projectPath,
		"targetFile":   targetFile,
		"instruction":  string(content),
		"contextFiles": contextFiles,
	}
	if verify != "" {
		body["verificationCommand"] = verify
		body["maxCorrectionTurns"] = turns
	}
	return body, nil
}

// SubmitTask POSTs a task payload to the daemon and returns the task ID and initial status.
func SubmitTask(addr string, body map[string]interface{}) (string, string, error) {
	reqJSON, err := json.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(addr+"/api/tasks", "application/json", bytes.NewReader(reqJSON))
	if err != nil {
		return "", "", fmt.Errorf("POST /api/tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("daemon returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	return result.TaskID, result.Status, nil
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
		switch t.Status {
		case "COMPLETED":
			if t.Logs != "" {
				fmt.Println("logs:", t.Logs)
			}
			return
		case "FAILED":
			if t.Logs != "" {
				fmt.Println("logs:", t.Logs)
			}
			os.Exit(1)
		case "CANCELLED":
			fmt.Fprintln(os.Stderr, "Task cancelled")
			os.Exit(1)
		case "TOO_LARGE":
			fmt.Fprintln(os.Stderr, "Task rejected: prompt exceeds model context limit")
			os.Exit(2)
		case "NO_PROVIDER":
			fmt.Fprintln(os.Stderr, "Task failed: no LLM provider available")
			os.Exit(3)
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
