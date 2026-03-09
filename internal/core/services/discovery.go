package services

import "nexus-ai/internal/core/ports"

// DiscoveryService probes registered LLM clients and returns the first active one.
type DiscoveryService struct {
	availableClients []ports.LLMClient
}

// NewDiscoveryService creates a DiscoveryService with the supplied LLM adapters.
func NewDiscoveryService(clients ...ports.LLMClient) *DiscoveryService {
	return &DiscoveryService{availableClients: clients}
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
