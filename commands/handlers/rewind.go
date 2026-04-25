package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RewindResult is the JSON payload returned by /rewind.
type RewindResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleRewindCommand returns a notice about checkpoint restore for /rewind.
// Mirrors TS src/commands/rewind/ (local-jsx with interactive checkpoint picker).
// gou-demo does not support checkpoint restore.
func HandleRewindCommand(args string) ([]byte, error) {
	extra := ""
	if a := strings.TrimSpace(args); a != "" {
		extra = fmt.Sprintf(" (args: %s)", a)
	}
	msg := RewindResult{
		Type: "text",
		Value: fmt.Sprintf("Checkpoint restore (/rewind) is not available in gou-demo.%s\n\n"+
			"Use the TS CLI to restore a checkpoint:\n"+
			"  claude /rewind              — Interactive checkpoint picker\n"+
			"  claude /rewind <id>         — Restore to a specific checkpoint", extra),
	}
	return json.Marshal(msg)
}
