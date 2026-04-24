package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ModelResult is the JSON payload returned by /model.
type ModelResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleModelCommand handles /model [modelName].
// Mirrors TS src/commands/model/ (local-jsx -> ModelPicker).
// In gou-demo, model is set via CLAUDE_CODE_MODEL env var.
func HandleModelCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)

	if args == "" {
		current := os.Getenv("CLAUDE_CODE_MODEL")
		if current == "" {
			return json.Marshal(ModelResult{
				Type:  "text",
				Value: "Current model: default (claude-sonnet-4-6)\nRun /model [modelName] to set a different model, or /model --help for usage.",
			})
		}
		return json.Marshal(ModelResult{
			Type:  "text",
			Value: fmt.Sprintf("Current model: %s", current),
		})
	}

	switch strings.ToLower(args) {
	case "help", "-h", "--help":
		return json.Marshal(ModelResult{
			Type:  "text",
			Value: "Usage: /model [modelName]\n\nSet the AI model for Claude Code.\nExamples:\n  /model claude-sonnet-4-6\n  /model claude-opus-4-6\n  /model default    (reset to default)",
		})
	case "default":
		_ = os.Unsetenv("CLAUDE_CODE_MODEL")
		return json.Marshal(ModelResult{
			Type:  "text",
			Value: "Set model to claude-sonnet-4-6 (default)",
		})
	default:
		_ = os.Setenv("CLAUDE_CODE_MODEL", args)
		return json.Marshal(ModelResult{
			Type:  "text",
			Value: fmt.Sprintf("Set model to %s", args),
		})
	}
}
