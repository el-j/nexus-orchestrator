package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/core/domain"
)

type orchestrateMockOrch struct {
	mockOrchestrator
	submitID  string
	submitErr error
	getCalls  int
	tasks     []domain.Task
	getErr    error
}

func (m *orchestrateMockOrch) SubmitTask(_ domain.Task) (string, error) {
	if m.submitErr != nil {
		return "", m.submitErr
	}
	return m.submitID, nil
}

func (m *orchestrateMockOrch) GetTask(_ string) (domain.Task, error) {
	if m.getErr != nil {
		return domain.Task{}, m.getErr
	}
	m.getCalls++
	if len(m.tasks) == 0 {
		return domain.Task{Status: domain.StatusCompleted}, nil
	}
	idx := m.getCalls - 1
	if idx >= len(m.tasks) {
		return m.tasks[len(m.tasks)-1], nil
	}
	return m.tasks[idx], nil
}

func TestOrchestrateCmd_InlineInstruction(t *testing.T) {
	orch := &mockOrchestrator{
		submitResult: "task-pipe-1",
		getTaskResult: domain.Task{
			ID:          "task-pipe-1",
			Status:      domain.StatusCompleted,
			Instruction: "implement pipeline command",
			TargetFile:  "cmd/pipeline.go",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	brain := &mockBrainService{
		getContextResult: domain.ContextResponse{
			ModelTier:      domain.ModelTierFrontier,
			TokensSaved:    120,
			SuggestedFiles: []string{"cmd/root.go"},
		},
	}

	cmd := cli.NewRootCmd(orch, brain)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"orchestrate",
		"implement pipeline command",
		"-t", "cmd/pipeline.go",
		"-m", "claude-3-5-sonnet",
		"-r", "coder",
		"-v", "go test ./...",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "[1/4] RAG Retrieval: tailored for frontier tier (saved 120 tokens)") {
		t.Errorf("expected RAG retrieval output, got: %s", out)
	}
	if !strings.Contains(out, "[2/4] Affinity Routing: role=coder") {
		t.Errorf("expected Affinity routing output, got: %s", out)
	}
	if !strings.Contains(out, "[3/4] Task Dispatched: ID=task-pipe-1") {
		t.Errorf("expected Task dispatched output, got: %s", out)
	}
	if !strings.Contains(out, "Pipeline Finished: status=COMPLETED") {
		t.Errorf("expected Pipeline finished output, got: %s", out)
	}
}

func TestOrchestrateCmd_JSONOutput(t *testing.T) {
	orch := &mockOrchestrator{
		submitResult: "task-pipe-json",
		getTaskResult: domain.Task{
			ID:          "task-pipe-json",
			Status:      domain.StatusCompleted,
			ModelID:     "qwen3-coder-next",
			Instruction: "lint project",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	brain := &mockBrainService{
		getContextResult: domain.ContextResponse{
			ModelTier:      domain.ModelTierLocal,
			TokensSaved:    50,
			SuggestedFiles: []string{"pkg/harness/harness.go"},
		},
	}

	cmd := cli.NewRootCmd(orch, brain)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"orchestrate",
		"lint project",
		"--json",
		"-m", "qwen3-coder-next",
		"-r", "linter",
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res cli.OrchestratePipelineResult
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON result: %v (output: %s)", err, buf.String())
	}

	if res.TaskID != "task-pipe-json" || res.Status != "COMPLETED" {
		t.Errorf("unexpected pipeline result: %+v", res)
	}
	if res.ModelTier != "local" {
		t.Errorf("expected local model tier, got %s", res.ModelTier)
	}
	if res.RAGTokensSaved != 50 {
		t.Errorf("expected 50 tokens saved, got %d", res.RAGTokensSaved)
	}
	if len(res.SuggestedFiles) != 1 || res.SuggestedFiles[0] != "pkg/harness/harness.go" {
		t.Errorf("expected suggested files, got: %v", res.SuggestedFiles)
	}
}

func TestOrchestrateCmd_SpecFile(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.md")
	if err := os.WriteFile(specFile, []byte("design distributed coordinator subsystem"), 0o600); err != nil {
		t.Fatal(err)
	}

	orch := &mockOrchestrator{
		submitResult: "task-spec-1",
		getTaskResult: domain.Task{
			ID:          "task-spec-1",
			Status:      domain.StatusCompleted,
			Instruction: "design distributed coordinator subsystem",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	cmd := cli.NewRootCmd(orch, nil) // test with nil brain
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{
		"orchestrate",
		specFile,
		"-p", tmpDir,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Pipeline Finished: status=COMPLETED") {
		t.Errorf("expected pipeline finished, got: %s", out)
	}
}

func TestOrchestrateCmd_Errors(t *testing.T) {
	orch := &mockOrchestrator{}
	cmd := cli.NewRootCmd(orch, nil)

	// 1. Missing argument
	cmd.SetArgs([]string{"orchestrate"})
	if err := cmd.Execute(); err == nil {
		t.Errorf("expected error for missing argument")
	}

	// 2. Empty instruction string
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"orchestrate", "   "})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected cannot be empty error, got: %v", err)
	}

	// 3. Submit error
	errOrch := &mockOrchestrator{
		submitErr: errors.New("daemon offline"),
	}
	cmdErr := cli.NewRootCmd(errOrch, nil)
	cmdErr.SetOut(buf)
	cmdErr.SetArgs([]string{"orchestrate", "do something"})
	if err := cmdErr.Execute(); err == nil || !strings.Contains(err.Error(), "daemon offline") {
		t.Errorf("expected daemon offline error, got: %v", err)
	}

	// 4. Failed task outcome with brain ingestion
	failedOrch := &mockOrchestrator{
		submitResult: "fail-task",
		getTaskResult: domain.Task{
			ID:          "fail-task",
			Status:      domain.StatusFailed,
			Instruction: "bad instruction",
			Logs:        "error: test failed",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	brain := &mockBrainService{}
	cmdFail := cli.NewRootCmd(failedOrch, brain)
	bufFail := new(bytes.Buffer)
	cmdFail.SetOut(bufFail)
	cmdFail.SetArgs([]string{"orchestrate", "bad instruction"})
	if err := cmdFail.Execute(); err == nil || !strings.Contains(err.Error(), "ended FAILED") {
		t.Fatalf("a failed task must fail the command, got: %v", err)
	}
	if !strings.Contains(bufFail.String(), "Pipeline Finished: status=FAILED") {
		t.Errorf("expected FAILED status, got: %s", bufFail.String())
	}

	// 5. Polling transition from QUEUED to COMPLETED (exercises loop sleep)
	pollOrch := &orchestrateMockOrch{
		submitID: "poll-task-1",
		tasks: []domain.Task{
			{ID: "poll-task-1", Status: domain.StatusQueued},
			{ID: "poll-task-1", Status: domain.StatusCompleted},
		},
	}
	cmdPoll := cli.NewRootCmd(pollOrch, nil)
	bufPoll := new(bytes.Buffer)
	cmdPoll.SetOut(bufPoll)
	cmdPoll.SetArgs([]string{"orchestrate", "poll instruction"})
	if err := cmdPoll.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(bufPoll.String(), "Pipeline Finished: status=COMPLETED") {
		t.Errorf("expected COMPLETED, got: %s", bufPoll.String())
	}

	// 6. Unreadable spec file error
	tmpDir := t.TempDir()
	unreadableFile := filepath.Join(tmpDir, "unreadable.md")
	if err := os.WriteFile(unreadableFile, []byte("secret"), 0o000); err == nil {
		cmdUnread := cli.NewRootCmd(orch, nil)
		cmdUnread.SetArgs([]string{"orchestrate", unreadableFile})
		_ = cmdUnread.Execute()
		_ = os.Chmod(unreadableFile, 0o600)
	}
}

// A task that never settles (a provider that waits for its limit to reset) does not hold the command
// beyond the user's Ctrl-C: it returns the task as it is and fails.
func TestOrchestrateCmd_StopsWaitingWhenInterrupted(t *testing.T) {
	orch := &orchestrateMockOrch{submitID: "w1", tasks: []domain.Task{{ID: "w1", Status: domain.StatusWaiting}}}
	cmd := cli.NewRootCmd(orch, nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"orchestrate", "wait forever"})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	err := cmd.ExecuteContext(ctx)
	if err == nil || !strings.Contains(err.Error(), "ended WAITING") {
		t.Errorf("err = %v, want the WAITING outcome", err)
	}
}

// With --json a failed task is still reported in full before the command fails.
func TestOrchestrateCmd_JSONReportsAFailedTaskAndFails(t *testing.T) {
	orch := &mockOrchestrator{submitResult: "j1", getTaskResult: domain.Task{ID: "j1", Status: domain.StatusFailed}}
	cmd := cli.NewRootCmd(orch, nil)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"orchestrate", "--json", "x"})
	if err := cmd.Execute(); err == nil || !strings.Contains(buf.String(), `"FAILED"`) {
		t.Errorf("err = %v, output %s", err, buf.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("pipe closed") }

// A report that cannot be written is the error, not a silent success.
func TestOrchestrateCmd_JSONReportsAWriteFailure(t *testing.T) {
	orch := &mockOrchestrator{submitResult: "j2", getTaskResult: domain.Task{ID: "j2", Status: domain.StatusCompleted}}
	cmd := cli.NewRootCmd(orch, nil)
	cmd.SetOut(brokenWriter{})
	cmd.SetArgs([]string{"orchestrate", "--json", "x"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "pipe closed") {
		t.Errorf("err = %v", err)
	}
}
