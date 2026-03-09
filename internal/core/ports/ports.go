package ports

import "nexus-ai/internal/core/domain"

// --- Outbound Ports (Driven Adapters) ---

// LLMClient is the port for any language model backend (LM Studio, Ollama, etc.).
type LLMClient interface {
	Ping() bool
	ProviderName() string
	GetAvailableModels() ([]string, error)
	GenerateCode(prompt string) (string, error)
}

// TaskRepository is the port for persisting and querying Tasks.
type TaskRepository interface {
	Save(t domain.Task) error
	GetByID(id string) (domain.Task, error)
	GetPending() ([]domain.Task, error)
	UpdateStatus(id string, status domain.TaskStatus) error
}

// FileWriter is the port for reading context from disk and writing generated code back.
type FileWriter interface {
	WriteCodeToFile(projectPath, targetFile, code string) error
	ReadContextFiles(projectPath string, files []string) (string, error)
}

// --- Inbound Ports (Driving Adapters) ---

// Orchestrator is the primary inbound port that the UI, CLI, and HTTP API call.
type Orchestrator interface {
	SubmitTask(task domain.Task) (string, error)
	GetQueue() ([]domain.Task, error)
	CancelTask(id string) error
}
