package anthropicmessages

import (
	"encoding/json"
	"os"
	"strings"
)

const toolSearchToolName = "ToolSearch"

const (
	toolSearchBeta1P = "advanced-tool-use-2025-11-20"
	toolSearchBeta3P = "tool-search-tool-2025-10-19"
)

func envTruthyBetas(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// AnthropicBetaHeaderForToolSearch returns the anthropic-beta token when tools[] still includes ToolSearch
// (dynamic tool loading). Mirrors [goc/internal/toolsearch.AnthropicBetaHeader] without importing toolsearch.
func AnthropicBetaHeaderForToolSearch() string {
	base := strings.ToLower(strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")))
	if strings.Contains(base, "vertex") || strings.Contains(base, "bedrock") {
		return toolSearchBeta3P
	}
	return toolSearchBeta1P
}

// BetasForToolsJSON scans a tools[] JSON array for name "ToolSearch" and returns anthropic-beta header values.
// When CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS is truthy, returns nil (matches toolsearch.BetasForWiredTools gate).
func BetasForToolsJSON(toolsJSON []byte) []string {
	if envTruthyBetas("CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS") {
		return nil
	}
	s := strings.TrimSpace(string(toolsJSON))
	if s == "" || s == "null" {
		return nil
	}
	var tools []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(toolsJSON, &tools); err != nil {
		return nil
	}
	for _, t := range tools {
		if strings.TrimSpace(t.Name) == toolSearchToolName {
			return []string{AnthropicBetaHeaderForToolSearch()}
		}
	}
	return nil
}
