// Package fakes provides scriptable test doubles for the core ports, shared by
// the tests of every adapter and entry point.
package fakes

import (
	"context"
	"sync"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Orchestrator is a fully scriptable ports.Orchestrator: every operation succeeds
// with canned data unless errs[<MethodName>] is set. Generated from the port.
type Orchestrator struct {
	mu   sync.Mutex
	errs map[string]error
	seen map[string][]any
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{errs: map[string]error{}, seen: map[string][]any{}}
}

func (f *Orchestrator) FailWith(op string, err error) *Orchestrator {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[op] = err
	return f
}

func (f *Orchestrator) op(name string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen[name] = args
	return f.errs[name]
}

var _ ports.Orchestrator = (*Orchestrator)(nil)

func (f *Orchestrator) SubmitTask(t domain.Task) (string, error) {
	if err := f.op("SubmitTask", t); err != nil {
		return "", err
	}
	return "task-1", nil
}

func (f *Orchestrator) GetTask(id string) (domain.Task, error) {
	if err := f.op("GetTask", id); err != nil {
		return domain.Task{}, err
	}
	return domain.Task{ID: id, Status: domain.StatusQueued}, nil
}

func (f *Orchestrator) GetQueue() ([]domain.Task, error) {
	if err := f.op("GetQueue"); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "q1"}}, nil
}

func (f *Orchestrator) GetQueueForProject(p string) ([]domain.Task, error) {
	if err := f.op("GetQueueForProject", p); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "q1", ProjectPath: p}}, nil
}

func (f *Orchestrator) GetAllTasks() ([]domain.Task, error) {
	if err := f.op("GetAllTasks"); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "a1", AISessionID: "sess-1"}, {ID: "a2"}}, nil
}

func (f *Orchestrator) GetTasksForProject(p string) ([]domain.Task, error) {
	if err := f.op("GetTasksForProject", p); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "a1", ProjectPath: p}}, nil
}

func (f *Orchestrator) GetProviders() ([]ports.ProviderInfo, error) {
	if err := f.op("GetProviders"); err != nil {
		return nil, err
	}
	return []ports.ProviderInfo{{Name: "p"}}, nil
}

func (f *Orchestrator) CancelTask(id string) error {
	return f.op("CancelTask", id)
}

func (f *Orchestrator) RegisterCloudProvider(c domain.ProviderConfig) error {
	return f.op("RegisterCloudProvider", c)
}

func (f *Orchestrator) RemoveProvider(n string) error {
	return f.op("RemoveProvider", n)
}

func (f *Orchestrator) GetProviderModels(n string) ([]string, error) {
	if err := f.op("GetProviderModels", n); err != nil {
		return nil, err
	}
	return []string{"m1"}, nil
}

func (f *Orchestrator) GetBacklog(p string) ([]domain.Task, error) {
	if err := f.op("GetBacklog", p); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "b1", Status: domain.StatusDraft}}, nil
}

func (f *Orchestrator) CreateDraft(t domain.Task) (string, error) {
	if err := f.op("CreateDraft", t); err != nil {
		return "", err
	}
	return "draft-1", nil
}

func (f *Orchestrator) PromoteTask(id string) (ports.PromoteResult, error) {
	if err := f.op("PromoteTask", id); err != nil {
		return ports.PromoteResult{}, err
	}
	return ports.PromoteResult{Promoted: true}, nil
}

func (f *Orchestrator) UpdateTask(id string, t domain.Task) (domain.Task, error) {
	if err := f.op("UpdateTask", id, t); err != nil {
		return domain.Task{}, err
	}
	return domain.Task{ID: id}, nil
}

func (f *Orchestrator) AddProviderConfig(ctx context.Context, c domain.ProviderConfig) (domain.ProviderConfig, error) {
	if err := f.op("AddProviderConfig", c); err != nil {
		return domain.ProviderConfig{}, err
	}
	return domain.ProviderConfig{ID: "pc1", Name: c.Name, Kind: c.Kind, APIKey: "sk-secret-1234"}, nil
}

func (f *Orchestrator) UpdateProviderConfig(ctx context.Context, c domain.ProviderConfig) (domain.ProviderConfig, error) {
	if err := f.op("UpdateProviderConfig", c); err != nil {
		return domain.ProviderConfig{}, err
	}
	return domain.ProviderConfig{ID: c.ID, Name: c.Name, APIKey: "sk-secret-1234"}, nil
}

func (f *Orchestrator) RemoveProviderConfig(ctx context.Context, id string) error {
	return f.op("RemoveProviderConfig", id)
}

func (f *Orchestrator) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	if err := f.op("ListProviderConfigs"); err != nil {
		return nil, err
	}
	return []domain.ProviderConfig{{ID: "pc1", Name: "n", APIKey: "sk-secret-1234"}, {ID: "pc2", Name: "m"}}, nil
}

func (f *Orchestrator) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	if err := f.op("GetDiscoveredProviders"); err != nil {
		return nil, err
	}
	return []domain.DiscoveredProvider{{ID: "d1"}}, nil
}

func (f *Orchestrator) TriggerScan(ctx context.Context) ([]domain.DiscoveredProvider, error) {
	if err := f.op("TriggerScan"); err != nil {
		return nil, err
	}
	return []domain.DiscoveredProvider{{ID: "d1"}}, nil
}

func (f *Orchestrator) PromoteProvider(ctx context.Context, id string) error {
	return f.op("PromoteProvider", id)
}

func (f *Orchestrator) RegisterAISession(ctx context.Context, s domain.AISession) (domain.AISession, error) {
	if err := f.op("RegisterAISession", s); err != nil {
		return domain.AISession{}, err
	}
	return domain.AISession{ID: "s1", AgentName: s.AgentName}, nil
}

func (f *Orchestrator) ListAISessions(ctx context.Context) ([]domain.AISession, error) {
	if err := f.op("ListAISessions"); err != nil {
		return nil, err
	}
	return []domain.AISession{{ID: "s1"}}, nil
}

func (f *Orchestrator) DeregisterAISession(ctx context.Context, id string) error {
	return f.op("DeregisterAISession", id)
}

func (f *Orchestrator) HeartbeatAISession(ctx context.Context, id string) error {
	return f.op("HeartbeatAISession", id)
}

func (f *Orchestrator) HeartbeatTask(ctx context.Context, taskID, sessionID string) error {
	return f.op("HeartbeatTask", taskID, sessionID)
}

func (f *Orchestrator) ClaimTask(ctx context.Context, taskID, sessionID string) (domain.Task, error) {
	if err := f.op("ClaimTask", taskID, sessionID); err != nil {
		return domain.Task{}, err
	}
	return domain.Task{ID: taskID, AISessionID: sessionID}, nil
}

func (f *Orchestrator) UpdateTaskStatus(ctx context.Context, taskID, sessionID string, st domain.TaskStatus, logs string) (domain.Task, error) {
	if err := f.op("UpdateTaskStatus", taskID, sessionID, st, logs); err != nil {
		return domain.Task{}, err
	}
	return domain.Task{ID: taskID, Status: st, Logs: logs}, nil
}

func (f *Orchestrator) PurgeDisconnectedSessions(ctx context.Context) (int, error) {
	if err := f.op("PurgeDisconnectedSessions"); err != nil {
		return 0, err
	}
	return 3, nil
}

func (f *Orchestrator) GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
	if err := f.op("GetDiscoveredAgents"); err != nil {
		return nil, err
	}
	return []domain.DiscoveredAgent{{ID: "a1"}}, nil
}

func (f *Orchestrator) DelegateToNexus(ctx context.Context, id string) (string, error) {
	if err := f.op("DelegateToNexus", id); err != nil {
		return "", err
	}
	return "do the work", nil
}

func (f *Orchestrator) TerminateAISession(ctx context.Context, id string, force bool) error {
	return f.op("TerminateAISession", id, force)
}

func (f *Orchestrator) GetDiscoveredPlanFiles(ctx context.Context, p string) ([]domain.DiscoveredPlanFile, error) {
	if err := f.op("GetDiscoveredPlanFiles", p); err != nil {
		return nil, err
	}
	return []domain.DiscoveredPlanFile{{ID: "pf1", ProjectPath: p}}, nil
}

func (f *Orchestrator) GetRuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	if err := f.op("GetRuntimeConfig"); err != nil {
		return domain.RuntimeConfig{}, err
	}
	return domain.RuntimeConfig{QueueCap: 50}, nil
}

func (f *Orchestrator) UpdateRuntimeConfig(ctx context.Context, u domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	if err := f.op("UpdateRuntimeConfig", u); err != nil {
		return domain.RuntimeConfig{}, err
	}
	return domain.RuntimeConfig{QueueCap: 7, APIToken: "new-api", MCPToken: "new-mcp"}, nil
}
