package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// EffortResult is the JSON payload returned by /effort.
type EffortResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleEffortCommand handles /effort [low|medium|high|max|auto].
// Mirrors TS src/commands/effort/effort.tsx (local-jsx).
// In gou-demo, effort is set via CLAUDE_CODE_EFFORT_LEVEL env var.
func HandleEffortCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)

	if args == "" || args == "current" || args == "status" {
		current := strings.TrimSpace(os.Getenv("CLAUDE_CODE_EFFORT_LEVEL"))
		if current == "" {
			return json.Marshal(EffortResult{
				Type:  "text",
				Value: "Current effort level: auto (default)",
			})
		}
		desc := effortDescription(current)
		return json.Marshal(EffortResult{
			Type:  "text",
			Value: fmt.Sprintf("Current effort level: %s (%s)", current, desc),
		})
	}

	switch strings.ToLower(args) {
	case "auto", "unset":
		_ = os.Unsetenv("CLAUDE_CODE_EFFORT_LEVEL")
		return json.Marshal(EffortResult{
			Type:  "text",
			Value: "Effort level set to auto",
		})
	case "low":
		_ = os.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "low")
		return json.Marshal(EffortResult{
			Type:  "text",
			Value: "Set effort level to low: Quick, straightforward implementation",
		})
	case "medium":
		_ = os.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "medium")
		return json.Marshal(EffortResult{
			Type:  "text",
			Value: "Set effort level to medium: Balanced approach with standard testing",
		})
	case "high":
		_ = os.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "high")
		return json.Marshal(EffortResult{
			Type:  "text",
			Value: "Set effort level to high: Comprehensive implementation with extensive testing",
		})
	case "max":
		_ = os.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "max")
		return json.Marshal(EffortResult{
			Type:  "text",
			Value: "Set effort level to max: Maximum capability with deepest reasoning (Opus 4.6 only)",
		})
	case "help", "-h", "--help":
		return json.Marshal(EffortResult{
			Type:  "text",
			Value: "Usage: /effort [low|medium|high|max|auto]\n\nEffort levels:\n- low: Quick, straightforward implementation\n- medium: Balanced approach with standard testing\n- high: Comprehensive implementation with extensive testing\n- max: Maximum capability with deepest reasoning (Opus 4.6 only)\n- auto: Use the default effort level for your model",
		})
	default:
		return json.Marshal(EffortResult{
			Type:  "text",
			Value: fmt.Sprintf("Invalid argument: %s. Valid options are: low, medium, high, max, auto", args),
		})
	}
}

func effortDescription(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "low":
		return "Quick, straightforward implementation"
	case "medium":
		return "Balanced approach with standard testing"
	case "high":
		return "Comprehensive implementation with extensive testing"
	case "max":
		return "Maximum capability with deepest reasoning"
	default:
		return "Default effort level"
	}
}
