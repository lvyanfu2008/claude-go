package handlers

import (
	"encoding/json"
)

// ContextResult is the JSON payload returned by /context.
type ContextResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleContextCommand returns a simple context summary for /context.
// In gou-demo the full context visualization (local-jsx) is not available,
// so this returns a basic text summary pointing to the store-based variant
// wired in the TUI.
func HandleContextCommand() ([]byte, error) {
	msg := ContextResult{
		Type:  "text",
		Value: "Context usage is session-scoped. The TUI shows message count in the status bar.",
	}
	return json.Marshal(msg)
}
