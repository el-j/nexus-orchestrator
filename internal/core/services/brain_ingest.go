package services

import (
	"context"
	"fmt"
	"nexus-orchestrator/internal/core/domain"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// sectionKinds are the heading keywords of each kind, tried in order; a heading that matches none (for
// example "Conventions", "Style" or "Error handling") is a convention.
var sectionKinds = []struct {
	kind     domain.KnowledgeKind
	keywords []string
}{
	{domain.KnowledgeArchitecture, []string{"architecture", "design", "hexagonal", "layer"}},
	{domain.KnowledgeFileMap, []string{"file", "structure", "directory", "map", "path", "layout"}},
	{domain.KnowledgeLearning, []string{"learning", "lesson", "discovered", "pitfall", "avoid"}},
	{domain.KnowledgeGlossary, []string{"glossary", "term", "definition", "vocabulary"}},
}

// classifySection maps a markdown heading to the most appropriate KnowledgeKind.
func classifySection(heading string) domain.KnowledgeKind {
	h := strings.ToLower(heading)
	for _, sk := range sectionKinds {
		for _, kw := range sk.keywords {
			if strings.Contains(h, kw) {
				return sk.kind
			}
		}
	}
	return domain.KnowledgeConvention
}

// formatSectionTopic returns the knowledge topic for an ingested section, qualifying it
// with the file path when the file is a doc or README rather than a primary instruction file.
func formatSectionTopic(src, heading string) string {
	if !strings.HasPrefix(src, "file:") {
		return heading
	}
	rel := strings.TrimPrefix(src, "file:")
	base := filepath.Base(rel)
	lower := strings.ToLower(base)
	if lower == "claude.md" || lower == "agents.md" || lower == "gemini.md" || lower == "copilot-instructions.md" {
		return heading
	}
	return fmt.Sprintf("%s (%s)", heading, filepath.ToSlash(rel))
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
	if maxBytes <= 3 {
		return cutUTF8(s, maxBytes)
	}
	return cutUTF8(s, maxBytes-3) + "..."
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
	if b.embedder != nil && len(k.Embedding) == 0 {
		if emb, err := b.embedder.Embed(ctx, k.Topic+"\n"+k.Content); err == nil {
			k.Embedding = emb
		}
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
	src := "file:" + filepath.Base(real)
	root, rerr := filepath.EvalSymlinks(filepath.Clean(projectPath))
	if rerr == nil {
		if rel, err := filepath.Rel(root, real); err == nil && rel != "" && rel != "." && !strings.HasPrefix(rel, "..") {
			src = "file:" + filepath.ToSlash(rel)
		}
	}
	return b.ingestMarkdown(ctx, projectPath, src, data)
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
			topic := formatSectionTopic(src, "Overview")
			kind := classifySection("Overview")
			_, err := b.IngestKnowledge(ctx, domain.ProjectKnowledge{
				ProjectPath: projectPath,
				Kind:        kind,
				Topic:       topic,
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
		topic := formatSectionTopic(src, heading)
		kind := classifySection(heading)
		_, err := b.IngestKnowledge(ctx, domain.ProjectKnowledge{
			ProjectPath: projectPath,
			Kind:        kind,
			Topic:       topic,
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

// scanDocDir recursively collects markdown files in dir, skipping archive and hidden folders.
func scanDocDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if e.IsDir() {
			if name == "archive" || name == "testdata" || name == "node_modules" || strings.HasPrefix(name, ".") {
				continue
			}
			out = append(out, scanDocDir(filepath.Join(dir, e.Name()))...)
			continue
		}
		if strings.HasSuffix(name, ".md") {
			if info, err := e.Info(); err == nil && info.Size() <= domain.MaxIngestBytes {
				out = append(out, filepath.Join(dir, e.Name()))
			}
		}
	}
	return out
}

// findDocFiles returns sorted paths to markdown documentation files in the project docs directory.
func findDocFiles(projectPath string) []string {
	out := scanDocDir(filepath.Join(projectPath, "docs"))
	sort.Strings(out)
	return out
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
