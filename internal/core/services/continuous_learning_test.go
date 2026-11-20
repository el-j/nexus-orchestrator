package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/services"
)

type mockBrainForLearning struct {
	ingested []domain.ProjectKnowledge
	err      error
}

func (m *mockBrainForLearning) IngestKnowledge(_ context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	if m.err != nil {
		return domain.ProjectKnowledge{}, m.err
	}
	m.ingested = append(m.ingested, k)
	return k, nil
}
func (m *mockBrainForLearning) GetContext(_ context.Context, _ domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{}, nil
}
func (m *mockBrainForLearning) GetFocusedContext(_ context.Context, _ domain.ContextQuery) (domain.ContextResponse, error) {
	return domain.ContextResponse{}, nil
}
func (m *mockBrainForLearning) IngestFromFile(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (m *mockBrainForLearning) SearchKnowledge(_ context.Context, _, _ string, _ int) ([]domain.ContextSection, error) {
	return nil, nil
}
func (m *mockBrainForLearning) GetFileMap(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockBrainForLearning) InitProject(_ context.Context, _, _ string) (domain.BrainStatus, error) {
	return domain.BrainStatus{}, nil
}
func (m *mockBrainForLearning) GetStatus(_ context.Context, _ string) (domain.BrainStatus, error) {
	return domain.BrainStatus{}, nil
}
func (m *mockBrainForLearning) ListKnowledge(_ context.Context, _, _ string) ([]domain.ProjectKnowledge, error) {
	return nil, nil
}
func (m *mockBrainForLearning) DeleteKnowledge(_ context.Context, _ string) error {
	return nil
}
func (m *mockBrainForLearning) GetOnboardingContext(_ context.Context, _ string, _ int) (string, error) {
	return "", nil
}

func TestExtractSummaryLearning(t *testing.T) {
	// Completed task
	taskComp := domain.Task{
		ID:                  "task-12345678",
		ProjectPath:         "/repo",
		Instruction:         "add authentication middleware",
		TargetFile:          "internal/auth/mw.go",
		VerificationCommand: "go test ./...",
	}
	sComp := services.ExtractSummaryLearning(taskComp, domain.StatusCompleted)
	if !strings.HasPrefix(sComp.Content, "Completed:") {
		t.Errorf("expected Completed prefix, got: %s", sComp.Content)
	}
	if !strings.Contains(sComp.Content, "target internal/auth/mw.go") {
		t.Errorf("expected target file in summary, got: %s", sComp.Content)
	}
	if !strings.Contains(sComp.Content, "verified with `go test ./...`") {
		t.Errorf("expected verification in summary, got: %s", sComp.Content)
	}

	// Failed task
	taskFail := domain.Task{
		ID:          "task-fail-1234",
		ProjectPath: "/repo",
		Instruction: "compile project",
		Logs:        "line 1\nsyntax error on line 42",
	}
	sFail := services.ExtractSummaryLearning(taskFail, domain.StatusFailed)
	if !strings.HasPrefix(sFail.Content, "Failed:") {
		t.Errorf("expected Failed prefix, got: %s", sFail.Content)
	}
	if !strings.Contains(sFail.Content, "syntax error on line 42") {
		t.Errorf("expected error log in summary, got: %s", sFail.Content)
	}
}

func TestExtractConventionEntries(t *testing.T) {
	// 1. Matched keyword with target file
	taskWithFile := domain.Task{
		ProjectPath: "/repo",
		Instruction: "enforce invariant: all HTTP handlers return JSON",
		TargetFile:  "internal/http/handler.go",
	}
	convs := services.ExtractConventionEntries(taskWithFile)
	if len(convs) != 1 {
		t.Fatalf("expected 1 convention, got %d", len(convs))
	}
	if convs[0].Topic != "handler.go" {
		t.Errorf("expected topic handler.go, got: %s", convs[0].Topic)
	}
	if !strings.HasPrefix(convs[0].Content, "* INVARIANT:") {
		t.Errorf("expected invariant prefix, got: %s", convs[0].Content)
	}

	// 2. Matched keyword with role but no target file
	taskWithRole := domain.Task{
		ProjectPath: "/repo",
		Role:        "architect",
		Instruction: "follow repository pattern for persistence",
	}
	convsRole := services.ExtractConventionEntries(taskWithRole)
	if len(convsRole) != 1 || convsRole[0].Topic != "architect" {
		t.Errorf("expected topic architect, got: %+v", convsRole)
	}

	// 3. Matched keyword without role or target file -> "General"
	taskGen := domain.Task{
		ProjectPath: "/repo",
		Instruction: "follow architectural decision AD-01",
	}
	convsGen := services.ExtractConventionEntries(taskGen)
	if len(convsGen) != 1 || convsGen[0].Topic != "General" {
		t.Errorf("expected topic General, got: %+v", convsGen)
	}

	// 4. No convention keywords -> nil
	taskPlain := domain.Task{
		Instruction: "fix small typo",
	}
	if got := services.ExtractConventionEntries(taskPlain); got != nil {
		t.Errorf("expected nil for non-convention task, got: %v", got)
	}
}

func TestExtractFileMapEntries(t *testing.T) {
	// Task with target file
	task := domain.Task{
		ProjectPath: "/repo",
		TargetFile:  "pkg/utils/strings.go",
		Instruction: "string manipulation helpers",
	}
	fmaps := services.ExtractFileMapEntries(task)
	if len(fmaps) != 1 {
		t.Fatalf("expected 1 file map entry, got %d", len(fmaps))
	}
	if fmaps[0].Topic != "strings.go" {
		t.Errorf("expected topic strings.go, got %s", fmaps[0].Topic)
	}
	if !strings.Contains(fmaps[0].Content, "pkg/utils/strings.go - string manipulation helpers") {
		t.Errorf("unexpected content: %s", fmaps[0].Content)
	}

	// Empty instruction uses fallback
	taskNoInst := domain.Task{
		TargetFile: "pkg/utils/empty.go",
	}
	fmapsNoInst := services.ExtractFileMapEntries(taskNoInst)
	if len(fmapsNoInst) != 1 || !strings.Contains(fmapsNoInst[0].Content, "Project source file") {
		t.Errorf("expected fallback purpose, got: %+v", fmapsNoInst)
	}

	// Empty target file -> nil
	if got := services.ExtractFileMapEntries(domain.Task{}); got != nil {
		t.Errorf("expected nil for empty target file")
	}
}

func TestExtractFailureLearnings(t *testing.T) {
	// Task with logs and target file
	task := domain.Task{
		ID:          "12345678",
		TargetFile:  "internal/db/repo.go",
		ProjectPath: "/repo",
		Logs:        "connecting to db\npanic: connection refused",
	}
	fails := services.ExtractFailureLearnings(task)
	if len(fails) != 1 {
		t.Fatalf("expected 1 failure entry, got %d", len(fails))
	}
	if !strings.Contains(fails[0].Content, "* AVOID: failure on internal/db/repo.go: panic: connection refused") {
		t.Errorf("unexpected content: %s", fails[0].Content)
	}

	// Task with logs but no target file uses shortID
	taskNoTarget := domain.Task{
		ID:   "abcdef12",
		Logs: "error compiling",
	}
	failsNoTarget := services.ExtractFailureLearnings(taskNoTarget)
	if len(failsNoTarget) != 1 || !strings.Contains(failsNoTarget[0].Content, "abcdef12") {
		t.Errorf("expected shortID in content, got: %+v", failsNoTarget)
	}

	// Task without logs -> nil
	if got := services.ExtractFailureLearnings(domain.Task{}); got != nil {
		t.Errorf("expected nil for task without logs")
	}

	// Task with empty whitespace logs -> nil
	if got := services.ExtractFailureLearnings(domain.Task{Logs: "   \n\n   "}); got != nil {
		t.Errorf("expected nil for empty whitespace logs")
	}
}

func TestDigestTaskOutcome(t *testing.T) {
	ctx := context.Background()

	// 1. Nil brain returns nil
	if got := services.DigestTaskOutcome(ctx, nil, domain.Task{}, domain.StatusCompleted); got != nil {
		t.Errorf("expected nil for nil brain")
	}

	// 2. Non-terminal status returns nil
	brain := &mockBrainForLearning{}
	if got := services.DigestTaskOutcome(ctx, brain, domain.Task{}, domain.StatusProcessing); got != nil {
		t.Errorf("expected nil for processing status")
	}

	// 3. Completed task with convention and file map
	taskComp := domain.Task{
		ID:                  "comp-1",
		ProjectPath:         "/projects/app",
		TargetFile:          "internal/router/routes.go",
		Instruction:         "follow convention: register all HTTP routes under /api/v1",
		VerificationCommand: "go test ./...",
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	ingested := services.DigestTaskOutcome(ctx, brain, taskComp, domain.StatusCompleted)
	// Expect 3 entries: summary, convention, filemap
	if len(ingested) != 3 {
		t.Errorf("expected 3 entries ingested, got %d", len(ingested))
	}

	// 4. Failed task with failure log
	brainFail := &mockBrainForLearning{}
	taskFail := domain.Task{
		ID:          "fail-1",
		ProjectPath: "/projects/app",
		Instruction: "run migration",
		Logs:        "error: table already exists",
	}
	ingestedFail := services.DigestTaskOutcome(ctx, brainFail, taskFail, domain.StatusFailed)
	// Expect 2 entries: summary, failure learning
	if len(ingestedFail) != 2 {
		t.Errorf("expected 2 entries ingested, got %d", len(ingestedFail))
	}

	// 5. Ingestion error tolerance
	errBrain := &mockBrainForLearning{err: errors.New("write failed")}
	ingestedErr := services.DigestTaskOutcome(ctx, errBrain, taskComp, domain.StatusCompleted)
	if len(ingestedErr) != 0 {
		t.Errorf("expected 0 successfully ingested, got %d", len(ingestedErr))
	}
}
