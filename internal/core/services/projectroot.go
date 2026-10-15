package services

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectRoot returns the repository an agent's working directory belongs to: the nearest
// ancestor with a .git entry (a folder, or a file for a worktree), never looking above the home
// directory. A directory outside any repository is returned as it is. An agent started in
// frontend/ or src/ works on the project that contains them; without this one project shows up
// as several.
func ProjectRoot(dir string) string {
	if dir == "" {
		return ""
	}
	dir = filepath.Clean(dir)
	home, _ := os.UserHomeDir()
	for cur := dir; ; {
		if cur == home || cur == filepath.Dir(cur) {
			return dir
		}
		if _, err := os.Stat(filepath.Join(cur, ".git")); err == nil {
			return cur
		}
		cur = filepath.Dir(cur)
	}
}

// IsScratchDir reports whether dir is a temporary directory (and not inside the home directory), where an agent runs a throwaway
// command and which is not a project anyone works on.
func IsScratchDir(dir string) bool {
	if dir == "" {
		return false
	}
	dir = filepath.Clean(dir)
	// Whatever is inside the user's home directory is theirs, wherever that directory lives.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if h := filepath.Clean(home); dir == h || strings.HasPrefix(dir, h+string(filepath.Separator)) {
			return false
		}
	}
	roots := []string{os.TempDir(), "/tmp", "/private/tmp", "/var/folders", "/private/var/folders"}
	for _, r := range roots {
		r = filepath.Clean(r)
		if dir == r || strings.HasPrefix(dir, r+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
