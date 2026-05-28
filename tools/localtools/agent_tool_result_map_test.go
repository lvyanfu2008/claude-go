package localtools

import (
	"strings"
	"testing"
)

func TestMapAgentToolResultToAssistantTextExplore(t *testing.T) {
	body := `{"data":{"output":"some explore result","agent_id":"abc123","agent_type":"Explore"}}`
	mapped, err := MapAgentToolResultToAssistantText(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mapped != "some explore result" {
		t.Fatalf("expected output without trailer, got: %q", mapped)
	}
}

func TestMapAgentToolResultToAssistantTextPlan(t *testing.T) {
	body := `{"data":{"output":"a plan output","agent_id":"def456","agent_type":"Plan"}}`
	mapped, err := MapAgentToolResultToAssistantText(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mapped != "a plan output" {
		t.Fatalf("expected output without trailer, got: %q", mapped)
	}
}

func TestMapAgentToolResultToAssistantTextGeneralAgent(t *testing.T) {
	body := `{"data":{"output":"general agent result","agent_id":"xyz999","agent_type":"general-purpose"}}`
	mapped, err := MapAgentToolResultToAssistantText(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(mapped, "agentId: xyz999") {
		t.Fatalf("expected agentId trailer, got: %q", mapped)
	}
	if !strings.Contains(mapped, "general agent result") {
		t.Fatalf("expected original output, got: %q", mapped)
	}
}

func TestMapAgentToolResultToAssistantTextEmpty(t *testing.T) {
	_, err := MapAgentToolResultToAssistantText("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestMapAgentToolResultToAssistantTextEmptyOutput(t *testing.T) {
	body := `{"data":{"output":"","agent_id":"abc","agent_type":"Explore"}}`
	mapped, err := MapAgentToolResultToAssistantText(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mapped != "" {
		t.Fatalf("expected empty output, got: %q", mapped)
	}
}

func TestMapAgentToolResultToAssistantTextExploreWithWorktree(t *testing.T) {
	body := `{"data":{"output":"explore result","agent_id":"abc123","agent_type":"Explore","worktree_path":"/tmp/wt"}}`
	mapped, err := MapAgentToolResultToAssistantText(body)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(mapped, "agentId: abc123") {
		t.Fatalf("expected agentId trailer for Explore with worktree, got: %q", mapped)
	}
}
