package handlers

import (
	"encoding/json"
)

// ClearResult is the JSON payload for /clear.
type ClearResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleClearCommand handles /clear.
func HandleClearCommand(args string) ([]byte, error) {
	return json.Marshal(ClearResult{
		Type: "text",
		Value: "To clear the current conversation:\n" +
			"  1. Use /compact to compress the conversation context\n" +
			"  2. Or exit and start a new session with: claude\n" +
			"  3. Or use Ctrl+L to clear the screen display",
	})
}
