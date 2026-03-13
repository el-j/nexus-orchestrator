package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// mockOrchestrator implements ports.Orchestrator with configurable responses.
type mockOrchestrator struct {
	submitResult  string
	submitErr     error
	queueResult   []domain.Task
	queueErr      error
	getTaskResult domain.Task
	getTaskErr    error
	cancelErr     error
	providersList []ports.ProviderInfo
	providersErr  error
}

func (m *mockOrchestrator) SubmitTask(_ domain.Task) (string, error) {
	return m.submitResult, m.submitErr
}

func (m *mockOrchestrator) GetTask(_ string) (domain.Task, error) {
	return m.getTaskResult, m.getTaskErr
}

func (m *mockOrchestrator) GetQueue() ([]domain.Task, error) {
	return m.queueResult, m.queueErr
}

func (m *mockOrchestrator) GetAllTasks() ([]domain.Task, error) {
	return m.queueResult, m.queueErr
	}

func (m *mockOrchestrator) GetProviders() ([]ports.ProviderInfo, error) {
	return m.providersList, m.providersErr
}

func (m *mockOrchestrator) CancelTask(_ string) error {
	return m.cancelErr
}
func (m *mockOrchestrator) RegisterCloudProvider(_ domain.ProviderConfig) error { return nil }
func (m *mockOrchestrator) RemoveProvider(_ string) error                       { return nil }
func (m *mockOrchestrator) GetProviderModels(_ string) ([]string, error)        { return nil, nil }
func (m *mockOrchestrator) AddProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, nil
}
func (m *mockOrchestrator) UpdateProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, nil
}
func (m *mockOrchestrator) RemoveProviderConfig(_ context.Context, _ string) error { return nil }
func (m *mockOrchestrator) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}
func (m *mockOrchestrator) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *mockOrchestrator) TriggerScan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *mockOrchestrator) PromoteProvider(_ context.Context, _ string) error { return nil }
func (m *mockOrchestrator) CreateDraft(_ domain.Task) (string, error)         { return "", nil }
func (m *mockOrchestrator) GetBacklog(_ string) ([]domain.Task, error)        { return nil, nil }
func (m *mockOrchestrator) PromoteTask(_ string) error                        { return nil }
func (m *mockOrchestrator) UpdateTask(_ string, _ domain.Task) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchestrator) RegisterAISession(_ context.Context, s domain.AISession) (domain.AISession, error) {
	return s, nil
}
func (m *mockOrchestrator) ListAISessions(_ context.Context) ([]domain.AISession, error) {
	return nil, nil
}
func (m *mockOrchestrator) DeregisterAISession(_ context.Context, _ string) error { return nil }
func (m *mockOrchestrator) HeartbeatAISession(_ context.Context, _ string) error  { return nil }

// captureStdout redirects os.Stdout while fn runs and returns the collected
// output. fn must not call t.Fatal/t.FailNow (runtime.Goexit) directly, as
// that would skip the pipe teardown; capture the command error via a closure
// variable instead and check it after captureStdout returns.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("captureStdout: os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("captureStdout: io.Copy: %v", err)
	}
	r.Close()
	return buf.String()
}

// --- queue list ---

func TestQueueList_Empty(t *testing.T) {
	mock := &mockOrchestrator{}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"queue", "list"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "Queue is empty.") {
		t.Errorf("expected output to contain %q; got: %q", "Queue is empty.", out)
	}
}

func TestQueueList_WithTasks(t *testing.T) {
	now := time.Now()
	mock := &mockOrchestrator{
		queueResult: []domain.Task{
			{
				ID:          "task-1",
				ProjectPath: "/proj/a",
				TargetFile:  "main.go",
				Instruction: "write tests",
				Status:      domain.StatusQueued,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          "task-2",
				ProjectPath: "/proj/b",
				TargetFile:  "handler.go",
				Instruction: "add logging",
				Status:      domain.StatusQueued,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		},
	}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"queue", "list"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, string(domain.StatusQueued)) {
		t.Errorf("expected output to contain status %q; got: %q", domain.StatusQueued, out)
	}
	if !strings.Contains(out, "write tests") {
		t.Errorf("expected output to contain %q; got: %q", "write tests", out)
	}
	if !strings.Contains(out, "add logging") {
		t.Errorf("expected output to contain %q; got: %q", "add logging", out)
	}
}

func TestQueueList_Error(t *testing.T) {
	mock := &mockOrchestrator{queueErr: errors.New("storage unavailable")}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"queue", "list"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	var errBuf bytes.Buffer
	root.SetErr(&errBuf)

	if err := root.Execute(); err == nil {
		t.Error("expected an error, got nil")
	}
}

// --- queue get ---

func TestQueueGet_Found(t *testing.T) {
	now := time.Now().Truncate(time.Second) // truncate for time.Time round-trip equality
	mock := &mockOrchestrator{
		getTaskResult: domain.Task{
			ID:          "abc123",
			ProjectPath: "/proj/x",
			TargetFile:  "service.go",
			Instruction: "refactor",
			Status:      domain.StatusQueued,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"queue", "get", "abc123"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	var got domain.Task
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %q", err, out)
	}
	if got.ID != "abc123" {
		t.Errorf("expected task ID %q, got %q", "abc123", got.ID)
	}
}

func TestQueueGet_NotFound(t *testing.T) {
	mock := &mockOrchestrator{
		getTaskErr: fmt.Errorf("repo: %w", domain.ErrNotFound),
	}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"queue", "get", "missing-id"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)

	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to contain %q; got: %q", "not found", err.Error())
	}
}

func TestQueueGet_MissingArg(t *testing.T) {
	mock := &mockOrchestrator{}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"queue", "get"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)

	if err := root.Execute(); err == nil {
		t.Error("expected an error for missing argument, got nil")
	}
}

// --- queue cancel ---

func TestQueueCancel_Success(t *testing.T) {
	mock := &mockOrchestrator{}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"queue", "cancel", "abc123"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "cancelled") {
		t.Errorf("expected output to contain %q; got: %q", "cancelled", out)
	}
}

func TestQueueCancel_Error(t *testing.T) {
	mock := &mockOrchestrator{cancelErr: errors.New("task already completed")}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"queue", "cancel", "abc123"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)

	if err := root.Execute(); err == nil {
		t.Error("expected an error, got nil")
	}
}

// --- providers ---

func TestProviders_Success(t *testing.T) {
	mock := &mockOrchestrator{
		providersList: []ports.ProviderInfo{
			{Name: "lmstudio", Active: true},
			{Name: "ollama", Active: false},
		},
	}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"providers"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	var got []ports.ProviderInfo
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("failed to parse JSON output: %v\noutput: %q", err, out)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(got))
	}
	names := make(map[string]bool, len(got))
	for _, p := range got {
		names[p.Name] = true
	}
	if !names["lmstudio"] {
		t.Errorf("expected provider %q in output", "lmstudio")
	}
	if !names["ollama"] {
		t.Errorf("expected provider %q in output", "ollama")
	}
}

func TestProviders_Empty(t *testing.T) {
	mock := &mockOrchestrator{}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"providers"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	var execErr error
	out := captureStdout(t, func() { execErr = root.Execute() })

	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if !strings.Contains(out, "No providers registered.") {
		t.Errorf("expected output to contain %q; got: %q", "No providers registered.", out)
	}
}

func TestProviders_Error(t *testing.T) {
	mock := &mockOrchestrator{providersErr: errors.New("LLM backend unreachable")}
	root := cli.NewRootCmd(mock)
	root.SetArgs([]string{"providers"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)

	if err := root.Execute(); err == nil {
		t.Error("expected an error, got nil")
	}
}
