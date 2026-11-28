package services

import (
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/ports"
)

type namedLLM struct{ ports.LLMClient }

func (namedLLM) ProviderName() string { return "p" }
func (namedLLM) ActiveModel() string  { return "m" }

// Auto-Mode asks which provider runs a task while the worker is still routing it: the two must not race
// (run with -race).
func TestActiveTaskProviderDoesNotRaceWithRouting(t *testing.T) {
	o := &OrchestratorService{running: map[string]*runHandle{}}
	h := &runHandle{}
	o.running["t"] = h
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			h.setRoute([]ports.LLMClient{namedLLM{}}, nil)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			o.activeTaskProvider("t")
		}
	}()
	wg.Wait()
	if p, m := o.activeTaskProvider("t"); p != "p" || m != "m" {
		t.Errorf("provider = %q/%q", p, m)
	}
	if p, _ := o.activeTaskProvider("none"); p != "" {
		t.Errorf("unknown task has provider %q", p)
	}
}
