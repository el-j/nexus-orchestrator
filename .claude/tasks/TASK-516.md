---
id: TASK-516
title: Add brain HTTP handlers
role: api
planId: PLAN-066
status: todo
dependencies: [TASK-513]
createdAt: 2026-04-13T00:00:00Z
---

## Context

Nine new HTTP endpoints under `/api/brain/` mirror the MCP brain tools. Pattern exactly matches existing handlers: decode request body, call service, writeJSON response. A `brainSvc ports.BrainService` field is added to the HTTP `Server` struct via the `WithBrainService` setter pattern (same as `WithActivityService`).

## Files to Read

- `internal/adapters/inbound/httpapi/server.go` — Server struct, Handler() route registration, WithActivityService pattern, writeJSON/writeJSONError helpers
- `internal/adapters/inbound/httpapi/handlers.go` — handler style: request decode, error handling, status codes
- `internal/core/ports/brain.go` — BrainService interface (TASK-510)
- `internal/core/domain/brain.go` — request/response types (TASK-509)

## Implementation Steps

1. **Add field to Server struct** in `server.go`:

   ```go
   brainSvc ports.BrainService
   ```

   Add setter (follow WithActivityService pattern):

   ```go
   func (s *Server) WithBrainService(svc ports.BrainService) *Server {
       s.brainSvc = svc
       return s
   }
   ```

2. **Register 9 routes** in `Handler()` method, after existing route groups:

   ```go
   // Brain (context intelligence)
   r.Post("/api/brain/init", s.handleBrainInit)
   r.Post("/api/brain/context", s.handleBrainGetContext)
   r.Post("/api/brain/focused", s.handleBrainGetFocused)
   r.Post("/api/brain/knowledge", s.handleBrainIngestKnowledge)
   r.Get("/api/brain/knowledge", s.handleBrainListKnowledge)
   r.Delete("/api/brain/knowledge/{id}", s.handleBrainDeleteKnowledge)
   r.Post("/api/brain/search", s.handleBrainSearch)
   r.Get("/api/brain/status", s.handleBrainStatus)
   r.Get("/api/brain/files", s.handleBrainFiles)
   ```

3. Create `internal/adapters/inbound/httpapi/handlers_brain.go`

4. Private guard helper at top of file:

   ```go
   func (s *Server) requireBrain(w http.ResponseWriter) bool {
       if s.brainSvc == nil {
           writeJSONError(w, "brain service not initialized", http.StatusNotImplemented)
           return false
       }
       return true
   }
   ```

5. Implement `handleBrainInit(w, r)`:
   - Decode body: `{projectPath string, claudeMdPath string}`
   - Call `s.brainSvc.InitProject(r.Context(), req.ProjectPath, req.ClaudeMdPath)`
   - Success: `writeJSON(w, http.StatusOK, status)`

6. Implement `handleBrainGetContext(w, r)`:
   - Decode body as `domain.ContextQuery`
   - Call `s.brainSvc.GetContext(r.Context(), query)`
   - Success: `writeJSON(w, http.StatusOK, resp)`

7. Implement `handleBrainGetFocused(w, r)`:
   - Decode body as `domain.ContextQuery` (Question field must be non-empty; 400 if missing)
   - Call `s.brainSvc.GetFocusedContext(r.Context(), query)`
   - Success: `writeJSON(w, http.StatusOK, resp)`

8. Implement `handleBrainIngestKnowledge(w, r)`:
   - Decode body as `domain.ProjectKnowledge`
   - Validate: ProjectPath, Kind, Topic, Content non-empty; 400 if any missing
   - Call `s.brainSvc.IngestKnowledge(r.Context(), k)`
   - Success: `writeJSON(w, http.StatusCreated, saved)`

9. Implement `handleBrainListKnowledge(w, r)`:
   - Query params: `project` (required), `kind` (optional)
   - If kind provided: call underlying repo via... check if BrainService exposes list-by-kind; if not, use GetContext with large budget and filter — OR expose a helper. Preferred: add `ListKnowledge(ctx, projectPath, kind string) ([]domain.ProjectKnowledge, error)` to BrainService interface (update TASK-510 port if needed, OR just call GetContext and flatten sections as a workaround if interface is already frozen).
   - Actually: for simplicity, have `handleBrainListKnowledge` call `s.brainSvc.GetContext` with MaxTokens=100000 and marshal as knowledge list. OR better: add `ListKnowledge` as a 9th method to BrainService interface in ports/brain.go, implement it in brain_service.go as `repo.GetByProject` / `repo.GetByProjectAndKind`. This is the clean approach.
   - Success: `writeJSON(w, http.StatusOK, entries)`

10. Implement `handleBrainDeleteKnowledge(w, r)`:
    - Path param: `id = chi.URLParam(r, "id")`
    - Need to call repo directly — expose `DeleteKnowledge` on BrainService or add `DeleteKnowledge(ctx, id string) error` as a 10th method. Add it cleanly to the interface.
    - On not-found: 404. On success: `writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})`

11. Implement `handleBrainSearch(w, r)`:
    - Decode body: `{projectPath, query string, maxTokens int}`
    - Call `s.brainSvc.SearchKnowledge(r.Context(), req.ProjectPath, req.Query, req.MaxTokens)`
    - Success: `writeJSON(w, http.StatusOK, sections)`

12. Implement `handleBrainStatus(w, r)`:
    - Query param: `project` (required)
    - Call `s.brainSvc.GetStatus(r.Context(), project)`
    - Success: `writeJSON(w, http.StatusOK, status)`

13. Implement `handleBrainFiles(w, r)`:
    - Query params: `project` (required), `focus` (optional, default "")
    - Call `s.brainSvc.GetFileMap(r.Context(), project, focus)`
    - Success: `writeJSON(w, http.StatusOK, files)`

**Note on interface additions**: If `ListKnowledge` and `DeleteKnowledge` are added to `ports.BrainService`, update `brain.go` port file (TASK-510 output) AND update `BrainServiceImpl` in `brain_service.go` (TASK-513 output) before implementing handlers.

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./cmd/nexus-cli/... ./cmd/nexus-daemon/...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] All 9 routes registered in Handler()
- [ ] Each handler calls `requireBrain` guard first
- [ ] Missing required fields return 400, not 500
- [ ] `WithBrainService` setter returns `*Server` for chaining

## Anti-patterns to Avoid

- NEVER call service methods without nil-checking brainSvc
- NEVER ignore request body decode errors
- NEVER use hardcoded strings for error messages — use descriptive messages
