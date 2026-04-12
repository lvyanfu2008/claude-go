package query

import (
	"testing"
)

func TestResolveOpenAIModel_envOverride(t *testing.T) {
	t.Setenv("OPENAI_MODEL", "deepseek-chat")
	if got := ResolveOpenAIModel("claude-opus-4-20250514"); got != "deepseek-chat" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveOpenAIModel_mapDefault(t *testing.T) {
	t.Setenv("OPENAI_MODEL", "")
	if got := ResolveOpenAIModel("claude-sonnet-4-20250514"); got != "gpt-4o" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveOpenAIModel_passThrough(t *testing.T) {
	t.Setenv("OPENAI_MODEL", "")
	if got := ResolveOpenAIModel("custom-model-id"); got != "custom-model-id" {
		t.Fatalf("got %q", got)
	}
}
