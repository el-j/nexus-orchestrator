package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/core/domain"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

type autoScripted struct {
	scriptedOrch
	run      domain.AutoModeRun
	nextRun  domain.AutoModeRun
	runs     []domain.AutoModeRun
	startErr error
	getErr   error
	stopErr  error
	gets     int
}

func (a *autoScripted) StartAutoMode(_ context.Context, req domain.AutoModeRequest) (domain.AutoModeRun, error) {
	if a.startErr != nil {
		return domain.AutoModeRun{}, a.startErr
	}
	r := a.run
	r.ProjectPath = req.ProjectPath
	r.Goal = req.Goal
	return r, nil
}
func (a *autoScripted) GetAutoModeRun(string) (domain.AutoModeRun, error) {
	a.gets++
	if a.getErr != nil {
		return domain.AutoModeRun{}, a.getErr
	}
	if a.nextRun.ID != "" {
		return a.nextRun, nil
	}
	return a.run, nil
}
func (a *autoScripted) ListAutoModeRuns(string) []domain.AutoModeRun { return a.runs }
func (a *autoScripted) StopAutoMode(string) error                    { return a.stopErr }

func TestAutoCmd(t *testing.T) {
	// Unavailable when orch does not implement AutoModeManager
	for _, args := range [][]string{{"auto", "start"}, {"auto", "status"}, {"auto", "stop", "r1"}} {
		if _, err := execute(&scriptedOrch{}, nil, args...); err == nil || !strings.Contains(err.Error(), "not available") {
			t.Fatalf("%v: expected not available, got %v", args, err)
		}
	}

	orch := &autoScripted{
		run:     domain.AutoModeRun{ID: "r1", Phase: domain.AutoPhaseExecuting, TaskIDs: []string{"t1"}},
		nextRun: domain.AutoModeRun{ID: "r1", Phase: domain.AutoPhaseCompleted, TaskIDs: []string{"t1"}, CompletedTasks: 1, Summary: "all done"},
		runs:    []domain.AutoModeRun{{ID: "r1", Phase: domain.AutoPhaseCompleted, ProjectPath: "/p", TaskIDs: []string{"t1"}, CompletedTasks: 1}},
	}

	// Start with --wait
	out, err := execute(orch, nil, "auto", "start", "implement feature", "--project", ".", "--wait")
	if err != nil || !strings.Contains(out, "all done") {
		t.Fatalf("auto start --wait: out=%q err=%v", out, err)
	}

	// Start with --json and failed outcome
	orch.run = domain.AutoModeRun{ID: "r2", Phase: domain.AutoPhaseFailed, Error: "verify failed"}
	orch.nextRun = domain.AutoModeRun{}
	if out, err := execute(orch, nil, "auto", "start", "--json"); err == nil || !strings.Contains(out, `"r2"`) {
		t.Fatalf("expected failed error with json output, out=%q err=%v", out, err)
	}

	// Start errors
	if _, err := execute(orch, nil, "auto", "start", "   "); err == nil {
		t.Fatal("expected error for blank goal")
	}
	orch.startErr = errors.New("already active")
	if _, err := execute(orch, nil, "auto", "start"); err == nil || !strings.Contains(err.Error(), "already active") {
		t.Fatalf("expected startErr, got %v", err)
	}
	orch.startErr = nil

	// Status list, json, empty, single ID, error
	out, err = execute(orch, nil, "auto", "status", "--project", ".")
	if err != nil || !strings.Contains(out, "r1") {
		t.Fatalf("auto status list: %q %v", out, err)
	}
	out, err = execute(orch, nil, "auto", "status", "--json")
	if err != nil || !strings.Contains(out, `"r1"`) {
		t.Fatalf("auto status json: %q %v", out, err)
	}
	orch.runs = nil
	out, err = execute(orch, nil, "auto", "status")
	if err != nil || !strings.Contains(out, "No Auto-Mode runs") {
		t.Fatalf("auto status empty: %q %v", out, err)
	}
	orch.run = domain.AutoModeRun{ID: "r1", Phase: domain.AutoPhaseCompleted}
	out, err = execute(orch, nil, "auto", "status", "r1")
	if err != nil || !strings.Contains(out, "r1") {
		t.Fatalf("auto status id: %q %v", out, err)
	}
	orch.getErr = errors.New("missing")
	if _, err := execute(orch, nil, "auto", "status", "r1"); err == nil {
		t.Fatal("expected status error")
	}

	// Stop OK & error
	out, err = execute(orch, nil, "auto", "stop", "r1")
	if err != nil || !strings.Contains(out, "stopped") {
		t.Fatalf("auto stop: %q %v", out, err)
	}
	orch.stopErr = errors.New("not found")
	if _, err := execute(orch, nil, "auto", "stop", "r1"); err == nil {
		t.Fatal("expected stop error")
	}

	// Non-JSON failed status prints error line.
	orch.getErr = nil
	orch.run = domain.AutoModeRun{ID: "r2", Phase: domain.AutoPhaseFailed, Error: "verify failed"}
	if out, err := execute(orch, nil, "auto", "status", "r2"); err == nil || !strings.Contains(out, "error: verify failed") {
		t.Fatalf("expected failed text output with error line, out=%q err=%v", out, err)
	}

	// waitForAutoRun exits cleanly when command context is cancelled.
	orch.run = domain.AutoModeRun{ID: "r3", Phase: domain.AutoPhaseExecuting}
	orch.nextRun = domain.AutoModeRun{}
	cmd := cli.NewRootCmd(orch, nil)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"auto", "start", "--wait"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cancelled wait returned unexpected error: %v", err)
	}

	// JSON encode write error is returned by printAutoRun.
	cmdErr := cli.NewRootCmd(orch, nil)
	cmdErr.SetOut(failingWriter{})
	cmdErr.SetErr(&buf)
	cmdErr.SetArgs([]string{"auto", "start", "--json"})
	if err := cmdErr.Execute(); err == nil {
		t.Fatal("expected JSON encode write error")
	}
}
