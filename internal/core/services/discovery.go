package services

import (
	"fmt"
	"strings"
	"sync"

	"nexus-orchestrator/internal/core/ports"
)

// DiscoveryService probes registered LLM clients and returns the first active one.
// All exported methods are safe for concurrent use.
type DiscoveryService struct {
	mu               sync.RWMutex
	availableClients []ports.LLMClient
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
	for _, client := range clients {
		if client.Ping() {
			return client
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

	// Pass 1: provider with the model already loaded / active.
	for _, c := range candidates {
		if c.Ping() && strings.EqualFold(c.ActiveModel(), modelID) {
			return c, nil
		}
	}

	// Pass 2: provider that can load the model (listed in its model catalogue).
	for _, c := range candidates {
		if !c.Ping() {
			continue
		}
		models, err := c.GetAvailableModels()
		if err != nil {
			continue
		}
		for _, m := range models {
			if strings.EqualFold(m, modelID) {
				return c, nil
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
		alive := c.Ping()
		info := ports.ProviderInfo{
			Name:    c.ProviderName(),
			Active:  alive,
			BaseURL: c.BaseURL(),
		}
		if alive {
			info.ActiveModel = c.ActiveModel()
			models, _ := c.GetAvailableModels()
			info.Models = models
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
