package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"

	"github.com/google/uuid"
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

// classifySection maps a markdown heading to the most appropriate KnowledgeKind.
func classifySection(heading string) domain.KnowledgeKind {
	h := strings.ToLower(heading)
	switch {
	case strings.Contains(h, "architecture") || strings.Contains(h, "design") ||
		strings.Contains(h, "hexagonal") || strings.Contains(h, "layer"):
		return domain.KnowledgeArchitecture
	case strings.Contains(h, "file") || strings.Contains(h, "structure") ||
		strings.Contains(h, "directory") || strings.Contains(h, "map") ||
		strings.Contains(h, "path") || strings.Contains(h, "layout"):
		return domain.KnowledgeFileMap
	case strings.Contains(h, "learning") || strings.Contains(h, "lesson") ||
		strings.Contains(h, "discovered") || strings.Contains(h, "pitfall") ||
		strings.Contains(h, "avoid"):
		return domain.KnowledgeLearning
	case strings.Contains(h, "glossary") || strings.Contains(h, "term") ||
		strings.Contains(h, "definition") || strings.Contains(h, "vocabulary"):
		return domain.KnowledgeGlossary
	case strings.Contains(h, "convention") || strings.Contains(h, "style") ||
		strings.Contains(h, "rule") || strings.Contains(h, "pattern") ||
		strings.Contains(h, "standard") || strings.Contains(h, "error"):
		return domain.KnowledgeConvention
	default:
		return domain.KnowledgeConvention
	}
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

// IngestKnowledge stores a knowledge entry, computing token count and setting timestamps.
func (b *BrainServiceImpl) IngestKnowledge(ctx context.Context, k domain.ProjectKnowledge) (domain.ProjectKnowledge, error) {
	k.ProjectPath = filepath.Clean(k.ProjectPath)
	if k.ID == "" {
		k.ID = uuid.New().String()
	}
	k.TokenCount = estimateStringTokens(k.Content)
	if k.RelevanceScore == 0 {
		k.RelevanceScore = 0.5
	}
	now := time.Now()
	if k.CreatedAt.IsZero() {
		k.CreatedAt = now
	}
	k.UpdatedAt = now
	if err := b.repo.SaveKnowledge(ctx, k); err != nil {
		return domain.ProjectKnowledge{}, fmt.Errorf("brain_service: ingest knowledge: %w", err)
	}
	return k, nil
}

// IngestFromFile parses a CLAUDE.md-style markdown file and ingests each ## section.
func (b *BrainServiceImpl) IngestFromFile(ctx context.Context, projectPath, filePath string) (int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("brain_service: ingest from file: %w", err)
	}
	src := "file:" + filepath.Base(filePath)
	// Split on "## " to get sections; the first element is preamble before the first ##.
	raw := strings.Split(string(data), "\n## ")
	count := 0
	var sectionErrors []error
	for i, section := range raw {
		if i == 0 {
			// Preamble — skip or treat as convention if non-trivial.
			trimmed := strings.TrimSpace(section)
			// Strip leading "# " title line if present.
			lines := strings.SplitN(trimmed, "\n", 2)
			if len(lines) == 2 {
				section = strings.TrimSpace(lines[1])
			} else {
				continue
			}
			if len(strings.TrimSpace(section)) < 20 {
				continue
			}
			// Ingest preamble as a general convention.
			_, err := b.IngestKnowledge(ctx, domain.ProjectKnowledge{
				ProjectPath: projectPath,
				Kind:        domain.KnowledgeConvention,
				Topic:       "Overview",
				Content:     strings.TrimSpace(section),
				Source:      src,
			})
			if err == nil {
				count++
			} else {
				sectionErrors = append(sectionErrors, err)
			}
			continue
		}
		// Each subsequent section starts with the heading (everything before first \n).
		nl := strings.IndexByte(section, '\n')
		if nl < 0 {
			continue
		}
		heading := strings.TrimSpace(section[:nl])
		content := strings.TrimSpace(section[nl+1:])
		if len(content) < 20 {
			continue
		}
		kind := classifySection(heading)
		_, err := b.IngestKnowledge(ctx, domain.ProjectKnowledge{
			ProjectPath: projectPath,
			Kind:        kind,
			Topic:       heading,
			Content:     content,
			Source:      src,
		})
		if err == nil {
			count++
		} else {
			sectionErrors = append(sectionErrors, err)
		}
	}
	if count == 0 && len(sectionErrors) > 0 {
		return 0, fmt.Errorf("brain_service: IngestFromFile: all %d sections failed to save: %w", len(sectionErrors), sectionErrors[0])
	}
	return count, nil
}

// SearchKnowledge performs BM25 search and returns ContextSection slices within budget.
func (b *BrainServiceImpl) SearchKnowledge(ctx context.Context, projectPath, query string, maxTokens int) ([]domain.ContextSection, error) {
	if maxTokens == 0 {
		maxTokens = 400
	}
	entries, err := b.repo.SearchFTS(ctx, filepath.Clean(projectPath), query, 20)
	if err != nil {
		return nil, fmt.Errorf("brain_service: search knowledge: %w", err)
	}
	var sections []domain.ContextSection
	used := 0
	for _, e := range entries {
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

// InitProject ingests the project CLAUDE.md (auto-detected if claudeMDPath is empty).
func (b *BrainServiceImpl) InitProject(ctx context.Context, projectPath, claudeMDPath string) (domain.BrainStatus, error) {
	if claudeMDPath == "" {
		candidates := []string{
			filepath.Join(projectPath, "CLAUDE.md"),
			filepath.Join(projectPath, ".claude", "CLAUDE.md"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				claudeMDPath = c
				break
			}
		}
	}
	if claudeMDPath != "" {
		if _, err := b.IngestFromFile(ctx, projectPath, claudeMDPath); err != nil {
			return domain.BrainStatus{}, fmt.Errorf("brain_service: init project: %w", err)
		}
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
