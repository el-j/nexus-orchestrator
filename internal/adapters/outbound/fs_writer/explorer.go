package fs_writer

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"nexus-orchestrator/internal/core/ports"
)

const (
	maxListDirEntries = 200
	maxSearchMatches  = 50
	maxSearchFileSize = 512 << 10
)

var _ ports.ProjectExplorer = (*Writer)(nil)

// skippedDirs are large generated or VCS directories skipped during workspace exploration.
var skippedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".venv":        true,
}

// ListDir returns sorted entry names in relDir under projectPath, suffixing directories with "/".
// Paths escaping projectPath are blocked by safePath.
func (w *Writer) ListDir(projectPath, relDir string) ([]string, error) {
	if strings.TrimSpace(relDir) == "" {
		relDir = "."
	}
	fullPath, err := safePath(projectPath, relDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("fs_writer: list dir %q: %w", relDir, err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if skippedDirs[name] {
				continue
			}
			name += "/"
		}
		out = append(out, name)
		if len(out) >= maxListDirEntries {
			break
		}
	}
	return out, nil
}

// SearchFiles searches projectPath for files whose relative path or text lines match query
// (case-insensitive substring), returning at most maxSearchMatches formatted hits.
func (w *Writer) SearchFiles(projectPath, query string) ([]string, error) {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nil, fmt.Errorf("fs_writer: search query is required")
	}
	root, err := safePath(projectPath, ".")
	if err != nil {
		return nil, err
	}
	var matches []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && skippedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		relSlash := filepath.ToSlash(rel)
		if strings.Contains(strings.ToLower(relSlash), needle) {
			matches = append(matches, relSlash)
			if len(matches) >= maxSearchMatches {
				return filepath.SkipAll
			}
		}
		info, infoErr := d.Info()
		if infoErr != nil || info.Size() > maxSearchFileSize || !info.Mode().IsRegular() {
			return nil
		}
		matches = scanFileForNeedle(path, relSlash, needle, matches)
		if len(matches) >= maxSearchMatches {
			return filepath.SkipAll
		}
		return nil
	})
	return matches, nil
}

// scanFileForNeedle scans lines of a regular file and appends "relSlash:line: text" matches.
func scanFileForNeedle(fullPath, relSlash, needle string, matches []string) []string {
	f, err := os.Open(fullPath)
	if err != nil {
		return matches
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 16*1024), 256*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if strings.Contains(strings.ToLower(line), needle) {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 160 {
				trimmed = trimmed[:160] + "..."
			}
			matches = append(matches, fmt.Sprintf("%s:%d: %s", relSlash, lineNo, trimmed))
			if len(matches) >= maxSearchMatches {
				break
			}
		}
	}
	return matches
}
