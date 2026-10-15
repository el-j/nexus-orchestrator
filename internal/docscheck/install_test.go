//go:build !windows

package docscheck

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// installer runs scripts/install.sh against a local server that stands in for the release page.
func installer(t *testing.T, files map[string][]byte, env []string, args ...string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	t.Cleanup(srv.Close)
	cmd := exec.Command("sh", append([]string{filepath.Join(root, "scripts", "install.sh")}, args...)...)
	cmd.Env = append(os.Environ(), append([]string{"NEXUS_RELEASE_BASE=" + srv.URL, "NEXUS_VERSION=v9.9.9-test"}, env...)...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func sumLine(name string, data []byte) []byte {
	return []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name))
}

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var sb strings.Builder
	gz := gzip.NewWriter(&sb)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = gz.Close()
	return []byte(sb.String())
}

func platform() (osName, arch string) {
	osName = runtime.GOOS
	arch = runtime.GOARCH
	return
}

func TestInstallScript_IsValidShellAndDocumentsItsOptions(t *testing.T) {
	script := filepath.Join(root, "scripts", "install.sh")
	if out, err := exec.Command("sh", "-n", script).CombinedOutput(); err != nil {
		t.Fatalf("syntax: %v\n%s", err, out)
	}
	out, err := exec.Command("sh", script, "--help").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "--desktop") || !strings.Contains(string(out), "--beta") {
		t.Errorf("--help: %v\n%s", err, out)
	}
	if out, err := exec.Command("sh", script, "--nope").CombinedOutput(); err == nil || !strings.Contains(string(out), "Unknown option") {
		t.Errorf("an unknown option must fail: %v\n%s", err, out)
	}
}

func TestInstallScript_InstallsTheCLIBinariesAndChecksTheirChecksum(t *testing.T) {
	goos, goarch := platform()
	archive := fmt.Sprintf("nexus-orchestrator-%s-%s.tar.gz", goos, map[string]string{"amd64": "amd64", "arm64": "arm64"}[goarch])
	stub := "#!/bin/sh\nexit 0\n"
	data := tarGz(t, map[string]string{"nexus-cli": stub, "nexus-daemon": stub, "nexus-submit": stub})
	bin := t.TempDir()

	out, err := installer(t, map[string][]byte{archive: data, "SHA256SUMS.txt": sumLine(archive, data)}, []string{"NEXUS_INSTALL_DIR=" + bin})
	if err != nil {
		t.Fatalf("install failed: %v\n%s", err, out)
	}
	for _, b := range []string{"nexus-cli", "nexus-daemon", "nexus-submit"} {
		if fi, err := os.Stat(filepath.Join(bin, b)); err != nil || fi.Mode()&0o111 == 0 {
			t.Errorf("%s not installed and executable: %v", b, err)
		}
	}

	// A corrupted download must be refused and install nothing.
	bin2 := t.TempDir()
	out, err = installer(t, map[string][]byte{archive: data, "SHA256SUMS.txt": sumLine(archive, []byte("other"))}, []string{"NEXUS_INSTALL_DIR=" + bin2})
	if err == nil || !strings.Contains(out, "Checksum mismatch") {
		t.Errorf("a wrong checksum must fail: %v\n%s", err, out)
	}
	if entries, _ := os.ReadDir(bin2); len(entries) != 0 {
		t.Errorf("nothing may be installed after a failed check, found %d files", len(entries))
	}
}

// The desktop app is not notarized, so a browser-downloaded zip is blocked by Gatekeeper. The
// installer fetches with curl, drops the quarantine flag and leaves a launchable app behind.
func TestInstallScript_DesktopInstallsAnAppWithoutTheQuarantineFlag(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the desktop installer is for macOS")
	}
	_, goarch := platform()
	src := t.TempDir()
	app := filepath.Join(src, "nexusOrchestrator.app", "Contents", "MacOS")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "nexusOrchestrator"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	zip := filepath.Join(t.TempDir(), "a.zip")
	if out, err := exec.Command("ditto", "-c", "-k", src, zip).CombinedOutput(); err != nil {
		t.Fatalf("ditto: %v\n%s", err, out)
	}
	data, _ := os.ReadFile(zip)
	archive := "nexus-orchestrator-desktop-darwin-" + goarch + ".zip"
	apps := t.TempDir()

	out, err := installer(t, map[string][]byte{archive: data, "SHA256SUMS-desktop.txt": sumLine(archive, data)}, []string{"NEXUS_APP_DIR=" + apps}, "--desktop")
	if err != nil {
		t.Fatalf("desktop install failed: %v\n%s", err, out)
	}
	installed := filepath.Join(apps, "nexusOrchestrator.app")
	if _, err := os.Stat(filepath.Join(installed, "Contents", "MacOS", "nexusOrchestrator")); err != nil {
		t.Fatalf("app not installed: %v", err)
	}
	if attrs, _ := exec.Command("xattr", installed).Output(); strings.Contains(string(attrs), "com.apple.quarantine") {
		t.Errorf("the installed app is quarantined: %s", attrs)
	}
	// Reinstalling replaces the app.
	if out, err := installer(t, map[string][]byte{archive: data, "SHA256SUMS-desktop.txt": sumLine(archive, data)}, []string{"NEXUS_APP_DIR=" + apps}, "--desktop"); err != nil {
		t.Errorf("reinstall failed: %v\n%s", err, out)
	}
}

func TestInstallScript_DesktopIsRefusedOutsideMacOS(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("covered by the real installer test on macOS")
	}
	out, err := installer(t, nil, nil, "--desktop")
	if err == nil || !strings.Contains(out, "--desktop is for macOS") {
		t.Errorf("want a clear refusal: %v\n%s", err, out)
	}
}
