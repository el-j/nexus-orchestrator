package main

import (
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

	taskID, status, err := SubmitTask(server.URL, body)
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

	_, _, err := SubmitTask(server.URL, map[string]interface{}{})
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
