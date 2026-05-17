package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// RenameResult is the JSON payload for /rename.
type RenameResult struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// HandleRenameCommand handles /rename [new-name].
func HandleRenameCommand(args string) ([]byte, error) {
	args = strings.TrimSpace(args)
	sessionID := os.Getenv("CLAUDE_SESSION_ID")

	if args == "" {
		return json.Marshal(RenameResult{
			Type: "text",
			Value: fmt.Sprintf("Current session ID: %s\nUse /rename [new-name] to rename this session.", sessionID),
		})
	}

	os.Setenv("CLAUDE_SESSION_NAME", args)
	return json.Marshal(RenameResult{
		Type:  "text",
		Value: fmt.Sprintf("Session renamed to: %s\n(Session ID: %s)", args, sessionID),
	})
}
