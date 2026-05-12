package handlers

import (
	"encoding/json"
	"fmt"
	"os"
)

// CompactResult is the JSON payload returned by /compact.
type CompactResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleCompactCommand handles the /compact local command.
// In the TUI, compaction is triggered via the conversation runtime.
// This handler provides info and a trigger hint for headless/non-TUI contexts.
func HandleCompactCommand() ([]byte, error) {
	lines := []string{
		"Compact (gou-demo)",
		"──────────────────",
	}

	// Check if auto-compact is enabled
	if os.Getenv("CLAUDE_CODE_AUTOCOMPACT_AFTER_TOKENS") != "" {
		lines = append(lines, fmt.Sprintf("  Auto-compact:  enabled (threshold: %s tokens)", os.Getenv("CLAUDE_CODE_AUTOCOMPACT_AFTER_TOKENS")))
	} else {
		lines = append(lines, "  Auto-compact:  enabled (default thresholds)")
	}

	if os.Getenv("CLAUDE_CODE_NO_SESSION_PERSISTENCE") != "" {
		lines = append(lines, "  Note:          Session persistence is off — compaction won't be saved")
	}

	lines = append(lines, "")
	lines = append(lines, "In the TUI, compaction runs automatically when the context window nears its limit.")
	lines = append(lines, "Use the TS CLI for manual compaction with optional custom summarization instructions.")

	msg := CompactResult{
		Type:  "text",
		Value: joinLines(lines),
	}
	return json.Marshal(msg)
}
