package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"nexus-orchestrator/internal/adapters/inbound/mcp"
	"nexus-orchestrator/internal/core/domain"
)

type routingOrch struct {
	mockOrch
	preview       domain.RoutingPreview
	err           error
	project, role string
}

func (r *routingOrch) PreviewRouting(_ context.Context, project, role string) (domain.RoutingPreview, error) {
	r.project, r.role = project, role
	return r.preview, r.err
}

func TestGetRouting_ReturnsTheOrder(t *testing.T) {
	o := &routingOrch{preview: domain.RoutingPreview{Role: "coder", Ladder: []string{"Ollama/*"}, QuotaThreshold: 0.9,
		Order: []domain.RoutingEntry{{Position: 1, Provider: "Ollama", Model: "qwen", Rung: 0}}}}
	srv := httptest.NewServer(mcp.NewMcpServer(o, nil))
	defer srv.Close()
	r := callTool(t, srv, 1, "get_routing", map[string]any{"projectPath": "/p", "role": " coder "})
	if r.Error != nil {
		t.Fatalf("error: %+v", r.Error)
	}
	var res struct {
		Content []struct{ Text string } `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &res); err != nil || len(res.Content) != 1 {
		t.Fatalf("%s %v", r.Result, err)
	}
	var got domain.RoutingPreview
	if err := json.Unmarshal([]byte(res.Content[0].Text), &got); err != nil || len(got.Order) != 1 || got.Order[0].Provider != "Ollama" {
		t.Errorf("%+v %v", got, err)
	}
	if o.project != "/p" || o.role != "coder" {
		t.Errorf("arguments: %q %q", o.project, o.role)
	}
}

func TestGetRouting_Errors(t *testing.T) {
	o := &routingOrch{err: errors.New("no provider")}
	srv := httptest.NewServer(mcp.NewMcpServer(o, nil))
	defer srv.Close()
	if r := callTool(t, srv, 1, "get_routing", map[string]any{}); r.Error == nil || !strings.Contains(r.Error.Message, "projectPath is required") {
		t.Errorf("missing projectPath: %+v", r.Error)
	}
	if r := callTool(t, srv, 2, "get_routing", map[string]any{"projectPath": "/p"}); r.Error == nil || !strings.Contains(r.Error.Message, "no provider") {
		t.Errorf("backend error: %+v", r.Error)
	}
	plain := httptest.NewServer(mcp.NewMcpServer(&mockOrch{}, nil))
	defer plain.Close()
	if r := callTool(t, plain, 3, "get_routing", map[string]any{"projectPath": "/p"}); r.Error == nil || !strings.Contains(r.Error.Message, "not available") {
		t.Errorf("no previewer: %+v", r.Error)
	}
}
