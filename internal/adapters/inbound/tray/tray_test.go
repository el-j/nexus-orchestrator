package tray_test

import (
	"context"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/tray"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

type mockOrchForTray struct{}

func (m *mockOrchForTray) SubmitTask(domain.Task) (string, error)      { return "", nil }
func (m *mockOrchForTray) GetTask(string) (domain.Task, error)         { return domain.Task{}, nil }
func (m *mockOrchForTray) GetQueue() ([]domain.Task, error)            { return nil, nil }
func (m *mockOrchForTray) GetAllTasks() ([]domain.Task, error)         { return nil, nil }
func (m *mockOrchForTray) GetProviders() ([]ports.ProviderInfo, error) { return nil, nil }
func (m *mockOrchForTray) GetRuntimeConfig(context.Context) (domain.RuntimeConfig, error) {
	return domain.RuntimeConfig{}, nil
}
func (m *mockOrchForTray) UpdateRuntimeConfig(context.Context, domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	return domain.RuntimeConfig{}, nil
}
func (m *mockOrchForTray) CancelTask(string) error                           { return nil }
func (m *mockOrchForTray) RegisterCloudProvider(domain.ProviderConfig) error { return nil }
func (m *mockOrchForTray) RemoveProvider(string) error                       { return nil }
func (m *mockOrchForTray) GetProviderModels(string) ([]string, error)        { return nil, nil }
func (m *mockOrchForTray) AddProviderConfig(context.Context, domain.ProviderConfig) (domain.ProviderConfig, error) {
	return domain.ProviderConfig{}, nil
}
func (m *mockOrchForTray) UpdateProviderConfig(context.Context, domain.ProviderConfig) (domain.ProviderConfig, error) {
	return domain.ProviderConfig{}, nil
}
func (m *mockOrchForTray) RemoveProviderConfig(context.Context, string) error { return nil }
func (m *mockOrchForTray) ListProviderConfigs(context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}
func (m *mockOrchForTray) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *mockOrchForTray) TriggerScan(context.Context) ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *mockOrchForTray) PromoteProvider(context.Context, string) error { return nil }
func (m *mockOrchForTray) CreateDraft(domain.Task) (string, error) {
	return "", nil
}
func (m *mockOrchForTray) GetBacklog(string) ([]domain.Task, error) {
	return nil, nil
}
func (m *mockOrchForTray) PromoteTask(string) (ports.PromoteResult, error) {
	return ports.PromoteResult{}, nil
}
func (m *mockOrchForTray) UpdateTask(string, domain.Task) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchForTray) RegisterAISession(context.Context, domain.AISession) (domain.AISession, error) {
	return domain.AISession{}, nil
}
func (m *mockOrchForTray) ListAISessions(context.Context) ([]domain.AISession, error) {
	return nil, nil
}
func (m *mockOrchForTray) DeregisterAISession(context.Context, string) error { return nil }
func (m *mockOrchForTray) TerminateAISession(context.Context, string, bool) error {
	return nil
}
func (m *mockOrchForTray) HeartbeatAISession(context.Context, string) error { return nil }
func (m *mockOrchForTray) HeartbeatTask(context.Context, string, string) error {
	return nil
}
func (m *mockOrchForTray) ClaimTask(context.Context, string, string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchForTray) UpdateTaskStatus(context.Context, string, string, domain.TaskStatus, string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockOrchForTray) PurgeDisconnectedSessions(context.Context) (int, error) {
	return 0, nil
}
func (m *mockOrchForTray) GetDiscoveredAgents(context.Context) ([]domain.DiscoveredAgent, error) {
	return nil, nil
}
func (m *mockOrchForTray) DelegateToNexus(context.Context, string) (string, error) {
	return "", nil
}
func (m *mockOrchForTray) GetDiscoveredPlanFiles(context.Context, string) ([]domain.DiscoveredPlanFile, error) {
	return nil, nil
}
func (m *mockOrchForTray) GetQueueForProject(projectPath string) ([]domain.Task, error) {
	return nil, nil
}
func (m *mockOrchForTray) GetTasksForProject(projectPath string) ([]domain.Task, error) {
	return nil, nil
}

func TestTrayAdapter_HonestStub(t *testing.T) {
	adapter := tray.NewTrayAdapter(&mockOrchForTray{}, func() {}, func() {})
	if adapter == nil {
		t.Fatal("expected non-nil TrayAdapter")
	}

	if adapter.Enabled() {
		t.Error("expected Enabled() to return false for stub tray adapter")
	}

	// Calling Start, UpdateStatus, and Stop should not panic
	adapter.Start()
	adapter.UpdateStatus()
	adapter.Stop()
	// Stop should be safe to call multiple times
	adapter.Stop()
}
