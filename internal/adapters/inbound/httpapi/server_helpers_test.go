package httpapi_test

import (
	"context"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// mockOrchestrator implements ports.Orchestrator with configurable responses.
// All fields are set at construction time and only read during handler execution,
// so no additional locking is required for these tests.
type mockOrchestrator struct {
	submitTaskID  string
	submitTaskErr error

	getTaskResult domain.Task
	getTaskErr    error

	getQueueResult []domain.Task
	getQueueErr    error

	getProvidersResult []ports.ProviderInfo
	getProvidersErr    error

	cancelTaskErr error

	registerProviderErr error

	removeProviderErr error

	getProviderModelsResult []string
	getProviderModelsErr    error

	createDraftID  string
	createDraftErr error

	getBacklogResult []domain.Task
	getBacklogErr    error

	promoteTaskErr error

	updateTaskResult domain.Task
	updateTaskErr    error

	discoveredProvidersResult []domain.DiscoveredProvider
	promoteProviderErr        error

	terminateAISessionErr error

	heartbeatAISessionErr error

	heartbeatTaskErr error

	claimTaskResult domain.Task
	claimTaskErr    error

	updateTaskStatusResult domain.Task
	updateTaskStatusErr    error

	delegateInstruction string
	delegateErr         error

	purgeCount int
	purgeErr   error

	addProviderConfigResult domain.ProviderConfig
	addProviderConfigErr    error

	updateProviderConfigErr error

	removeProviderConfigErr error

	listProviderConfigsResult []domain.ProviderConfig
	listProviderConfigsErr    error
}

func (m *mockOrchestrator) SubmitTask(_ domain.Task) (string, error) {
	return m.submitTaskID, m.submitTaskErr
}

func (m *mockOrchestrator) GetTask(_ string) (domain.Task, error) {
	return m.getTaskResult, m.getTaskErr
}

func (m *mockOrchestrator) GetQueue() ([]domain.Task, error) {
	return m.getQueueResult, m.getQueueErr
}

func (m *mockOrchestrator) GetQueueForProject(projectPath string) ([]domain.Task, error) {
	var out []domain.Task
	for _, t := range m.getQueueResult {
		if t.ProjectPath == projectPath {
			out = append(out, t)
		}
	}
	return out, m.getQueueErr
}

func (m *mockOrchestrator) GetAllTasks() ([]domain.Task, error) {
	return m.getQueueResult, m.getQueueErr
}

func (m *mockOrchestrator) GetTasksForProject(projectPath string) ([]domain.Task, error) {
	var out []domain.Task
	for _, t := range m.getQueueResult {
		if t.ProjectPath == projectPath {
			out = append(out, t)
		}
	}
	return out, m.getQueueErr
}

func (m *mockOrchestrator) GetProviders() ([]ports.ProviderInfo, error) {
	return m.getProvidersResult, m.getProvidersErr
}

func (m *mockOrchestrator) CancelTask(_ string) error {
	return m.cancelTaskErr
}

func (m *mockOrchestrator) RegisterCloudProvider(_ domain.ProviderConfig) error {
	return m.registerProviderErr
}

func (m *mockOrchestrator) RemoveProvider(_ string) error {
	return m.removeProviderErr
}

func (m *mockOrchestrator) GetProviderModels(_ string) ([]string, error) {
	return m.getProviderModelsResult, m.getProviderModelsErr
}

func (m *mockOrchestrator) AddProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	if m.addProviderConfigErr != nil {
		return domain.ProviderConfig{}, m.addProviderConfigErr
	}
	if m.addProviderConfigResult.Name != "" {
		return m.addProviderConfigResult, nil
	}
	return cfg, nil
}

func (m *mockOrchestrator) UpdateProviderConfig(_ context.Context, cfg domain.ProviderConfig) (domain.ProviderConfig, error) {
	return cfg, m.updateProviderConfigErr
}

func (m *mockOrchestrator) RemoveProviderConfig(_ context.Context, _ string) error {
	return m.removeProviderConfigErr
}

func (m *mockOrchestrator) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return m.listProviderConfigsResult, m.listProviderConfigsErr
}

func (m *mockOrchestrator) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return m.discoveredProvidersResult, nil
}

func (m *mockOrchestrator) TriggerScan(_ context.Context) ([]domain.DiscoveredProvider, error) {
	return nil, nil
}

func (m *mockOrchestrator) PromoteProvider(_ context.Context, _ string) error {
	return m.promoteProviderErr
}

// configurable fields for backlog operations
var _ ports.Orchestrator = (*mockOrchestrator)(nil)

func (m *mockOrchestrator) CreateDraft(_ domain.Task) (string, error) {
	return m.createDraftID, m.createDraftErr
}

func (m *mockOrchestrator) GetBacklog(_ string) ([]domain.Task, error) {
	return m.getBacklogResult, m.getBacklogErr
}

func (m *mockOrchestrator) PromoteTask(_ string) (ports.PromoteResult, error) {
	return ports.PromoteResult{Promoted: m.promoteTaskErr == nil}, m.promoteTaskErr
}

func (m *mockOrchestrator) UpdateTask(_ string, _ domain.Task) (domain.Task, error) {
	return m.updateTaskResult, m.updateTaskErr
}

func (m *mockOrchestrator) RegisterAISession(_ context.Context, s domain.AISession) (domain.AISession, error) {
	return s, nil
}

func (m *mockOrchestrator) ListAISessions(_ context.Context) ([]domain.AISession, error) {
	return nil, nil
}

func (m *mockOrchestrator) DeregisterAISession(_ context.Context, _ string) error {
	return nil
}

func (m *mockOrchestrator) HeartbeatAISession(_ context.Context, _ string) error {
	return m.heartbeatAISessionErr
}

func (m *mockOrchestrator) HeartbeatTask(_ context.Context, _, _ string) error {
	return m.heartbeatTaskErr
}

func (m *mockOrchestrator) ClaimTask(_ context.Context, _ string, _ string) (domain.Task, error) {
	return m.claimTaskResult, m.claimTaskErr
}

func (m *mockOrchestrator) UpdateTaskStatus(_ context.Context, _ string, _ string, _ domain.TaskStatus, _ string) (domain.Task, error) {
	return m.updateTaskStatusResult, m.updateTaskStatusErr
}

func (m *mockOrchestrator) PurgeDisconnectedSessions(_ context.Context) (int, error) {
	return m.purgeCount, m.purgeErr
}

func (m *mockOrchestrator) GetDiscoveredAgents(_ context.Context) ([]domain.DiscoveredAgent, error) {
	return nil, nil
}

func (m *mockOrchestrator) DelegateToNexus(_ context.Context, _ string) (string, error) {
	return m.delegateInstruction, m.delegateErr
}

func (m *mockOrchestrator) TerminateAISession(_ context.Context, _ string, _ bool) error {
	return m.terminateAISessionErr
}

func (m *mockOrchestrator) GetDiscoveredPlanFiles(_ context.Context, _ string) ([]domain.DiscoveredPlanFile, error) {
	return nil, nil
}

func (m *mockOrchestrator) GetRuntimeConfig(_ context.Context) (domain.RuntimeConfig, error) {
	return domain.RuntimeConfig{QueueCap: 50}, nil
}

func (m *mockOrchestrator) UpdateRuntimeConfig(_ context.Context, update domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	cfg := domain.RuntimeConfig{QueueCap: 50}
	if update.QueueCap != nil {
		cfg.QueueCap = *update.QueueCap
	}
	if update.APIToken != nil {
		cfg.APIToken = *update.APIToken
	}
	if update.MCPToken != nil {
		cfg.MCPToken = *update.MCPToken
	}
	return cfg, nil
}
