package query

import (
	"os"
	"strings"
)

// providerType returns the current OpenAI-compatible provider variant.
func providerType() string {
	if envTruthy("CLAUDE_CODE_USE_GROK") {
		return "grok"
	}
	if envTruthy("CLAUDE_CODE_USE_GEMINI") {
		return "gemini"
	}
	if UseOpenAIChatProvider() {
		return "openai"
	}
	return "anthropic"
}

// UseGrokProvider returns true when the Grok provider is active.
func UseGrokProvider() bool {
	return envTruthy("CLAUDE_CODE_USE_GROK")
}

// grokAPIKey returns the Grok API key from standard env vars.
func grokAPIKey() string {
	if k := strings.TrimSpace(os.Getenv("GROK_API_KEY")); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("XAI_API_KEY"))
}

// grokBaseURL returns the Grok API base URL.
func grokBaseURL() string {
	if b := strings.TrimSpace(os.Getenv("GROK_BASE_URL")); b != "" {
		return strings.TrimSuffix(b, "/")
	}
	return "https://api.x.ai/v1"
}

// resolveGrokModel maps Anthropic-style model IDs to Grok model IDs.
// When the input already looks like a Grok model name, pass it through.
func resolveGrokModel(model string) string {
	clean := strings.TrimSpace(model)
	low := strings.ToLower(clean)

	if strings.Contains(low, "grok") {
		return clean
	}

	switch {
	case strings.Contains(low, "haiku"):
		return "grok-4.1-mini"
	case strings.Contains(low, "opus"):
		return "grok-4.1"
	case strings.Contains(low, "sonnet"):
		return "grok-4.1"
	default:
		return "grok-4.1"
	}
}
