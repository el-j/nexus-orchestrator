package services_test

import (
	"sync"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// scanAnnouncer is a broadcaster that also understands the optional
// provider-discovery announcement, as the SSE hub does.
type scanAnnouncer struct {
	capturingBroadcaster
	mu     sync.Mutex
	counts []int
}

func (s *scanAnnouncer) BroadcastProviderDiscovered(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts = append(s.counts, n)
}

func TestTriggerScan_AnnouncesTheResultToConnectedUIs(t *testing.T) {
	orch, _ := newProviderOrch(t, nil, nil)
	b := &scanAnnouncer{}
	orch.SetBroadcaster(b)
	orch.WithSystemScanner(&mockScanner{results: []domain.DiscoveredProvider{{ID: "a"}, {ID: "b"}, {ID: "c"}}})

	if _, err := orch.TriggerScan(bg); err != nil {
		t.Fatal(err)
	}
	if len(b.counts) != 1 || b.counts[0] != 3 {
		t.Fatalf("announcements = %v, want [3]", b.counts)
	}
}

func TestTriggerScan_AFailedScanAnnouncesNothing(t *testing.T) {
	orch, _ := newProviderOrch(t, nil, nil)
	b := &scanAnnouncer{}
	orch.SetBroadcaster(b)
	orch.WithSystemScanner(&mockScanner{err: errInjected})
	_, _ = orch.TriggerScan(bg)
	if len(b.counts) != 0 {
		t.Fatalf("a failed scan must not be announced: %v", b.counts)
	}
}

// A broadcaster without the optional method (or no broadcaster at all) must keep working.
func TestTriggerScan_WorksWithBroadcastersThatCannotAnnounce(t *testing.T) {
	orch, _ := newProviderOrch(t, nil, nil)
	orch.SetBroadcaster(&capturingBroadcaster{})
	orch.WithSystemScanner(&mockScanner{results: []domain.DiscoveredProvider{{ID: "a"}}})
	if _, err := orch.TriggerScan(bg); err != nil {
		t.Fatal(err)
	}
	bare, _ := newProviderOrch(t, nil, nil)
	bare.WithSystemScanner(&mockScanner{})
	if _, err := bare.TriggerScan(bg); err != nil {
		t.Fatal(err)
	}
}
