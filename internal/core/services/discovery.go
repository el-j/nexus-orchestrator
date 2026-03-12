package services

import (
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
}

// DiscoveryService probes registered LLM clients and returns the first active one.
// All exported methods are safe for concurrent use.
type DiscoveryService struct {
	mu               sync.RWMutex
	availableClients []ports.LLMClient
	cacheMu          sync.Mutex
	healthCache      map[string]*providerHealth
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

// DetectActive returns the first LLM provider that responds to a Ping,
// or nil when none are reachable.
func (s *DiscoveryService) DetectActive() ports.LLMClient {
	s.mu.RLock()
	clients := make([]ports.LLMClient, len(s.availableClients))
	copy(clients, s.availableClients)
	s.mu.RUnlock()

	for _, c := range clients {
		if s.cachedHealth(c).alive {
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
	for _, c := range candidates {
		snapshots = append(snapshots, candidateHealth{c, s.cachedHealth(c)})
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

// ListProviders returns the liveness and model information for every registered
// LLM backend without modifying internal state.
func (s *DiscoveryService) ListProviders() []ports.ProviderInfo {
	s.mu.RLock()
	clients := make([]ports.LLMClient, len(s.availableClients))
	copy(clients, s.availableClients)
	s.mu.RUnlock()

	result := make([]ports.ProviderInfo, 0, len(clients))
	for _, c := range clients {
		h := s.cachedHealth(c)
		info := ports.ProviderInfo{
			Name:    c.ProviderName(),
			Active:  h.alive,
			BaseURL: c.BaseURL(),
		}
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

// cachedHealth returns the providerHealth for c, re-probing only when the cache
// entry is absent or has exceeded its TTL. Thread-safe via s.cacheMu.
func (s *DiscoveryService) cachedHealth(c ports.LLMClient) *providerHealth {
	name := c.ProviderName()

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if s.healthCache == nil {
		s.healthCache = make(map[string]*providerHealth)
	}

	h, ok := s.healthCache[name]
	if ok && time.Since(h.checkedAt) < staleTTL(h) {
		return h // cache is fresh — no network call
	}

	// Cache is stale or missing — probe now.
	alive := c.Ping()
	updated := &providerHealth{
		alive:     alive,
		checkedAt: time.Now(),
	}
	if ok {
		updated.consecutiveFails = h.consecutiveFails
	}
	if alive {
		updated.activeModel = c.ActiveModel()
		if models, mErr := c.GetAvailableModels(); mErr != nil {
			log.Printf("discovery: get models from %s: %v", name, mErr)
		} else {
			updated.models = models
		}
		updated.consecutiveFails = 0
	} else {
		updated.consecutiveFails++
	}
	s.healthCache[name] = updated
	return updated
}

// InvalidateHealthCache clears all cached health entries so the next
// ListProviders / DetectActive call will re-probe all providers.
func (s *DiscoveryService) InvalidateHealthCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.healthCache = make(map[string]*providerHealth)
}
