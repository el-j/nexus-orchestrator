package sys_scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// shimDirMarkers identify directories a terminal app puts first on PATH to wrap a tool
// (cmux creates one per terminal session). A path inside one stops working when that session
// ends, so it must not be saved as a provider's command.
var shimDirMarkers = []string{"cmux-cli-shims", "cmux.app/contents/resources/bin"}

// lookPathReal finds binary like exec.LookPath but skips terminal-shim directories; when the
// shim is the only copy it is returned anyway, so the tool is still reported as installed.
func lookPathReal(binary string) (string, error) {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" || isShimDir(dir) {
			continue
		}
		if p, err := exec.LookPath(filepath.Join(dir, binary)); err == nil {
			return p, nil
		}
	}
	return exec.LookPath(binary)
}

// isShimDir reports whether dir is one of the terminal-shim directories.
func isShimDir(dir string) bool {
	d := strings.ToLower(filepath.ToSlash(dir))
	for _, m := range shimDirMarkers {
		if strings.Contains(d, m) {
			return true
		}
	}
	return false
}
