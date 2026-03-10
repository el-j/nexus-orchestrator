package ports

import "nexus-orchestrator/internal/core/domain"

// --- Outbound Ports (Driven Adapters) ---

// LLMClient is the port for any language model backend (LM Studio, Ollama, etc.).
type LLMClient interface {
	Ping() bool
	ProviderName() string
	// ActiveModel returns the model ID currently loaded or configured on this
	// provider.  Returns empty string when unknown.
	ActiveModel() string
	GetAvailableModels() ([]string, error)
	// ContextLimit returns the maximum number of input tokens the currently
	// loaded model can accept.  Returns 0 if unknown; when 0, no pre-flight
	// token check is performed.
	ContextLimit() int
	// GenerateCode sends a single prompt and returns generated text.
	// Prefer Chat for multi-turn session-aware generation.
	GenerateCode(prompt string) (string, error)
	// Chat sends a full conversation history and returns the assistant reply.
	// Used by OrchestratorService for per-project session isolation.
	Chat(messages []domain.Message) (string, error)
}

// TaskRepository is the port for persisting and querying Tasks.
type TaskRepository interface {
	Save(t domain.Task) error
	GetByID(id string) (domain.Task, error)
	GetPending() ([]domain.Task, error)
	UpdateStatus(id string, status domain.TaskStatus) error
	// UpdateLogs replaces the Logs field on the task identified by id.
	UpdateLogs(id, logs string) error
}

// FileWriter is the port for reading context from disk and writing generated code back.
type FileWriter interface {
	WriteCodeToFile(projectPath, targetFile, code string) error
	ReadContextFiles(projectPath string, files []string) (string, error)
}

// --- Inbound Ports (Driving Adapters) ---

// ProviderInfo summarises the liveness status of a single LLM backend.
type ProviderInfo struct {
	Name        string   `json:"name"`
	Active      bool     `json:"active"`
	ActiveModel string   `json:"activeModel,omitempty"`
	Models      []string `json:"models,omitempty"`
}

// SessionRepository is the port for persisting per-project conversation history.
// ProjectPath (filepath.Clean'd) is the isolation key.
type SessionRepository interface {
	Save(s domain.Session) error
	// GetByProjectPath returns the session for the given project, or domain.ErrNotFound.
	GetByProjectPath(projectPath string) (domain.Session, error)
	// AppendMessage adds a message to the session for projectPath.
	// If no session exists for that path, one is created automatically.
	AppendMessage(projectPath string, msg domain.Message) error
}

// Orchestrator is the primary inbound port that the UI, CLI, and HTTP API call.
type Orchestrator interface {
	SubmitTask(task domain.Task) (string, error)
	// GetTask returns the task with the given ID, or domain.ErrNotFound.
	GetTask(id string) (domain.Task, error)
	GetQueue() ([]domain.Task, error)
	// GetProviders returns a snapshot of all registered LLM backends and their liveness.
	GetProviders() ([]ProviderInfo, error)
	CancelTask(id string) error
	// RegisterCloudProvider dynamically adds a new LLM backend using the supplied
	// configuration. Returns an error if the kind is unknown or the name is already
	// registered.
	RegisterCloudProvider(cfg domain.ProviderConfig) error
	// RemoveProvider deregisters the provider with the given name.
	// Returns domain.ErrNotFound when no provider with that name exists.
	RemoveProvider(providerName string) error
	// GetProviderModels returns the model catalogue of the named provider.
	// Returns domain.ErrNotFound when no provider with that name exists.
	GetProviderModels(providerName string) ([]string, error)
}

// EventType identifies a task lifecycle event.
type EventType string

const (
	EventTaskQueued     EventType = "task.queued"
	EventTaskProcessing EventType = "task.processing"
	EventTaskCompleted  EventType = "task.completed"
	EventTaskFailed     EventType = "task.failed"
	EventTaskCancelled  EventType = "task.cancelled"
	EventTaskTooLarge   EventType = "task.too_large"
	EventTaskNoProvider EventType = "task.no_provider"
)

// TaskEvent is emitted by OrchestratorService on task lifecycle changes.
type TaskEvent struct {
	Type   EventType         `json:"type"`
	TaskID string            `json:"taskId"`
	Status domain.TaskStatus `json:"status"`
}

// EventBroadcaster is the optional outbound port for publishing task lifecycle events.
// It must be safe for concurrent use. Implementations must be non-blocking.
type EventBroadcaster interface {
	Broadcast(event TaskEvent)
}
