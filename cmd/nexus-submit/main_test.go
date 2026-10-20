package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRequestBody_Valid(t *testing.T) {
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "TASK-123.md")
	content := "Implement authentication service"
	if err := os.WriteFile(taskPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	body, err := BuildRequestBody(taskPath, dir, "internal/auth/auth.go", "go.mod, internal/types.go", "go test ./...", 3)
	if err != nil {
		t.Fatalf("BuildRequestBody failed: %v", err)
	}

	if body["projectPath"] != dir {
		t.Errorf("expected projectPath %s, got %v", dir, body["projectPath"])
	}
	if body["targetFile"] != "internal/auth/auth.go" {
		t.Errorf("expected targetFile internal/auth/auth.go, got %v", body["targetFile"])
	}
	if body["instruction"] != content {
		t.Errorf("expected instruction %s, got %v", content, body["instruction"])
	}
	if body["verificationCommand"] != "go test ./..." {
		t.Errorf("expected verificationCommand 'go test ./...', got %v", body["verificationCommand"])
	}
	if body["maxCorrectionTurns"] != 3 {
		t.Errorf("expected maxCorrectionTurns 3, got %v", body["maxCorrectionTurns"])
	}

	ctxFiles, ok := body["contextFiles"].([]string)
	if !ok || len(ctxFiles) != 2 || ctxFiles[0] != "go.mod" || ctxFiles[1] != "internal/types.go" {
		t.Errorf("expected parsed contextFiles [go.mod internal/types.go], got %v", body["contextFiles"])
	}
}

func TestBuildRequestBody_MissingFile(t *testing.T) {
	_, err := BuildRequestBody("/nonexistent/file.md", "/proj", "target.go", "", "", 2)
	if err == nil {
		t.Error("expected error when task file does not exist, got nil")
	}
}

func TestSubmitTask_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/tasks" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["targetFile"] != "main.go" {
			t.Errorf("unexpected targetFile %v", payload["targetFile"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"task_id":"task-abc-123","status":"QUEUED"}`))
	}))
	defer server.Close()

	body := map[string]interface{}{
		"targetFile":  "main.go",
		"instruction": "package main",
	}

	taskID, status, err := SubmitTask(context.Background(), server.URL, body)
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	if taskID != "task-abc-123" {
		t.Errorf("expected task-abc-123, got %s", taskID)
	}
	if status != "QUEUED" {
		t.Errorf("expected QUEUED, got %s", status)
	}
}

func TestSubmitTask_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":"invalid task payload"}`)
	}))
	defer server.Close()

	_, _, err := SubmitTask(context.Background(), server.URL, map[string]interface{}{})
	if err == nil {
		t.Error("expected error on 400 response, got nil")
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_NEXUS_VAR", "configured_value")
	defer os.Unsetenv("TEST_NEXUS_VAR")

	if got := getEnv("TEST_NEXUS_VAR", "default"); got != "configured_value" {
		t.Errorf("expected configured_value, got %s", got)
	}
	if got := getEnv("NONEXISTENT_VAR", "default"); got != "default" {
		t.Errorf("expected default, got %s", got)
	}
}

func TestApplyRouting(t *testing.T) {
	body := map[string]interface{}{}
	ApplyRouting(body, "", "", "", "", 0, "")
	if len(body) != 0 {
		t.Fatalf("unset flags must not be sent, got %v", body)
	}
	ApplyRouting(body, "GitHub Copilot", "Ollama", "gpt-4o", "architect", 1, " a, ,b ")
	want := map[string]interface{}{
		"providerName": "GitHub Copilot", "providerHint": "Ollama", "modelId": "gpt-4o",
		"role": "architect", "priority": 1, "tags": []string{"a", "b"},
	}
	got, _ := json.Marshal(body)
	exp, _ := json.Marshal(want)
	if !bytes.Equal(got, exp) {
		t.Errorf("body = %s, want %s", got, exp)
	}
}

func TestSubmitTask_SendsBearerTokenWhenEnvSet(t *testing.T) {
	t.Setenv("NEXUS_API_TOKEN", "secret-token-xyz")
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"task_id":"task-auth-1","status":"QUEUED"}`))
	}))
	defer server.Close()

	_, _, err := SubmitTask(context.Background(), server.URL, map[string]interface{}{"instruction": "hi"})
	if err != nil {
		t.Fatalf("SubmitTask: %v", err)
	}
	if gotAuth != "Bearer secret-token-xyz" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer secret-token-xyz")
	}
}

func TestWaitForCompletion_LogsWaitingUntilAndCompletes(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Write([]byte(`{"status":"WAITING","waitUntil":"2026-10-20T15:00:00Z"}`))
			return
		}
		w.Write([]byte(`{"status":"COMPLETED","logs":"done"}`))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := waitForCompletion(context.Background(), &stdout, &stderr, server.URL, "task-wait-1", 0)
	if code != exitOK {
		t.Fatalf("code = %d, want %d (stderr: %s)", code, exitOK, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("waiting until 2026-10-20T15:00:00Z")) {
		t.Errorf("stdout should include waiting until timestamp, got: %s", stdout.String())
	}
}
