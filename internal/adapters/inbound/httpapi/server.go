package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"nexus-ai/internal/core/domain"
	"nexus-ai/internal/core/ports"
)

// StartServer starts the VS Code extension HTTP API on the given address (e.g. ":9999").
// It blocks until the server exits.
func StartServer(orch ports.Orchestrator, addr string) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// POST /api/tasks — submit a new task
	r.Post("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		var req domain.Task
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		taskID, err := orch.SubmitTask(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"task_id": taskID,
			"status":  string(domain.StatusQueued),
		})
	})

	// GET /api/tasks — list pending tasks
	r.Get("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		tasks, err := orch.GetQueue()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tasks)
	})

	// DELETE /api/tasks/{id} — cancel a queued task
	r.Delete("/api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := orch.CancelTask(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	fmt.Println("🚀 NexusAI VS Code API listening on", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		fmt.Printf("httpapi: server error: %v\n", err)
	}
}
