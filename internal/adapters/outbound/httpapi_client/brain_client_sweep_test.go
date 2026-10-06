package httpapi_client_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/outbound/httpapi_client"
	"nexus-orchestrator/internal/core/domain"
)

type brainCase struct {
	name    string
	run     func(c *httpapi_client.BrainClient) error
	success int
	body    string // healthy response body
	decodes bool
}

func brainCases() []brainCase {
	ctx := context.Background()
	return []brainCase{
		{"GetContext", func(c *httpapi_client.BrainClient) error {
			_, err := c.GetContext(ctx, domain.ContextQuery{ProjectPath: "/p"})
			return err
		}, 200, `{}`, true},
		{"GetFocusedContext", func(c *httpapi_client.BrainClient) error {
			_, err := c.GetFocusedContext(ctx, domain.ContextQuery{ProjectPath: "/p", Question: "q"})
			return err
		}, 200, `{}`, true},
		{"IngestFromFile", func(c *httpapi_client.BrainClient) error {
			_, err := c.IngestFromFile(ctx, "/p", "/f.md")
			return err
		}, 200, `{"ingestedSections":3}`, true},
		{"SearchKnowledge", func(c *httpapi_client.BrainClient) error {
			_, err := c.SearchKnowledge(ctx, "/p", "q", 5)
			return err
		}, 200, `{"results":[]}`, true},
		{"GetFileMap", func(c *httpapi_client.BrainClient) error { _, err := c.GetFileMap(ctx, "/p", "auth"); return err }, 200, `{"filePaths":[]}`, true},
		{"InitProject", func(c *httpapi_client.BrainClient) error { _, err := c.InitProject(ctx, "/p", ""); return err }, 200, `{}`, true},
		{"GetStatus", func(c *httpapi_client.BrainClient) error { _, err := c.GetStatus(ctx, "/p"); return err }, 200, `{}`, true},
		{"ListKnowledge", func(c *httpapi_client.BrainClient) error {
			_, err := c.ListKnowledge(ctx, "/p", "learning")
			return err
		}, 200, `[]`, true},
		{"DeleteKnowledge", func(c *httpapi_client.BrainClient) error { return c.DeleteKnowledge(ctx, "k") }, 204, ``, false},
		{"GetOnboardingContext", func(c *httpapi_client.BrainClient) error {
			_, err := c.GetOnboardingContext(ctx, "/p", 100)
			return err
		}, 200, `{"content":"hi"}`, true},
	}
}

func TestEveryBrainClientMethod_FailureModes(t *testing.T) {
	for _, tc := range brainCases() {
		t.Run(tc.name, func(t *testing.T) {
			dead := fixedServer(200, "{}")
			url := dead.URL
			dead.Close()
			if err := tc.run(httpapi_client.NewBrainClient(url)); err == nil {
				t.Error("unreachable daemon must be an error")
			}
			if err := tc.run(httpapi_client.NewBrainClient("http://bad host")); err == nil || !strings.Contains(err.Error(), "build request") {
				t.Errorf("bad URL: %v", err)
			}
			srv := fixedServer(500, `{"error":"boom"}`)
			err := tc.run(httpapi_client.NewBrainClient(srv.URL))
			srv.Close()
			if err == nil || !strings.Contains(err.Error(), "unexpected status 500") || !strings.Contains(err.Error(), "boom") {
				t.Errorf("500: %v", err)
			}
			if tc.decodes {
				srv = fixedServer(tc.success, "not json")
				err = tc.run(httpapi_client.NewBrainClient(srv.URL))
				srv.Close()
				if err == nil || !strings.Contains(err.Error(), "decode") {
					t.Errorf("malformed body: %v", err)
				}
			}
			srv = fixedServer(tc.success, tc.body)
			err = tc.run(httpapi_client.NewBrainClient(srv.URL))
			srv.Close()
			if err != nil {
				t.Errorf("healthy daemon: %v", err)
			}
		})
	}
}

func TestBrainClient_ResultsAndQueryEncoding(t *testing.T) {
	var gotQuery map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		switch r.URL.Path {
		case "/api/brain/ingest":
			_, _ = w.Write([]byte(`{"ingestedSections":4}`))
		case "/api/brain/knowledge":
			_, _ = w.Write([]byte(`null`))
		case "/api/brain/onboarding":
			_, _ = w.Write([]byte(`{"content":"welcome"}`))
		case "/api/brain/file-map":
			_, _ = w.Write([]byte(`{"filePaths":["a.go","b.go"]}`))
		case "/api/brain/search":
			_, _ = w.Write([]byte(`{"results":[{"topic":"T"}]}`))
		}
	}))
	defer srv.Close()
	c := httpapi_client.NewBrainClient(srv.URL)
	ctx := context.Background()

	if n, err := c.IngestFromFile(ctx, "/p", "/f.md"); err != nil || n != 4 {
		t.Errorf("ingest: %d %v", n, err)
	}
	got, err := c.ListKnowledge(ctx, "/My Project&Co", "")
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("a JSON null list must become an empty non-nil slice: %#v %v", got, err)
	}
	if gotQuery["projectPath"][0] != "/My Project&Co" {
		t.Errorf("project path must survive the query string, got %q", gotQuery["projectPath"])
	}
	if text, err := c.GetOnboardingContext(ctx, "/p", 0); err != nil || text != "welcome" {
		t.Errorf("onboarding: %q %v", text, err)
	}
	if _, ok := gotQuery["maxTokens"]; ok {
		t.Error("maxTokens must be omitted when not positive")
	}
	if _, err := c.GetOnboardingContext(ctx, "/p", 250); err != nil || gotQuery["maxTokens"][0] != "250" {
		t.Errorf("maxTokens not sent: %v %v", gotQuery, err)
	}
	if files, err := c.GetFileMap(ctx, "/p", "x"); err != nil || len(files) != 2 {
		t.Errorf("file map: %v %v", files, err)
	}
	if secs, err := c.SearchKnowledge(ctx, "/p", "a&b", 7); err != nil || len(secs) != 1 || gotQuery["q"][0] != "a&b" || gotQuery["limit"][0] != "7" {
		t.Errorf("search: %v %v %v", secs, gotQuery, err)
	}
	if _, err := c.IngestKnowledge(ctx, domain.ProjectKnowledge{}); err == nil {
		t.Error("IngestKnowledge is intentionally unsupported over HTTP")
	}
}
