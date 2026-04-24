package query

import (
	"os"
	"strings"
)

// defaultOpenAIModelMap mirrors src/api-client/openai/modelMapping.ts DEFAULT_MODEL_MAP.
var defaultOpenAIModelMap = map[string]string{
	"claude-sonnet-4-20250514":   "gpt-4o",
	"claude-sonnet-4-5-20250929": "gpt-4o",
	"claude-sonnet-4-6":          "gpt-4o",
	"claude-opus-4-20250514":     "o3",
	"claude-opus-4-1-20250805":   "o3",
	"claude-opus-4-5-20251101":   "o3",
	"claude-opus-4-6":            "o3",
	"claude-haiku-4-5-20251001":  "gpt-4o-mini",
	"claude-3-5-haiku-20241022":  "gpt-4o-mini",
	"claude-3-7-sonnet-20250219": "gpt-4o",
	"claude-3-5-sonnet-20241022": "gpt-4o",
}

func openaiModelFamilyUpper(model string) (string, bool) {
	low := strings.ToLower(model)
	switch {
	case strings.Contains(low, "haiku"):
		return "HAIKU", true
	case strings.Contains(low, "opus"):
		return "OPUS", true
	case strings.Contains(low, "sonnet"):
		return "SONNET", true
	default:
		return "", false
	}
}

// ResolveOpenAIModel mirrors src/api-client/openai/modelMapping.ts resolveOpenAIModel
// except the env override: Go uses CCB_ENGINE_MODEL (same as [goc/modelenv] first key) so the
// wire model id and main-loop / TUI model stay one value.
func ResolveOpenAIModel(anthropicModel string) string {
	if v := strings.TrimSpace(os.Getenv("CCB_ENGINE_MODEL")); v != "" {
		return v
	}
	clean := strings.TrimSuffix(strings.TrimSpace(anthropicModel), "[1m]")
	if family, ok := openaiModelFamilyUpper(clean); ok {
		key := "ANTHROPIC_DEFAULT_" + family + "_MODEL"
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	if m, ok := defaultOpenAIModelMap[clean]; ok {
		return m
	}
	if clean == "" {
		return "gpt-4o"
	}
	return clean
}
