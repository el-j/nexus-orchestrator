//go:build !windows

package main

import (
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestRunMain_ReturnsZeroAfterASignalStopsTheDaemon(t *testing.T) {
	isolate(t)
	listen := freeAddr(t)
	t.Setenv("NEXUS_DB_PATH", filepath.Join(t.TempDir(), "d.db"))
	t.Setenv("NEXUS_LISTEN_ADDR", listen)
	t.Setenv("NEXUS_MCP_ADDR", freeAddr(t))
	done := make(chan int, 1)
	go func() { done <- runMain() }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if resp, err := http.Get("http://" + listen + "/api/health"); err == nil {
			_ = resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the daemon never answered")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code %d, want 0 after a signal", code)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("the daemon did not stop after SIGTERM")
	}
}
