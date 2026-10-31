package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"nexus-orchestrator/internal/core/ports"
)

const (
	healthCacheTTL = 30 * time.Second // TTL for healthy providers
	maxBackoffTTL  = 10 * time.Minute // ceiling for circuit-breaker backoff
)

// providerHealth is the cached liveness snapshot for one LLM provider.
type providerHealth struct {
	alive            bool
	activeModel      string
	models           []string
	checkedAt        time.Time
	consecutiveFails int
	latencyMs        int64  // last probe round-trip in ms
	lastError        string // last error message (empty when healthy)
}

// probeCall is one liveness probe in flight; callers that need the same provider wait on done.
type probeCall struct {
	done chan struct{}
	h    *providerHealth
}

// DiscoveryService probes registered LLM clients and returns the first active one.
// All exported methods are safe for concurrent use.
type DiscoveryService struct {
	mu               sync.RWMutex
	availableClients []ports.LLMClient
	cacheMu          sync.Mutex // guards healthCache and probing; never held while a provider is probed
	healthCache      map[string]*providerHealth
	probing          map[string]*probeCall // probes in flight, so concurrent callers share one
	cooldowns        cooldownBook          // providers that hit a limit; see cooldown.go
	quotas           quotaBook             // the plans' own allowance readings; see quota.go
}

// NewDiscoveryService creates a DiscoveryService with the supplied LLM adapters.
func NewDiscoveryService(clients ...ports.LLMClient) *DiscoveryService {
	return &DiscoveryService{availableClients: clients}
}

// RegisterProvider adds a new LLM adapter at runtime. Safe for concurrent use.
func (s *DiscoveryService) RegisterProvider(c ports.LLMClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.availableClients = append(s.availableClients, c)
}

// ReplaceProvider registers c, first dropping any provider with the same name and active model:
// registering the same backend again (a restarted bridge, a re-sent config) must not leave a
// stale copy next to it. A different model under the same name is a different provider and stays.
func (s *DiscoveryService) ReplaceProvider(c ports.LLMClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.availableClients[:0:0]
	for _, old := range s.availableClients {
		if strings.EqualFold(old.ProviderName(), c.ProviderName()) && old.ActiveModel() == c.ActiveModel() {
			continue
		}
		kept = append(kept, old)
	}
	s.availableClients = append(kept, c)
}

// RemoveProvider removes the first registered provider whose ProviderName()
// matches name (case-insensitive). Returns true if a provider was removed.
func (s *DiscoveryService) RemoveProvider(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.availableClients {
		if strings.EqualFold(c.ProviderName(), name) {
			s.availableClients = append(s.availableClients[:i], s.availableClients[i+1:]...)
			return true
		}
	}
	return false
}

// GetClientByName returns the first registered provider whose ProviderName()
// matches name (case-insensitive), along with a found flag.
func (s *DiscoveryService) GetClientByName(name string) (ports.LLMClient, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.availableClients {
		if strings.EqualFold(c.ProviderName(), name) {
			return c, true
		}
	}
	return nil, false
}

// GetAllClients returns a copy of all registered LLM clients.
func (s *DiscoveryService) GetAllClients() []ports.LLMClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clients := make([]ports.LLMClient, len(s.availableClients))
	copy(clients, s.availableClients)
	return clients
}

// DetectActive returns the first LLM provider that responds to a Ping,
// or nil when none are reachable.
func (s *DiscoveryService) DetectActive() ports.LLMClient {
	s.mu.RLock()
	clients := make([]ports.LLMClient, len(s.availableClients))
	copy(clients, s.availableClients)
	s.mu.RUnlock()

	// Every stale provider is probed at once; the answer is the first alive one in order, and
	// a slow or hung provider behind it is not waited for.
	pending := s.probeAll(clients)
	for i, c := range clients {
		if (<-pending[i]).alive {
			return c
		}
	}
	return nil
}

// FindForModel returns a live provider that can serve the requested model.
//
// When modelID is empty it falls back to DetectActive(). Otherwise:
//   - Pass 1: provider whose ActiveModel() matches modelID (currently loaded)
//   - Pass 2: provider that lists modelID in GetAvailableModels()
//
// providerHint (case-insensitive substring of ProviderName) is tried first in
// each pass. Returns an error when no suitable provider is found.
func (s *DiscoveryService) FindForModel(modelID, providerHint string) (ports.LLMClient, error) {
	if modelID == "" {
		c := s.DetectActive()
		if c == nil {
			return nil, fmt.Errorf("discovery: no active provider available")
		}
		return c, nil
	}

	candidates := s.orderedCandidates(providerHint)

	// Snapshot health for all candidates once (avoids repeated lock/unlock).
	type candidateHealth struct {
		client ports.LLMClient
		health *providerHealth
	}
	snapshots := make([]candidateHealth, 0, len(candidates))
	pending := s.probeAll(candidates)
	for i, c := range candidates {
		snapshots = append(snapshots, candidateHealth{c, <-pending[i]})
	}

	// Pass 1: provider with the model already loaded / active.
	for _, snap := range snapshots {
		if snap.health.alive && strings.EqualFold(snap.health.activeModel, modelID) {
			return snap.client, nil
		}
	}

	// Pass 2: provider that lists the model in its catalogue.
	for _, snap := range snapshots {
		if !snap.health.alive {
			continue
		}
		for _, m := range snap.health.models {
			if strings.EqualFold(m, modelID) {
				return snap.client, nil
			}
		}
	}

	return nil, fmt.Errorf("discovery: model %q not available on any registered provider", modelID)
}

// probeTimeout bounds each liveness / model-list probe made on behalf of the API.
const probeTimeout = 10 * time.Second

// ListProviders returns the liveness and model information for every registered
// LLM backend without modifying internal state.
func (s *DiscoveryService) ListProviders() []ports.ProviderInfo {
	s.mu.RLock()
	clients := make([]ports.LLMClient, len(s.availableClients))
	copy(clients, s.availableClients)
	s.mu.RUnlock()

	probeCtx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	now := time.Now()
	result := make([]ports.ProviderInfo, 0, len(clients))
	pending := s.probeAll(clients)
	for i, c := range clients {
		h := <-pending[i]
		info := ports.ProviderInfo{
			Name:             c.ProviderName(),
			Active:           h.alive,
			BaseURL:          c.BaseURL(),
			ContextLimit:     c.ContextLimit(probeCtx),
			LastChecked:      h.checkedAt,
			ConsecutiveFails: h.consecutiveFails,
		}
		info.LatencyMs = h.latencyMs
		if cd, ok := s.cooldowns.active(cooldownKey(c), now); ok {
			info.RateLimited = true
			info.RateLimitedUntil = cd.until
			info.CooldownReason = cd.reason
		}
		if q, ok := s.quotas.current(cooldownKey(c), now); ok {
			info.Quota = &q
		}
		info.LastError = h.lastError
		if h.alive {
			info.ActiveModel = h.activeModel
			info.Models = h.models
		} else {
			info.Error = fmt.Sprintf("discovery: %s: provider unreachable", c.ProviderName())
		}
		result = append(result, info)
	}
	return result
}

// orderedCandidates returns a snapshot of all clients with hint-matching providers first.
// Callers must NOT hold s.mu when calling this method.
func (s *DiscoveryService) orderedCandidates(hint string) []ports.LLMClient {
	s.mu.RLock()
	all := make([]ports.LLMClient, len(s.availableClients))
	copy(all, s.availableClients)
	s.mu.RUnlock()

	if hint == "" {
		return all
	}
	var first, rest []ports.LLMClient
	for _, c := range all {
		if strings.Contains(strings.ToLower(c.ProviderName()), strings.ToLower(hint)) {
			first = append(first, c)
		} else {
			rest = append(rest, c)
		}
	}
	return append(first, rest...)
}

// staleTTL returns how long a cached providerHealth entry remains valid.
// Healthy providers (or fewer than 3 consecutive failures) use healthCacheTTL.
// Each extra failure beyond 3 doubles the TTL (capped at maxBackoffTTL).
func staleTTL(h *providerHealth) time.Duration {
	if h.consecutiveFails < 3 {
		return healthCacheTTL
	}
	exp := h.consecutiveFails - 3
	if exp > 5 {
		exp = 5
	}
	d := healthCacheTTL * (1 << uint(exp))
	if d > maxBackoffTTL {
		return maxBackoffTTL
	}
	return d
}

// probeAll starts cachedHealth for every client at once and returns one channel per client, in
// order, that yields its health. Reading the channels in order gives the first answers first.
func (s *DiscoveryService) probeAll(clients []ports.LLMClient) []<-chan *providerHealth {
	out := make([]<-chan *providerHealth, len(clients))
	for i, c := range clients {
		ch := make(chan *providerHealth, 1)
		out[i] = ch
		go func() { ch <- s.cachedHealth(c) }()
	}
	return out
}

// cachedHealth returns the providerHealth for c, re-probing only when the cache entry is absent
// or has exceeded its TTL. The probe runs without holding cacheMu (it can take seconds: a model
// list fetched over the network, a host that does not answer), and callers that arrive while a
// probe of the same provider is in flight wait for that probe instead of starting another.
func (s *DiscoveryService) cachedHealth(c ports.LLMClient) *providerHealth {
	name := c.ProviderName()

	s.cacheMu.Lock()
	if s.healthCache == nil {
		s.healthCache = make(map[string]*providerHealth)
	}
	h, ok := s.healthCache[name]
	if ok && time.Since(h.checkedAt) < staleTTL(h) {
		s.cacheMu.Unlock()
		return h // cache is fresh — no network call
	}
	if call, running := s.probing[name]; running {
		s.cacheMu.Unlock()
		<-call.done
		return call.h
	}
	if s.probing == nil {
		s.probing = make(map[string]*probeCall)
	}
	call := &probeCall{done: make(chan struct{})}
	s.probing[name] = call
	s.cacheMu.Unlock()

	updated := probeProvider(c, h)

	s.cacheMu.Lock()
	s.healthCache[name] = updated
	delete(s.probing, name)
	call.h = updated
	s.cacheMu.Unlock()
	close(call.done)
	return updated
}

// probeProvider pings c and, when it answers, reads its models. prev is the earlier result
// (nil the first time) and carries the count of consecutive failures.
func probeProvider(c ports.LLMClient, prev *providerHealth) *providerHealth {
	name := c.ProviderName()
	// Never wait on a hung provider forever.
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	start := time.Now()
	alive := c.Ping(ctx)
	updated := &providerHealth{
		alive:     alive,
		checkedAt: time.Now(),
		latencyMs: time.Since(start).Milliseconds(),
	}
	if prev != nil {
		updated.consecutiveFails = prev.consecutiveFails
	}
	if alive {
		updated.activeModel = c.ActiveModel()
		if models, mErr := c.GetAvailableModels(ctx); mErr != nil {
			log.Printf("discovery: get models from %s: %v", name, mErr)
		} else {
			updated.models = models
		}
		updated.consecutiveFails = 0
		updated.lastError = ""
	} else {
		updated.consecutiveFails++
		updated.lastError = name + " unreachable"
	}
	return updated
}

// InvalidateHealthCache clears all cached health entries so the next
// ListProviders / DetectActive call will re-probe all providers.
func (s *DiscoveryService) InvalidateHealthCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.healthCache = make(map[string]*providerHealth)
}
