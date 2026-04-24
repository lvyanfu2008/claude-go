package query

import (
	"testing"
)

func TestResolveOpenAIModel(t *testing.T) {
	t.Run("CCB_ENGINE_MODEL wins over mapping", func(t *testing.T) {
		t.Setenv("CCB_ENGINE_MODEL", "deepseek-v4-pro")
		t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "should-not-use")
		got := ResolveOpenAIModel("claude-sonnet-4-20250514")
		if got != "deepseek-v4-pro" {
			t.Fatalf("got %q want deepseek-v4-pro", got)
		}
	})
	t.Run("default map when CCB unset", func(t *testing.T) {
		t.Setenv("CCB_ENGINE_MODEL", "")
		got := ResolveOpenAIModel("claude-sonnet-4-20250514")
		if got != "gpt-4o" {
			t.Fatalf("got %q want gpt-4o", got)
		}
	})
}
