// Package main is the entry point for the nexus CLI binary.
// It connects to a running NexusAI daemon via the HTTP API.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"nexus-ai/internal/adapters/inbound/cli"
	"nexus-ai/internal/core/domain"
)

func main() {
	// Use a lightweight HTTP-backed orchestrator stub that talks to the daemon.
	orch := &remoteOrchestrator{baseURL: "http://127.0.0.1:9999"}

	root := cli.NewRootCmd(orch)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// remoteOrchestrator forwards calls to the running NexusAI HTTP API.
type remoteOrchestrator struct{ baseURL string }

func (r *remoteOrchestrator) SubmitTask(task domain.Task) (string, error) {
	body, _ := json.Marshal(task)
	resp, err := http.Post(r.baseURL+"/api/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("remote: submit task: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("remote: decode response: %w", err)
	}
	return result["task_id"], nil
}

func (r *remoteOrchestrator) GetQueue() ([]domain.Task, error) {
	resp, err := http.Get(r.baseURL + "/api/tasks")
	if err != nil {
		return nil, fmt.Errorf("remote: get queue: %w", err)
	}
	defer resp.Body.Close()

	var tasks []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("remote: decode queue: %w", err)
	}
	return tasks, nil
}

func (r *remoteOrchestrator) CancelTask(id string) error {
	req, err := http.NewRequest(http.MethodDelete, r.baseURL+"/api/tasks/"+id, nil)
	if err != nil {
		return fmt.Errorf("remote: build cancel request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: cancel task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: cancel task: status %d", resp.StatusCode)
	}
	return nil
}
