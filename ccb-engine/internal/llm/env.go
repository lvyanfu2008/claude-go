package llm

import (
	"os"
	"strings"

	"goc/ccb-engine/internal/anthropic"
)

// UseOpenAICompat returns true when the engine should call OpenAI-style /chat/completions (DeepSeek, etc.).
func UseOpenAICompat() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("CCB_ENGINE_LLM"))) {
	case "openai", "openai-compat", "deepseek":
		return true
	}
	if os.Getenv("CLAUDE_CODE_USE_OPENAI") == "1" {
		return true
	}
	// Heuristic: DeepSeek host is always OpenAI-compatible, not Anthropic /v1/messages.
	if strings.Contains(strings.ToLower(os.Getenv("ANTHROPIC_BASE_URL")), "deepseek") {
		return true
	}
	return false
}

// NewFromEnv returns a TurnCompleter: OpenAI-compatible or Anthropic Messages.
func NewFromEnv() TurnCompleter {
	if UseOpenAICompat() {
		return newOpenAICompatFromEnv()
	}
	return &AnthropicAdapter{Client: anthropic.NewClient()}
}
