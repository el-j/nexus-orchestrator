package main

import (
	"context"
	"fmt"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails application struct. Its exported methods are bound to the
// JavaScript frontend.
type App struct {
	ctx          context.Context
	orchestrator ports.Orchestrator
	brainSvc     ports.BrainService
	httpAddr     string
	activitySvc  ports.ActivityServicePort
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

// ShowWindow reveals the desktop window if the Wails runtime is ready.
func (a *App) ShowWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
}

// QuitApp requests a graceful Wails shutdown so deferred cleanup can run.
func (a *App) QuitApp() {
	if a.ctx == nil {
		return
	}
	runtime.Quit(a.ctx)
}

// SubmitTask forwards a task from the frontend to the orchestrator.
func (a *App) SubmitTask(task domain.Task) (string, error) {
	return a.orchestrator.SubmitTask(task)
}

// GetTask retrieves a specific task by ID.
func (a *App) GetTask(id string) (domain.Task, error) {
	t, err := a.orchestrator.GetTask(id)
	if err != nil {
		return t, err
	}
	t.ComputeDuration()
	return t, nil
}

// GetQueue returns all pending tasks for the dashboard.
func (a *App) GetQueue() ([]domain.Task, error) {
	tasks, err := a.orchestrator.GetQueue()
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].ComputeDuration()
	}
	return tasks, nil
}

// GetAllTasks returns every task regardless of status.
func (a *App) GetAllTasks() ([]domain.Task, error) {
	tasks, err := a.orchestrator.GetAllTasks()
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].ComputeDuration()
	}
	return tasks, nil
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
	tasks, err := a.orchestrator.GetBacklog(projectPath)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].ComputeDuration()
	}
	return tasks, nil
}

// PromoteTask transitions a DRAFT or BACKLOG task to QUEUED and enqueues it for execution.
func (a *App) PromoteTask(id string) (ports.PromoteResult, error) {
	return a.orchestrator.PromoteTask(id)
}

// UpdateTask updates mutable fields on an existing task.
func (a *App) UpdateTask(id string, updates domain.Task) (domain.Task, error) {
	t, err := a.orchestrator.UpdateTask(id, updates)
	if err != nil {
		return t, err
	}
	t.ComputeDuration()
	return t, nil
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

// PurgeDisconnectedSessions deletes all AI sessions that are disconnected and
// older than 2 hours. Returns the number of sessions purged.
func (a *App) PurgeDisconnectedSessions() (int, error) {
	n, err := a.orchestrator.PurgeDisconnectedSessions(context.Background())
	if err != nil {
		return 0, fmt.Errorf("app: purge disconnected sessions: %w", err)
	}
	return n, nil
}

// ClaimTask assigns a QUEUED task to the given AI session.
func (a *App) ClaimTask(taskID string, sessionID string) (domain.Task, error) {
	t, err := a.orchestrator.ClaimTask(context.Background(), taskID, sessionID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("app: claim task: %w", err)
	}
	t.ComputeDuration()
	return t, nil
}

// UpdateTaskStatus allows an external AI session to report task completion or failure.
func (a *App) UpdateTaskStatus(taskID string, sessionID string, status string, logs string) (domain.Task, error) {
	t, err := a.orchestrator.UpdateTaskStatus(context.Background(), taskID, sessionID, domain.TaskStatus(status), logs)
	if err != nil {
		return domain.Task{}, fmt.Errorf("app: update task status: %w", err)
	}
	t.ComputeDuration()
	return t, nil
}

// withActivityService injects an ActivityServicePort into the App for observatory bindings.
func (a *App) withActivityService(svc ports.ActivityServicePort) *App {
	a.activitySvc = svc
	return a
}

// withBrainService injects a BrainService into the App for project knowledge intelligence.
func (a *App) withBrainService(svc ports.BrainService) *App {
	a.brainSvc = svc
	return a
}

// IngestKnowledge processes a markdown file to be added to the project's brain.
func (a *App) IngestKnowledge(projectPath, filePath string) (int, error) {
	if a.brainSvc == nil {
		return 0, fmt.Errorf("brain service not available")
	}
	return a.brainSvc.IngestFromFile(context.Background(), projectPath, filePath)
}

// GetBrainStatus retrieves the state of the project knowledge store.
func (a *App) GetBrainStatus(projectPath string) (domain.BrainStatus, error) {
	if a.brainSvc == nil {
		return domain.BrainStatus{}, fmt.Errorf("brain service not available")
	}
	return a.brainSvc.GetStatus(context.Background(), projectPath)
}

// GetProjectContext fetches top-level context for the project footprint.
func (a *App) GetProjectContext(projectPath string, maxTokens int) (domain.ContextResponse, error) {
	if a.brainSvc == nil {
		return domain.ContextResponse{}, fmt.Errorf("brain service not available")
	}
	return a.brainSvc.GetContext(context.Background(), domain.ContextQuery{
		ProjectPath: projectPath,
		MaxTokens:   maxTokens,
	})
}

// GetFocusedContext searches the brain context relative to a specific reasoning query.
func (a *App) GetFocusedContext(projectPath, question string, maxTokens int) (domain.ContextResponse, error) {
	if a.brainSvc == nil {
		return domain.ContextResponse{}, fmt.Errorf("brain service not available")
	}
	return a.brainSvc.GetFocusedContext(context.Background(), domain.ContextQuery{
		ProjectPath: projectPath,
		Question:    question,
		MaxTokens:   maxTokens,
	})
}

// SearchKnowledge allows full text search on the sqlite index store.
func (a *App) SearchKnowledge(projectPath, query string, limit int) ([]domain.ContextSection, error) {
	if a.brainSvc == nil {
		return nil, fmt.Errorf("brain service not available")
	}
	return a.brainSvc.SearchKnowledge(context.Background(), projectPath, query, limit)
}

// InitProject initialises the project brain, optionally seeding from a CLAUDE.md file.
func (a *App) InitProject(projectPath, claudeMDPath string) (domain.BrainStatus, error) {
	if a.brainSvc == nil {
		return domain.BrainStatus{}, fmt.Errorf("brain service not available")
	}
	return a.brainSvc.InitProject(context.Background(), projectPath, claudeMDPath)
}

// ListKnowledge returns all knowledge entries for a project, optionally filtered by kind.
func (a *App) ListKnowledge(projectPath, kind string) ([]domain.ProjectKnowledge, error) {
	if a.brainSvc == nil {
		return nil, fmt.Errorf("brain service not available")
	}
	return a.brainSvc.ListKnowledge(context.Background(), projectPath, kind)
}

// DeleteKnowledge removes a knowledge entry by ID.
func (a *App) DeleteKnowledge(id string) error {
	if a.brainSvc == nil {
		return fmt.Errorf("brain service not available")
	}
	return a.brainSvc.DeleteKnowledge(context.Background(), id)
}

// GetFileMap returns the ordered list of file paths from file_map knowledge entries.
func (a *App) GetFileMap(projectPath, focusArea string) ([]string, error) {
	if a.brainSvc == nil {
		return nil, fmt.Errorf("brain service not available")
	}
	return a.brainSvc.GetFileMap(context.Background(), projectPath, focusArea)
}

// GetRecentActivities returns recent AI activities for the observatory dashboard.
func (a *App) GetRecentActivities(agentName, projectPath, activityType, sinceRFC3339 string, limit int) ([]domain.AIActivity, error) {
	if a.activitySvc == nil {
		return nil, fmt.Errorf("app: activity service not available")
	}
	f := domain.ActivityFilter{
		AgentName:   agentName,
		ProjectPath: projectPath,
	}
	if activityType != "" {
		f.Type = domain.ActivityType(activityType)
	}
	if sinceRFC3339 != "" {
		if t, err := time.Parse(time.RFC3339, sinceRFC3339); err == nil {
			f.Since = t
		}
	}
	if limit > 0 {
		f.Limit = limit
	}
	return a.activitySvc.GetRecentActivities(context.Background(), f)
}

// GetActivityTimeline returns a cross-agent chronological activity feed.
func (a *App) GetActivityTimeline(sinceRFC3339 string, limit int) ([]domain.AIActivity, error) {
	if a.activitySvc == nil {
		return nil, fmt.Errorf("app: activity service not available")
	}
	since := time.Now().Add(-24 * time.Hour)
	if sinceRFC3339 != "" {
		if t, err := time.Parse(time.RFC3339, sinceRFC3339); err == nil {
			since = t
		}
	}
	if limit == 0 {
		limit = 100
	}
	return a.activitySvc.GetTimeline(context.Background(), since, limit)
}

// OpenFileDialog opens a native file picker dialog and returns the selected file path.
// Returns an empty string if the user cancels.
func (a *App) OpenFileDialog(title string, filters []map[string]string) (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app: context not initialised")
	}
	var dialogFilters []runtime.FileFilter
	for _, f := range filters {
		dialogFilters = append(dialogFilters, runtime.FileFilter{
			DisplayName: f["displayName"],
			Pattern:     f["pattern"],
		})
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: dialogFilters,
	})
	if err != nil {
		return "", fmt.Errorf("app: open file dialog: %w", err)
	}
	return path, nil
}

// GetRuntimeConfig returns the current effective runtime configuration.
func (a *App) GetRuntimeConfig() (domain.RuntimeConfig, error) {
	cfg, err := a.orchestrator.GetRuntimeConfig(context.Background())
	if err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("app: get runtime config: %w", err)
	}
	return cfg, nil
}

// UpdateRuntimeConfig applies a partial update to the runtime configuration.
func (a *App) UpdateRuntimeConfig(update domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	cfg, err := a.orchestrator.UpdateRuntimeConfig(context.Background(), update)
	if err != nil {
		return domain.RuntimeConfig{}, fmt.Errorf("app: update runtime config: %w", err)
	}
	return cfg, nil
}
