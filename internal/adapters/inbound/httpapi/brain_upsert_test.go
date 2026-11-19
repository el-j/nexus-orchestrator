package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/httpapi"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// upsertBrain records the entry it is asked to store.
type upsertBrain struct {
	*mockBrainService
	got domain.ProjectKnowledge
	err error
}

func (u *upsertBrain) IngestKnowledge(_ context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	u.got = k
	if u.err != nil {
		return domain.ProjectKnowledge{}, u.err
	}
	k.ID, k.TokenCount = "stored-1", 7
	return k, nil
}

func postKnowledge(t *testing.T, brain ports.BrainService, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/brain/knowledge", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	httpapi.NewServer(&mockOrchestrator{}, brain, nil).Handler().ServeHTTP(rec, req)
	return rec
}

func TestUpsertKnowledge_StoresOneEntryAndReturnsIt(t *testing.T) {
	b := &upsertBrain{mockBrainService: &mockBrainService{}}
	rec := postKnowledge(t, b, `{"projectPath":"/p","kind":"learning","topic":"Quota","content":"A 429 puts the provider on cooldown.","source":"agent"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if b.got.ProjectPath != "/p" || b.got.Kind != domain.KnowledgeLearning || b.got.Topic != "Quota" || b.got.Source != "agent" {
		t.Errorf("stored: %+v", b.got)
	}
	if !strings.Contains(rec.Body.String(), `"id":"stored-1"`) {
		t.Errorf("response: %s", rec.Body)
	}
}

func TestUpsertKnowledge_RejectsBadRequests(t *testing.T) {
	b := &upsertBrain{mockBrainService: &mockBrainService{}}
	cases := map[string]struct {
		body string
		want int
	}{
		"not json":     {`{`, http.StatusBadRequest},
		"no project":   {`{"kind":"learning","topic":"t","content":"c"}`, http.StatusBadRequest},
		"no topic":     {`{"projectPath":"/p","kind":"learning","content":"c"}`, http.StatusBadRequest},
		"no content":   {`{"projectPath":"/p","kind":"learning","topic":"t"}`, http.StatusBadRequest},
		"unknown kind": {`{"projectPath":"/p","kind":"gossip","topic":"t","content":"c"}`, http.StatusBadRequest},
		"missing kind": {`{"projectPath":"/p","topic":"t","content":"c"}`, http.StatusBadRequest},
		"too large":    {`{"projectPath":"/p","kind":"learning","topic":"t","content":"` + strings.Repeat("x", domain.MaxIngestBytes+1) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for name, c := range cases {
		if rec := postKnowledge(t, b, c.body); rec.Code != c.want {
			t.Errorf("%s: status %d, want %d (%s)", name, rec.Code, c.want, rec.Body)
		}
	}
}

func TestUpsertKnowledge_BackendProblems(t *testing.T) {
	good := `{"projectPath":"/p","kind":"learning","topic":"t","content":"c"}`
	if rec := postKnowledge(t, &upsertBrain{mockBrainService: &mockBrainService{}, err: errors.New("db: /secret")}, good); rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "secret") {
		t.Errorf("a backend error must not leak: %d %s", rec.Code, rec.Body)
	}
	if rec := postKnowledge(t, nil, good); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("no brain: %d", rec.Code)
	}
}
