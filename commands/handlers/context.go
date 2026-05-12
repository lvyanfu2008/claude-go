package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"goc/modelenv"
)

// ContextResult is the JSON payload returned by /context.
type ContextResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleContextCommand returns a context window usage summary for /context.
func HandleContextCommand() ([]byte, error) {
	model := modelenv.EffectiveMainLoopModel()

	lines := []string{
		"Context Window (gou-demo)",
		"─────────────────────────",
		fmt.Sprintf("  Model:             %s", model),
	}

	// Context window sizes by model family
	ctxWindows := map[string]int{
		"claude-sonnet-4-6": 200000,
		"claude-opus-4-7":   200000,
		"claude-haiku-4-5":  200000,
		"claude-sonnet-4-0": 200000,
		"claude-opus-4-0":   200000,
	}
	for key, size := range ctxWindows {
		if strings.Contains(strings.ToLower(model), key) {
			lines = append(lines, fmt.Sprintf("  Context window:    %d tokens", size))
			break
		}
	}

	// Max turns
	if mt := os.Getenv("CLAUDE_CODE_MAX_TURNS"); mt != "" {
		lines = append(lines, fmt.Sprintf("  Max turns:         %s", mt))
	}

	// Token budget
	if budget := os.Getenv("CLAUDE_CODE_TOKEN_BUDGET"); budget != "" {
		lines = append(lines, fmt.Sprintf("  Token budget:      %s", budget))
	}

	// Thinking mode
	if thinking := os.Getenv("CLAUDE_CODE_THINKING"); thinking != "" {
		lines = append(lines, fmt.Sprintf("  Thinking:          %s", thinking))
	}

	// Effort level
	if effort := os.Getenv("CLAUDE_CODE_EFFORT_LEVEL"); effort != "" {
		lines = append(lines, fmt.Sprintf("  Effort:            %s", effort))
	}

	// Output format
	outputFmt := os.Getenv("CLAUDE_CODE_OUTPUT_FORMAT")
	if outputFmt == "" {
		outputFmt = "text"
	}
	lines = append(lines, fmt.Sprintf("  Output format:     %s", outputFmt))

	// Session persistence
	if os.Getenv("CLAUDE_CODE_NO_SESSION_PERSISTENCE") != "" {
		lines = append(lines, "  Session persist:   off")
	} else {
		lines = append(lines, "  Session persist:   on")
	}

	// Auto-compact threshold from env
	if acTokens := os.Getenv("CLAUDE_CODE_AUTOCOMPACT_AFTER_TOKENS"); acTokens != "" {
		if n, err := strconv.Atoi(acTokens); err == nil && n > 0 {
			lines = append(lines, fmt.Sprintf("  Auto-compact:      after ~%d tokens", n))
		}
	} else {
		lines = append(lines, "  Auto-compact:      enabled (default thresholds)")
	}

	lines = append(lines, "")
	lines = append(lines, "Per-message context tracking requires an active session in the TUI.")

	msg := ContextResult{
		Type:  "text",
		Value: joinLines(lines),
	}
	return json.Marshal(msg)
}
