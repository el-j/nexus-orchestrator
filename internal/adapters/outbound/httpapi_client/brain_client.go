package httpapi_client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// Compile-time proof that BrainClient satisfies the brain port.
var _ ports.BrainService = (*BrainClient)(nil)

// BrainClient implements ports.BrainService by calling the daemon's
// /api/brain/* endpoints. It shares the transport, bearer-token handling and
// error conventions of Client.
type BrainClient struct {
	c *Client
}

// NewBrainClient returns a BrainClient for the daemon at baseURL. The bearer
// token, when required, is read from NEXUS_API_TOKEN.
func NewBrainClient(baseURL string) *BrainClient {
	return &BrainClient{c: NewClient(baseURL)}
}

// GetContext assembles a token-budgeted context response for a project.
func (r *BrainClient) GetContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	var out domain.ContextResponse
	err := r.c.send(ctx, call{op: "brain get context", method: http.MethodPost, path: "/api/brain/context", body: q}, &out)
	return out, err
}

// GetFocusedContext assembles context relevant to q.Question using full-text search.
func (r *BrainClient) GetFocusedContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	var out domain.ContextResponse
	err := r.c.send(ctx, call{op: "brain get focused context", method: http.MethodPost, path: "/api/brain/focused-context", body: q}, &out)
	return out, err
}

// IngestKnowledge is not available over HTTP: the daemon only exposes bulk
// ingestion from a file (see IngestFromFile).
func (r *BrainClient) IngestKnowledge(_ context.Context, _ domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	return domain.ProjectKnowledge{}, fmt.Errorf("brain_client: IngestKnowledge: direct knowledge upsert not supported via HTTP client; use IngestFromFile")
}

// IngestFromFile asks the daemon to ingest a CLAUDE.md-style file and returns
// the number of sections stored.
func (r *BrainClient) IngestFromFile(ctx context.Context, projectPath, filePath string) (int, error) {
	var out struct {
		IngestedSections int `json:"ingestedSections"`
	}
	body := map[string]string{"projectPath": projectPath, "filePath": filePath}
	err := r.c.send(ctx, call{op: "brain ingest from file", method: http.MethodPost, path: "/api/brain/ingest", body: body}, &out)
	return out.IngestedSections, err
}

// SearchKnowledge runs a full-text search and returns at most limit sections.
func (r *BrainClient) SearchKnowledge(ctx context.Context, projectPath, query string, limit int) ([]domain.ContextSection, error) {
	q := url.Values{"projectPath": {projectPath}, "q": {query}, "limit": {strconv.Itoa(limit)}}
	var out struct {
		Results []domain.ContextSection `json:"results"`
	}
	err := r.c.send(ctx, call{op: "brain search", method: http.MethodGet, path: "/api/brain/search?" + q.Encode()}, &out)
	return out.Results, err
}

// GetFileMap returns the project's file-map entries, optionally narrowed by focusArea.
func (r *BrainClient) GetFileMap(ctx context.Context, projectPath, focusArea string) ([]string, error) {
	q := url.Values{"projectPath": {projectPath}, "focusArea": {focusArea}}
	var out struct {
		FilePaths []string `json:"filePaths"`
	}
	err := r.c.send(ctx, call{op: "brain file map", method: http.MethodGet, path: "/api/brain/file-map?" + q.Encode()}, &out)
	return out.FilePaths, err
}

// InitProject ingests the project's CLAUDE.md (auto-detected when claudeMDPath
// is empty) and returns the resulting brain status.
func (r *BrainClient) InitProject(ctx context.Context, projectPath, claudeMDPath string) (domain.BrainStatus, error) {
	var out domain.BrainStatus
	body := map[string]string{"projectPath": projectPath, "claudeMDPath": claudeMDPath}
	err := r.c.send(ctx, call{op: "brain init project", method: http.MethodPost, path: "/api/brain/init", body: body}, &out)
	return out, err
}

// GetStatus returns the knowledge-base status for a project.
func (r *BrainClient) GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error) {
	var out domain.BrainStatus
	err := r.c.send(ctx, call{op: "brain status", method: http.MethodGet, path: "/api/brain/status" + query("projectPath", projectPath)}, &out)
	return out, err
}

// ListKnowledge lists a project's knowledge entries, optionally filtered by kind.
// The result is never nil.
func (r *BrainClient) ListKnowledge(ctx context.Context, projectPath, kind string) ([]domain.ProjectKnowledge, error) {
	q := url.Values{"projectPath": {projectPath}, "kind": {kind}}
	out := []domain.ProjectKnowledge{}
	if err := r.c.send(ctx, call{op: "brain list knowledge", method: http.MethodGet, path: "/api/brain/knowledge?" + q.Encode()}, &out); err != nil {
		return nil, err
	}
	if out == nil { // the daemon answered with JSON null
		out = []domain.ProjectKnowledge{}
	}
	return out, nil
}

// DeleteKnowledge removes a knowledge entry by ID.
func (r *BrainClient) DeleteKnowledge(ctx context.Context, id string) error {
	return r.c.send(ctx, call{op: "brain delete knowledge", method: http.MethodDelete, path: "/api/brain/knowledge/" + esc(id), ok: statusNoBody}, nil)
}

// GetOnboardingContext returns a concise onboarding summary for an agent
// starting work on a project; maxTokens <= 0 lets the daemon choose.
func (r *BrainClient) GetOnboardingContext(ctx context.Context, projectPath string, maxTokens int) (string, error) {
	q := url.Values{"projectPath": {projectPath}}
	if maxTokens > 0 {
		q.Set("maxTokens", strconv.Itoa(maxTokens))
	}
	var out struct {
		Content string `json:"content"`
	}
	err := r.c.send(ctx, call{op: "brain onboarding context", method: http.MethodGet, path: "/api/brain/onboarding?" + q.Encode()}, &out)
	return out.Content, err
}
