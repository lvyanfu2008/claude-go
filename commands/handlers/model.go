package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/modelenv"
)

// ModelResult is the JSON payload returned by /model.
type ModelResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleModelCommand handles /model [modelName].
// Mirrors TS src/commands/model/ (local-jsx -> ModelPicker).
// Sets CLAUDE_CODE_MODEL, which [modelenv.LookupKeys] reads first so the next turn matches this display.
func HandleModelCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)

	if args == "" {
		current := modelenv.EffectiveMainLoopModel()
		return json.Marshal(ModelResult{
			Type:  "text",
			Value: fmt.Sprintf("Current model: %s\nRun /model [modelName] to set a different model, or /model --help for usage.", current),
		})
	}

	switch strings.ToLower(args) {
	case "help", "-h", "--help":
		return json.Marshal(ModelResult{
			Type:  "text",
			Value: fmt.Sprintf("Usage: /model [modelName]\n\nSet the AI model for Claude Code (sets CLAUDE_CODE_MODEL).\nExamples:\n  /model %s\n  /model claude-opus-4-20250514\n  /model default    (unset CLAUDE_CODE_MODEL; fall back to other env or %s)",
				modelenv.DefaultMainLoopModelID, modelenv.DefaultMainLoopModelID),
		})
	case "default":
		_ = os.Unsetenv("CLAUDE_CODE_MODEL")
		return json.Marshal(ModelResult{
			Type:  "text",
			Value: fmt.Sprintf("Unset CLAUDE_CODE_MODEL. Effective model: %s", modelenv.EffectiveMainLoopModel()),
		})
	default:
		_ = os.Setenv("CLAUDE_CODE_MODEL", args)
		return json.Marshal(ModelResult{
			Type:  "text",
			Value: fmt.Sprintf("Set model to %s", args),
		})
	}
}
