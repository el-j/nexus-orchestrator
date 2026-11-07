package mcp

import (
	"encoding/json"
	"fmt"
)

// jsonResult answers a tool call with v encoded as JSON text. A value JSON cannot write (a time outside
// the years it supports) is reported as an error naming the tool.
func jsonResult(tool string, v any) (callToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: %s: marshal: %w", tool, err)
	}
	return textResult(string(b)), nil
}
