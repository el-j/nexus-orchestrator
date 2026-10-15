package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
)

// compile-time interface check
var _ ports.BrainService = (*BrainServiceImpl)(nil)

// BrainServiceImpl implements ports.BrainService.
type BrainServiceImpl struct {
	repo     ports.ProjectKnowledgeRepository
	taskRepo ports.TaskRepository
}

// NewBrainService returns a BrainServiceImpl using the given repositories.
func NewBrainService(repo ports.ProjectKnowledgeRepository, taskRepo ports.TaskRepository) *BrainServiceImpl {
	return &BrainServiceImpl{repo: repo, taskRepo: taskRepo}
}

// estimateStringTokens approximates token count using GPT-family heuristics (~4 chars/token).
func estimateStringTokens(text string) int {
	n := len(text) / 4
	if n < 1 {
		return 1
	}
	return n
}

// cutUTF8 returns the longest prefix of s that is at most maxBytes long and does
// not end in the middle of a multi-byte character.
func cutUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}

// truncateUTF8 shortens s to at most maxBytes (rune-safe) and appends "..." when
// anything was removed.
func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return cutUTF8(s, maxBytes) + "..."
}

// kindPriority defines the assembly order for GetContext.
var kindPriority = []domain.KnowledgeKind{
	domain.KnowledgeArchitecture,
	domain.KnowledgeConvention,
	domain.KnowledgeFileMap,
	domain.KnowledgeLearning,
	domain.KnowledgeGlossary,
	domain.KnowledgeTaskSummary,
}

// GetContext assembles a ContextResponse in priority order within the MaxTokens budget (default 400).
func (b *BrainServiceImpl) GetContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	if q.MaxTokens == 0 {
		q.MaxTokens = 400
	}
	entries, err := b.repo.GetByProject(ctx, filepath.Clean(q.ProjectPath))
	if err != nil {
		return domain.ContextResponse{}, fmt.Errorf("brain_service: get context: %w", err)
	}

	// Group by kind.
	byKind := make(map[domain.KnowledgeKind][]domain.ProjectKnowledge)
	for _, e := range entries {
		byKind[e.Kind] = append(byKind[e.Kind], e)
	}
	// Sort each group by relevance descending.
	for k := range byKind {
		sort.Slice(byKind[k], func(i, j int) bool {
			return byKind[k][i].RelevanceScore > byKind[k][j].RelevanceScore
		})
	}

	focus := strings.ToLower(q.FocusArea)
	resp := domain.ContextResponse{
		ProjectPath: filepath.Clean(q.ProjectPath),
		TokenBudget: q.MaxTokens,
		Sections:    []domain.ContextSection{},
	}

	for _, kind := range kindPriority {
		group := byKind[kind]
		for _, e := range group {
			// Filter convention and file_map by focus area when requested.
			if focus != "" && focus != "all" {
				if kind == domain.KnowledgeConvention || kind == domain.KnowledgeFileMap {
					if !strings.Contains(strings.ToLower(e.Topic), focus) {
						continue
					}
				}
			}
			if resp.TokensUsed+e.TokenCount > q.MaxTokens {
				resp.Truncated = true
				break
			}
			resp.Sections = append(resp.Sections, domain.ContextSection{
				Kind:    e.Kind,
				Topic:   e.Topic,
				Content: e.Content,
				Tokens:  e.TokenCount,
			})
			resp.TokensUsed += e.TokenCount
			if kind == domain.KnowledgeFileMap {
				resp.SuggestedFiles = append(resp.SuggestedFiles, e.Content)
			}
		}
		if resp.Truncated {
			break
		}
	}
	return resp, nil
}

// GetFocusedContext uses BM25 search to assemble context relevant to a specific question.
func (b *BrainServiceImpl) GetFocusedContext(ctx context.Context, q domain.ContextQuery) (domain.ContextResponse, error) {
	if q.MaxTokens == 0 {
		q.MaxTokens = 400
	}
	entries, err := b.repo.SearchFTS(ctx, filepath.Clean(q.ProjectPath), q.Question, 20)
	if err != nil {
		return domain.ContextResponse{}, fmt.Errorf("brain_service: focused context: %w", err)
	}
	resp := domain.ContextResponse{
		ProjectPath: filepath.Clean(q.ProjectPath),
		TokenBudget: q.MaxTokens,
		Sections:    []domain.ContextSection{},
	}
	for _, e := range entries {
		if resp.TokensUsed+e.TokenCount > q.MaxTokens {
			resp.Truncated = true
			break
		}
		resp.Sections = append(resp.Sections, domain.ContextSection{
			Kind:    e.Kind,
			Topic:   e.Topic,
			Content: e.Content,
			Tokens:  e.TokenCount,
		})
		resp.TokensUsed += e.TokenCount
	}
	return resp, nil
}

// SearchKnowledge performs BM25 search and returns ContextSection slices within budget.
func (b *BrainServiceImpl) SearchKnowledge(ctx context.Context, projectPath, query string, maxTokens int) ([]domain.ContextSection, error) {
	itemLimit := 0
	if maxTokens > 0 && maxTokens <= 20 {
		// Caller passed an item count limit (e.g. limit=5 or limit=10), not a token budget.
		itemLimit = maxTokens
		maxTokens = 400
	} else if maxTokens <= 0 {
		maxTokens = 400
	}
	// itemLimit is at most 20 (see above), so a fixed fetch of 20 always covers it.
	const fetchCount = 20
	entries, err := b.repo.SearchFTS(ctx, filepath.Clean(projectPath), query, fetchCount)
	if err != nil {
		return nil, fmt.Errorf("brain_service: search knowledge: %w", err)
	}
	var sections []domain.ContextSection
	used := 0
	for _, e := range entries {
		if itemLimit > 0 && len(sections) >= itemLimit {
			break
		}
		if used+e.TokenCount > maxTokens {
			break
		}
		sections = append(sections, domain.ContextSection{
			Kind:    e.Kind,
			Topic:   e.Topic,
			Content: e.Content,
			Tokens:  e.TokenCount,
		})
		used += e.TokenCount
	}
	return sections, nil
}

// GetFileMap returns file path strings from file_map knowledge entries, optionally filtered.
func (b *BrainServiceImpl) GetFileMap(ctx context.Context, projectPath, focusArea string) ([]string, error) {
	entries, err := b.repo.GetByProjectAndKind(ctx, filepath.Clean(projectPath), domain.KnowledgeFileMap)
	if err != nil {
		return nil, fmt.Errorf("brain_service: get file map: %w", err)
	}
	focus := strings.ToLower(focusArea)
	var files []string
	for _, e := range entries {
		if focus != "" && !strings.Contains(strings.ToLower(e.Topic), focus) {
			continue
		}
		files = append(files, e.Content)
	}
	return files, nil
}

// instructionFiles are the files in which the agents that work on a project keep their
// instructions. The first of a group that exists is ingested, so a project that has both
// CLAUDE.md and .claude/CLAUDE.md is not read twice.
var instructionFiles = [][]string{
	{"CLAUDE.md", filepath.Join(".claude", "CLAUDE.md")},
	{"AGENTS.md"},
	{"GEMINI.md"},
	{filepath.Join(".github", "copilot-instructions.md")},
}

// InitProject ingests the project's agent instructions. With an explicit claudeMDPath only that
// file is read; with none, the project's CLAUDE.md, AGENTS.md, GEMINI.md and Copilot
// instructions are found by themselves, whichever agent works on the project.
func (b *BrainServiceImpl) InitProject(ctx context.Context, projectPath, claudeMDPath string) (domain.BrainStatus, error) {
	var paths []string
	if claudeMDPath != "" {
		paths = []string{claudeMDPath}
	} else {
		for _, group := range instructionFiles {
			for _, rel := range group {
				if _, err := os.Stat(filepath.Join(projectPath, rel)); err == nil {
					paths = append(paths, filepath.Join(projectPath, rel))
					break
				}
			}
		}
	}
	for _, path := range paths {
		if _, err := b.IngestFromFile(ctx, projectPath, path); err != nil {
			return domain.BrainStatus{}, fmt.Errorf("brain_service: init project: %w", err)
		}
	}
	// A project that is introduced to the orchestrator also gets its rule file, once.
	// Failing to write it (read-only checkout, a file named .nexus) must not fail the init.
	if created, err := EnsureProjectPolicy(projectPath); err != nil {
		log.Printf("brain_service: init project %s: policy: %v", projectPath, err)
	} else if created {
		log.Printf("brain_service: init project %s: wrote default %s", projectPath, PolicyFile)
	}
	return b.repo.GetStatus(ctx, filepath.Clean(projectPath))
}

// GetStatus returns the current brain status for a project.
func (b *BrainServiceImpl) GetStatus(ctx context.Context, projectPath string) (domain.BrainStatus, error) {
	return b.repo.GetStatus(ctx, filepath.Clean(projectPath))
}

// ListKnowledge returns all knowledge entries for a project, optionally filtered by kind string.
func (b *BrainServiceImpl) ListKnowledge(ctx context.Context, projectPath, kind string) ([]domain.ProjectKnowledge, error) {
	clean := filepath.Clean(projectPath)
	if kind != "" {
		return b.repo.GetByProjectAndKind(ctx, clean, domain.KnowledgeKind(kind))
	}
	return b.repo.GetByProject(ctx, clean)
}

// DeleteKnowledge removes a knowledge entry by ID.
func (b *BrainServiceImpl) DeleteKnowledge(ctx context.Context, id string) error {
	return b.repo.DeleteKnowledge(ctx, id)
}

// GetOnboardingContext returns a concise, token-budgeted Tier 0/1 onboarding summary
// for an AI agent starting a session on a project.
func (b *BrainServiceImpl) GetOnboardingContext(ctx context.Context, projectPath string, maxTokens int) (string, error) {
	return b.buildOnboarding(ctx, projectPath, maxTokens, true)
}

// GetProjectBrief is GetOnboardingContext without the workflow rules. The engine
// states the rules itself, in full and once, so the briefing must not repeat them.
func (b *BrainServiceImpl) GetProjectBrief(ctx context.Context, projectPath string, maxTokens int) (string, error) {
	return b.buildOnboarding(ctx, projectPath, maxTokens, false)
}

func (b *BrainServiceImpl) buildOnboarding(ctx context.Context, projectPath string, maxTokens int, withRules bool) (string, error) {
	if maxTokens <= 0 {
		maxTokens = 800
	}
	clean := filepath.Clean(projectPath)
	projectName := filepath.Base(clean)

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Project Onboarding: %s\n\n", projectName)
	fmt.Fprintf(&sb, "**Path:** `%s`\n", clean)

	// 1. Stack detection
	var stackParts []string
	if _, err := os.Stat(filepath.Join(clean, "go.mod")); err == nil {
		stackParts = append(stackParts, "Go")
	}
	if _, err := os.Stat(filepath.Join(clean, "package.json")); err == nil {
		stackParts = append(stackParts, "TypeScript/JavaScript (Node)")
	}
	if _, err := os.Stat(filepath.Join(clean, "Cargo.toml")); err == nil {
		stackParts = append(stackParts, "Rust")
	}
	if _, err := os.Stat(filepath.Join(clean, "pyproject.toml")); err == nil {
		stackParts = append(stackParts, "Python")
	} else if _, err := os.Stat(filepath.Join(clean, "requirements.txt")); err == nil {
		stackParts = append(stackParts, "Python")
	}

	stackStr := "Generic"
	if len(stackParts) > 0 {
		stackStr = strings.Join(stackParts, ", ")
	}
	fmt.Fprintf(&sb, "**Stack:** %s\n\n", stackStr)

	// 2. Active Development State
	sb.WriteString("## Active Development State\n")
	activePlan := "None"
	orchestratorPath := filepath.Join(clean, ".claude", "orchestrator.json")
	if data, err := os.ReadFile(orchestratorPath); err == nil {
		var orchConfig struct {
			ActivePlanID string `json:"activePlanId"`
		}
		if err := json.Unmarshal(data, &orchConfig); err == nil && orchConfig.ActivePlanID != "" {
			activePlan = orchConfig.ActivePlanID
		}
	}
	// Agents that do not keep orchestrator.json up to date still leave plan and task files.
	planFiles, taskFiles := readPlanFiles(clean)
	if activePlan == "None" {
		if p, ok := activePlanFromFiles(planFiles); ok {
			activePlan = p.ID + " — " + p.Title
		}
	}
	fmt.Fprintf(&sb, "- **Active Plan:** %s\n", activePlan)
	if summary := taskFileSummary(taskFiles); summary != "" {
		fmt.Fprintf(&sb, "- **Plan files:** %s\n", summary)
	}

	if b.taskRepo != nil {
		tasks, _ := b.taskRepo.GetByProjectPath(ctx, clean)
		var queued, processing, completed, failed int
		var activeTaskDesc string
		for _, t := range tasks {
			switch t.Status {
			case domain.StatusQueued:
				queued++
			case domain.StatusProcessing:
				processing++
				if activeTaskDesc == "" {
					activeTaskDesc = fmt.Sprintf("%s (%s)", t.ID, t.Instruction)
				}
			case domain.StatusCompleted:
				completed++
			case domain.StatusFailed:
				failed++
			}
		}
		fmt.Fprintf(&sb, "- **Tasks:** %d queued, %d processing, %d completed, %d failed\n",
			queued, processing, completed, failed)
		if activeTaskDesc != "" {
			fmt.Fprintf(&sb, "- **Current Task:** %s\n", activeTaskDesc)
		}
	}
	sb.WriteString("\n")

	// 2b. Project rules go before everything that can be truncated: a fresh session
	// must learn them even on a tight budget.
	policy, hasPolicy, policyErr := LoadProjectPolicy(clean)
	switch {
	case policyErr != nil:
		fmt.Fprintf(&sb, "## Workflow rules\n- The project's policy file is invalid and no rules are applied until it is fixed: %v\n\n", policyErr)
	case hasPolicy && withRules:
		sb.WriteString(renderPolicy(policy))
		sb.WriteString("\n")
	}

	// 3. Architecture & Invariants
	archEntries, _ := b.repo.GetByProjectAndKind(ctx, clean, domain.KnowledgeArchitecture)
	if len(archEntries) > 0 {
		sb.WriteString("## Architectural Overview\n")
		for _, a := range archEntries {
			content := truncateUTF8(strings.TrimSpace(a.Content), 300)
			fmt.Fprintf(&sb, "### %s\n%s\n\n", a.Topic, content)
		}
	}

	// 4. Key Conventions & Rules
	convEntries, _ := b.repo.GetByProjectAndKind(ctx, clean, domain.KnowledgeConvention)
	if len(convEntries) > 0 {
		sb.WriteString("## Key Conventions & Constraints\n")
		count := 0
		for _, c := range convEntries {
			content := truncateUTF8(strings.TrimSpace(c.Content), 250)
			fmt.Fprintf(&sb, "- **%s:** %s\n", c.Topic, content)
			count++
			if count >= 5 {
				break
			}
		}
		sb.WriteString("\n")
	}

	// 5. Mandatory Verification Commands (a policy states the real one; guessing would contradict it)
	if hasPolicy {
		return b.fitOnboarding(sb.String(), maxTokens), nil
	}
	sb.WriteString("## Verification Commands\n")
	var verifCommands []string
	if strings.Contains(stackStr, "Go") {
		verifCommands = append(verifCommands, "CGO_ENABLED=1 CGO_CFLAGS=\"-DSQLITE_ENABLE_FTS5\" go test -race -count=1 ./...")
	}
	if strings.Contains(stackStr, "TypeScript") || strings.Contains(stackStr, "Node") {
		verifCommands = append(verifCommands, "npm test")
	}
	if strings.Contains(stackStr, "Rust") {
		verifCommands = append(verifCommands, "cargo test")
	}
	if strings.Contains(stackStr, "Python") {
		verifCommands = append(verifCommands, "pytest")
	}
	if len(verifCommands) == 0 {
		verifCommands = append(verifCommands, "Run project-specific test suite before committing.")
	}
	for _, cmd := range verifCommands {
		fmt.Fprintf(&sb, "- `%s`\n", cmd)
	}

	return b.fitOnboarding(sb.String(), maxTokens), nil
}

// fitOnboarding cuts text to the token budget, marking the cut.
func (b *BrainServiceImpl) fitOnboarding(result string, maxTokens int) string {
	if estimateStringTokens(result) > maxTokens {
		if maxChars := maxTokens * 4; len(result) > maxChars {
			result = cutUTF8(result, maxChars) + "\n\n...[Onboarding context truncated to fit token budget]"
		}
	}
	return result
}
