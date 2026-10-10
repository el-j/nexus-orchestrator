package fakes

import (
	"context"
	"sync"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Brain is a scriptable ports.BrainService: every operation succeeds with
// canned data unless errs[<MethodName>] is set.
type Brain struct {
	mu   sync.Mutex
	errs map[string]error
}

func NewBrain() *Brain { return &Brain{errs: map[string]error{}} }

func (f *Brain) FailWith(op string, err error) *Brain {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[op] = err
	return f
}

func (f *Brain) op(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.errs[name]
}

func (f *Brain) GetContext(context.Context, domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{TokensUsed: 5}, f.op("GetContext")
}
func (f *Brain) GetFocusedContext(context.Context, domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{TokensUsed: 5}, f.op("GetFocusedContext")
}
func (f *Brain) IngestKnowledge(_ context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	return k, f.op("IngestKnowledge")
}
func (f *Brain) IngestFromFile(context.Context, string, string) (int, error) {
	return 2, f.op("IngestFromFile")
}
func (f *Brain) SearchKnowledge(context.Context, string, string, int) ([]domain.ContextSection, error) {
	return []domain.ContextSection{{Topic: "t"}}, f.op("SearchKnowledge")
}
func (f *Brain) GetFileMap(context.Context, string, string) ([]string, error) {
	return []string{"a.go"}, f.op("GetFileMap")
}
func (f *Brain) InitProject(context.Context, string, string) (domain.BrainStatus, error) {
	return domain.BrainStatus{Initialized: true}, f.op("InitProject")
}
func (f *Brain) GetStatus(context.Context, string) (domain.BrainStatus, error) {
	return domain.BrainStatus{Initialized: true}, f.op("GetStatus")
}
func (f *Brain) ListKnowledge(context.Context, string, string) ([]domain.ProjectKnowledge, error) {
	return []domain.ProjectKnowledge{{ID: "k1"}}, f.op("ListKnowledge")
}
func (f *Brain) DeleteKnowledge(context.Context, string) error { return f.op("DeleteKnowledge") }
func (f *Brain) GetOnboardingContext(context.Context, string, int) (string, error) {
	return "welcome", f.op("GetOnboardingContext")
}

var _ ports.BrainService = (*Brain)(nil)
