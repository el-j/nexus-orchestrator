package mcp

import (
	"reflect"
	"testing"

	"nexus-orchestrator/internal/core/domain"
)

// Every routing field a task can carry over HTTP must also be settable through
// MCP, otherwise agents cannot pick a provider, model or tier (issue #53).
var routingFields = []string{"providerName", "providerHint", "modelId", "role", "priority", "tags"}

func TestSubmitAndDraftToolsExposeEveryRoutingField(t *testing.T) {
	for _, tool := range []string{"submit_task", "create_draft"} {
		var schema map[string]property
		for _, def := range toolList() {
			if def.Name == tool {
				schema = def.InputSchema.Properties
			}
		}
		if schema == nil {
			t.Fatalf("tool %s not found", tool)
		}
		for _, f := range routingFields {
			if _, ok := schema[f]; !ok {
				t.Errorf("%s does not advertise %q", tool, f)
			}
		}
	}
}

func TestRoutingFieldsAreRealTaskFields(t *testing.T) {
	known := map[string]bool{}
	typ := reflect.TypeOf(domain.Task{})
	for i := 0; i < typ.NumField(); i++ {
		known[typ.Field(i).Tag.Get("json")] = true
	}
	for _, f := range routingFields {
		if !known[f+",omitempty"] && !known[f] {
			t.Errorf("%q is not a domain.Task JSON field", f)
		}
	}
}
