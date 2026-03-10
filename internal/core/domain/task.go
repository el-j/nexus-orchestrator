package domain

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a task cannot be found by its ID.
var ErrNotFound = errors.New("task not found")

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

// Task is the central domain entity that represents a single unit of AI work.
type Task struct {
	ID           string     `json:"id"`
	ProjectPath  string     `json:"projectPath"`
	TargetFile   string     `json:"targetFile"`
	Instruction  string     `json:"instruction"`
	ContextFiles []string   `json:"contextFiles"`
	// ModelID constrains which LLM model must process this task.
	// Empty means "use whatever the active provider has loaded".
	ModelID      string `json:"modelId,omitempty"`
	// ProviderHint is a preference (provider name) when multiple providers carry
	// the same model.  Empty means no preference.
	ProviderHint string `json:"providerHint,omitempty"`
	Status       TaskStatus `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Logs         string     `json:"logs,omitempty"`
}
