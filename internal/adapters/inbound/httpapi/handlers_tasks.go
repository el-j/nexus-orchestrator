package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"nexus-orchestrator/internal/core/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req domain.Task
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	taskID, err := s.orch.SubmitTask(req)
	if err != nil {
		if errors.Is(err, domain.ErrNoPlan) {
			writeJSONError(w, "planning required before execution; submit a 'plan' task first", http.StatusUnprocessableEntity)
			return
		}
		log.Printf("httpapi: create task: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"task_id": taskID,
		"status":  string(domain.StatusQueued),
	})
}

// computeTaskDurations sets the DurationMs field on each task in the slice.
func computeTaskDurations(tasks []domain.Task) {
	for i := range tasks {
		tasks[i].ComputeDuration()
	}
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.orch.GetQueue()
	if err != nil {
		log.Printf("httpapi: list tasks: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	computeTaskDurations(tasks)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleGetAllTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.orch.GetAllTasks()
	if err != nil {
		log.Printf("httpapi: get all tasks: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	projectPath := r.URL.Query().Get("projectPath")
	if projectPath != "" {
		filtered := make([]domain.Task, 0, len(tasks))
		for _, t := range tasks {
			if t.ProjectPath == projectPath {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	computeTaskDurations(tasks)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := s.orch.GetTask(id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: get task %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	task.ComputeDuration()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(task)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orch.CancelTask(id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "cannot cancel task with status") {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("httpapi: cancel task %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateDraft(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	id, err := s.orch.CreateDraft(task)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "DRAFT"})
}

func (s *Server) handleGetBacklog(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("project")
	tasks, err := s.orch.GetBacklog(projectPath)
	if err != nil {
		log.Printf("httpapi: get backlog: %v", err)
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	computeTaskDurations(tasks)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

func (s *Server) handlePromoteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := s.orch.PromoteTask(id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var updates domain.Task
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	updated, err := s.orch.UpdateTask(id, updates)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	updated.ComputeDuration()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (s *Server) handleClaimTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" {
		writeJSONError(w, "sessionId is required", http.StatusBadRequest)
		return
	}
	task, err := s.orch.ClaimTask(r.Context(), id, body.SessionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task or session not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "not QUEUED") || strings.Contains(err.Error(), "is disconnected") {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("httpapi: claim task %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	task.ComputeDuration()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(task)
}

func (s *Server) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		SessionID string `json:"sessionId"`
		Status    string `json:"status"`
		Logs      string `json:"logs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" || body.Status == "" {
		writeJSONError(w, "sessionId and status are required", http.StatusBadRequest)
		return
	}
	if body.Status != "COMPLETED" && body.Status != "FAILED" {
		writeJSONError(w, "status must be COMPLETED or FAILED", http.StatusBadRequest)
		return
	}
	task, err := s.orch.UpdateTaskStatus(r.Context(), id, body.SessionID, domain.TaskStatus(body.Status), body.Logs)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "does not own") {
			writeJSONError(w, err.Error(), http.StatusForbidden)
			return
		}
		if strings.Contains(err.Error(), "not PROCESSING") || strings.Contains(err.Error(), "invalid target status") {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("httpapi: update task status %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	task.ComputeDuration()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(task)
}

func (s *Server) handleHeartbeatTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	var body struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" {
		writeJSONError(w, "session_id is required", http.StatusBadRequest)
		return
	}
	if err := s.orch.HeartbeatTask(r.Context(), taskID, body.SessionID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task or session not found", http.StatusNotFound)
			return
		}
		log.Printf("httpapi: heartbeat task %s: %v", taskID, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetSessionTasks(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// We use GetAllTasks and filter by AISessionID since there's no direct port method on Orchestrator
	// for session-scoped task query. The repo method is on TaskRepository, not Orchestrator.
	allTasks, err := s.orch.GetAllTasks()
	if err != nil {
		log.Printf("httpapi: get session tasks %s: %v", id, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	var sessionTasks []domain.Task
	for _, t := range allTasks {
		if t.AISessionID == id {
			sessionTasks = append(sessionTasks, t)
		}
	}
	if sessionTasks == nil {
		sessionTasks = []domain.Task{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sessionTasks)
}
