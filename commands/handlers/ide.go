package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// IDEResult is the JSON payload for /ide.
type IDEResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleIDECommand handles /ide.
func HandleIDECommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	connected := os.Getenv("CLAUDE_CODE_IDE_CONNECTED") == "1"

	if args == "" {
		status := "not connected"
		if connected {
			status = "connected"
		}
		return json.Marshal(IDEResult{
			Type:  "text",
			Value: fmt.Sprintf("IDE status: %s.\nUse /ide connect to auto-connect, or set --ide flag on launch.", status),
		})
	}

	return json.Marshal(IDEResult{
		Type: "text", Value: "Usage: /ide — shows IDE connection status.",
	})
}
