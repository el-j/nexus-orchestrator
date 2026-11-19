package services

import (
	"strings"
	"testing"
	"time"

	"nexus-orchestrator/internal/core/domain"
)

func TestPromptCaching_AgentSystemPromptIsByteIdenticalAcrossTasks(t *testing.T) {
	prompt1 := agentSystemPrompt(true)
	prompt2 := agentSystemPrompt(true)

	if prompt1 != prompt2 {
		t.Fatalf("agentSystemPrompt must be byte-identical across calls for prompt caching: %q vs %q", prompt1, prompt2)
	}

	promptNoVerify := agentSystemPrompt(false)
	if strings.Contains(promptNoVerify, "verify {}") {
		t.Fatalf("promptNoVerify should not contain verify tool: %s", promptNoVerify)
	}
	if !strings.Contains(prompt1, "verify {}") {
		t.Fatalf("prompt1 should contain verify tool: %s", prompt1)
	}

	// Ensure system prompt does not contain dynamic task details
	if strings.Contains(prompt1, "Primary target file:") {
		t.Fatalf("system prompt should not embed dynamic target file: %s", prompt1)
	}
}

func TestPromptCaching_DelegationInstructionHasPrefixStability(t *testing.T) {
	orch := &OrchestratorService{daemonAddr: "http://127.0.0.1:63987"}

	now1 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now2 := time.Date(2026, 6, 1, 15, 30, 0, 0, time.UTC)

	sess1 := domain.AISession{ID: "sess-aaa", ProjectPath: "/proj/a"}
	sess2 := domain.AISession{ID: "sess-bbb", ProjectPath: "/proj/b"}

	inst1 := orch.delegationInstruction(sess1, now1)
	inst2 := orch.delegationInstruction(sess2, now2)

	// Both instructions must share an identical workflow prefix up to "Session details:"
	splitToken := "Session details:"
	parts1 := strings.Split(inst1, splitToken)
	parts2 := strings.Split(inst2, splitToken)

	if len(parts1) != 2 || len(parts2) != 2 {
		t.Fatalf("expected instruction to be partitioned by %q", splitToken)
	}

	if parts1[0] != parts2[0] {
		t.Fatalf("workflow instructions prefix must be byte-identical for caching:\nPrefix 1:\n%s\nPrefix 2:\n%s", parts1[0], parts2[0])
	}
}
