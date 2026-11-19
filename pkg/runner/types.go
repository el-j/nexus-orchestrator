// Package runner provides a standalone verification gate, invariant enforcers,
// and self-healing task execution engine.
package runner

import (
	"context"
)

// Policy defines invariant limits and rules for verification gates.
type Policy struct {
	// MaxSourceLines is the maximum allowed lines for a source file (default: 500).
	MaxSourceLines int `json:"max_source_lines"`
	// MaxTestLines is the maximum allowed lines for a test file (default: 800).
	MaxTestLines int `json:"max_test_lines"`
	// EnforceTestFirst requires tests to be present or modified alongside code changes.
	EnforceTestFirst bool `json:"enforce_test_first"`
	// VerificationCmd is the command to run for verification (e.g. "go test ./...").
	VerificationCmd string `json:"verification_cmd,omitempty"`
	// SkipPatterns are regex or substring patterns for paths to exclude from verification.
	SkipPatterns []string `json:"skip_patterns,omitempty"`
	// FileExtensions are the extensions to inspect (e.g. [".go", ".ts", ".vue"]).
	FileExtensions []string `json:"file_extensions,omitempty"`
}

// Violation represents an invariant policy violation.
type Violation struct {
	// File is the relative path of the violating file.
	File string `json:"file"`
	// Rule is the identifier of the broken rule.
	Rule string `json:"rule"`
	// Details explains why the rule was violated.
	Details string `json:"details"`
	// LineCount is the recorded line count if applicable.
	LineCount int `json:"line_count,omitempty"`
	// Limit is the policy limit if applicable.
	Limit int `json:"limit,omitempty"`
}

// VerificationResult contains the outcome of invariant and command verification.
type VerificationResult struct {
	// Passed indicates whether all invariants and verification commands succeeded.
	Passed bool `json:"passed"`
	// Command is the verification command that was executed.
	Command string `json:"command,omitempty"`
	// Output is the combined stdout and stderr of the verification command.
	Output string `json:"output,omitempty"`
	// ExitCode is the process exit code of the verification command.
	ExitCode int `json:"exit_code"`
	// DurationMs is the execution time of the verification command in milliseconds.
	DurationMs int64 `json:"duration_ms"`
	// Violations holds any detected policy or invariant violations.
	Violations []Violation `json:"violations,omitempty"`
}

// Generator provides code generation capability for task execution and healing.
type Generator interface {
	// Generate generates code or a response for the given prompt.
	Generate(ctx context.Context, prompt string) (string, error)
	// ProviderName returns the name of the underlying model provider.
	ProviderName() string
}

// ExecuteConfig configures the task execution and self-healing loop.
type ExecuteConfig struct {
	// TaskID is an optional identifier for the task.
	TaskID string `json:"task_id,omitempty"`
	// TaskFilePath is an optional path to a task file (e.g. .claude/tasks/TASK-NNN.md).
	TaskFilePath string `json:"task_file_path,omitempty"`
	// Instruction is the prompt or instructions for the generator.
	Instruction string `json:"instruction,omitempty"`
	// TargetFile is the destination file where generated code will be written.
	TargetFile string `json:"target_file,omitempty"`
	// VerificationCmd is the command executed to verify the generated code.
	VerificationCmd string `json:"verification_cmd,omitempty"`
	// MaxTurns is the maximum number of correction turns (default: 2).
	MaxTurns int `json:"max_turns,omitempty"`
	// Heal enables the self-healing correction loop upon verification failure.
	Heal bool `json:"heal,omitempty"`
	// ProjectPath is the root path of the project.
	ProjectPath string `json:"project_path,omitempty"`
}

// ExecuteResult represents the outcome of task execution and healing.
type ExecuteResult struct {
	// Success indicates whether the task and all verifications succeeded.
	Success bool `json:"success"`
	// Output is the final generated code or result.
	Output string `json:"output,omitempty"`
	// Turns is the total number of execution turns performed.
	Turns int `json:"turns"`
	// Violations contains any invariant policy violations.
	Violations []Violation `json:"violations,omitempty"`
	// Verification holds the final verification command result.
	Verification VerificationResult `json:"verification"`
	// Logs contains execution events and healing turn records.
	Logs []string `json:"logs,omitempty"`
}

// InspectionReport details the state of project invariants, files, and git status.
type InspectionReport struct {
	// ProjectPath is the root directory that was inspected.
	ProjectPath string `json:"project_path"`
	// TotalFiles is the count of scanned files.
	TotalFiles int `json:"total_files"`
	// SourceFiles is the count of non-test source files.
	SourceFiles int `json:"source_files"`
	// TestFiles is the count of test files.
	TestFiles int `json:"test_files"`
	// Violations lists any policy or invariant violations found.
	Violations []Violation `json:"violations"`
	// GitStatus summarizes the git working tree state if available.
	GitStatus string `json:"git_status,omitempty"`
	// Passed indicates whether the project meets all invariants.
	Passed bool `json:"passed"`
}
