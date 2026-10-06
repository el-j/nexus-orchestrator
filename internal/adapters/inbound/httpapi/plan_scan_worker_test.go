package httpapi_test

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
)

// planWorkerOrch records which projects were plan-scanned and whether the
// (unrelated) provider scan was ever triggered.
type planWorkerOrch struct {
	*failOrch
	mu            sync.Mutex
	scanned       []string
	tasks         []domain.Task
	sessions      []domain.AISession
	failProject   string
	providerScans int
}

func (p *planWorkerOrch) GetAllTasks() ([]domain.Task, error) { return p.tasks, nil }
func (p *planWorkerOrch) ListAISessions(context.Context) ([]domain.AISession, error) {
	return p.sessions, nil
}
func (p *planWorkerOrch) TriggerScan(context.Context) ([]domain.DiscoveredProvider, error) {
	p.mu.Lock()
	p.providerScans++
	p.mu.Unlock()
	return nil, nil
}
func (p *planWorkerOrch) GetDiscoveredPlanFiles(_ context.Context, project string) ([]domain.DiscoveredPlanFile, error) {
	p.mu.Lock()
	p.scanned = append(p.scanned, project)
	p.mu.Unlock()
	if project == p.failProject {
		return nil, errors.New("scan failed")
	}
	return nil, nil
}

func (p *planWorkerOrch) snapshot() ([]string, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.scanned...), p.providerScans
}

func TestPlanScanWorker_ScansPlanFilesOfEveryKnownProject(t *testing.T) {
	orch := &planWorkerOrch{
		failOrch: newFailOrch(),
		tasks: []domain.Task{
			{ProjectPath: "/work/b"}, {ProjectPath: "/work/a/"}, {ProjectPath: ""}, {ProjectPath: "/work/b"},
		},
		sessions:    []domain.AISession{{ProjectPath: "/work/a"}, {ProjectPath: "/work/c"}},
		failProject: "/work/a", // an erroring project must not stop the sweep
	}
	ctx, cancel := context.WithCancel(context.Background())
	httpapi.StartPlanScanWorker(ctx, orch, 10*time.Millisecond)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if scanned, _ := orch.snapshot(); len(scanned) >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()

	scanned, providerScans := orch.snapshot()
	if len(scanned) < 3 {
		t.Fatalf("scanned %v", scanned)
	}
	first := append([]string(nil), scanned[:3]...)
	if !sort.StringsAreSorted(first) || first[0] != "/work/a" || first[1] != "/work/b" || first[2] != "/work/c" {
		t.Errorf("each distinct, cleaned project is scanned once per sweep, in order: %v", first)
	}
	if providerScans != 0 {
		t.Errorf("the plan worker must not trigger provider scans (the daemon does that itself); got %d", providerScans)
	}

	// After cancellation the worker goes quiet.
	time.Sleep(30 * time.Millisecond)
	before, _ := orch.snapshot()
	time.Sleep(60 * time.Millisecond)
	if after, _ := orch.snapshot(); len(after) != len(before) {
		t.Errorf("worker kept scanning after cancel: %d -> %d", len(before), len(after))
	}
}

func TestPlanScanWorker_NonPositiveIntervalFallsBackToDefaultAndStops(t *testing.T) {
	orch := &planWorkerOrch{failOrch: newFailOrch(), tasks: []domain.Task{{ProjectPath: "/p"}}}
	ctx, cancel := context.WithCancel(context.Background())
	httpapi.StartPlanScanWorker(ctx, orch, 0) // 5-minute default: nothing runs in this test
	time.Sleep(20 * time.Millisecond)
	cancel()
	if scanned, _ := orch.snapshot(); len(scanned) != 0 {
		t.Errorf("default interval is minutes, got scans: %v", scanned)
	}
}
