package query

import (
	"os"
	"strings"
)

// resolveGeminiModel mirrors TS resolveGeminiModel:
// GEMINI_MODEL > GEMINI_DEFAULT_{HAIKU|SONNET|OPUS}_MODEL > ANTHROPIC_DEFAULT_{family}_MODEL.
func resolveGeminiModel(anthropicModel string) (string, error) {
	if m := strings.TrimSpace(os.Getenv("GEMINI_MODEL")); m != "" {
		return m, nil
	}

	clean := strings.TrimSpace(anthropicModel)
	low := strings.ToLower(clean)

	var family string
	switch {
	case strings.Contains(low, "haiku"):
		family = "HAIKU"
	case strings.Contains(low, "opus"):
		family = "OPUS"
	case strings.Contains(low, "sonnet"):
		family = "SONNET"
	default:
		return clean, nil
	}

	if m := strings.TrimSpace(os.Getenv("GEMINI_DEFAULT_" + family + "_MODEL")); m != "" {
		return m, nil
	}
	if m := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_" + family + "_MODEL")); m != "" {
		return m, nil
	}

	// Sensible defaults
	switch family {
	case "HAIKU":
		return "gemini-2.0-flash", nil
	case "SONNET":
		return "gemini-2.5-flash", nil
	case "OPUS":
		return "gemini-2.5-pro", nil
	}
	return "gemini-2.5-flash", nil
}
