package wailsbind

import (
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// NexusApp exposes orchestrator methods to the Wails/JS frontend.
// All exported methods are automatically bound to the frontend by Wails.
type NexusApp struct {
	orch ports.Orchestrator
}

// New creates a NexusApp bound to the given orchestrator.
func New(orch ports.Orchestrator) *NexusApp {
	return &NexusApp{orch: orch}
}

// SubmitTask forwards a task submission from the UI to the orchestrator.
func (n *NexusApp) SubmitTask(task domain.Task) (string, error) {
	return n.orch.SubmitTask(task)
}

// GetTask retrieves a specific task by ID.
func (n *NexusApp) GetTask(id string) (domain.Task, error) {
	return n.orch.GetTask(id)
}

// GetQueue returns the current task queue for display in the dashboard.
func (n *NexusApp) GetQueue() ([]domain.Task, error) {
	return n.orch.GetQueue()
}

// GetProviders returns the liveness status of all registered LLM backends.
func (n *NexusApp) GetProviders() ([]ports.ProviderInfo, error) {
	return n.orch.GetProviders()
}

// CancelTask cancels a pending task by ID.
func (n *NexusApp) CancelTask(id string) error {
	return n.orch.CancelTask(id)
}
