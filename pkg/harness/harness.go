// Package harness provides a configurable test-driven development (TDD) execution
// and simulation harness with adjustable quality gates, command ladders, and telemetry.
package harness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"nexus-orchestrator/pkg/runner"
)

// QualityGateMode configures the enforcement level of TDD quality gates.
type QualityGateMode string

const (
	// QualityGatePermissive runs verifications without strict TDD ordering checks.
	QualityGatePermissive QualityGateMode = "permissive"
	// QualityGateStrict enforces that tests must fail first (red phase) before passing (green phase).
	QualityGateStrict QualityGateMode = "strict"
)

// Config configures the test-driven execution and simulation harness.
type Config struct {
	TurnBudget     int             `json:"turnBudget"`
	Timeout        time.Duration   `json:"timeout"`
	CommandLadder  []string        `json:"commandLadder"`
	QualityGate    QualityGateMode `json:"qualityGate"`
	MaxASTDepth    int             `json:"maxAstDepth"`
	SimulationMode bool            `json:"simulationMode"`
}

// LadderStepResult captures the outcome of one verification ladder step.
type LadderStepResult struct {
	Command  string        `json:"command"`
	Passed   bool          `json:"passed"`
	ExitCode int           `json:"exitCode"`
	Output   string        `json:"output"`
	Duration time.Duration `json:"duration"`
}

// CommandExecutor abstracts the execution of verification ladder commands.
type CommandExecutor interface {
	Execute(ctx context.Context, cmdStr string, workDir string) (LadderStepResult, error)
}

// OSCommandExecutor runs commands against the local host operating system.
type OSCommandExecutor struct{}

// Execute executes cmdStr using sh -c in workDir.
func (e *OSCommandExecutor) Execute(ctx context.Context, cmdStr string, workDir string) (LadderStepResult, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	if workDir != "" {
		cmd.Dir = workDir
	}
	out, err := cmd.CombinedOutput()
	dur := time.Since(start)

	exitCode := 0
	passed := true
	if err != nil {
		passed = false
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return LadderStepResult{
		Command:  cmdStr,
		Passed:   passed,
		ExitCode: exitCode,
		Output:   string(out),
		Duration: dur,
	}, nil
}

// ModelSimulator simulates an agent generating code modifications across turns.
type ModelSimulator interface {
	Generate(ctx context.Context, turn int, lastFeedback string) (map[string]string, int, error)
}

// ScriptedSimulator provides deterministic scripted model turns for testing and simulation.
type ScriptedSimulator struct {
	Turns []SimulatedTurn
}

// SimulatedTurn represents a single scripted turn response.
type SimulatedTurn struct {
	Files      map[string]string
	TokensUsed int
	Err        error
}

// Generate returns the turn response matching the turn index (1-based).
func (s *ScriptedSimulator) Generate(_ context.Context, turn int, _ string) (map[string]string, int, error) {
	idx := turn - 1
	if idx < 0 || idx >= len(s.Turns) {
		return nil, 0, fmt.Errorf("harness: no scripted turn configured for turn %d", turn)
	}
	t := s.Turns[idx]
	return t.Files, t.TokensUsed, t.Err
}

// Report contains the evaluation and benchmark metrics from a harness execution.
type Report struct {
	Success            bool               `json:"success"`
	TurnsExecuted      int                `json:"turnsExecuted"`
	TurnBudget         int                `json:"turnBudget"`
	InitialStateFailed bool               `json:"initialStateFailed"`
	FinalStatePassed   bool               `json:"finalStatePassed"`
	TotalTokensSpent   int                `json:"totalTokensSpent"`
	TokensSaved        int                `json:"tokensSaved"`
	LadderResults      []LadderStepResult `json:"ladderResults"`
	Duration           time.Duration      `json:"duration"`
	FailureReason      string             `json:"failureReason,omitempty"`
}

// Harness orchestrates test-driven verification loops with adjustable controls.
type Harness struct {
	cfg      Config
	executor CommandExecutor
	model    ModelSimulator
}

// NewHarness constructs a new Harness with the given configuration, executor, and simulator.
func NewHarness(cfg Config, executor CommandExecutor, model ModelSimulator) *Harness {
	if cfg.TurnBudget <= 0 {
		cfg.TurnBudget = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if len(cfg.CommandLadder) == 0 {
		cfg.CommandLadder = []string{"go test ./..."}
	}
	if cfg.QualityGate == "" {
		cfg.QualityGate = QualityGatePermissive
	}
	if cfg.MaxASTDepth <= 0 {
		cfg.MaxASTDepth = 5
	}
	if executor == nil {
		executor = &OSCommandExecutor{}
	}
	return &Harness{
		cfg:      cfg,
		executor: executor,
		model:    model,
	}
}

// runLadder executes the command ladder sequentially in workDir.
func (h *Harness) runLadder(ctx context.Context, workDir string) ([]LadderStepResult, bool) {
	var results []LadderStepResult
	allPassed := true

	for _, cmd := range h.cfg.CommandLadder {
		res, err := h.executor.Execute(ctx, cmd, workDir)
		if err != nil {
			res.Passed = false
			res.Output = err.Error()
		}
		results = append(results, res)
		if !res.Passed {
			allPassed = false
			break
		}
	}
	return results, allPassed
}

// applyFiles writes the generated files to disk in workDir.
func applyFiles(workDir string, files map[string]string) error {
	for relPath, content := range files {
		target := filepath.Join(workDir, relPath)
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return fmt.Errorf("harness: mkdir %s: %w", filepath.Dir(target), err)
		}
		if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
			return fmt.Errorf("harness: write file %s: %w", target, err)
		}
	}
	return nil
}

// Run executes the complete TDD lifecycle and returns an evaluation report.
func (h *Harness) Run(ctx context.Context, workDir string) (Report, error) {
	start := time.Now()
	timeoutCtx, cancel := context.WithTimeout(ctx, h.cfg.Timeout)
	defer cancel()

	report := Report{
		TurnBudget: h.cfg.TurnBudget,
	}

	// 1. Initial State Check (Red phase check)
	initialResults, initialPassed := h.runLadder(timeoutCtx, workDir)
	report.InitialStateFailed = !initialPassed
	report.LadderResults = initialResults

	var lastFeedback string
	if !initialPassed && len(initialResults) > 0 {
		lastFeedback = initialResults[len(initialResults)-1].Output
	}

	finalPassed := false

	for turn := 1; turn <= h.cfg.TurnBudget; turn++ {
		if timeoutCtx.Err() != nil {
			report.FailureReason = timeoutCtx.Err().Error()
			break
		}
		report.TurnsExecuted = turn

		files, tokens, err := h.model.Generate(timeoutCtx, turn, lastFeedback)
		if err != nil {
			report.FailureReason = fmt.Sprintf("model generation error on turn %d: %v", turn, err)
			break
		}
		report.TotalTokensSpent += tokens

		if !h.cfg.SimulationMode && len(files) > 0 {
			if err := applyFiles(workDir, files); err != nil {
				report.FailureReason = err.Error()
				break
			}
		}

		ladderResults, ladderPassed := h.runLadder(timeoutCtx, workDir)
		report.LadderResults = ladderResults

		if ladderPassed {
			finalPassed = true
			break
		}

		// Self-healing feedback with AST diagnostics
		lastOutput := ladderResults[len(ladderResults)-1].Output
		diags := runner.ParseDiagnostics(lastOutput)
		if len(diags) > 0 {
			var diagLines []string
			for _, d := range diags {
				diagLines = append(diagLines, fmt.Sprintf("%s:%d: %s", d.File, d.Line, d.Message))
			}
			compactFeedback := strings.Join(diagLines, "\n")
			// Tokens saved via AST extraction vs raw full log
			saved := (len(lastOutput) - len(compactFeedback)) / 4
			if saved > 0 {
				report.TokensSaved += saved
			}
			lastFeedback = compactFeedback
		} else {
			lastFeedback = lastOutput
		}
	}

	report.FinalStatePassed = finalPassed
	report.Duration = time.Since(start)

	// Quality gate evaluation
	if h.cfg.QualityGate == QualityGateStrict && !report.InitialStateFailed {
		report.Success = false
		report.FailureReason = "strict quality gate failed: red phase (failing test) was not observed before implementation"
		return report, nil
	}

	report.Success = finalPassed
	if !finalPassed && report.FailureReason == "" {
		report.FailureReason = "turn budget exhausted without passing verification ladder"
	}

	return report, nil
}
