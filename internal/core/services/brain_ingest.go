package services

import (
	"context"
	"fmt"
	"nexus-orchestrator/internal/core/domain"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
//
// filePath is client-supplied, so it is confined: relative paths are taken from
// projectPath and the resolved file must lie inside it (see resolveIngestPath).
// Violations wrap domain.ErrPathNotAllowed.
func (b *BrainServiceImpl) IngestFromFile(ctx context.Context, projectPath, filePath string) (int, error) {
	real, err := resolveIngestPath(projectPath, filePath)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(real)
	if err != nil {
		return 0, fmt.Errorf("brain_service: ingest from file: %w", err)
	}
	return b.ingestMarkdown(ctx, projectPath, "file:"+filepath.Base(real), data)
}

// IngestContent ingests markdown supplied by the caller (for example the contents
// of a file picked in a browser, which has no path the daemon could read). name is
// only a label for the source; content is parsed exactly like a CLAUDE.md file.
// Nothing is read from disk, so there is no path to confine; the size is capped at
// MaxIngestBytes and larger content wraps domain.ErrInputTooLarge.
func (b *BrainServiceImpl) IngestContent(ctx context.Context, projectPath, name, content string) (int, error) {
	if strings.TrimSpace(projectPath) == "" {
		return 0, fmt.Errorf("brain_service: ingest content: projectPath is required")
	}
	if len(content) > MaxIngestBytes {
		return 0, fmt.Errorf("brain_service: ingest content: %w: content is larger than %d bytes", domain.ErrInputTooLarge, MaxIngestBytes)
	}
	label := filepath.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	if label == "" || label == "." || label == "/" {
		label = "upload"
	}
	return b.ingestMarkdown(ctx, projectPath, "upload:"+label, []byte(content))
}

// ingestMarkdown splits a CLAUDE.md-style document into sections and stores each.
func (b *BrainServiceImpl) ingestMarkdown(ctx context.Context, projectPath, src string, data []byte) (int, error) {
	// Split on "## " to get sections; the first element is preamble before the first ##.
	// A leading "\n" makes a file that starts directly with "## Heading" split into
	// an empty preamble plus a normal first section, instead of the heading being
	// mistaken for a "# Title" line and discarded.
	raw := strings.Split("\n"+string(data), "\n## ")
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
