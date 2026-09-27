package httpapi_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// planScanMockOrch implements only the TriggerScan method needed for the worker test.
type planScanMockOrch struct {
	triggerScanCount atomic.Int32
}

func (m *planScanMockOrch) TriggerScan(context.Context) ([]domain.DiscoveredProvider, error) {
	m.triggerScanCount.Add(1)
	return nil, nil
}

// Minimal stubs for other Orchestrator methods used by compilation. They are
// never called by the worker tests.
func (m *planScanMockOrch) SubmitTask(domain.Task) (string, error)             { return "", nil }
func (m *planScanMockOrch) GetTask(string) (domain.Task, error)                { return domain.Task{}, nil }
func (m *planScanMockOrch) GetQueue() ([]domain.Task, error)                   { return nil, nil }
func (m *planScanMockOrch) GetQueueForProject(_ string) ([]domain.Task, error) { return nil, nil }
func (m *planScanMockOrch) GetAllTasks() ([]domain.Task, error)                { return nil, nil }
func (m *planScanMockOrch) GetTasksForProject(_ string) ([]domain.Task, error) { return nil, nil }
func (m *planScanMockOrch) GetProviders() ([]ports.ProviderInfo, error)        { return nil, nil }
func (m *planScanMockOrch) GetRuntimeConfig(context.Context) (domain.RuntimeConfig, error) {
	return domain.RuntimeConfig{}, nil
}
func (m *planScanMockOrch) UpdateRuntimeConfig(context.Context, domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	return domain.RuntimeConfig{}, nil
}
func (m *planScanMockOrch) CancelTask(string) error                           { return nil }
func (m *planScanMockOrch) RegisterCloudProvider(domain.ProviderConfig) error { return nil }
func (m *planScanMockOrch) RemoveProvider(string) error                       { return nil }
func (m *planScanMockOrch) GetProviderModels(string) ([]string, error)        { return nil, nil }
func (m *planScanMockOrch) AddProviderConfig(context.Context, domain.ProviderConfig) (domain.ProviderConfig, error) {
	return domain.ProviderConfig{}, nil
}
func (m *planScanMockOrch) UpdateProviderConfig(context.Context, domain.ProviderConfig) (domain.ProviderConfig, error) {
	return domain.ProviderConfig{}, nil
}
func (m *planScanMockOrch) RemoveProviderConfig(context.Context, string) error { return nil }
func (m *planScanMockOrch) ListProviderConfigs(context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}
func (m *planScanMockOrch) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	return nil, nil
}
func (m *planScanMockOrch) PromoteProvider(context.Context, string) error { return nil }
func (m *planScanMockOrch) CreateDraft(domain.Task) (string, error)       { return "", nil }
func (m *planScanMockOrch) GetBacklog(string) ([]domain.Task, error)      { return nil, nil }
func (m *planScanMockOrch) PromoteTask(string) (ports.PromoteResult, error) {
	return ports.PromoteResult{}, nil
}
func (m *planScanMockOrch) UpdateTask(string, domain.Task) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *planScanMockOrch) RegisterAISession(context.Context, domain.AISession) (domain.AISession, error) {
	return domain.AISession{}, nil
}
func (m *planScanMockOrch) ListAISessions(context.Context) ([]domain.AISession, error) {
	return nil, nil
}
func (m *planScanMockOrch) DeregisterAISession(context.Context, string) error      { return nil }
func (m *planScanMockOrch) TerminateAISession(context.Context, string, bool) error { return nil }
func (m *planScanMockOrch) HeartbeatAISession(context.Context, string) error       { return nil }
func (m *planScanMockOrch) ClaimTask(context.Context, string, string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *planScanMockOrch) UpdateTaskStatus(context.Context, string, string, domain.TaskStatus, string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *planScanMockOrch) HeartbeatTask(context.Context, string, string) error    { return nil }
func (m *planScanMockOrch) PurgeDisconnectedSessions(context.Context) (int, error) { return 0, nil }
func (m *planScanMockOrch) GetDiscoveredAgents(context.Context) ([]domain.DiscoveredAgent, error) {
	return nil, nil
}
func (m *planScanMockOrch) DelegateToNexus(context.Context, string) (string, error) { return "", nil }
func (m *planScanMockOrch) GetDiscoveredPlanFiles(context.Context, string) ([]domain.DiscoveredPlanFile, error) {
	return nil, nil
}

func TestStartPlanScanWorker_CallsTriggerScanPeriodically(t *testing.T) {
	orch := &planScanMockOrch{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpapi.StartPlanScanWorker(ctx, orch, 20*time.Millisecond)

	time.Sleep(80 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	count := orch.triggerScanCount.Load()
	if count < 2 {
		t.Errorf("expected at least 2 scans in 80ms with 20ms interval, got %d", count)
	}
}

func TestStartPlanScanWorker_StopsOnContextCancel(t *testing.T) {
	orch := &planScanMockOrch{}

	ctx, cancel := context.WithCancel(context.Background())
	httpapi.StartPlanScanWorker(ctx, orch, 10*time.Millisecond)
	time.Sleep(25 * time.Millisecond)
	cancel()
	countAfterCancel := orch.triggerScanCount.Load()
	time.Sleep(30 * time.Millisecond)
	if orch.triggerScanCount.Load() > countAfterCancel+1 {
		t.Error("worker continued scanning after context cancel")
	}
}
