package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nexus-orchestrator/internal/core/domain"
)

// MaxIngestBytes caps the size of a file the brain will read (1 MiB).
const MaxIngestBytes = 1 << 20

// IngestAllowedDirsEnv names an optional list of extra directories (separated
// like PATH) that ingestion may read from in addition to the project itself, for
// shared templates kept outside a repository.
const IngestAllowedDirsEnv = "NEXUS_INGEST_ALLOWED_DIRS"

// sensitiveSegments are directory names that hold credentials. A file below one of
// them is refused even when it is technically inside the project, so pointing a
// "project" at ~/.ssh does not turn ingestion into a key reader.
var sensitiveSegments = map[string]bool{
	".ssh": true, ".gnupg": true, ".aws": true, ".kube": true, ".azure": true,
	".docker": true, ".git": true,
}

// resolveIngestPath validates a client-supplied file path and returns the real
// path to read. A relative filePath is taken relative to projectPath. The result,
// after symlink resolution, must lie inside projectPath (or an allowed directory),
// must be a regular file no larger than MaxIngestBytes, and must not be a
// credentials location. Violations wrap domain.ErrPathNotAllowed.
func resolveIngestPath(projectPath, filePath string) (string, error) {
	if strings.TrimSpace(projectPath) == "" {
		return "", fmt.Errorf("%w: projectPath is required", domain.ErrPathNotAllowed)
	}
	if strings.TrimSpace(filePath) == "" {
		return "", fmt.Errorf("%w: filePath is required", domain.ErrPathNotAllowed)
	}

	root, err := filepath.EvalSymlinks(filepath.Clean(projectPath))
	if err != nil {
		return "", fmt.Errorf("%w: project directory is not accessible", domain.ErrPathNotAllowed)
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: project path is not a directory", domain.ErrPathNotAllowed)
	}
	if root == filepath.Dir(root) { // "/" or a volume root
		return "", fmt.Errorf("%w: the filesystem root cannot be a project", domain.ErrPathNotAllowed)
	}

	candidate := filePath
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(root, candidate)
	}
	real, err := filepath.EvalSymlinks(filepath.Clean(candidate))
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("brain_service: ingest from file: %w", err)
		}
		return "", fmt.Errorf("%w: file is not accessible", domain.ErrPathNotAllowed)
	}

	if !withinAny(real, append([]string{root}, ingestAllowedDirs()...)) {
		return "", fmt.Errorf("%w: %s is outside the project", domain.ErrPathNotAllowed, filepath.Base(real))
	}
	if sensitivePath(real) {
		return "", fmt.Errorf("%w: %s is a credentials location", domain.ErrPathNotAllowed, filepath.Base(real))
	}

	info, err := os.Stat(real)
	if err != nil {
		return "", fmt.Errorf("brain_service: ingest from file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %s is not a regular file", domain.ErrPathNotAllowed, filepath.Base(real))
	}
	if info.Size() > MaxIngestBytes {
		return "", fmt.Errorf("%w: %s is larger than %d bytes", domain.ErrPathNotAllowed, filepath.Base(real), MaxIngestBytes)
	}
	return real, nil
}

// ingestAllowedDirs returns the real paths of the directories named in
// NEXUS_INGEST_ALLOWED_DIRS; entries that do not exist are ignored.
func ingestAllowedDirs() []string {
	var dirs []string
	for _, d := range filepath.SplitList(os.Getenv(IngestAllowedDirsEnv)) {
		if d = strings.TrimSpace(d); d == "" {
			continue
		}
		if real, err := filepath.EvalSymlinks(filepath.Clean(d)); err == nil {
			dirs = append(dirs, real)
		}
	}
	return dirs
}

// withinAny reports whether path is dir or lies below one of dirs.
func withinAny(path string, dirs []string) bool {
	for _, d := range dirs {
		rel, err := filepath.Rel(d, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}

// sensitivePath reports whether any directory component, or the file name itself,
// marks the path as a credentials location.
func sensitivePath(path string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		if sensitiveSegments[seg] {
			return true
		}
	}
	base := filepath.Base(path)
	return base == ".env" || strings.HasPrefix(base, ".env.") || base == ".netrc" || base == ".npmrc" || base == "id_rsa" || base == "id_ed25519"
}
