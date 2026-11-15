package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// The route table, one function per area. Within an area, literal segments are registered before the
// wildcard that could swallow them.

// routeUI registers the root redirect, the full UI (when this binary has it) and the built-in dashboard.
func (s *Server) routeUI(r chi.Router) {
	// The root goes to the full UI when this binary has it, else to the built-in dashboard,
	// which stays available at /ui either way.
	home := "/ui"
	if s.ui != nil {
		home = "/app/"
		r.Get("/app", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/app/", http.StatusFound) })
		r.Handle("/app/*", s.ui)
	}
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, home, http.StatusFound)
	})
	r.Get("/ui", s.handleUI)
}

// routeTasks registers the task endpoints: queue, backlog, claim, external status and heartbeat.
func (s *Server) routeTasks(r chi.Router) {
	r.Post("/api/tasks", s.handleCreateTask)
	r.Get("/api/tasks", s.handleListTasks)
	r.Get("/api/tasks/all", s.handleGetAllTasks)
	r.Post("/api/tasks/draft", s.handleCreateDraft)
	r.Get("/api/tasks/backlog", s.handleGetBacklog)
	r.Get("/api/tasks/{id}", s.handleGetTask)
	r.Delete("/api/tasks/{id}", s.handleCancelTask)
	r.Post("/api/tasks/{id}/promote", s.handlePromoteTask)
	r.Put("/api/tasks/{id}", s.handleUpdateTask)
	r.Post("/api/tasks/{id}/claim", s.handleClaimTask)
	r.Put("/api/tasks/{id}/status", s.handleUpdateTaskStatus)
	r.Post("/api/tasks/{id}/heartbeat", s.handleHeartbeatTask)
}

// routeProviders registers the provider endpoints: running providers, discovery and stored configurations.
func (s *Server) routeProviders(r chi.Router) {
	r.Get("/api/providers", s.handleProviders)
	r.Post("/api/providers", s.handleRegisterProvider)

	r.Get("/api/providers/discovered", s.handleGetDiscoveredProviders)
	r.Post("/api/providers/discovered/scan", s.handleTriggerScan)
	r.Post("/api/providers/promote/{id}", s.handlePromoteProvider)

	r.Delete("/api/providers/{name}", s.handleRemoveProvider)
	r.Get("/api/providers/{name}/models", s.handleProviderModels)

	// Stored configurations (the API key is masked in every response).
	r.Post("/api/providers/config", s.handleAddProviderConfig)
	r.Get("/api/providers/config", s.handleListProviderConfigs)
	r.Put("/api/providers/config/{id}", s.handleUpdateProviderConfig)
	r.Delete("/api/providers/config/{id}", s.handleRemoveProviderConfig)
}

// routeSessions registers the AI session endpoints.
func (s *Server) routeSessions(r chi.Router) {
	r.Post("/api/ai-sessions", s.handleRegisterAISession)
	r.Get("/api/ai-sessions", s.handleListAISessions)
	r.Delete("/api/ai-sessions", s.handlePurgeDisconnectedSessions)
	r.Get("/api/ai-sessions/discovered", s.handleGetDiscoveredAgents)
	r.Delete("/api/ai-sessions/{id}", s.handleDeregisterAISession)
	r.Post("/api/ai-sessions/{id}/heartbeat", s.handleHeartbeatAISession)
	r.Post("/api/ai-sessions/{id}/terminate", s.handleTerminateAISession)
	r.Get("/api/ai-sessions/{id}/tasks", s.handleGetSessionTasks)
	r.Post("/api/ai-sessions/{id}/delegate", s.handleDelegateToNexus)
}

// routeSystem registers health, logs, settings, the event stream, the observatory (usage, routing,
// policy, activity), plan files and the discovery documents.
func (s *Server) routeSystem(r chi.Router) {
	r.Get("/api/health", s.handleHealth)
	r.Get("/api/logs", s.handleGetLogs)
	r.Get("/api/config", s.handleGetConfig)
	r.Put("/api/config", s.handlePutConfig)
	r.Get("/api/events", s.handleEvents)

	r.Get("/api/usage", s.handleUsage)
	r.Get("/api/routing", s.handleRouting)
	r.Get("/api/policy", s.handleGetPolicy)
	r.Put("/api/policy", s.handlePutPolicy)
	r.Get("/api/activities", s.handleListActivities)
	r.Get("/api/activities/timeline", s.handleActivityTimeline)

	r.Get("/api/plans/discovered", s.handleGetDiscoveredPlanFiles)
	r.Post("/api/plans/discovered/scan", s.handleScanPlanFiles)

	r.Get("/api/howto", s.handleHowto)
	r.Get("/.well-known/nexus.json", s.handleWellKnownNexus)
}

// routeBrain registers the knowledge endpoints.
func (s *Server) routeBrain(r chi.Router) {
	r.Post("/api/brain/ingest", s.handleIngestKnowledge)
	r.Post("/api/brain/ingest-content", s.handleIngestContent)
	r.Post("/api/brain/knowledge", s.handleUpsertKnowledge)
	r.Get("/api/brain/status", s.handleGetBrainStatus)
	r.Post("/api/brain/context", s.handleGetProjectContext)
	r.Post("/api/brain/focused-context", s.handleGetFocusedContext)
	r.Get("/api/brain/search", s.handleSearchKnowledge)
	r.Post("/api/brain/init", s.handleInitProject)
	r.Get("/api/brain/knowledge", s.handleListKnowledge)
	r.Delete("/api/brain/knowledge/{id}", s.handleDeleteKnowledge)
	r.Get("/api/brain/file-map", s.handleGetFileMap)
	r.Get("/api/brain/onboarding", s.handleGetOnboardingContext)
}
