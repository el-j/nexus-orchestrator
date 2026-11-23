package harness_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/pkg/harness"
)

type mockExecutor struct {
	results map[string][]harness.LadderStepResult
	calls   map[string]int
	err     error
}

func (m *mockExecutor) Execute(_ context.Context, cmdStr string, _ string) (harness.LadderStepResult, error) {
	if m.err != nil {
		return harness.LadderStepResult{Command: cmdStr}, m.err
	}
	m.calls[cmdStr]++
	resList := m.results[cmdStr]
	if len(resList) == 0 {
		return harness.LadderStepResult{Command: cmdStr, Passed: true}, nil
	}
	callIdx := m.calls[cmdStr] - 1
	if callIdx >= len(resList) {
		return resList[len(resList)-1], nil
	}
	return resList[callIdx], nil
}

func TestNewHarness_Defaults(t *testing.T) {
	h := harness.NewHarness(harness.Config{}, nil, nil)
	if h == nil {
		t.Fatal("expected non-nil harness")
	}
}

func TestOSCommandExecutor(t *testing.T) {
	exec := &harness.OSCommandExecutor{}
	ctx := context.Background()

	// 1. Successful command
	res, err := exec.Execute(ctx, "echo 'hello world'", "")
	if err != nil || !res.Passed || res.ExitCode != 0 {
		t.Fatalf("expected passed command, got: %+v, err: %v", res, err)
	}
	if !strings.Contains(res.Output, "hello world") {
		t.Errorf("expected output to contain hello world, got: %s", res.Output)
	}

	// 2. Failing command with non-zero exit code
	failRes, err := exec.Execute(ctx, "exit 42", "")
	if err != nil || failRes.Passed || failRes.ExitCode != 42 {
		t.Fatalf("expected failed command with code 42, got: %+v, err: %v", failRes, err)
	}

	// 3. Command error without ExitError (canceled context)
	cancCtx, cancel := context.WithCancel(ctx)
	cancel()
	cancRes, _ := exec.Execute(cancCtx, "sleep 1", "")
	if cancRes.Passed || cancRes.ExitCode != 1 {
		t.Errorf("expected exit code 1 for canceled context, got: %d", cancRes.ExitCode)
	}

	// 3. Command in specific workDir
	tmpDir := t.TempDir()
	pwdRes, err := exec.Execute(ctx, "pwd", tmpDir)
	if err != nil || !pwdRes.Passed {
		t.Fatalf("pwd failed: %v", err)
	}
	// On macOS /var might be symlinked to /private/var
	cleanTmp, _ := filepath.EvalSymlinks(tmpDir)
	cleanOut, _ := filepath.EvalSymlinks(strings.TrimSpace(pwdRes.Output))
	if cleanTmp != cleanOut && !strings.Contains(pwdRes.Output, tmpDir) {
		t.Errorf("expected output to contain %s, got: %s", tmpDir, pwdRes.Output)
	}
}

func TestScriptedSimulator(t *testing.T) {
	sim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{
				Files:      map[string]string{"main.go": "package main"},
				TokensUsed: 150,
			},
		},
	}

	// Valid turn
	files, tokens, err := sim.Generate(context.Background(), 1, "")
	if err != nil || tokens != 150 || files["main.go"] != "package main" {
		t.Errorf("unexpected turn 1 output: %v, %d, %v", files, tokens, err)
	}

	// Out of bounds turn
	_, _, err = sim.Generate(context.Background(), 2, "")
	if err == nil || !strings.Contains(err.Error(), "no scripted turn configured") {
		t.Errorf("expected out of bounds error, got: %v", err)
	}
	_, _, err = sim.Generate(context.Background(), 0, "")
	if err == nil {
		t.Errorf("expected error for turn 0")
	}
}

func TestHarness_Permissive_RedToGreenWorkflow(t *testing.T) {
	verboseOutput := strings.Repeat("verbose build system log line\n", 20) + "main.go:10: undefined: Answer\n"
	mockExec := &mockExecutor{
		calls: make(map[string]int),
		results: map[string][]harness.LadderStepResult{
			"go test ./...": {
				{Command: "go test ./...", Passed: false, ExitCode: 1, Output: "initial fail"},
				{Command: "go test ./...", Passed: false, ExitCode: 1, Output: verboseOutput},
				{Command: "go test ./...", Passed: true, ExitCode: 0, Output: "ok"},
			},
		},
	}

	sim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{
				Files:      map[string]string{"main.go": "package main\n"},
				TokensUsed: 120,
			},
			{
				Files:      map[string]string{"main.go": "package main\nconst Answer = 42\n"},
				TokensUsed: 120,
			},
		},
	}

	cfg := harness.Config{
		TurnBudget:     3,
		Timeout:        5 * time.Second,
		CommandLadder:  []string{"go test ./..."},
		QualityGate:    harness.QualityGatePermissive,
		SimulationMode: true,
	}

	h := harness.NewHarness(cfg, mockExec, sim)
	report, err := h.Run(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !report.Success {
		t.Errorf("expected success, got failure: %s", report.FailureReason)
	}
	if !report.InitialStateFailed {
		t.Errorf("expected initial state failed (red phase)")
	}
	if !report.FinalStatePassed {
		t.Errorf("expected final state passed (green phase)")
	}
	if report.TurnsExecuted != 2 {
		t.Errorf("expected 2 turns executed, got %d", report.TurnsExecuted)
	}
	if report.TotalTokensSpent != 240 {
		t.Errorf("expected 240 tokens spent, got %d", report.TotalTokensSpent)
	}
	if report.TokensSaved <= 0 {
		t.Errorf("expected positive tokens saved via AST diagnostic extraction, got %d", report.TokensSaved)
	}
}

func TestHarness_Strict_RequiresInitialRedPhase(t *testing.T) {
	// Already passing from start
	mockExec := &mockExecutor{
		calls: make(map[string]int),
		results: map[string][]harness.LadderStepResult{
			"go test ./...": {
				{Command: "go test ./...", Passed: true, ExitCode: 0, Output: "ok"},
			},
		},
	}

	sim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{TokensUsed: 50},
		},
	}

	cfg := harness.Config{
		TurnBudget:     2,
		QualityGate:    harness.QualityGateStrict,
		SimulationMode: true,
	}

	h := harness.NewHarness(cfg, mockExec, sim)
	report, err := h.Run(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Success {
		t.Errorf("expected strict quality gate to reject run without red phase")
	}
	if !strings.Contains(report.FailureReason, "strict quality gate failed") {
		t.Errorf("expected strict quality gate failure reason, got: %s", report.FailureReason)
	}
}

func TestHarness_TurnBudgetExhausted(t *testing.T) {
	mockExec := &mockExecutor{
		calls: make(map[string]int),
		results: map[string][]harness.LadderStepResult{
			"go test ./...": {
				{Command: "go test ./...", Passed: false, ExitCode: 1, Output: "failing"},
				{Command: "go test ./...", Passed: false, ExitCode: 1, Output: "still failing"},
			},
		},
	}

	sim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{TokensUsed: 100},
			{TokensUsed: 100},
		},
	}

	cfg := harness.Config{
		TurnBudget:     2,
		SimulationMode: true,
	}

	h := harness.NewHarness(cfg, mockExec, sim)
	report, err := h.Run(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Success || report.FinalStatePassed {
		t.Errorf("expected failure when turn budget exhausted")
	}
	if report.FailureReason != "turn budget exhausted without passing verification ladder" {
		t.Errorf("unexpected failure reason: %s", report.FailureReason)
	}
}

func TestHarness_ModelGenerationError(t *testing.T) {
	mockExec := &mockExecutor{
		calls: make(map[string]int),
		results: map[string][]harness.LadderStepResult{
			"go test ./...": {
				{Command: "go test ./...", Passed: false, ExitCode: 1, Output: "fail"},
			},
		},
	}

	sim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{Err: errors.New("provider rate limit exceeded")},
		},
	}

	cfg := harness.Config{
		TurnBudget:     2,
		SimulationMode: true,
	}

	h := harness.NewHarness(cfg, mockExec, sim)
	report, err := h.Run(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Success {
		t.Errorf("expected failure on model generation error")
	}
	if !strings.Contains(report.FailureReason, "provider rate limit exceeded") {
		t.Errorf("expected model error in failure reason, got: %s", report.FailureReason)
	}
}

func TestHarness_Timeout(t *testing.T) {
	mockExec := &mockExecutor{
		calls: make(map[string]int),
		results: map[string][]harness.LadderStepResult{
			"go test ./...": {
				{Command: "go test ./...", Passed: false, ExitCode: 1, Output: "fail"},
			},
		},
	}

	sim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{TokensUsed: 10},
		},
	}

	cfg := harness.Config{
		TurnBudget:     5,
		Timeout:        1 * time.Nanosecond, // immediate timeout
		SimulationMode: true,
	}

	h := harness.NewHarness(cfg, mockExec, sim)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	report, err := h.Run(ctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Success {
		t.Errorf("expected failure on timeout")
	}
}

func TestHarness_ApplyFiles(t *testing.T) {
	mockExec := &mockExecutor{
		calls: make(map[string]int),
		results: map[string][]harness.LadderStepResult{
			"go test ./...": {
				{Command: "go test ./...", Passed: false, ExitCode: 1, Output: "fail"},
				{Command: "go test ./...", Passed: true, ExitCode: 0, Output: "ok"},
			},
		},
	}

	tmpDir := t.TempDir()
	sim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{
				Files: map[string]string{
					"sub/pkg/file.go": "package pkg\n",
				},
				TokensUsed: 50,
			},
		},
	}

	cfg := harness.Config{
		TurnBudget:     2,
		SimulationMode: false, // will write to disk
	}

	h := harness.NewHarness(cfg, mockExec, sim)
	report, err := h.Run(context.Background(), tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Success {
		t.Fatalf("expected success, got failure: %s", report.FailureReason)
	}

	// Verify file was written to disk
	writtenFile := filepath.Join(tmpDir, "sub", "pkg", "file.go")
	content, err := os.ReadFile(writtenFile)
	if err != nil || string(content) != "package pkg\n" {
		t.Errorf("file was not written properly: %v, %s", err, string(content))
	}

	// Test applyFiles write error (e.g. directory exists as file)
	conflictDir := filepath.Join(tmpDir, "conflict")
	if err := os.WriteFile(conflictDir, []byte("i am a file"), 0o600); err != nil {
		t.Fatal(err)
	}
	errSim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{
				Files: map[string]string{
					"conflict/nested/file.go": "content",
				},
			},
		},
	}
	mockExec2 := &mockExecutor{
		calls: make(map[string]int),
		results: map[string][]harness.LadderStepResult{
			"go test ./...": {{Passed: false}},
		},
	}
	hErr := harness.NewHarness(cfg, mockExec2, errSim)
	repErr, _ := hErr.Run(context.Background(), tmpDir)
	if repErr.Success {
		t.Errorf("expected failure when file cannot be created")
	}

	// Test applyFiles write error (e.g. target is existing directory)
	targetDirConflict := filepath.Join(tmpDir, "target_is_dir")
	if err := os.MkdirAll(targetDirConflict, 0o755); err != nil {
		t.Fatal(err)
	}
	errSimTarget := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{
				Files: map[string]string{
					"target_is_dir": "cannot write to dir",
				},
			},
		},
	}
	hErrTarget := harness.NewHarness(cfg, mockExec2, errSimTarget)
	repErrTarget, _ := hErrTarget.Run(context.Background(), tmpDir)
	if repErrTarget.Success {
		t.Errorf("expected failure when target is directory")
	}
}

func TestHarness_ExecutorError(t *testing.T) {
	failingExec := &mockExecutor{
		err: errors.New("command not found"),
	}
	sim := &harness.ScriptedSimulator{
		Turns: []harness.SimulatedTurn{
			{TokensUsed: 10},
		},
	}
	cfg := harness.Config{
		TurnBudget:     1,
		SimulationMode: true,
	}
	h := harness.NewHarness(cfg, failingExec, sim)
	report, err := h.Run(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Success {
		t.Errorf("expected failure when executor fails")
	}
}
