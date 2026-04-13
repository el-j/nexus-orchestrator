package httpapi_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

type BrainClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewBrainClient(baseURL string) *BrainClient {
	return &BrainClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   strings.TrimSpace(os.Getenv("NEXUS_API_TOKEN")),
		client:  http.DefaultClient,
	}
}

func (r *BrainClient) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}
	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}
	return req, nil
}

func (r *BrainClient) do(req *http.Request) (*http.Response, error) {
	return r.client.Do(req)
}

func (r *BrainClient) GetContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	body, _ := json.Marshal(q)
	req, err := r.newRequest(ctx, http.MethodPost, "/api/brain/context", body)
	if err != nil {
		return domain.ContextResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return domain.ContextResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.ContextResponse{}, fmt.Errorf("remote: status %d", resp.StatusCode)
	}
	var out domain.ContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.ContextResponse{}, err
	}
	return out, nil
}

func (r *BrainClient) GetFocusedContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	body, _ := json.Marshal(q)
	req, err := r.newRequest(ctx, http.MethodPost, "/api/brain/focused-context", body)
	if err != nil {
		return domain.ContextResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return domain.ContextResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.ContextResponse{}, fmt.Errorf("remote: status %d", resp.StatusCode)
	}
	var out domain.ContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.ContextResponse{}, err
	}
	return out, nil
}

func (r *BrainClient) IngestKnowledge(ctx context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	return domain.ProjectKnowledge{}, fmt.Errorf("IngestKnowledge not implemented in client")
}

func (r *BrainClient) IngestFromFile(ctx context.Context, projectPath, filePath string) (int, error) {
	body, _ := json.Marshal(map[string]string{"projectPath": projectPath, "filePath": filePath})
	req, err := r.newRequest(ctx, http.MethodPost, "/api/brain/ingest", body)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("remote: status %d", resp.StatusCode)
	}
	var out map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	return out["ingestedSections"], nil
}

func (r *BrainClient) SearchKnowledge(ctx context.Context, projectPath, query string, limit int) ([]domain.ContextSection, error) {
	u := fmt.Sprintf("/api/brain/search?projectPath=%s&q=%s&limit=%d", url.QueryEscape(projectPath), url.QueryEscape(query), limit)
	req, err := r.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: status %d", resp.StatusCode)
	}
	var out struct {
		Results []domain.ContextSection `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Results, nil
}

func (r *BrainClient) GetFileMap(ctx context.Context, projectPath, focusArea string) ([]string, error) {
	return nil, fmt.Errorf("GetFileMap not implemented in client")
}

func (r *BrainClient) InitProject(ctx context.Context, projectPath, claudeMDPath string) (domain.BrainStatus, error) {
	return domain.BrainStatus{}, fmt.Errorf("InitProject not implemented in client")
}

func (r *BrainClient) GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error) {
	u := fmt.Sprintf("/api/brain/status?projectPath=%s", url.QueryEscape(projectPath))
	req, err := r.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return domain.BrainStatus{}, err
	}
	resp, err := r.do(req)
	if err != nil {
		return domain.BrainStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.BrainStatus{}, fmt.Errorf("remote: status %d", resp.StatusCode)
	}
	var out domain.BrainStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.BrainStatus{}, err
	}
	return out, nil
}

func (r *BrainClient) ListKnowledge(ctx context.Context, projectPath, kind string) ([]domain.ProjectKnowledge, error) {
	return nil, fmt.Errorf("ListKnowledge not implemented in client")
}

func (r *BrainClient) DeleteKnowledge(ctx context.Context, id string) error {
	return fmt.Errorf("DeleteKnowledge not implemented in client")
}
