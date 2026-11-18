package embedding_client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/embedding_client"
)

func TestClient_Embed_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("expected /v1/embeddings, got %s", r.URL.Path)
		}
		var req struct {
			Model string `json:"model"`
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Input != "hello world" {
			t.Errorf("expected input 'hello world', got %q", req.Input)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"embedding": []float32{0.1, 0.2, 0.3}},
			},
		})
	}))
	defer ts.Close()

	c := embedding_client.NewClient(embedding_client.Config{
		BaseURL: ts.URL,
		Model:   "test-model",
	})

	emb, err := c.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(emb) != 3 || emb[0] != 0.1 || emb[1] != 0.2 || emb[2] != 0.3 {
		t.Errorf("unexpected embedding: %+v", emb)
	}
}

func TestClient_Embed_Errors(t *testing.T) {
	// 1. Empty text
	c := embedding_client.NewClient(embedding_client.Config{})
	if _, err := c.Embed(context.Background(), "   "); err == nil {
		t.Errorf("expected error for empty text")
	}

	// 2. Non-200 status code
	tsErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer tsErr.Close()

	cErr := embedding_client.NewClient(embedding_client.Config{BaseURL: tsErr.URL})
	if _, err := cErr.Embed(context.Background(), "test"); err == nil {
		t.Errorf("expected error for 500 status")
	}

	// 3. API Error payload
	tsAPIErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "model not found"},
		})
	}))
	defer tsAPIErr.Close()

	cAPIErr := embedding_client.NewClient(embedding_client.Config{BaseURL: tsAPIErr.URL})
	if _, err := cAPIErr.Embed(context.Background(), "test"); err == nil {
		t.Errorf("expected error for API error payload")
	}

	// 4. Invalid JSON
	tsBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not-json`))
	}))
	defer tsBadJSON.Close()

	cBadJSON := embedding_client.NewClient(embedding_client.Config{BaseURL: tsBadJSON.URL})
	if _, err := cBadJSON.Embed(context.Background(), "test"); err == nil {
		t.Errorf("expected error for invalid JSON")
	}

	// 5. Empty data
	tsEmptyData := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{},
		})
	}))
	defer tsEmptyData.Close()

	cEmptyData := embedding_client.NewClient(embedding_client.Config{BaseURL: tsEmptyData.URL})
	if _, err := cEmptyData.Embed(context.Background(), "test"); err == nil {
		t.Errorf("expected error for empty data")
	}

	// 6. Network connection error
	cNetErr := embedding_client.NewClient(embedding_client.Config{BaseURL: "http://127.0.0.1:1"})
	if _, err := cNetErr.Embed(context.Background(), "test"); err == nil {
		t.Errorf("expected network error")
	}

	// 7. Invalid URL error
	cBadURL := embedding_client.NewClient(embedding_client.Config{BaseURL: "http://[::1]:bad port"})
	if _, err := cBadURL.Embed(context.Background(), "test"); err == nil {
		t.Errorf("expected URL creation error")
	}

	// 8. Body read error
	cReadErr := embedding_client.NewClient(embedding_client.Config{
		BaseURL: "http://dummy",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(errReader{}),
				}, nil
			}),
		},
	})
	if _, err := cReadErr.Embed(context.Background(), "test"); err == nil {
		t.Errorf("expected body read error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
