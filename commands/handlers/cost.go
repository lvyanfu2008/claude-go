package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"goc/modelenv"
)

// CostResult is the JSON payload returned by /cost.
type CostResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleCostCommand returns session token usage for the /cost local command.
func HandleCostCommand() ([]byte, error) {
	model := modelenv.EffectiveMainLoopModel()

	lines := []string{
		"Token Usage (gou-demo)",
		"──────────────────────",
		fmt.Sprintf("  Model:       %s", model),
	}

	// Per-model approximate pricing (per 1M tokens)
	pricing := map[string]string{
		"claude-sonnet-4-6":  "$3 / $15",
		"claude-opus-4-7":    "$15 / $75",
		"claude-haiku-4-5":   "$0.80 / $4",
		"claude-sonnet-4-0":  "$3 / $15",
		"claude-opus-4-0":    "$15 / $75",
	}
	for key, price := range pricing {
		if strings.Contains(strings.ToLower(model), key) {
			lines = append(lines, fmt.Sprintf("  Pricing:     %s (input/output per 1M tokens)", price))
			break
		}
	}

	// Token budget from env
	if budget := os.Getenv("CLAUDE_CODE_TOKEN_BUDGET"); budget != "" {
		lines = append(lines, fmt.Sprintf("  Token budget: %s", budget))
	}

	// Max turns
	if mt := os.Getenv("CLAUDE_CODE_MAX_TURNS"); mt != "" {
		lines = append(lines, fmt.Sprintf("  Max turns:   %s", mt))
	}

	lines = append(lines, "")
	lines = append(lines, "Full per-session cost tracking is available in the TS CLI with an active session.")

	msg := CostResult{
		Type:  "text",
		Value: joinLines(lines),
	}
	return json.Marshal(msg)
}
