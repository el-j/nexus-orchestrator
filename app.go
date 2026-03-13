package main

import (
	"context"
	"fmt"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// App is the Wails application struct. Its exported methods are bound to the
// JavaScript frontend.
type App struct {
	ctx          context.Context
	orchestrator ports.Orchestrator
	httpAddr     string
}

// NewApp creates a new App instance.
func NewApp(orch ports.Orchestrator, httpAddr string) *App {
	return &App{orchestrator: orch, httpAddr: httpAddr}
}

// GetServerAddr returns the base HTTP URL of the embedded API server so the
// frontend can derive EventSource and fetch URLs without hardcoding the port.
func (a *App) GetServerAddr() string {
	return "http://" + a.httpAddr
}

// startup is called by Wails when the application starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SubmitTask forwards a task from the frontend to the orchestrator.
func (a *App) SubmitTask(task domain.Task) (string, error) {
	return a.orchestrator.SubmitTask(task)
}

// GetTask retrieves a specific task by ID.
func (a *App) GetTask(id string) (domain.Task, error) {
	return a.orchestrator.GetTask(id)
}

// GetQueue returns all pending tasks for the dashboard.
func (a *App) GetQueue() ([]domain.Task, error) {
	return a.orchestrator.GetQueue()
}

// GetAllTasks returns every task regardless of status.
func (a *App) GetAllTasks() ([]domain.Task, error) {
	return a.orchestrator.GetAllTasks()
}

// GetProviders returns the status of all registered LLM backends.
func (a *App) GetProviders() ([]ports.ProviderInfo, error) {
	return a.orchestrator.GetProviders()
}

// CancelTask cancels a queued task.
func (a *App) CancelTask(id string) error {
	return a.orchestrator.CancelTask(id)
}

// RegisterCloudProvider adds a new LLM backend from the UI.
func (a *App) RegisterCloudProvider(cfg domain.ProviderConfig) error {
	return a.orchestrator.RegisterCloudProvider(cfg)
}

// RemoveProvider deregisters the named LLM backend.
func (a *App) RemoveProvider(name string) error {
	return a.orchestrator.RemoveProvider(name)
}

// GetProviderModels returns the model catalogue of the named provider.
func (a *App) GetProviderModels(providerName string) ([]string, error) {
	return a.orchestrator.GetProviderModels(providerName)
}

// AddProviderConfig persists a new provider configuration and registers its adapter.
func (a *App) AddProviderConfig(cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return a.orchestrator.AddProviderConfig(context.Background(), cfg)
}

// ListProviderConfigs returns all persisted provider configuration records.
func (a *App) ListProviderConfigs() ([]domain.ProviderConfig, error) {
	return a.orchestrator.ListProviderConfigs(context.Background())
}

// UpdateProviderConfig overwrites an existing provider configuration.
func (a *App) UpdateProviderConfig(cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return a.orchestrator.UpdateProviderConfig(context.Background(), cfg)
}

// RemoveProviderConfig deletes a persisted provider configuration by ID.
func (a *App) RemoveProviderConfig(id string) error {
	return a.orchestrator.RemoveProviderConfig(context.Background(), id)
}

// Greet is the default Wails example method — kept for scaffolding compatibility.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello, %s! nexusOrchestrator is running.", name)
}

// GetDiscoveredProviders returns system-detected AI tools (not yet promoted).
// The frontend calls this to populate the Discovery panel.
func (a *App) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return a.orchestrator.GetDiscoveredProviders()
}

// TriggerScan requests an immediate system scan for AI providers.
func (a *App) TriggerScan() error {
	_, err := a.orchestrator.TriggerScan(context.Background())
	return err
}

// CreateDraft creates a task idea (DRAFT status) without entering the execution queue.
func (a *App) CreateDraft(task domain.Task) (string, error) {
	return a.orchestrator.CreateDraft(task)
}

// GetBacklog returns DRAFT and BACKLOG tasks for the given project path.
func (a *App) GetBacklog(projectPath string) ([]domain.Task, error) {
	return a.orchestrator.GetBacklog(projectPath)
}

// PromoteTask transitions a DRAFT or BACKLOG task to QUEUED and enqueues it for execution.
func (a *App) PromoteTask(id string) error {
	return a.orchestrator.PromoteTask(id)
}

// UpdateTask updates mutable fields on an existing task.
func (a *App) UpdateTask(id string, updates domain.Task) (domain.Task, error) {
	return a.orchestrator.UpdateTask(id, updates)
}

// ListAISessions returns all registered AI sessions.
func (a *App) ListAISessions() ([]domain.AISession, error) {
	sessions, err := a.orchestrator.ListAISessions(context.Background())
	if err != nil {
		return nil, fmt.Errorf("app: list ai sessions: %w", err)
	}
	return sessions, nil
}

// RegisterAISession registers a new AI session and returns it with a server-assigned ID.
func (a *App) RegisterAISession(session domain.AISession) (domain.AISession, error) {
	result, err := a.orchestrator.RegisterAISession(context.Background(), session)
	if err != nil {
		return domain.AISession{}, fmt.Errorf("app: register ai session: %w", err)
	}
	return result, nil
}

// DeregisterAISession removes the AI session with the given ID.
func (a *App) DeregisterAISession(id string) error {
	if err := a.orchestrator.DeregisterAISession(context.Background(), id); err != nil {
		return fmt.Errorf("app: deregister ai session: %w", err)
	}
	return nil
}

// HeartbeatAISession refreshes the last-activity timestamp of the AI session with the given ID.
func (a *App) HeartbeatAISession(id string) error {
	if err := a.orchestrator.HeartbeatAISession(context.Background(), id); err != nil {
		return fmt.Errorf("app: heartbeat ai session: %w", err)
	}
	return nil
}
