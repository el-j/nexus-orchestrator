package usagebox

import (
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

func TestBox(t *testing.T) {
	var b Box
	if _, ok := b.LastUsage(); ok {
		t.Fatal("nothing reported yet")
	}
	b.Set(domain.Usage{Model: "m", InputTokens: 3, OutputTokens: 4})
	if u, ok := b.LastUsage(); !ok || u.InputTokens != 3 || u.Model != "m" {
		t.Errorf("got %+v %v", u, ok)
	}
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); b.Set(domain.Usage{InputTokens: i}); b.LastUsage() }()
	}
	wg.Wait()
}
