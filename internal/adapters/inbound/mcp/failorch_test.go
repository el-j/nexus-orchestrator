package mcp_test

import (
	"context"
	"sync"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// failOrch is a fully scriptable ports.Orchestrator: every operation succeeds
// with canned data unless errs[<MethodName>] is set. Generated from the port.
type failOrch struct {
	mu   sync.Mutex
	errs map[string]error
	seen map[string][]any
}

func newFailOrch() *failOrch { return &failOrch{errs: map[string]error{}, seen: map[string][]any{}} }

func (f *failOrch) failWith(op string, err error) *failOrch {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[op] = err
	return f
}

func (f *failOrch) op(name string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen[name] = args
	return f.errs[name]
}

var _ ports.Orchestrator = (*failOrch)(nil)
var _ = context.Background

func (f *failOrch) SubmitTask(t domain.Task) (string, error) {
	if err := f.op("SubmitTask", t); err != nil {
		return "", err
	}
	return "task-1", nil
}

func (f *failOrch) GetTask(id string) (domain.Task, error) {
	if err := f.op("GetTask", id); err != nil {
		return domain.Task{}, err
	}
	return domain.Task{ID: id, Status: domain.StatusQueued}, nil
}

func (f *failOrch) GetQueue() ([]domain.Task, error) {
	if err := f.op("GetQueue"); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "q1"}}, nil
}

func (f *failOrch) GetQueueForProject(p string) ([]domain.Task, error) {
	if err := f.op("GetQueueForProject", p); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "q1", ProjectPath: p}}, nil
}

func (f *failOrch) GetAllTasks() ([]domain.Task, error) {
	if err := f.op("GetAllTasks"); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "a1", AISessionID: "sess-1"}, {ID: "a2"}}, nil
}

func (f *failOrch) GetTasksForProject(p string) ([]domain.Task, error) {
	if err := f.op("GetTasksForProject", p); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "a1", ProjectPath: p}}, nil
}

func (f *failOrch) GetProviders() ([]ports.ProviderInfo, error) {
	if err := f.op("GetProviders"); err != nil {
		return nil, err
	}
	return []ports.ProviderInfo{{Name: "p"}}, nil
}

func (f *failOrch) CancelTask(id string) error {
	return f.op("CancelTask", id)
}

func (f *failOrch) RegisterCloudProvider(c domain.ProviderConfig) error {
	return f.op("RegisterCloudProvider", c)
}

func (f *failOrch) RemoveProvider(n string) error {
	return f.op("RemoveProvider", n)
}

func (f *failOrch) GetProviderModels(n string) ([]string, error) {
	if err := f.op("GetProviderModels", n); err != nil {
		return nil, err
	}
	return []string{"m1"}, nil
}

func (f *failOrch) GetBacklog(p string) ([]domain.Task, error) {
	if err := f.op("GetBacklog", p); err != nil {
		return nil, err
	}
	return []domain.Task{{ID: "b1", Status: domain.StatusDraft}}, nil
}

func (f *failOrch) CreateDraft(t domain.Task) (string, error) {
	if err := f.op("CreateDraft", t); err != nil {
		return "", err
	}
	return "draft-1", nil
}

func (f *failOrch) PromoteTask(id string) (ports.PromoteResult, error) {
	if err := f.op("PromoteTask", id); err != nil {
		return ports.PromoteResult{}, err
	}
	return ports.PromoteResult{Promoted: true}, nil
}

func (f *failOrch) UpdateTask(id string, t domain.Task) (domain.Task, error) {
	if err := f.op("UpdateTask", id, t); err != nil {
		return domain.Task{}, err
	}
	return domain.Task{ID: id}, nil
}

func (f *failOrch) AddProviderConfig(ctx context.Context, c domain.ProviderConfig) (domain.ProviderConfig, error) {
	if err := f.op("AddProviderConfig", c); err != nil {
		return domain.ProviderConfig{}, err
	}
	return domain.ProviderConfig{ID: "pc1", Name: c.Name, Kind: c.Kind, APIKey: "sk-secret-1234"}, nil
}

func (f *failOrch) UpdateProviderConfig(ctx context.Context, c domain.ProviderConfig) (domain.ProviderConfig, error) {
	if err := f.op("UpdateProviderConfig", c); err != nil {
		return domain.ProviderConfig{}, err
	}
	return domain.ProviderConfig{ID: c.ID, Name: c.Name, APIKey: "sk-secret-1234"}, nil
}

func (f *failOrch) RemoveProviderConfig(ctx context.Context, id string) error {
	return f.op("RemoveProviderConfig", id)
}

func (f *failOrch) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	if err := f.op("ListProviderConfigs"); err != nil {
		return nil, err
	}
	return []domain.ProviderConfig{{ID: "pc1", Name: "n", APIKey: "sk-secret-1234"}, {ID: "pc2", Name: "m"}}, nil
}

func (f *failOrch) GetDiscoveredProviders() ([]domain.DiscoveredProvider, error) {
	if err := f.op("GetDiscoveredProviders"); err != nil {
		return nil, err
	}
	return []domain.DiscoveredProvider{{ID: "d1"}}, nil
}

func (f *failOrch) TriggerScan(ctx context.Context) ([]domain.DiscoveredProvider, error) {
	if err := f.op("TriggerScan"); err != nil {
		return nil, err
	}
	return []domain.DiscoveredProvider{{ID: "d1"}}, nil
}

func (f *failOrch) PromoteProvider(ctx context.Context, id string) error {
	return f.op("PromoteProvider", id)
}

func (f *failOrch) RegisterAISession(ctx context.Context, s domain.AISession) (domain.AISession, error) {
	if err := f.op("RegisterAISession", s); err != nil {
		return domain.AISession{}, err
	}
	return domain.AISession{ID: "s1", AgentName: s.AgentName}, nil
}

func (f *failOrch) ListAISessions(ctx context.Context) ([]domain.AISession, error) {
	if err := f.op("ListAISessions"); err != nil {
		return nil, err
	}
	return []domain.AISession{{ID: "s1"}}, nil
}

func (f *failOrch) DeregisterAISession(ctx context.Context, id string) error {
	return f.op("DeregisterAISession", id)
}

func (f *failOrch) HeartbeatAISession(ctx context.Context, id string) error {
	return f.op("HeartbeatAISession", id)
}

func (f *failOrch) HeartbeatTask(ctx context.Context, taskID, sessionID string) error {
	return f.op("HeartbeatTask", taskID, sessionID)
}

func (f *failOrch) ClaimTask(ctx context.Context, taskID, sessionID string) (domain.Task, error) {
	if err := f.op("ClaimTask", taskID, sessionID); err != nil {
		return domain.Task{}, err
	}
	return domain.Task{ID: taskID, AISessionID: sessionID}, nil
}

func (f *failOrch) UpdateTaskStatus(ctx context.Context, taskID, sessionID string, st domain.TaskStatus, logs string) (domain.Task, error) {
	if err := f.op("UpdateTaskStatus", taskID, sessionID, st, logs); err != nil {
		return domain.Task{}, err
	}
	return domain.Task{ID: taskID, Status: st, Logs: logs}, nil
}

func (f *failOrch) PurgeDisconnectedSessions(ctx context.Context) (int, error) {
	if err := f.op("PurgeDisconnectedSessions"); err != nil {
		return 0, err
	}
	return 3, nil
}

func (f *failOrch) GetDiscoveredAgents(ctx context.Context) ([]domain.DiscoveredAgent, error) {
	if err := f.op("GetDiscoveredAgents"); err != nil {
		return nil, err
	}
	return []domain.DiscoveredAgent{{ID: "a1"}}, nil
}

func (f *failOrch) DelegateToNexus(ctx context.Context, id string) (string, error) {
	if err := f.op("DelegateToNexus", id); err != nil {
		return "", err
	}
	return "do the work", nil
}

func (f *failOrch) TerminateAISession(ctx context.Context, id string, force bool) error {
	return f.op("TerminateAISession", id, force)
}

func (f *failOrch) GetDiscoveredPlanFiles(ctx context.Context, p string) ([]domain.DiscoveredPlanFile, error) {
	if err := f.op("GetDiscoveredPlanFiles", p); err != nil {
		return nil, err
	}
	return []domain.DiscoveredPlanFile{{ID: "pf1", ProjectPath: p}}, nil
}

func (f *failOrch) GetRuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error) {
	if err := f.op("GetRuntimeConfig"); err != nil {
		return domain.RuntimeConfig{}, err
	}
	return domain.RuntimeConfig{QueueCap: 50}, nil
}

func (f *failOrch) UpdateRuntimeConfig(ctx context.Context, u domain.RuntimeConfigUpdate) (domain.RuntimeConfig, error) {
	if err := f.op("UpdateRuntimeConfig", u); err != nil {
		return domain.RuntimeConfig{}, err
	}
	return domain.RuntimeConfig{QueueCap: 7, APIToken: "new-api", MCPToken: "new-mcp"}, nil
}

// failBrain is a scriptable ports.BrainService: every operation succeeds with
// canned data unless errs[<MethodName>] is set.
type failBrain struct {
	mu   sync.Mutex
	errs map[string]error
}

func newFailBrain() *failBrain { return &failBrain{errs: map[string]error{}} }

func (f *failBrain) failWith(op string, err error) *failBrain {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[op] = err
	return f
}

func (f *failBrain) op(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.errs[name]
}

func (f *failBrain) GetContext(context.Context, domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{TokensUsed: 5}, f.op("GetContext")
}
func (f *failBrain) GetFocusedContext(context.Context, domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{TokensUsed: 5}, f.op("GetFocusedContext")
}
func (f *failBrain) IngestKnowledge(_ context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	return k, f.op("IngestKnowledge")
}
func (f *failBrain) IngestFromFile(context.Context, string, string) (int, error) {
	return 2, f.op("IngestFromFile")
}
func (f *failBrain) SearchKnowledge(context.Context, string, string, int) ([]domain.ContextSection, error) {
	return []domain.ContextSection{{Topic: "t"}}, f.op("SearchKnowledge")
}
func (f *failBrain) GetFileMap(context.Context, string, string) ([]string, error) {
	return []string{"a.go"}, f.op("GetFileMap")
}
func (f *failBrain) InitProject(context.Context, string, string) (domain.BrainStatus, error) {
	return domain.BrainStatus{Initialized: true}, f.op("InitProject")
}
func (f *failBrain) GetStatus(context.Context, string) (domain.BrainStatus, error) {
	return domain.BrainStatus{Initialized: true}, f.op("GetStatus")
}
func (f *failBrain) ListKnowledge(context.Context, string, string) ([]domain.ProjectKnowledge, error) {
	return []domain.ProjectKnowledge{{ID: "k1"}}, f.op("ListKnowledge")
}
func (f *failBrain) DeleteKnowledge(context.Context, string) error { return f.op("DeleteKnowledge") }
func (f *failBrain) GetOnboardingContext(context.Context, string, int) (string, error) {
	return "welcome", f.op("GetOnboardingContext")
}

var _ ports.BrainService = (*failBrain)(nil)
