package query

import (
	"os"
	"strconv"
	"strings"

	goccontext "goc/context"
)

// TS parity: src/services/api/openai/index.ts [clampOpenAICompatibleMaxTokens] and request max_tokens wiring.
const (
	envOpenAIMaxOutputTokensCapTS = "CLAUDE_CODE_OPENAI_MAX_OUTPUT_TOKENS_CAP"
	// envOpenAIMaxOutputTokensCapGo is a Go-only legacy alias read only when the TS-named env is unset.
	envOpenAIMaxOutputTokensCapGo = "GOC_AUTOCOMPACT_OPENAI_MAX_COMPLETION_TOKENS"
	defaultOpenAICompatibleMaxCap = 8192
)

// openAICompatibleMaxTokensCapFromEnv returns the per-request max_tokens ceiling for OpenAI-compatible
// chat.completions (default 8192). Prefer CLAUDE_CODE_OPENAI_MAX_OUTPUT_TOKENS_CAP (TS); if unset,
// GOC_AUTOCOMPACT_OPENAI_MAX_COMPLETION_TOKENS is accepted for backward compatibility.
func openAICompatibleMaxTokensCapFromEnv() int {
	if raw := strings.TrimSpace(os.Getenv(envOpenAIMaxOutputTokensCapTS)); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			return n
		}
		return defaultOpenAICompatibleMaxCap
	}
	if raw := strings.TrimSpace(os.Getenv(envOpenAIMaxOutputTokensCapGo)); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 {
			return n
		}
	}
	return defaultOpenAICompatibleMaxCap
}

// ClampOpenAICompatibleMaxTokens mirrors TS [clampOpenAICompatibleMaxTokens]: max(1, trunc(request)) capped by env (default 8192).
func ClampOpenAICompatibleMaxTokens(maxTokens int) int {
	cap := openAICompatibleMaxTokensCapFromEnv()
	x := maxTokens
	if x < 1 {
		x = 1
	}
	if x > cap {
		return cap
	}
	return x
}

// openAIMaxTokensForChatCompletion mirrors TS queryModelOpenAI: requested = maxOutputTokensOverride
// ?? getMaxOutputTokensForModel(model), then [ClampOpenAICompatibleMaxTokens].
func openAIMaxTokensForChatCompletion(params QueryParams, anthropicModelID string) int {
	m := strings.TrimSpace(anthropicModelID)
	requested := goccontext.GetMaxOutputTokensForModel(m)
	if params.MaxOutputTokensOverride != nil {
		requested = *params.MaxOutputTokensOverride
	}
	return ClampOpenAICompatibleMaxTokens(requested)
}
