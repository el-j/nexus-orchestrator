package domain

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a task cannot be found by its ID.
var ErrNotFound = errors.New("task not found")

// ErrNoPlan is returned when an execute task is submitted but no prior plan exists for the project.
var ErrNoPlan = errors.New("no plan exists; planning required before execution")

// TaskStatus represents the lifecycle state of a Task.
type TaskStatus string

const (
	StatusQueued     TaskStatus = "QUEUED"
	StatusProcessing TaskStatus = "PROCESSING"
	StatusCompleted  TaskStatus = "COMPLETED"
	StatusFailed     TaskStatus = "FAILED"
	StatusCancelled  TaskStatus = "CANCELLED"
	// StatusTooLarge is set when the assembled prompt exceeds the loaded model's
	// context window.  The task is never sent to the LLM; the user should shorten
	// the instruction or reduce the number of context files.
	StatusTooLarge TaskStatus = "TOO_LARGE"
	// StatusNoProvider is set when no registered LLM provider has the requested
	// model available or all providers are unreachable.  The task is never sent
	// to any LLM; the user should choose a different model or start a provider.
	StatusNoProvider TaskStatus = "NO_PROVIDER"
)

// String returns the underlying string value of the TaskStatus.
func (s TaskStatus) String() string { return string(s) }

// CommandType classifies a task as planning work, execution work, or auto-routed.
type CommandType string

const (
	// CommandPlan indicates this task is for planning/orchestration (creating plans, task docs).
	CommandPlan CommandType = "plan"
	// CommandExecute indicates this task is for code implementation.
	CommandExecute CommandType = "execute"
	// CommandAuto lets the orchestrator decide (default when empty or unspecified).
	CommandAuto CommandType = "auto"
)

// String returns the underlying string value of the CommandType.
func (c CommandType) String() string { return string(c) }

// IsValid returns true for the three valid command types and empty string (treated as auto).
func (c CommandType) IsValid() bool {
	switch c {
	case CommandPlan, CommandExecute, CommandAuto, "":
		return true
	default:
		return false
	}
}

// Task is the central domain entity that represents a single unit of AI work.
type Task struct {
	ID           string   `json:"id"`
	ProjectPath  string   `json:"projectPath"`
	TargetFile   string   `json:"targetFile"`
	Instruction  string   `json:"instruction"`
	ContextFiles []string `json:"contextFiles"`
	// ModelID constrains which LLM model must process this task.
	// Empty means "use whatever the active provider has loaded".
	ModelID string `json:"modelId,omitempty"`
	// ProviderHint is a preference (provider name) when multiple providers carry
	// the same model.  Empty means no preference.
	ProviderHint string `json:"providerHint,omitempty"`
	// Command classifies the task as planning, execution, or auto-routed.
	// Empty is treated as CommandAuto.
	Command   CommandType `json:"command,omitempty"`
	Status    TaskStatus  `json:"status"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	Logs      string      `json:"logs,omitempty"`
}
