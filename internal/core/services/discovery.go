package services

import (
	"fmt"
	"strings"

	"nexus-orchestrator/internal/core/ports"
)

// DiscoveryService probes registered LLM clients and returns the first active one.
type DiscoveryService struct {
	availableClients []ports.LLMClient
}

// NewDiscoveryService creates a DiscoveryService with the supplied LLM adapters.
func NewDiscoveryService(clients ...ports.LLMClient) *DiscoveryService {
	return &DiscoveryService{availableClients: clients}
}

// RegisterProvider adds a new LLM adapter at runtime.
func (s *DiscoveryService) RegisterProvider(c ports.LLMClient) {
	s.availableClients = append(s.availableClients, c)
}

// DetectActive returns the first LLM provider that responds to a Ping,
// or nil when none are reachable.
func (s *DiscoveryService) DetectActive() ports.LLMClient {
	for _, client := range s.availableClients {
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
// providerHint (case-insensitive prefix of ProviderName) is tried first in
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
	result := make([]ports.ProviderInfo, 0, len(s.availableClients))
	for _, c := range s.availableClients {
		alive := c.Ping()
		info := ports.ProviderInfo{
			Name:   c.ProviderName(),
			Active: alive,
		}
		if alive {
			info.ActiveModel = c.ActiveModel()
			models, _ := c.GetAvailableModels()
			info.Models = models
		}
		result = append(result, info)
	}
	return result
}

// orderedCandidates returns all clients with hint-matching providers first.
func (s *DiscoveryService) orderedCandidates(hint string) []ports.LLMClient {
	if hint == "" {
		return s.availableClients
	}
	var first, rest []ports.LLMClient
	for _, c := range s.availableClients {
		if strings.Contains(strings.ToLower(c.ProviderName()), strings.ToLower(hint)) {
			first = append(first, c)
		} else {
			rest = append(rest, c)
		}
	}
	return append(first, rest...)
}
