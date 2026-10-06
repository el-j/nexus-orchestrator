package mcp

import (
	"encoding/json"
	"fmt"

	"nexus-orchestrator/internal/core/domain"
)

// ----- Task Handlers -----

func (s *Server) toolSubmitTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath         string   `json:"projectPath"`
		TargetFile          string   `json:"targetFile"`
		Instruction         string   `json:"instruction"`
		ContextFiles        []string `json:"contextFiles"`
		Command             string   `json:"command"`
		VerificationCommand string   `json:"verificationCommand"`
		MaxCorrectionTurns  int      `json:"maxCorrectionTurns"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: submit_task: invalid arguments: %w", err)
	}
	t := domain.Task{
		ProjectPath:         p.ProjectPath,
		TargetFile:          p.TargetFile,
		Instruction:         p.Instruction,
		ContextFiles:        p.ContextFiles,
		Command:             domain.CommandType(p.Command),
		VerificationCommand: p.VerificationCommand,
		MaxCorrectionTurns:  p.MaxCorrectionTurns,
	}
	id, err := s.orch.SubmitTask(t)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: submit_task: %w", err)
	}
	b, _ := json.Marshal(map[string]string{"id": id})
	return textResult(string(b)), nil
}

func (s *Server) toolGetTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: invalid arguments: %w", err)
	}
	task, err := s.orch.GetTask(p.ID)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: %w", err)
	}
	b, err := json.Marshal(task)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_task: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

// decodeOptionalArgs decodes tool arguments where every field is optional.
// Absent, empty and JSON null arguments leave v untouched; anything else that
// does not decode is an invalid-params error. Silently ignoring a malformed
// projectPath would make list tools fall back to the unfiltered, cross-project
// result.
func decodeOptionalArgs(args json.RawMessage, v any) error {
	if len(args) == 0 || string(args) == "null" {
		return nil
	}
	if err := json.Unmarshal(args, v); err != nil {
		return &mcpError{code: codeInvalidParams, msg: "invalid arguments: " + err.Error()}
	}
	return nil
}

func (s *Server) toolGetQueue(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
	}
	if err := decodeOptionalArgs(args, &p); err != nil {
		return callToolResult{}, err
	}
	var (
		tasks []domain.Task
		err   error
	)
	if p.ProjectPath != "" {
		tasks, err = s.orch.GetQueueForProject(p.ProjectPath)
	} else {
		tasks, err = s.orch.GetQueue()
	}
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_queue: %w", err)
	}
	b, err := json.Marshal(tasks)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_queue: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolGetAllTasks(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
	}
	if err := decodeOptionalArgs(args, &p); err != nil {
		return callToolResult{}, err
	}
	var (
		tasks []domain.Task
		err   error
	)
	if p.ProjectPath != "" {
		tasks, err = s.orch.GetTasksForProject(p.ProjectPath)
	} else {
		tasks, err = s.orch.GetAllTasks()
	}
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_all_tasks: %w", err)
	}
	b, err := json.Marshal(tasks)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_all_tasks: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolCancelTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: cancel_task: invalid arguments: %w", err)
	}
	if err := s.orch.CancelTask(p.ID); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: cancel_task: %w", err)
	}
	b, _ := json.Marshal(map[string]bool{"cancelled": true})
	return textResult(string(b)), nil
}

func (s *Server) toolCreateDraft(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath  string   `json:"projectPath"`
		Instruction  string   `json:"instruction"`
		TargetFile   string   `json:"targetFile"`
		ProviderName string   `json:"providerName"`
		ModelID      string   `json:"modelId"`
		Priority     int      `json:"priority"`
		Tags         []string `json:"tags"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: create_draft: invalid arguments: %w", err)
	}
	t := domain.Task{
		ProjectPath:  p.ProjectPath,
		Instruction:  p.Instruction,
		TargetFile:   p.TargetFile,
		ProviderName: p.ProviderName,
		ModelID:      p.ModelID,
		Priority:     p.Priority,
		Tags:         p.Tags,
	}
	id, err := s.orch.CreateDraft(t)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: create_draft: %w", err)
	}
	b, _ := json.Marshal(map[string]string{"id": id, "status": string(domain.StatusDraft)})
	return textResult(string(b)), nil
}

func (s *Server) toolGetBacklog(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ProjectPath string `json:"projectPath"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_backlog: invalid arguments: %w", err)
	}
	tasks, err := s.orch.GetBacklog(p.ProjectPath)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_backlog: %w", err)
	}
	b, err := json.Marshal(tasks)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: get_backlog: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

func (s *Server) toolPromoteTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: promote_task: invalid arguments: %w", err)
	}
	result, err := s.orch.PromoteTask(p.ID)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: promote_task: %w", err)
	}
	resp := map[string]any{"promoted": result.Promoted}
	if result.Warning != "" {
		resp["warning"] = result.Warning
	}
	b, _ := json.Marshal(resp)
	return textResult(string(b)), nil
}

func (s *Server) toolUpdateTask(args json.RawMessage) (callToolResult, error) {
	var p struct {
		ID           string   `json:"id"`
		Instruction  string   `json:"instruction"`
		Priority     int      `json:"priority"`
		ProviderName string   `json:"providerName"`
		ModelID      string   `json:"modelId"`
		Tags         []string `json:"tags"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_task: invalid arguments: %w", err)
	}
	updates := domain.Task{
		Instruction:  p.Instruction,
		Priority:     p.Priority,
		ProviderName: p.ProviderName,
		ModelID:      p.ModelID,
		Tags:         p.Tags,
	}
	updated, err := s.orch.UpdateTask(p.ID, updates)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_task: %w", err)
	}
	b, err := json.Marshal(updated)
	if err != nil {
		return callToolResult{}, fmt.Errorf("mcp: update_task: marshal: %w", err)
	}
	return textResult(string(b)), nil
}

// ----- Task Schema Definitions -----

func taskToolDefs() []toolDef {
	return []toolDef{
		{
			Name:        "submit_task",
			Description: "Submit a new code-generation task to the orchestrator.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath":         {Type: "string", Description: "Absolute path to the project root."},
					"targetFile":          {Type: "string", Description: "Relative path of the file to generate or modify."},
					"instruction":         {Type: "string", Description: "Natural-language instruction for the LLM."},
					"contextFiles":        {Type: "array", Description: "Optional list of relative file paths to include as context.", Items: &propertyItems{Type: "string"}},
					"command":             {Type: "string", Description: "Task type: plan, execute, or auto (default: auto)."},
					"verificationCommand": {Type: "string", Description: "Optional command (e.g. 'go test ./...', 'npm test') to verify output and trigger self-healing if it fails."},
					"maxCorrectionTurns":  {Type: "integer", Description: "Max self-healing attempts if verification fails (default: 2)."},
				},
				Required: []string{"projectPath", "targetFile", "instruction"},
			},
		},
		{
			Name:        "get_task",
			Description: "Get the current status and output of a task by ID.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{"id": {Type: "string", Description: "Task ID returned by submit_task."}},
				Required:   []string{"id"},
			},
		},
		{
			Name:        "get_queue",
			Description: "List all tasks currently in the queue, optionally filtered by project.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Optional absolute path to filter tasks by project."},
				},
			},
		},
		{
			Name:        "get_all_tasks",
			Description: "Return every task regardless of status, optionally filtered by project.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Optional absolute path to filter tasks by project."},
				},
			},
		},
		{
			Name:        "cancel_task",
			Description: "Cancel a pending task by ID.",
			InputSchema: inputSchema{
				Type:       "object",
				Properties: map[string]property{"id": {Type: "string", Description: "Task ID to cancel."}},
				Required:   []string{"id"},
			},
		},
		{
			Name:        "create_draft",
			Description: "Create a draft idea for a project without entering the execution queue.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath":  {Type: "string", Description: "Absolute path to the project."},
					"instruction":  {Type: "string", Description: "What the task should do."},
					"targetFile":   {Type: "string", Description: "File to write output to (optional)."},
					"providerName": {Type: "string", Description: "Exact provider name for routing (optional)."},
					"modelId":      {Type: "string", Description: "Model to use (optional)."},
					"priority":     {Type: "integer", Description: "Priority 1=high, 2=medium, 3=low (default 2)."},
					"tags":         {Type: "array", Description: "Labels (optional).", Items: &propertyItems{Type: "string"}},
				},
				Required: []string{"projectPath", "instruction"},
			},
		},
		{
			Name:        "get_backlog",
			Description: "List draft and backlog items for a project, ordered by priority.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"projectPath": {Type: "string", Description: "Absolute path to the project."},
				},
				Required: []string{"projectPath"},
			},
		},
		{
			Name:        "promote_task",
			Description: "Promote a draft or backlog task to the execution queue.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id": {Type: "string", Description: "Task ID to promote."},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "update_task",
			Description: "Update mutable fields on an existing task (instruction, priority, provider, tags, status).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]property{
					"id":           {Type: "string", Description: "Task ID to update."},
					"instruction":  {Type: "string", Description: "Updated instruction."},
					"priority":     {Type: "integer", Description: "Updated priority."},
					"providerName": {Type: "string", Description: "Updated provider name."},
					"modelId":      {Type: "string", Description: "Updated model ID."},
					"tags":         {Type: "array", Description: "Updated tags.", Items: &propertyItems{Type: "string"}},
				},
				Required: []string{"id"},
			},
		},
	}
}
