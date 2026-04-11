package toolsearch

import (
	"os"
	"strings"
)

const (
	toolSearchBeta1P = "advanced-tool-use-2025-11-20"
	toolSearchBeta3P = "tool-search-tool-2025-10-19"
)

// ModelSupportsToolReference mirrors TS modelSupportsToolReference (default blocks haiku).
func modelSupportsToolReference(modelID string) bool {
	m := strings.ToLower(strings.TrimSpace(modelID))
	if m == "" {
		return true
	}
	return !strings.Contains(m, "haiku")
}

// AnthropicBetaHeader returns the anthropic-beta header value for tool search (1P vs 3P heuristic).
func AnthropicBetaHeader() string {
	base := strings.ToLower(strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")))
	if strings.Contains(base, "vertex") || strings.Contains(base, "bedrock") {
		return toolSearchBeta3P
	}
	return toolSearchBeta1P
}
