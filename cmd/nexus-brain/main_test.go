package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNexusBrain_VersionAndHelpExitZero(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"--version"}, &out, &errb); code != 0 {
		t.Fatalf("--version: code=%d stderr=%q", code, errb.String())
	}
	if !strings.Contains(out.String(), "dev (unknown unknown)") {
		t.Errorf("version output = %q", out.String())
	}

	out.Reset()
	if code := run([]string{"--help"}, &out, &errb); code != 0 || !strings.Contains(out.String(), "search") {
		t.Fatalf("--help: code=%d out=%q", code, out.String())
	}
}

func TestNexusBrain_UnknownCommandFails(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"unknown-subcommand"}, &out, &errb); code != 1 {
		t.Fatalf("code=%d, want 1", code)
	}
	if !strings.Contains(errb.String(), "unknown command") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestNexusBrain_WithRemoteAddr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"projectPath":"/test","initialized":true,"entryCount":5}`)
	}))
	defer srv.Close()

	var out, errb bytes.Buffer
	code := run([]string{"--addr", srv.URL, "status", "--project", "/test"}, &out, &errb)
	if code != 0 {
		t.Fatalf("run with --addr: code=%d stderr=%q", code, errb.String())
	}
	if !strings.Contains(out.String(), `"entryCount": 5`) {
		t.Errorf("expected status output, got %q", out.String())
	}

	// Also test --addr=URL syntax
	out.Reset()
	errb.Reset()
	code = run([]string{"--addr=" + srv.URL, "status", "--project", "/test"}, &out, &errb)
	if code != 0 {
		t.Fatalf("run with --addr=...: code=%d stderr=%q", code, errb.String())
	}
}

func TestNexusBrain_WithLocalSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "brain.db")

	var out, errb bytes.Buffer
	// Test --db flag
	code := run([]string{"--db", dbPath, "init", "--project", tmpDir}, &out, &errb)
	if code != 0 {
		t.Fatalf("run init: code=%d stderr=%q", code, errb.String())
	}
	if !strings.Contains(out.String(), `"projectPath"`) {
		t.Errorf("expected init output, got %q", out.String())
	}

	// Test --db=path syntax and search
	out.Reset()
	errb.Reset()
	code = run([]string{"--db=" + dbPath, "search", "--all", "anything"}, &out, &errb)
	if code != 0 {
		t.Fatalf("run search: code=%d stderr=%q", code, errb.String())
	}
}

func TestResolveDatabasePath(t *testing.T) {
	// Explicit flag takes highest precedence
	if got := resolveDatabasePath("/custom/path.db"); got != "/custom/path.db" {
		t.Errorf("got %q, want /custom/path.db", got)
	}

	// NEXUS_BRAIN_DB env var
	t.Setenv("NEXUS_BRAIN_DB", "/env/brain.db")
	t.Setenv("NEXUS_DB_PATH", "/env/other.db")
	if got := resolveDatabasePath(""); got != "/env/brain.db" {
		t.Errorf("got %q, want /env/brain.db", got)
	}

	// NEXUS_DB_PATH env var
	t.Setenv("NEXUS_BRAIN_DB", "")
	if got := resolveDatabasePath(""); got != "/env/other.db" {
		t.Errorf("got %q, want /env/other.db", got)
	}
	t.Setenv("NEXUS_DB_PATH", "")

	// Check fallback
	def := resolveDatabasePath("")
	if !strings.HasSuffix(def, ".db") {
		t.Errorf("expected .db suffix, got %q", def)
	}
}

func TestNexusBrain_InvalidDatabasePath(t *testing.T) {
	// Trying to create database inside an unwriteable directory (e.g. dev/null as a directory)
	var out, errb bytes.Buffer
	code := run([]string{"--db", "/dev/null/impossible/brain.db", "status", "--project", "/test"}, &out, &errb)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestNexusBrain_OpenDatabaseFailsOnDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Pass directory itself as the db file path to force sqlite open failure
	var out, errb bytes.Buffer
	code := run([]string{"--db", tmpDir, "status", "--project", "/test"}, &out, &errb)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
}

func TestResolveDatabasePath_LocalNexusExists(t *testing.T) {
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(orig) }()

	_ = os.MkdirAll(".nexus", 0750)
	f, _ := os.Create(filepath.Join(".nexus", "brain.db"))
	_ = f.Close()

	if got := resolveDatabasePath(""); got != filepath.Join(".nexus", "brain.db") {
		t.Errorf("got %q, want .nexus/brain.db", got)
	}
}

func TestResolveDatabasePath_GlobalNexusExists(t *testing.T) {
	tmp := t.TempDir()
	origHome := userHomeDir
	userHomeDir = func() (string, error) { return tmp, nil }
	defer func() { userHomeDir = origHome }()

	orig, _ := os.Getwd()
	emptyDir := t.TempDir()
	_ = os.Chdir(emptyDir)
	defer func() { _ = os.Chdir(orig) }()

	_ = os.MkdirAll(filepath.Join(tmp, ".nexus"), 0750)
	f, _ := os.Create(filepath.Join(tmp, ".nexus", "orchestrator.db"))
	_ = f.Close()

	if got := resolveDatabasePath(""); got != filepath.Join(tmp, ".nexus", "orchestrator.db") {
		t.Errorf("got %q, want %s", got, filepath.Join(tmp, ".nexus", "orchestrator.db"))
	}
}

func TestResolveDatabasePath_FallbackToLocalWhenHomeFails(t *testing.T) {
	origHome := userHomeDir
	userHomeDir = func() (string, error) { return "", fmt.Errorf("no home") }
	defer func() { userHomeDir = origHome }()

	orig, _ := os.Getwd()
	emptyDir := t.TempDir()
	_ = os.Chdir(emptyDir)
	defer func() { _ = os.Chdir(orig) }()

	if got := resolveDatabasePath(""); got != filepath.Join(".nexus", "brain.db") {
		t.Errorf("got %q, want .nexus/brain.db", got)
	}
}
