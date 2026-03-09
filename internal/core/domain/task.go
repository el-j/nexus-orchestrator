package domain

import "time"

// TaskStatus represents the lifecycle state of a Task.
type TaskStatus string

const (
	StatusQueued     TaskStatus = "QUEUED"
	StatusProcessing TaskStatus = "PROCESSING"
	StatusCompleted  TaskStatus = "COMPLETED"
	StatusFailed     TaskStatus = "FAILED"
	StatusCancelled  TaskStatus = "CANCELLED"
)

// Task is the central domain entity that represents a single unit of AI work.
type Task struct {
	ID           string
	ProjectPath  string
	TargetFile   string
	Instruction  string
	ContextFiles []string
	Status       TaskStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Logs         string
}
