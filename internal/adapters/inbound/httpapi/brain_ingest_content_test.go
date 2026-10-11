package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// uploadBrain is a brain service that can ingest uploaded content.
type uploadBrain struct {
	*mockBrainService
	count int
	err   error
	got   struct{ project, name, content string }
}

func (u *uploadBrain) IngestContent(_ context.Context, project, name, content string) (int, error) {
	u.got.project, u.got.name, u.got.content = project, name, content
	return u.count, u.err
}

func postContent(t *testing.T, brain ports.BrainService, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/brain/ingest-content", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	httpapi.NewServer(&mockOrchestrator{}, brain, nil).Handler().ServeHTTP(rec, req)
	return rec
}

func jsonBody(project, name, content string) string {
	b, _ := json.Marshal(map[string]string{"projectPath": project, "name": name, "content": content})
	return string(b)
}

func TestHandleIngestContent_StoresWhatTheUIUploaded(t *testing.T) {
	b := &uploadBrain{mockBrainService: &mockBrainService{}, count: 4}
	rec := postContent(t, b, jsonBody("/p", "notes.md", "## A\nbody"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]int
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["ingestedSections"] != 4 {
		t.Errorf("response = %v", resp)
	}
	if b.got.project != "/p" || b.got.name != "notes.md" || b.got.content != "## A\nbody" {
		t.Errorf("service got %+v", b.got)
	}
}

func TestHandleIngestContent_RejectsBadRequests(t *testing.T) {
	b := &uploadBrain{mockBrainService: &mockBrainService{}}
	for name, body := range map[string]string{
		"not json":        `{`,
		"missing content": jsonBody("/p", "n", ""),
		"missing project": jsonBody("", "n", "c"),
	} {
		if rec := postContent(t, b, body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, rec.Code)
		}
	}
}

func TestHandleIngestContent_RefusesOversizedUploads(t *testing.T) {
	b := &uploadBrain{mockBrainService: &mockBrainService{}}
	// Larger than the request cap: refused before the body is buffered.
	huge := jsonBody("/p", "n", strings.Repeat("x", 3*domain.MaxIngestBytes))
	if rec := postContent(t, b, huge); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("huge body: status %d, want 413", rec.Code)
	}
	if b.got.content != "" {
		t.Error("an oversized body must never reach the service")
	}
	// Within the cap but over the content limit: the service says so.
	b.err = domain.ErrInputTooLarge
	if rec := postContent(t, b, jsonBody("/p", "n", "c")); rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("service limit: status %d, want 413", rec.Code)
	}
}

func TestHandleIngestContent_DoesNotLeakInternalErrors(t *testing.T) {
	b := &uploadBrain{mockBrainService: &mockBrainService{}, err: errors.New("open /var/lib/nexus/secret.db: boom")}
	rec := postContent(t, b, jsonBody("/p", "n", "c"))
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "/var/lib") {
		t.Errorf("status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestHandleIngestContent_ServiceWithoutUploadSupportAndMissingService(t *testing.T) {
	if rec := postContent(t, &mockBrainService{}, jsonBody("/p", "n", "c")); rec.Code != http.StatusNotImplemented {
		t.Errorf("brain without upload support: status %d, want 501", rec.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/brain/ingest-content", bytes.NewBufferString(jsonBody("/p", "n", "c")))
	rec := httptest.NewRecorder()
	serveNilBrain(req, rec)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("no brain: status %d, want 503", rec.Code)
	}
}
