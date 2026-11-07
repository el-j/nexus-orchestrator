package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"nexus-orchestrator/internal/core/domain"

	"github.com/go-chi/chi/v5"
)

// handleCreateTask queues a task for the workers.
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
		if errors.Is(err, domain.ErrQueueFull) {
			writeQueueFull(w)
			return
		}
		log.Printf("httpapi: create task: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
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

// handleListTasks lists the tasks that are waiting or running, for one project or all.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	var (
		tasks []domain.Task
		err   error
	)
	// When a caller supplies projectPath we return only that project's queue,
	// preventing agents on different projects from seeing each other's work.
	if projectPath := r.URL.Query().Get("projectPath"); projectPath != "" {
		tasks, err = s.orch.GetQueueForProject(projectPath)
	} else {
		tasks, err = s.orch.GetQueue()
	}
	if err != nil {
		log.Printf("httpapi: list tasks: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	computeTaskDurations(tasks)
	writeJSON(w, http.StatusOK, tasks)
}

// DefaultTaskPage is how many tasks GET /api/tasks/all returns when no limit is given, and MaxTaskPage the
// most a caller may ask for: the history grows without end and one response must not.
const (
	DefaultTaskPage = 1000
	MaxTaskPage     = 5000
)

// pageParams reads limit and offset from the query. A limit above MaxTaskPage is lowered to it; a limit
// below 1, an offset below 0 or a value that is not a number is an error.
func pageParams(r *http.Request) (limit, offset int, err error) {
	limit = DefaultTaskPage
	q := r.URL.Query()
	if v := q.Get("limit"); v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 {
			return 0, 0, errors.New("limit must be a whole number of at least 1")
		}
		limit = min(limit, MaxTaskPage)
	}
	if v := q.Get("offset"); v != "" {
		if offset, err = strconv.Atoi(v); err != nil || offset < 0 {
			return 0, 0, errors.New("offset must be a whole number of at least 0")
		}
	}
	return limit, offset, nil
}

// handleGetAllTasks lists tasks, finished ones included, newest first, for one project or all. The list
// is paged (limit, offset; see pageParams) and the total is in the X-Total-Count header.
func (s *Server) handleGetAllTasks(w http.ResponseWriter, r *http.Request) {
	limit, offset, perr := pageParams(r)
	if perr != nil {
		writeJSONError(w, perr.Error(), http.StatusBadRequest)
		return
	}
	var (
		tasks []domain.Task
		err   error
	)
	// Filter at the database level when projectPath is provided.
	if projectPath := r.URL.Query().Get("projectPath"); projectPath != "" {
		tasks, err = s.orch.GetTasksForProject(projectPath)
	} else {
		tasks, err = s.orch.GetAllTasks()
	}
	if err != nil {
		log.Printf("httpapi: get all tasks: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(len(tasks)))
	tasks = tasks[min(offset, len(tasks)):]
	tasks = tasks[:min(limit, len(tasks))]
	if tasks == nil {
		tasks = []domain.Task{}
	}
	computeTaskDurations(tasks)
	writeJSON(w, http.StatusOK, tasks)
}

// handleGetTask returns one task; 404 when it does not exist.
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
	writeJSON(w, http.StatusOK, task)
}

// handleCancelTask cancels a task that has not finished.
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

// handleCreateDraft stores a task in the backlog without running it.
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
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "DRAFT"})
}

// handleGetBacklog lists the drafts of a project.
func (s *Server) handleGetBacklog(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("project")
	tasks, err := s.orch.GetBacklog(projectPath)
	if err != nil {
		log.Printf("httpapi: get backlog: %v", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	computeTaskDurations(tasks)
	writeJSON(w, http.StatusOK, tasks)
}

// handlePromoteTask moves a draft into the queue.
func (s *Server) handlePromoteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	result, err := s.orch.PromoteTask(id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, domain.ErrQueueFull) {
			writeQueueFull(w)
			return
		}
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleUpdateTask edits a task that has not started.
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
	writeJSON(w, http.StatusOK, updated)
}

// handleClaimTask lets a session take a queued task to work on itself.
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
		if strings.Contains(err.Error(), "does not match task project") {
			writeJSONError(w, err.Error(), http.StatusForbidden)
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
	writeJSON(w, http.StatusOK, task)
}

// handleUpdateTaskStatus lets the claiming session report progress, a result or a failure.
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
	writeJSON(w, http.StatusOK, task)
}

// handleHeartbeatTask keeps a claimed task from being released as stale.
func (s *Server) handleHeartbeatTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	// "sessionId" is canonical (as for /claim and /status); "session_id" is
	// accepted for older callers and the MCP parameter naming.
	var body struct {
		SessionID       string `json:"sessionId"`
		LegacySessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "sessionId is required", http.StatusBadRequest)
		return
	}
	sessionID := body.SessionID
	if sessionID == "" {
		sessionID = body.LegacySessionID
	}
	if sessionID == "" {
		writeJSONError(w, "sessionId is required", http.StatusBadRequest)
		return
	}
	if err := s.orch.HeartbeatTask(r.Context(), taskID, sessionID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeJSONError(w, "task or session not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "not processing") ||
			strings.Contains(err.Error(), "not claimed by session") ||
			strings.Contains(err.Error(), "status changed") {
			writeJSONError(w, err.Error(), http.StatusConflict)
			return
		}
		log.Printf("httpapi: heartbeat task %s: %v", taskID, err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetSessionTasks lists the tasks a session has claimed.
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
	writeJSON(w, http.StatusOK, sessionTasks)
}
