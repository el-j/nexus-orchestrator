// Package mcp — tool handlers and schema definitions.
// This file contains the handleToolCall dispatcher, the textResult helper,
// and the composite toolList() function. Domain-specific tool handlers and
// schemas are organized into tools_tasks.go, tools_providers.go,
// tools_sessions.go, tools_howto.go, and brain_tools.go.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ----- Tool dispatch -----

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	Meta      progressMeta    `json:"_meta"`
}

// handleToolCall runs one tools/call: it decodes the call, dispatches on the tool name and turns the
// result or error into the JSON-RPC response.
func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	var p callToolParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		writeError(w, req.ID, codeInvalidParams, "invalid params")
		return
	}

	handler, ok := toolHandlers[p.Name]
	if !ok {
		writeError(w, req.ID, codeMethodNotFound, fmt.Sprintf("unknown tool: %s", p.Name))
		return
	}
	call := toolCall{ctx: r.Context(), args: p.Arguments}
	if p.Name == "wait_for_task" {
		call.progress, w = startProgress(w, r, p.Meta)
	}
	result, err := handler(s, call)

	if err != nil {
		var me *mcpError
		if errors.As(err, &me) {
			writeError(w, req.ID, me.code, me.msg)
			return
		}
		writeError(w, req.ID, codeInternalError, err.Error())
		return
	}

	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
	_ = json.NewEncoder(w).Encode(resp)
}

// ----- Helpers -----

// textResult wraps a string as the single text item of a tool result.
func textResult(text string) callToolResult {
	return callToolResult{Content: []contentItem{{Type: "text", Text: text}}}
}

// ----- Composite Tool Catalogue -----

// toolList is the catalogue of all tools the server offers, in the order of the groups below.
func toolList() []toolDef {
	var list []toolDef
	list = append(list, taskToolDefs()...)
	list = append(list, waitToolDefs()...)
	list = append(list, providerToolDefs()...)
	list = append(list, sessionToolDefs()...)
	list = append(list, usageToolDefs()...)
	list = append(list, routingToolDefs()...)
	list = append(list, howtoToolDefs()...)
	list = append(list, brainToolDefs()...)
	return list
}
