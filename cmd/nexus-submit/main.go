// Command nexus-submit submits task files to a running nexusOrchestrator daemon for LLM code generation.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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

// Exit codes returned by run. 1 covers every generic failure (bad usage,
// unreadable task file, daemon unreachable, task FAILED/CANCELLED, timeout).
const (
	exitOK         = 0
	exitFailure    = 1
	exitTooLarge   = 2 // task prompt exceeds the model's context limit
	exitNoProvider = 3 // no LLM provider was available for the task
)

// httpClient bounds every daemon call; the default client would wait forever.
var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, 3*time.Second))
}

// run is the testable body of main. It parses args, submits the task and, with
// --wait, polls every pollEvery until the task reaches a terminal state. It
// returns the process exit code.
func run(args []string, stdout, stderr io.Writer, pollEvery time.Duration) int {
	fs := flag.NewFlagSet("nexus-submit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		taskFile = fs.String("task-file", "", "path to .claude/tasks/TASK-NNN.md (required)")
		project  = fs.String("project", "", "project root path (default: $PWD)")
		target   = fs.String("target", "", "relative target file path for LLM output (e.g. internal/foo/bar.go)")
		ctxFiles = fs.String("context", "", "comma-separated relative file paths to include as context")
		addr     = fs.String("addr", getEnv("NEXUS_ADDR", "http://127.0.0.1:63987"), "daemon base URL")
		wait     = fs.Bool("wait", false, "poll until task completes and print result")
		timeout  = fs.Duration("timeout", 5*time.Minute, "max wait time when --wait is set")
		verify   = fs.String("verify", "", "verification command (e.g. 'go test ./...', 'npm test') to trigger self-healing verification gates")
		turns    = fs.Int("turns", 2, "max self-healing correction turns if verification fails")
		provider = fs.String("provider", "", "lock routing to this exact provider name")
		hint     = fs.String("provider-hint", "", "preferred provider when several serve the model")
		model    = fs.String("model", "", "model ID that must process the task")
		role     = fs.String("role", "", "routing tier hint: architect, techlead, coder, tester or linter")
		priority = fs.Int("priority", 0, "priority 1=high, 2=medium, 3=low (0 = server default)")
		tags     = fs.String("tags", "", "comma-separated labels")
		agentic  = fs.Bool("agentic", false, "let the model read/write project files over several turns and re-verify before finishing")
		maxTurns = fs.Int("max-agent-turns", 0, "upper bound on model turns for --agentic (0 = daemon default)")
		showVer  = fs.Bool("version", false, "print version information and exit")
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitFailure
	}

	if *showVer {
		fmt.Fprintf(stdout, "nexus-submit %s (%s %s)\n", version, commit, buildDate)
		return exitOK
	}

	if *taskFile == "" {
		fmt.Fprintln(stderr, "error: --task-file is required")
		fs.Usage()
		return exitFailure
	}

	body, err := BuildRequestBody(*taskFile, *project, *target, *ctxFiles, *verify, *turns)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitFailure
	}

	ApplyRouting(body, *provider, *hint, *model, *role, *priority, *tags)
	if *agentic {
		body["agentic"] = true
		if *maxTurns > 0 {
			body["maxAgentTurns"] = *maxTurns
		}
	}

	taskID, status, err := SubmitTask(context.Background(), *addr, body)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitFailure
	}

	fmt.Fprintf(stdout, "submitted: task_id=%s status=%s\n", taskID, status)
	fmt.Fprintf(stdout, "track: %s/api/tasks/%s\n", *addr, taskID)
	fmt.Fprintf(stdout, "ui:    %s/ui\n", *addr)

	if *wait {
		// --timeout bounds the whole wait, including a poll request still in flight.
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		return waitForCompletion(ctx, stdout, stderr, *addr, taskID, pollEvery)
	}
	return exitOK
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

// ApplyRouting copies the routing flags onto the request body; unset flags are
// left out so the daemon applies its own defaults.
func ApplyRouting(body map[string]interface{}, provider, hint, model, role string, priority int, tags string) {
	for k, v := range map[string]string{"providerName": provider, "providerHint": hint, "modelId": model, "role": role} {
		if v != "" {
			body[k] = v
		}
	}
	if priority > 0 {
		body["priority"] = priority
	}
	var list []string
	for _, t := range strings.Split(tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			list = append(list, t)
		}
	}
	if len(list) > 0 {
		body["tags"] = list
	}
}

// SubmitTask POSTs a task payload to the daemon and returns the task ID and initial status.
// Cancelling ctx aborts the request.
func SubmitTask(ctx context.Context, addr string, body map[string]interface{}) (string, string, error) {
	reqJSON, err := json.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr+"/api/tasks", bytes.NewReader(reqJSON))
	if err != nil {
		return "", "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
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

// waitForCompletion polls the task every pollEvery until it reaches a terminal
// state or ctx is done (its deadline is the --timeout), and returns the exit code
// describing the outcome.
func waitForCompletion(ctx context.Context, stdout, stderr io.Writer, addr, taskID string, pollEvery time.Duration) int {
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(stderr, "error: timed out waiting for task completion")
			return exitFailure
		case <-time.After(pollEvery):
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr+"/api/tasks/"+taskID, nil)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitFailure
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			if ctx.Err() == nil {
				fmt.Fprintln(stderr, "poll error:", err)
			}
			continue
		}
		var t struct {
			Status string `json:"status"`
			Logs   string `json:"logs"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&t)
		resp.Body.Close()

		fmt.Fprintf(stdout, "  [%s] status=%s\n", time.Now().Format("15:04:05"), t.Status)
		switch t.Status {
		case "COMPLETED":
			if t.Logs != "" {
				fmt.Fprintln(stdout, "logs:", t.Logs)
			}
			return exitOK
		case "FAILED":
			if t.Logs != "" {
				fmt.Fprintln(stdout, "logs:", t.Logs)
			}
			return exitFailure
		case "CANCELLED":
			fmt.Fprintln(stderr, "Task cancelled")
			return exitFailure
		case "TOO_LARGE":
			fmt.Fprintln(stderr, "Task rejected: prompt exceeds model context limit")
			return exitTooLarge
		case "NO_PROVIDER":
			fmt.Fprintln(stderr, "Task failed: no LLM provider available")
			return exitNoProvider
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
