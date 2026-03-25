package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.orch.GetProviders()
	if err != nil {
		log.Printf("httpapi: get providers: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if providers == nil {
		providers = []ports.ProviderInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providers)
}

func (s *Server) handleRegisterProvider(w http.ResponseWriter, r *http.Request) {
	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if cfg.Name == "" || cfg.Kind == "" {
		writeJSONError(w, "name and kind are required", http.StatusBadRequest)
		return
	}
	if err := s.orch.RegisterCloudProvider(cfg); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		writeJSONError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"name": cfg.Name, "kind": string(cfg.Kind)})
}

func (s *Server) handleRemoveProvider(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := s.orch.RemoveProvider(name); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: remove provider %s: %v", name, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleProviderModels(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	models, err := s.orch.GetProviderModels(name)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: provider models %s: %v", name, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if models == nil {
		models = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(models)
}

// maskAPIKey returns a masked representation of an API key:
// if longer than 4 characters, the last 4 are preserved; otherwise "****".
func maskAPIKey(key string) string {
	if len(key) > 4 {
		return "****" + key[len(key)-4:]
	}
	return "****"
}

// maskedProviderConfig returns a copy of cfg with the APIKey field masked.
func maskedProviderConfig(cfg domain.ProviderConfig) domain.ProviderConfig {
	if cfg.APIKey != "" {
		cfg.APIKey = maskAPIKey(cfg.APIKey)
	}
	return cfg
}

func (s *Server) handleAddProviderConfig(w http.ResponseWriter, r *http.Request) {
	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if cfg.Name == "" || cfg.Kind == "" {
		writeJSONError(w, "name and kind are required", http.StatusBadRequest)
		return
	}
	created, err := s.orch.AddProviderConfig(r.Context(), cfg)
	if err != nil {
		log.Printf("httpapi: add provider config: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(maskedProviderConfig(created))
}

func (s *Server) handleListProviderConfigs(w http.ResponseWriter, r *http.Request) {
	cfgs, err := s.orch.ListProviderConfigs(r.Context())
	if err != nil {
		log.Printf("httpapi: list provider configs: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	masked := make([]domain.ProviderConfig, len(cfgs))
	for i, c := range cfgs {
		masked[i] = maskedProviderConfig(c)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(masked)
}

func (s *Server) handleUpdateProviderConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var cfg domain.ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSONError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg.ID = id
	updated, err := s.orch.UpdateProviderConfig(r.Context(), cfg)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider config not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: update provider config %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(maskedProviderConfig(updated))
}

func (s *Server) handleRemoveProviderConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.RemoveProviderConfig(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider config not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: remove provider config %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetDiscoveredProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.orch.GetDiscoveredProviders()
	if err != nil {
		writeJSONError(w, "failed to get discovered providers", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providers)
}

func (s *Server) handleTriggerScan(w http.ResponseWriter, r *http.Request) {
	providers, err := s.orch.TriggerScan(r.Context())
	if err != nil {
		writeJSONError(w, "scan failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(providers)
}

func (s *Server) handlePromoteProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := s.orch.PromoteProvider(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "provider not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, "failed to promote provider", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
