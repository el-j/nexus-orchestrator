package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// plannedStep is one task returned by the goal-decomposition prompt.
type plannedStep struct {
	Instruction   string `json:"instruction"`
	TargetFile    string `json:"targetFile"`
	Role          string `json:"role"`
	VerifyCommand string `json:"verifyCommand"`
	DependsOn     []int  `json:"dependsOn"`
}

// autoPlanTasks either decomposes req.Goal into dependency-ordered agentic tasks or promotes backlog items.
func (o *OrchestratorService) autoPlanTasks(ctx context.Context, req domain.AutoModeRequest, ragContext, verifyCmd string) ([]string, error) {
	if strings.TrimSpace(req.Goal) == "" {
		backlog, err := o.GetBacklog(req.ProjectPath)
		if err != nil {
			return nil, fmt.Errorf("automode: load backlog: %w", err)
		}
		var ids []string
		for _, t := range backlog {
			if len(ids) >= req.MaxTasks {
				break
			}
			if _, err := o.PromoteTask(t.ID); err != nil {
				return nil, fmt.Errorf("automode: promote %s: %w", t.ID, err)
			}
			ids = append(ids, t.ID)
		}
		return ids, nil
	}

	steps := o.decomposeGoal(ctx, req, ragContext, verifyCmd)
	ids := make([]string, 0, len(steps))
	for _, st := range steps {
		var deps []string
		for _, depIdx := range st.DependsOn {
			if depIdx >= 0 && depIdx < len(ids) {
				deps = append(deps, ids[depIdx])
			}
		}
		stepVerify := strings.TrimSpace(st.VerifyCommand)
		if stepVerify == "" {
			stepVerify = verifyCmd
		}
		role := strings.TrimSpace(st.Role)
		if role == "" {
			role = "coder"
		}
		id, err := o.SubmitTask(domain.Task{
			ProjectPath:         req.ProjectPath,
			TargetFile:          strings.TrimSpace(st.TargetFile),
			Instruction:         strings.TrimSpace(st.Instruction),
			Role:                role,
			Agentic:             true,
			MaxAgentTurns:       req.MaxAgentTurns,
			VerificationCommand: stepVerify,
			DependsOn:           deps,
			Tags:                []string{"auto-mode"},
		})
		if err != nil {
			return nil, fmt.Errorf("automode: submit planned task: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// decomposeGoal asks the best available provider to break req.Goal into ordered steps,
// falling back to a single agentic task if decomposition fails or returns non-JSON.
func (o *OrchestratorService) decomposeGoal(ctx context.Context, req domain.AutoModeRequest, ragContext, verifyCmd string) []plannedStep {
	fallback := []plannedStep{{Instruction: req.Goal, Role: "coder", VerifyCommand: verifyCmd}}
	planTask := domain.Task{ProjectPath: req.ProjectPath, Instruction: req.Goal, Role: "architect", Command: domain.CommandPlan}
	chain, err := o.resolveProviderWithFallback(ctx, planTask)
	if err != nil || len(chain) == 0 {
		return fallback
	}
	chain, _ = o.routeChain(chain, planTask)
	var prompt strings.Builder
	prompt.WriteString("Decompose the following project goal into at most ")
	fmt.Fprintf(&prompt, "%d small, dependency-ordered engineering tasks.\n", req.MaxTasks)
	prompt.WriteString("Reply ONLY with a JSON array of objects: [{\"instruction\":\"...\",\"targetFile\":\"...\",\"role\":\"coder|tester|architect\",\"verifyCommand\":\"...\",\"dependsOn\":[0]}].\n")
	if ragContext != "" {
		prompt.WriteString("\nProject Context:\n")
		prompt.WriteString(ragContext)
	}
	prompt.WriteString("\nGoal:\n")
	prompt.WriteString(req.Goal)

	reply, err := chain[0].Chat(ctx, []domain.Message{
		{Role: domain.RoleSystem, Content: "You are a principal software architect. Output valid JSON only."},
		{Role: domain.RoleUser, Content: prompt.String()},
	})
	if err != nil {
		return fallback
	}
	steps := parsePlannedSteps(reply, req.MaxTasks)
	if len(steps) == 0 {
		return fallback
	}
	return steps
}

// parsePlannedSteps extracts a JSON array of plannedStep from reply (stripping optional markdown fences).
func parsePlannedSteps(reply string, maxTasks int) []plannedStep {
	raw := strings.TrimSpace(reply)
	if idx := strings.Index(raw, "["); idx >= 0 {
		if end := strings.LastIndex(raw, "]"); end > idx {
			raw = raw[idx : end+1]
		}
	}
	var parsed []plannedStep
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil
	}
	out := make([]plannedStep, 0, len(parsed))
	for _, s := range parsed {
		if strings.TrimSpace(s.Instruction) == "" {
			continue
		}
		out = append(out, s)
		if len(out) >= maxTasks {
			break
		}
	}
	return out
}
