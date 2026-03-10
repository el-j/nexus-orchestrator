// Package main is the entry point for the nexus CLI binary.
// It connects to a running NexusAI daemon via the HTTP API.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"nexus-orchestrator/internal/adapters/inbound/cli"
	"nexus-orchestrator/internal/core/domain"
	"nexus-orchestrator/internal/core/ports"
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
	body, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("remote: marshal task: %w", err)
	}
	resp, err := http.Post(r.baseURL+"/api/tasks", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("remote: submit task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("remote: submit task: unexpected status %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("remote: decode response: %w", err)
	}
	return result["task_id"], nil
}

func (r *remoteOrchestrator) GetTask(id string) (domain.Task, error) {
	resp, err := http.Get(r.baseURL + "/api/tasks/" + url.PathEscape(id))
	if err != nil {
		return domain.Task{}, fmt.Errorf("remote: get task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return domain.Task{}, fmt.Errorf("remote: get task: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return domain.Task{}, fmt.Errorf("remote: get task: unexpected status %d", resp.StatusCode)
	}

	var task domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return domain.Task{}, fmt.Errorf("remote: decode task: %w", err)
	}
	return task, nil
}

func (r *remoteOrchestrator) GetQueue() ([]domain.Task, error) {
	resp, err := http.Get(r.baseURL + "/api/tasks")
	if err != nil {
		return nil, fmt.Errorf("remote: get queue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get queue: unexpected status %d", resp.StatusCode)
	}

	var tasks []domain.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("remote: decode queue: %w", err)
	}
	return tasks, nil
}

func (r *remoteOrchestrator) GetProviders() ([]ports.ProviderInfo, error) {
	resp, err := http.Get(r.baseURL + "/api/providers")
	if err != nil {
		return nil, fmt.Errorf("remote: get providers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get providers: unexpected status %d", resp.StatusCode)
	}

	var providers []ports.ProviderInfo
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return nil, fmt.Errorf("remote: decode providers: %w", err)
	}
	return providers, nil
}

func (r *remoteOrchestrator) CancelTask(id string) error {
	req, err := http.NewRequest(http.MethodDelete, r.baseURL+"/api/tasks/"+url.PathEscape(id), nil)
	if err != nil {
		return fmt.Errorf("remote: build cancel request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: cancel task: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: cancel task: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) RegisterCloudProvider(cfg domain.ProviderConfig) error {
	body, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("remote: marshal provider config: %w", err)
	}
	resp, err := http.Post(r.baseURL+"/api/providers", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("remote: register provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("remote: register provider: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) RemoveProvider(name string) error {
	req, err := http.NewRequest(http.MethodDelete, r.baseURL+"/api/providers/"+url.PathEscape(name), nil)
	if err != nil {
		return fmt.Errorf("remote: build remove-provider request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: remove provider: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("remote: remove provider: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("remote: remove provider: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (r *remoteOrchestrator) GetProviderModels(name string) ([]string, error) {
	resp, err := http.Get(r.baseURL + "/api/providers/" + url.PathEscape(name) + "/models")
	if err != nil {
		return nil, fmt.Errorf("remote: get provider models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("remote: get provider models: %w", domain.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote: get provider models: unexpected status %d", resp.StatusCode)
	}
	var models []string
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("remote: decode models: %w", err)
	}
	return models, nil
}
