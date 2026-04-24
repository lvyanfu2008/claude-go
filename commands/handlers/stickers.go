package handlers

import (
	"encoding/json"
)

// StickersResult is the JSON payload returned by /stickers.
type StickersResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleStickersCommand returns info about ordering stickers for /stickers.
// Mirrors TS src/commands/stickers/ (local -> opens browser).
// gou-demo shows the URL without opening a browser.
func HandleStickersCommand() ([]byte, error) {
	msg := StickersResult{
		Type:  "text",
		Value: "Order Claude Code stickers at:\n  https://www.stickermule.com/claudecode\n\n(Open this URL in your browser to order.)",
	}
	return json.Marshal(msg)
}
