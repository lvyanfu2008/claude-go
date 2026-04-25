package query

import (
	"encoding/json"
	"os"
	"testing"

	"goc/types"
)

func TestOpenAIEnforcesReasoningInThinkingMode_DepSeekV4Pro(t *testing.T) {
	prev := os.Getenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING")
		} else {
			_ = os.Setenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING", prev)
		}
	})
	_ = os.Setenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING", "1")
	if !OpenAIEnforcesReasoningInThinkingMode("DeepSeek-V4-Pro", true) {
		t.Fatal("expected enforce for v4-pro + thinking on")
	}
}

func TestOpenAIEnforcesReasoningInThinkingMode_Disabled(t *testing.T) {
	prev := os.Getenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING")
		} else {
			_ = os.Setenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING", prev)
		}
	})
	_ = os.Setenv("CLAUDE_CODE_DEEPSEEK_STRICT_THINKING", "0")
	if OpenAIEnforcesReasoningInThinkingMode("deepseek-v4-pro", true) {
		t.Fatal("expected off when CLAUDE_CODE_DEEPSEEK_STRICT_THINKING=0")
	}
}

func TestAssistantWireMessageHasNonEmptyThinkingBlock(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"role": "assistant",
		"content": []any{
			map[string]any{"type": "thinking", "thinking": "abc"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !assistantWireMessageHasNonEmptyThinkingBlock(raw) {
		t.Fatal("expected true for non-empty thinking")
	}
}

func TestOpenAIEnableThinkingForRequest_Transcript(t *testing.T) {
	prev := os.Getenv("OPENAI_ENABLE_THINKING")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("OPENAI_ENABLE_THINKING")
		} else {
			_ = os.Setenv("OPENAI_ENABLE_THINKING", prev)
		}
	})
	_ = os.Unsetenv("OPENAI_ENABLE_THINKING")
	inner, err := json.Marshal(map[string]any{
		"role":    "assistant",
		"content": []any{map[string]any{"type": "thinking", "thinking": "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	work := []types.Message{{
		Type:    types.MessageTypeAssistant,
		Message: inner,
	}}
	// gpt-4o is not a thinking model name, but prior assistant row has thinking → enable echo path
	if !openAIEnableThinkingForRequest("gpt-4o", work) {
		t.Fatal("expected enableThinking when transcript has thinking")
	}
}
