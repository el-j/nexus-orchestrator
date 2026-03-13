// Package ports defines the hexagonal architecture port interfaces for nexusOrchestrator.
// Inbound ports (Orchestrator) are implemented by core services.
// Outbound ports (LLMClient, TaskRepository, FileWriter, SessionRepository) are implemented by adapters.
package ports

import (
	"context"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

// --- Outbound Ports (Driven Adapters) ---

// LLMClient is the port for any language model backend (LM Studio, Ollama, etc.).
type LLMClient interface {
	Ping() bool
	ProviderName() string
	// ActiveModel returns the model ID currently loaded or configured on this
	// provider.  Returns empty string when unknown.
	ActiveModel() string
	// BaseURL returns the configured endpoint URL for this provider.
	BaseURL() string
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
	ClaimNextQueued() (domain.Task, error)
	// GetByProjectPath returns all tasks for the given project path.
	GetByProjectPath(projectPath string) ([]domain.Task, error)
	UpdateStatus(id string, status domain.TaskStatus) error
	UpdateStatusIfCurrent(id string, from, to domain.TaskStatus) (bool, error)
	// UpdateLogs replaces the Logs field on the task identified by id.
	UpdateLogs(id, logs string) error
	// GetAll returns every task, ordered by creation time descending.
	GetAll() ([]domain.Task, error)
	// GetByProjectPathAndStatus returns tasks for a project filtered by one or more statuses.
	GetByProjectPathAndStatus(projectPath string, statuses ...domain.TaskStatus) ([]domain.Task, error)
	// Update persists changes to an existing task's mutable fields.
	Update(t domain.Task) error
	// GetTasksBySessionID returns all tasks claimed by the given AI session.
	GetTasksBySessionID(sessionID string) ([]domain.Task, error)
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
	BaseURL     string   `json:"baseURL,omitempty"`
	Error       string   `json:"error,omitempty"`
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
	// GetAllTasks returns every task regardless of status.
	GetAllTasks() ([]domain.Task, error)
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
	// AddProviderConfig persists a new provider configuration and registers the
	// adapter when Enabled is true. Returns the saved config with the generated ID.
	AddProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error)
	// UpdateProviderConfig overwrites an existing provider configuration identified
	// by cfg.ID and refreshes the in-process adapter registration.
	UpdateProviderConfig(ctx context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error)
	// RemoveProviderConfig deletes the persisted provider configuration with the
	// given ID and deregisters its adapter from the discovery service.
	RemoveProviderConfig(ctx context.Context, id string) error
	// ListProviderConfigs returns all persisted provider configuration records.
	ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error)
	// GetDiscoveredProviders returns auto-detected AI tools from the local system
	// that have NOT yet been promoted to active/configured providers.
	GetDiscoveredProviders() ([]domain.DiscoveredProvider, error)
	// TriggerScan requests an immediate re-scan and returns the discovered providers.
	TriggerScan(ctx context.Context) ([]domain.DiscoveredProvider, error)
	// PromoteProvider converts a discovered provider into an active registered backend.
	PromoteProvider(ctx context.Context, discoveredID string) error
	// CreateDraft creates a task with StatusDraft. It does NOT enter the execution queue.
	CreateDraft(task domain.Task) (string, error)
	// GetBacklog returns DRAFT and BACKLOG tasks for the given project, ordered by priority then creation time.
	GetBacklog(projectPath string) ([]domain.Task, error)
	// PromoteTask transitions a DRAFT or BACKLOG task to QUEUED and enqueues it.
	PromoteTask(id string) error
	// UpdateTask updates mutable fields (instruction, priority, providerName, tags, status) on an existing task.
	UpdateTask(id string, updates domain.Task) (domain.Task, error)
	// RegisterAISession registers a new external AI agent session and persists it.
	// If ExternalID is set and a session with that ExternalID already exists, it
	// updates the existing session's last activity and returns it (idempotent).
	RegisterAISession(ctx context.Context, s domain.AISession) (domain.AISession, error)
	// ListAISessions returns all persisted AI agent sessions.
	ListAISessions(ctx context.Context) ([]domain.AISession, error)
	// DeregisterAISession marks the session identified by id as disconnected.
	DeregisterAISession(ctx context.Context, id string) error
	// HeartbeatAISession refreshes the last-activity timestamp of a session.
	HeartbeatAISession(ctx context.Context, id string) error
	// ClaimTask assigns a QUEUED task to the given AI session, transitioning it to PROCESSING.
	// Returns domain.ErrNotFound if the task or session does not exist.
	ClaimTask(ctx context.Context, taskID string, sessionID string) (domain.Task, error)
	// UpdateTaskStatus allows an external AI session to report task completion or failure.
	// Only the session that claimed the task (matching AISessionID) may update its status.
	UpdateTaskStatus(ctx context.Context, taskID string, sessionID string, status domain.TaskStatus, logs string) (domain.Task, error)
	// PurgeDisconnectedSessions deletes all AI sessions with status "disconnected"
	// that have been inactive for more than 2 hours. Returns the count deleted.
	PurgeDisconnectedSessions(ctx context.Context) (int, error)
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
	EventTaskDraft      EventType = "task.draft"
	EventTaskBacklog    EventType = "task.backlog"
	EventTaskUpdated    EventType = "task.updated"
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
	BroadcastAISessionEvent(event domain.AISessionEvent)
}

// SystemScanner scans the local system for AI providers/agents.
type SystemScanner interface {
	Scan(ctx context.Context) ([]domain.DiscoveredProvider, error)
}

// ProviderConfigRepository is the outbound port for persisting and querying
// provider configuration records across restarts.
type ProviderConfigRepository interface {
	SaveProviderConfig(ctx context.Context, cfg domain.ProviderConfig) error
	ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error)
	GetProviderConfig(ctx context.Context, id string) (domain.ProviderConfig, error)
	DeleteProviderConfig(ctx context.Context, id string) error
}

// AISessionRepository is the outbound port for persisting AI agent session entities.
type AISessionRepository interface {
	SaveAISession(ctx context.Context, s domain.AISession) error
	GetAISessionByID(ctx context.Context, id string) (domain.AISession, error)
	// GetAISessionByExternalID looks up a session by its external identifier.
	// Returns domain.ErrNotFound when no match exists.
	GetAISessionByExternalID(ctx context.Context, externalID string) (domain.AISession, error)
	ListAISessions(ctx context.Context) ([]domain.AISession, error)
	UpdateAISessionStatus(ctx context.Context, id string, status domain.AISessionStatus, lastActivity time.Time) error
	DeleteAISession(ctx context.Context, id string) error
	// AppendRoutedTaskID adds a task ID to the session's routed task list (no duplicates).
	AppendRoutedTaskID(ctx context.Context, sessionID string, taskID string) error
	// PurgeDisconnected deletes all sessions with status "disconnected" whose
	// last_activity is older than olderThan. Returns the number of rows deleted.
	PurgeDisconnected(ctx context.Context, olderThan time.Duration) (int, error)
}

// AISessionMonitor is the optional inbound port for push-based session discovery adapters.
type AISessionMonitor interface {
	RegisterSession(s domain.AISession) error
	ListActive() ([]domain.AISession, error)
}
