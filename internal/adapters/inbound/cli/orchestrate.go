package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
	"nexus-orchestrator/internal/core/services"

	"github.com/spf13/cobra"
)

// OrchestrateOptions contains the command-line options for the orchestrate command.
type OrchestrateOptions struct {
	ProjectPath string
	ModelID     string
	Role        string
	TargetFile  string
	VerifyCmd   string
	MaxTurns    int
	JSON        bool
}

// OrchestratePipelineResult contains the telemetry and results of a completed pipeline run.
type OrchestratePipelineResult struct {
	TaskID         string   `json:"taskId"`
	Status         string   `json:"status"`
	PromptTokens   int      `json:"promptTokens"`
	RAGTokensSaved int      `json:"ragTokensSaved"`
	ModelTier      string   `json:"modelTier"`
	AssignedModel  string   `json:"assignedModel"`
	TurnsExecuted  int      `json:"turnsExecuted"`
	Duration       string   `json:"duration"`
	LearningsSaved int      `json:"learningsSaved"`
	SuggestedFiles []string `json:"suggestedFiles,omitempty"`
}

// newOrchestrateCmd creates the `nexus orchestrate` pipeline command.
func newOrchestrateCmd(orch ports.Orchestrator, brain ports.BrainService) *cobra.Command {
	opts := OrchestrateOptions{}

	cmd := &cobra.Command{
		Use:   "orchestrate [spec-file | instruction]",
		Short: "Execute an end-to-end unified orchestration pipeline",
		Long: `The orchestrate command drives the complete unified development cycle:
1. Dynamic token-budgeted RAG retrieval tailored to target model tier.
2. Capability and cost affinity matching for model assignment.
3. Test-driven generation and self-healing AST verification.
4. Continuous knowledge digestion updating project memory and file maps.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrchestratePipeline(cmd.Context(), cmd.OutOrStdout(), orch, brain, args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ProjectPath, "project", "p", ".", "Path to the target project directory")
	cmd.Flags().StringVarP(&opts.ModelID, "model", "m", "", "Explicit model ID constraint (e.g. claude-3-5-sonnet, qwen3-coder-next)")
	cmd.Flags().StringVarP(&opts.Role, "role", "r", "coder", "Task role (architect, techlead, coder, linter, tester)")
	cmd.Flags().StringVarP(&opts.TargetFile, "target", "t", "", "Target file to create or modify")
	cmd.Flags().StringVarP(&opts.VerifyCmd, "verify", "v", "", "Verification command to validate code (e.g. go test ./...)")
	cmd.Flags().IntVar(&opts.MaxTurns, "max-turns", 5, "Maximum self-healing turns")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Output structured telemetry in JSON format")

	return cmd
}

// resolveInstruction loads the instruction from a file path if it exists, or returns the raw string.
func resolveInstruction(specOrInstruction string) (string, error) {
	instruction := strings.TrimSpace(specOrInstruction)
	if fi, err := os.Stat(instruction); err == nil && !fi.IsDir() {
		data, err := os.ReadFile(instruction)
		if err != nil {
			return "", fmt.Errorf("read spec file: %w", err)
		}
		instruction = string(data)
	}
	if instruction == "" {
		return "", errors.New("orchestrate: instruction or spec file content cannot be empty")
	}
	return instruction, nil
}

// retrieveRAGContext fetches tailored context and token savings from the brain.
func retrieveRAGContext(ctx context.Context, brain ports.BrainService, cleanPath, modelID string) (int, domain.ModelTier, []string) {
	ragTokensSaved := 0
	tier := domain.ModelTierStandard
	var suggestedFiles []string

	if brain != nil {
		resp, err := brain.GetContext(ctx, domain.ContextQuery{
			ProjectPath: cleanPath,
			ModelID:     modelID,
			FocusArea:   "all",
		})
		if err == nil {
			ragTokensSaved = resp.TokensSaved
			if resp.ModelTier.IsValid() {
				tier = resp.ModelTier
			}
			suggestedFiles = resp.SuggestedFiles
		}
	}
	return ragTokensSaved, tier, suggestedFiles
}

// waitForTaskSettlement polls the orchestrator until the task reaches a terminal state.
func waitForTaskSettlement(orch ports.Orchestrator, taskID string) domain.Task {
	finalTask := domain.Task{ID: taskID, Status: domain.StatusQueued}
	for i := 0; i < 60; i++ {
		t, err := orch.GetTask(taskID)
		if err == nil {
			finalTask = t
			if t.Status == domain.StatusCompleted || t.Status == domain.StatusFailed ||
				t.Status == domain.StatusCancelled || t.Status == domain.StatusNoProvider ||
				t.Status == domain.StatusTooLarge {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return finalTask
}

// runOrchestratePipeline executes the cohesive pipeline from RAG to task execution and knowledge ingestion.
func runOrchestratePipeline(
	ctx context.Context,
	w io.Writer,
	orch ports.Orchestrator,
	brain ports.BrainService,
	specOrInstruction string,
	opts OrchestrateOptions,
) error {
	start := time.Now()
	cleanPath, _ := filepath.Abs(opts.ProjectPath)

	instruction, err := resolveInstruction(specOrInstruction)
	if err != nil {
		return err
	}

	ragTokensSaved, tier, suggestedFiles := retrieveRAGContext(ctx, brain, cleanPath, opts.ModelID)

	task := domain.Task{
		ProjectPath:         cleanPath,
		Instruction:         instruction,
		TargetFile:          opts.TargetFile,
		VerificationCommand: opts.VerifyCmd,
		ModelID:             opts.ModelID,
		Role:                opts.Role,
		Agentic:             opts.VerifyCmd != "",
		MaxAgentTurns:       opts.MaxTurns,
	}

	profile := services.EstimateTaskAffinity(task, 0)
	if !opts.JSON {
		fmt.Fprintf(w, "[1/4] RAG Retrieval: tailored for %s tier (saved %d tokens)\n", tier, ragTokensSaved)
		fmt.Fprintf(w, "[2/4] Affinity Routing: role=%s, complexity=%s, target=%s\n", profile.RequiredRole, profile.Complexity, opts.TargetFile)
	}

	taskID, err := orch.SubmitTask(task)
	if err != nil {
		return fmt.Errorf("submit task: %w", err)
	}

	if !opts.JSON {
		fmt.Fprintf(w, "[3/4] Task Dispatched: ID=%s, status=%s\n", taskID, domain.StatusQueued)
	}

	finalTask := waitForTaskSettlement(orch, taskID)

	learningsCount := 0
	if brain != nil && (finalTask.Status == domain.StatusCompleted || finalTask.Status == domain.StatusFailed) {
		ingested := services.DigestTaskOutcome(ctx, brain, finalTask, finalTask.Status)
		learningsCount = len(ingested)
	}

	durStr := time.Since(start).Round(time.Millisecond).String()

	if !opts.JSON {
		fmt.Fprintf(w, "[4/4] Knowledge Digested: %d entries ingested into project memory\n", learningsCount)
		fmt.Fprintf(w, "Pipeline Finished: status=%s, elapsed=%s\n", finalTask.Status, durStr)
		return nil
	}

	result := OrchestratePipelineResult{
		TaskID:         finalTask.ID,
		Status:         finalTask.Status.String(),
		PromptTokens:   profile.EstimatedPromptTokens,
		RAGTokensSaved: ragTokensSaved,
		ModelTier:      string(tier),
		AssignedModel:  finalTask.ModelID,
		TurnsExecuted:  1,
		Duration:       durStr,
		LearningsSaved: learningsCount,
		SuggestedFiles: suggestedFiles,
	}
	return json.NewEncoder(w).Encode(result)
}
