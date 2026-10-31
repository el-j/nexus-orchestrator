package services_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/services"
)

// slowClient is a provider whose liveness probe takes a while, like a model list fetched over
// the network or a host that does not answer.
type slowClient struct {
	mockLLMClient
	delay time.Duration
	hold  chan struct{} // when set, Ping blocks until it is closed
	pings atomic.Int32
}

func (c *slowClient) Ping(ctx context.Context) bool {
	c.pings.Add(1)
	if c.hold != nil {
		select {
		case <-c.hold:
		case <-ctx.Done():
		}
	} else {
		time.Sleep(c.delay)
	}
	return c.alive
}

func TestListProviders_ProbesStaleProvidersInParallel(t *testing.T) {
	svc := services.NewDiscoveryService()
	for _, n := range []string{"a", "b", "c", "d"} {
		c := &slowClient{mockLLMClient: mockLLMClient{alive: true, name: n}, delay: 250 * time.Millisecond}
		svc.RegisterProvider(c)
	}
	start := time.Now()
	if got := svc.ListProviders(); len(got) != 4 {
		t.Fatalf("got %d providers", len(got))
	}
	if took := time.Since(start); took > 600*time.Millisecond {
		t.Errorf("4 probes of 250 ms took %v: they ran one after another", took)
	}
}

func TestCachedHealth_ConcurrentCallsShareOneProbe(t *testing.T) {
	c := &slowClient{mockLLMClient: mockLLMClient{alive: true, name: "x"}, delay: 150 * time.Millisecond}
	svc := services.NewDiscoveryService(c)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); svc.ListProviders() }()
	}
	wg.Wait()
	if n := c.pings.Load(); n != 1 {
		t.Errorf("provider probed %d times for 8 simultaneous requests, want 1", n)
	}
}

func TestDetectActive_DoesNotWaitForSlowerProvidersBehindTheFirstAliveOne(t *testing.T) {
	hung := &slowClient{mockLLMClient: mockLLMClient{alive: true, name: "hung"}, hold: make(chan struct{})}
	defer close(hung.hold)
	svc := services.NewDiscoveryService(&mockLLMClient{alive: true, name: "fast"}, hung)
	done := make(chan string, 1)
	go func() { done <- svc.DetectActive().ProviderName() }()
	select {
	case name := <-done:
		if name != "fast" {
			t.Errorf("got %q", name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("routing waited for a hung provider that comes after a healthy one")
	}
}

func TestDetectActive_AProbeInFlightOnOneProviderDoesNotBlockRoutingToAnother(t *testing.T) {
	hung := &slowClient{mockLLMClient: mockLLMClient{alive: true, name: "hung"}, hold: make(chan struct{})}
	defer close(hung.hold)
	svc := services.NewDiscoveryService(&mockLLMClient{alive: true, name: "ok"}, hung)
	// One caller is probing the hung provider (it holds no lock while it waits)...
	go svc.ListProviders()
	time.Sleep(50 * time.Millisecond)
	// ...which must not stop another caller from using the healthy one.
	done := make(chan struct{})
	go func() { svc.DetectActive(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a probe in flight on one provider blocked routing to another")
	}
}

func TestTriggerScan_RefillsTheHealthCacheInTheBackground(t *testing.T) {
	orch, disc := newProviderOrch(t, nil, nil)
	c := &slowClient{mockLLMClient: mockLLMClient{alive: true, name: "p"}}
	disc.RegisterProvider(c)
	orch.WithSystemScanner(&mockScanner{})
	if _, err := orch.TriggerScan(bg); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for c.pings.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the cache was cleared but nothing probed the provider again")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
